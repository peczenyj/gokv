package fallback

import (
	"fmt"

	"github.com/philippgille/gokv"
)

type fallbackMiddleware struct {
	stable          gokv.Store
	fallback        gokv.Store
	errorClassifier func(error) bool
}

func (m *fallbackMiddleware) Set(k string, v interface{}) error {
	err := m.stable.Set(k, v)
	if m.errorClassifier(err) {
		return m.fallback.Set(k, v)
	}

	return err
}

func (m *fallbackMiddleware) Get(k string, v interface{}) (found bool, err error) {
	found, err = m.stable.Get(k, v)
	if m.errorClassifier(err) {
		return m.fallback.Get(k, v)
	}

	return found, err
}

func (m *fallbackMiddleware) Delete(k string) error {
	err := m.stable.Delete(k)
	if m.errorClassifier(err) {
		return m.fallback.Delete(k)
	}

	return err
}

func (m *fallbackMiddleware) Close() error {
	var errs []error

	if err := m.fallback.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := m.stable.Close(); err != nil {
		errs = append(errs, err)
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("%s\n%s", errs[0].Error(), errs[1].Error())
	}
}

func defaultErrorClassifier(err error) bool {
	return err != nil
}

type conf struct {
	errorClassifier func(error) bool
}

func (c *conf) SetDefaults() {
	c.errorClassifier = defaultErrorClassifier
}

type Option func(*conf)

func WithErrorClassifier(errorClassifier func(error) bool) Option {
	return func(c *conf) {
		c.errorClassifier = errorClassifier
	}
}

func Wrap(stable, fallback gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	return &fallbackMiddleware{
		stable:          stable,
		fallback:        fallback,
		errorClassifier: c.errorClassifier,
	}
}
