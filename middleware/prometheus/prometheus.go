package prometheus

import (
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"

	prom "github.com/prometheus/client_golang/prometheus"
)

type conf struct {
	timeHistogramEnabled bool
	timeHistogramOptions prom.HistogramOpts
	timeHistogramVec     *prom.HistogramVec

	counterOpts prom.CounterOpts
	counterVet  *prom.CounterVec
}

func (c *conf) SetDefaults() {
	c.timeHistogramEnabled = false
	c.counterOpts = prom.CounterOpts{
		Name: "gokv_handled_total",
		Help: "total number of calls to gokv.Store per operation.",
	}
	c.timeHistogramOptions = prom.HistogramOpts{
		Name:    "gokv_handling_seconds",
		Help:    "histogram of response latency	(seconds) of the gokv.Store per operation until it is finished by the application.",
		Buckets: prom.DefBuckets,
	}
}

type Option func(*conf)

func WithCounterOpts(opts ...CounterOption) Option {
	return func(c *conf) {
		for _, opt := range opts {
			opt(&c.counterOpts)
		}
	}
}

func EnableHandlingTimeHistogram(opts ...HistogramOption) Option {
	return func(c *conf) {
		for _, opt := range opts {
			opt(&c.timeHistogramOptions)
		}

		if !c.timeHistogramEnabled {
			c.timeHistogramVec = prom.NewHistogramVec(c.timeHistogramOptions,
				[]string{"operation"})
		}

		c.timeHistogramEnabled = true
	}
}

// A CounterOption lets you add options to Counter metrics using With* funcs.
type CounterOption func(*prom.CounterOpts)

// WithConstLabels allows you to add ConstLabels to Counter metrics.
func WithConstLabels(labels prom.Labels) CounterOption {
	return func(o *prom.CounterOpts) {
		o.ConstLabels = labels
	}
}

// WithCounterName allow you to rename the prometheus counter (default is `gokv_handled_total`)
func WithCounterName(name string) CounterOption {
	return func(o *prom.CounterOpts) {
		o.Name = name
	}
}

// A HistogramOption lets you add options to Histogram metrics using With*
// funcs.
type HistogramOption func(*prom.HistogramOpts)

// WithHistogramBuckets allows you to specify custom bucket ranges for histograms if EnableHandlingTimeHistogram is on.
func WithHistogramBuckets(buckets []float64) HistogramOption {
	return func(o *prom.HistogramOpts) { o.Buckets = buckets }
}

// WithHistogramConstLabels allows you to add custom ConstLabels to
// histograms metrics.
func WithHistogramConstLabels(labels prom.Labels) HistogramOption {
	return func(o *prom.HistogramOpts) {
		o.ConstLabels = labels
	}
}

// WithCounterName allow you to rename the prometheus histogram (default is `gokv_handling_seconds`)
func WithHistogramName(name string) HistogramOption {
	return func(o *prom.HistogramOpts) {
		o.Name = name
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	var c conf

	c.SetDefaults()

	for _, opt := range opts {
		opt(&c)
	}

	middlewareOpts := []middleware.Option{
		middleware.WithSetHitCounter(c.counterVet.WithLabelValues("set")),
		middleware.WithGetHitCounter(c.counterVet.WithLabelValues("get")),
		middleware.WithDeleteHitCounter(c.counterVet.WithLabelValues("delete")),
	}

	if c.timeHistogramEnabled {
		middlewareOpts = append(middlewareOpts,
			middleware.WithSetDurationObserver(c.timeHistogramVec.WithLabelValues("set")),
			middleware.WithGetDurationObserver(c.timeHistogramVec.WithLabelValues("get")),
			middleware.WithDeleteDurationObserver(c.timeHistogramVec.WithLabelValues("delete")),
		)
	}

	return middleware.Wrap(inner,
		middlewareOpts...,
	)
}
