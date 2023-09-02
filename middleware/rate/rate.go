package rate

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/philippgille/gokv"
)

type rateMiddleware struct {
	store   gokv.Store
	limiter *rate.Limiter
}

func (m *rateMiddleware) Set(k string, v interface{}) error {
	m.limiter.Wait(context.Background())

	return m.store.Set(k, v)
}

func (m *rateMiddleware) Get(k string, v interface{}) (found bool, err error) {
	m.limiter.Wait(context.Background())

	return m.store.Get(k, v)
}

func (m *rateMiddleware) Delete(k string) error {
	m.limiter.Wait(context.Background())

	return m.store.Delete(k)
}

func (m *rateMiddleware) Close() error {
	return m.store.Close()
}

type conf struct {
	limit rate.Limit
	burst int
}

func (c *conf) SetDefaults() {
	c.limit = rate.Inf
	c.burst = 10
}

type Option func(*conf)

func WithRateLimit(limit rate.Limit) Option {
	return func(c *conf) {
		c.limit = limit
	}
}

func WithRateBurst(burst int) Option {
	return func(c *conf) {
		c.burst = burst
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	limiter := rate.NewLimiter(c.limit, c.burst)

	return &rateMiddleware{
		store:   inner,
		limiter: limiter,
	}
}
