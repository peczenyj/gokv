package shard

import (
	"errors"
	"strings"

	"github.com/OneOfOne/xxhash"

	"github.com/philippgille/gokv"
)

var _ gokv.Store = (*shardMiddleware)(nil)

type shardMiddleware struct {
	shards []gokv.Store
}

func (m *shardMiddleware) getShard(k string) gokv.Store {
	if len(m.shards) == 1 {
		return m.shards[0]
	}

	return m.shards[xxhash.ChecksumString64(k)%uint64(len(m.shards))]
}

func (m *shardMiddleware) Set(k string, v interface{}) error {
	return m.getShard(k).Set(k, v)
}

func (m *shardMiddleware) Get(k string, v interface{}) (found bool, err error) {
	return m.getShard(k).Get(k, v)
}

func (m *shardMiddleware) Delete(k string) error {
	return m.getShard(k).Delete(k)
}

func (m *shardMiddleware) Close() error {
	var errs []error

	for _, store := range m.shards {
		err := store.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	switch n := len(errs); n {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		// this is an "equivalent" to errors.Join() but it was added on go 1.20
		var errMsg strings.Builder
		for i, err := range errs {
			if i > 0 {
				_ = errMsg.WriteByte('\n')
			}
			_, _ = errMsg.WriteString(err.Error())
		}

		return errors.New(errMsg.String())
	}
}

func Wrap(shards ...gokv.Store) gokv.Store {
	_ = shards[0]

	if len(shards) == 1 {
		return shards[0]
	}

	return &shardMiddleware{shards: shards}
}
