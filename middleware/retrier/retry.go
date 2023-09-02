package retrier

import (
	"time"

	"github.com/eapache/go-resiliency/retrier"

	"github.com/philippgille/gokv"
)

type retrierMiddleware struct {
	store   gokv.Store
	retrier *retrier.Retrier
}

func (m *retrierMiddleware) Set(k string, v interface{}) error {
	return m.retrier.Run(func() error {
		return m.store.Set(k, v)
	})
}

func (m *retrierMiddleware) Get(k string, v interface{}) (found bool, err error) {
	rerr := m.retrier.Run(func() error {
		found, err = m.store.Get(k, v)

		return err
	})

	return found, rerr
}

func (m *retrierMiddleware) Delete(k string) error {
	return m.retrier.Run(func() error {
		return m.store.Delete(k)
	})
}

func (m *retrierMiddleware) Close() error {
	return m.store.Close()
}

type conf struct {
	backoff    []time.Duration
	classifier retrier.Classifier
}

const (
	defaultN             = 3
	defaultInitialAmount = 200 * time.Millisecond
)

func (c *conf) SetDefaults() {
	c.backoff = retrier.ExponentialBackoff(defaultN, defaultInitialAmount)
	c.classifier = retrier.DefaultClassifier{}
}

type Option func(*conf)

func WithBackoff(backoff []time.Duration) Option {
	return func(c *conf) {
		c.backoff = backoff
	}
}

func WithClassifier(classifier retrier.Classifier) Option {
	return func(c *conf) {
		c.classifier = classifier
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	return &retrierMiddleware{
		store:   inner,
		retrier: retrier.New(c.backoff, c.classifier),
	}
}
