package readonly

import (
	"errors"

	"github.com/philippgille/gokv"
)

type readonlyMiddleware struct {
	gokv.Store
	defaultError error
}

func (m *readonlyMiddleware) Set(k string, v interface{}) error {
	return m.defaultError
}

func (m *readonlyMiddleware) Delete(k string) error {
	return m.defaultError
}

type conf struct {
	defaultError error
}

func (c *conf) SetDefaults() {
	c.defaultError = errors.New("operation not permitted: gokv.Store is read-only")
}

type Option func(*conf)

func WithDefaultError(defaultError error) Option {
	return func(c *conf) {
		c.defaultError = defaultError
	}
}

func WithoutDefaultError() Option {
	return func(c *conf) {
		c.defaultError = nil
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	return &readonlyMiddleware{
		Store:        inner,
		defaultError: c.defaultError,
	}
}
