package expvar

import (
	"expvar"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"
)

var _ middleware.Adder = (*expvar.Int)(nil)

var (
	defaultGetHitAdder  = expvar.NewInt("gokv_get_hit")
	defaultGetMissAdder = expvar.NewInt("gokv_get_miss")
)

type conf struct {
	getHitAdder  middleware.Adder
	getMissAdder middleware.Adder
}

func (c *conf) SetDefaults() {
	c.getHitAdder = defaultGetHitAdder
	c.getMissAdder = defaultGetMissAdder
}

type Option func(*conf)

func WithGetHitExpvar(adder middleware.Adder) Option {
	return func(c *conf) {
		c.getHitAdder = adder
	}
}

func WithGetMissExpvar(adder middleware.Adder) Option {
	return func(c *conf) {
		c.getMissAdder = adder
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	return middleware.Wrap(inner,
		middleware.WithGetHitCounter(middleware.WrapAdderToIncrementer(c.getHitAdder)),
		middleware.WithGetMissCounter(middleware.WrapAdderToIncrementer(c.getHitAdder)),
	)
}
