package trace

import (
	"context"

	"golang.org/x/net/trace"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"
)

func Wrap(inner gokv.Store) gokv.Store {
	return middleware.Wrap(inner,
		middleware.WithTraceCallback(func(ctx context.Context, method string, _ map[string]interface{}) (done func(context.Context, error)) {
			tr := trace.New("gokv.Store", method)

			return func(ctx context.Context, err error) {
				if err != nil {
					tr.LazyPrintf("error while handle operation %q: %v", method, err)

					tr.SetError()
				}

				tr.Finish()
			}
		}))
}
