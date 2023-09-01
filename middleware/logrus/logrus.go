package logrus

import (
	"context"
	"time"

	"github.com/philippgille/gokv"
	gokv_middleware "github.com/philippgille/gokv/middleware"

	"github.com/sirupsen/logrus"
)

type logDecorator struct {
	logger         *logrus.Logger
	msgNormalLevel logrus.Level
	msgErrorLevel  logrus.Level
}

func (m *logDecorator) handleLog(ctx context.Context, err error, fields logrus.Fields) {
	if traceID, ok := gokv_middleware.GetTraceIDFromContext(ctx); ok {
		fields["trace-id"] = traceID
	}

	l := m.logger.WithFields(fields)

	if err != nil {
		l.WithError(err).Log(m.msgErrorLevel, "gokv.Store return error")
	} else {
		l.Log(m.msgNormalLevel, "gokv.Store return success")
	}
}

type conf struct {
	msgNormalLevel        logrus.Level
	msgErrorLevel         logrus.Level
	baseMiddlewareOptions []gokv_middleware.Option
}

func (c *conf) SetDefaults() {
	c.msgNormalLevel = logrus.DebugLevel
	c.msgErrorLevel = logrus.WarnLevel
}

// Option functional option type.
type Option func(*conf)

// WithMessageNormalLevel change the message normal level (default is debug)
func WithMessageNormalLevel(level logrus.Level) Option {
	return func(c *conf) {
		c.msgNormalLevel = level
	}
}

// WithMessageNormalLevel change the message error level (default is warning)
func WithMessageErrorLevel(level logrus.Level) Option {
	return func(c *conf) {
		c.msgErrorLevel = level
	}
}
func WithBaseMiddlewareOption(opt gokv_middleware.Option) Option {
	return func(c *conf) {
		c.baseMiddlewareOptions = append(c.baseMiddlewareOptions, opt)
	}
}

// WithLogrus decorate the inner gokv.Store with an instance of logrus
func WithLogrus(inner gokv.Store, logger *logrus.Logger, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	la := &logDecorator{
		logger:         logger,
		msgNormalLevel: c.msgNormalLevel,
		msgErrorLevel:  c.msgErrorLevel,
	}

	middlewareOpts := []gokv_middleware.Option{
		gokv_middleware.WithTraceCallback(func(ctx context.Context, method string, args map[string]interface{}) {
			if !logger.IsLevelEnabled(logrus.TraceLevel) {
				return
			}

			if args == nil {
				args = map[string]interface{}{}
			}

			args["operation"] = method

			if traceID, ok := gokv_middleware.GetTraceIDFromContext(ctx); ok {
				args["trace-id"] = traceID
			}

			logger.WithFields(args).Trace("trace call to gokv.Store")
		}),
		gokv_middleware.WithSetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, err error) {
			la.handleLog(ctx, err, logrus.Fields{
				"operation": "set",
				"key":       k,
				"duration":  duration,
			})
		}),
		gokv_middleware.WithGetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, found bool, err error) {
			la.handleLog(ctx, err, logrus.Fields{
				"operation": "get",
				"key":       k,
				"found":     found,
				"duration":  duration,
			})
		}),
		gokv_middleware.WithDeleteCallback(func(ctx context.Context, duration time.Duration, k string, err error) {
			la.handleLog(ctx, err, logrus.Fields{
				"operation": "delete",
				"key":       k,
				"duration":  duration,
			})
		}),
		gokv_middleware.WithCloseCallback(func(ctx context.Context, duration time.Duration, err error) {
			la.handleLog(ctx, err, logrus.Fields{
				"operation": "close",
				"duration":  duration,
			})
		}),
	}

	middlewareOpts = append(middlewareOpts, c.baseMiddlewareOptions...)

	return gokv_middleware.Wrap(inner, middlewareOpts...)
}
