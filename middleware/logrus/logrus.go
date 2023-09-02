package logrus

import (
	"context"
	"time"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"

	"github.com/sirupsen/logrus"
)

type logDecorator struct {
	logger         *logrus.Logger
	msgNormalLevel logrus.Level
	msgErrorLevel  logrus.Level
}

func (d *logDecorator) handleLog(ctx context.Context, err error, fields logrus.Fields) {
	if traceID, ok := middleware.GetTraceIDFromContext(ctx); ok {
		fields["trace-id"] = traceID
	}

	l := d.logger.WithFields(fields)

	if err != nil {
		l.WithError(err).Log(d.msgErrorLevel, "gokv.Store return error")
	} else {
		l.Log(d.msgNormalLevel, "gokv.Store return success")
	}
}

type conf struct {
	msgNormalLevel logrus.Level
	msgErrorLevel  logrus.Level

	logger *logrus.Logger

	baseMiddlewareOptions []middleware.Option
}

func (c *conf) SetDefaults() {
	c.msgNormalLevel = logrus.DebugLevel
	c.msgErrorLevel = logrus.WarnLevel
	c.logger = logrus.StandardLogger()
}

// Option functional option type.
type Option func(*conf)

func WithLogger(logger *logrus.Logger) Option {
	return func(c *conf) {
		c.logger = logger
	}
}

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

func WithBaseMiddlewareOption(opt middleware.Option) Option {
	return func(c *conf) {
		c.baseMiddlewareOptions = append(c.baseMiddlewareOptions, opt)
	}
}

// WithLogrus decorate the inner gokv.Store with an instance of logrus
// TODO use default logrus unless we set it via functional option
// TODO add basic const fields to add on logrus
func WithLogrus(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	d := &logDecorator{
		logger:         c.logger,
		msgNormalLevel: c.msgNormalLevel,
		msgErrorLevel:  c.msgErrorLevel,
	}

	middlewareOpts := []middleware.Option{
		middleware.WithTraceCallback(func(ctx context.Context, method string, args map[string]interface{}) (done func(context.Context, error)) {
			if !c.logger.IsLevelEnabled(logrus.TraceLevel) {
				return func(context.Context, error) {}
			}

			if args == nil {
				args = map[string]interface{}{}
			}

			args["operation"] = method

			if traceID, ok := middleware.GetTraceIDFromContext(ctx); ok {
				args["trace-id"] = traceID
			}

			c.logger.WithFields(args).Trace("trace call to gokv.Store")

			return func(context.Context, error) {}
		}),
		middleware.WithSetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, err error) {
			d.handleLog(ctx, err, logrus.Fields{
				"operation": "set",
				"key":       k,
				"duration":  duration,
			})
		}),
		middleware.WithGetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, found bool, err error) {
			d.handleLog(ctx, err, logrus.Fields{
				"operation": "get",
				"key":       k,
				"found":     found,
				"duration":  duration,
			})
		}),
		middleware.WithDeleteCallback(func(ctx context.Context, duration time.Duration, k string, err error) {
			d.handleLog(ctx, err, logrus.Fields{
				"operation": "delete",
				"key":       k,
				"duration":  duration,
			})
		}),
		middleware.WithCloseCallback(func(ctx context.Context, duration time.Duration, err error) {
			d.handleLog(ctx, err, logrus.Fields{
				"operation": "close",
				"duration":  duration,
			})
		}),
	}

	middlewareOpts = append(middlewareOpts, c.baseMiddlewareOptions...)

	return middleware.Wrap(inner, middlewareOpts...)
}
