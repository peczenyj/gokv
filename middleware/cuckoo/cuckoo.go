package cuckoo

import (
	"github.com/philippgille/gokv"

	cuckoo "github.com/seiflotfy/cuckoofilter"
)

var (
	_ CuckooFilter = (*cuckoo.Filter)(nil)
	_ CuckooFilter = (*cuckoo.ScalableCuckooFilter)(nil)
)

type CuckooFilter interface {
	Insert(k []byte) bool
	Lookup(k []byte) bool
	Delete(k []byte) bool
}

type cuckooMiddleware struct {
	store        gokv.Store
	cuckooFilter CuckooFilter
}

func (m *cuckooMiddleware) Set(k string, v interface{}) error {
	err := m.store.Set(k, v)
	if err != nil {
		return err
	}

	_ = m.cuckooFilter.Insert([]byte(k))

	return nil
}

func (m *cuckooMiddleware) Get(k string, v interface{}) (found bool, err error) {
	if !m.cuckooFilter.Lookup([]byte(k)) {
		return false, nil
	}

	found, err = m.store.Get(k, v)
	if err != nil {
		return found, err
	}

	if found {
		_ = m.cuckooFilter.Insert([]byte(k))
	}

	return found, nil
}

func (m *cuckooMiddleware) Delete(k string) error {
	_ = m.cuckooFilter.Delete([]byte(k))

	return m.store.Delete(k)
}

func (m *cuckooMiddleware) Close() error {
	return m.store.Close()
}

type conf struct {
	capacity             uint
	scalableCuckooFilter bool
}

func (c *conf) SetDefaults() {
	c.capacity = cuckoo.DefaultCapacity
	c.scalableCuckooFilter = false
}

type Option func(*conf)

func WithDefaultCapacity(capacity uint) Option {
	return func(c *conf) {
		c.capacity = capacity
	}
}

func WithScalableCuckooFilter() Option {
	return func(c *conf) {
		c.scalableCuckooFilter = true
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	var filter CuckooFilter

	if c.scalableCuckooFilter {
		filter = cuckoo.NewScalableCuckooFilter()
	} else {
		filter = cuckoo.NewFilter(c.capacity)
	}

	return &cuckooMiddleware{
		store:        inner,
		cuckooFilter: filter,
	}
}
