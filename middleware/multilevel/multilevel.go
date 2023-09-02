package multilevel

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/philippgille/gokv"
)

type multilevelMiddleware struct {
	levels []gokv.Store
}

func (m *multilevelMiddleware) Set(k string, v interface{}) error {
	for i, level := range m.levels {
		if err := level.Set(k, v); err != nil {
			return fmt.Errorf("unable to perforn Set(%q,...) on gokv.Store #%d level: %v", k, i, err)
		}
	}

	return nil
}

func (m *multilevelMiddleware) Get(k string, v interface{}) (found bool, err error) {
	for i, level := range m.levels {
		found, err = level.Get(k, v)
		if err != nil {
			return found, fmt.Errorf("unable to perforn Get(%q,...) on gokv.Store #%d level: %v", k, i, err)
		}

		if found {
			m.setUpToLevelPassive(i, k, v)

			return found, err
		}
	}

	return false, nil
}

func (m *multilevelMiddleware) setUpToLevelPassive(limit int, k string, v interface{}) {
	for i := 0; i < limit; i++ {
		if err := m.levels[i].Set(k, v); err != nil {
			log.Printf("unexpected error while set up the gokv.Store level #%d on key %q: %v", i, k, err)
		}
	}
}

func (m *multilevelMiddleware) Delete(k string) error {
	for i, level := range m.levels {
		if err := level.Delete(k); err != nil {
			return fmt.Errorf("unable to perforn Delete(%q) on gokv.Store #%d level: %v", k, i, err)
		}
	}

	return nil
}

func (m *multilevelMiddleware) Close() error {
	var errs []error

	for _, level := range m.levels {
		if err := level.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
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

func Wrap(inners ...gokv.Store) gokv.Store {
	_ = inners[1]

	return &multilevelMiddleware{
		levels: inners,
	}
}
