package slog

import (
	"context"
	"time"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"

	"golang.org/x/exp/slog"
)

type conf struct {
	msgNormalLevel slog.Level
	msgErrorLevel  slog.Level

	logger *slog.Logger
}

func (c *conf) SetDefaults() {
	c.msgNormalLevel = slog.LevelDebug
	c.msgErrorLevel = slog.LevelWarn

	c.logger = slog.Default()
}

type Option func(*conf)

func WithLogger(logger *slog.Logger) Option {
	return func(c *conf) {
		c.logger = logger
	}
}

type logDecorator struct {
	logger         *slog.Logger
	msgNormalLevel slog.Level
	msgErrorLevel  slog.Level
}

func (d *logDecorator) handleLog(ctx context.Context, err error, fields ...interface{}) {
	logger := d.logger.With(fields...)

	if traceID, ok := middleware.GetTraceIDFromContext(ctx); ok {
		logger = logger.With("trace-id", traceID)
	}

	if err != nil {
		d.logger.Log(ctx, d.msgErrorLevel, "gokv.Store return error", "error", err)

		return
	}

	d.logger.Log(ctx, d.msgNormalLevel, "gokv.Store return success")
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
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

	return middleware.Wrap(inner,
		middleware.WithSetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, err error) {
			d.handleLog(ctx, err,
				"operation", "set",
				"key", k,
				"duration", duration,
			)
		}),
		middleware.WithGetCallback(func(ctx context.Context, duration time.Duration, k string, v interface{}, found bool, err error) {
			d.handleLog(ctx, err,
				"operation", "get",
				"key", k,
				"found", found,
				"duration", duration,
			)
		}),
		middleware.WithDeleteCallback(func(ctx context.Context, duration time.Duration, k string, err error) {
			d.handleLog(ctx, err,
				"operation", "delete",
				"key", k,
				"duration", duration,
			)
		}),
		middleware.WithCloseCallback(func(ctx context.Context, duration time.Duration, err error) {
			d.handleLog(ctx, err,
				"operation", "close",
				"duration", duration,
			)
		}),
	)
}
