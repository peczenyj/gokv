package middleware

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/philippgille/gokv"
)

var traceIDKey traceIDKeyType = "trace-id"

type traceIDKeyType string

type TraceID uint64

var nilTraceID TraceID

// String returns the hex string representation form of a TraceID.
func (t TraceID) String() string {
	return fmt.Sprintf("%#016x", uint64(t))
}

// GetTraceIDFromContext extract the trace id from context, if any
func GetTraceIDFromContext(ctx context.Context) (TraceID, bool) {
	v := ctx.Value(traceIDKey)
	if v == nil {
		return nilTraceID, false
	}

	traceID, ok := v.(TraceID)

	return traceID, ok
}

type TraceCallback func(ctx context.Context, method string, args map[string]interface{}) (done func(context.Context, error))

type SetCallback func(ctx context.Context, duration time.Duration, k string, v interface{}, err error)

type GetCallback func(ctx context.Context, duration time.Duration, k string, v interface{}, found bool, err error)

type DeleteCallback func(ctx context.Context, duration time.Duration, k string, err error)

type CloseCallback func(ctx context.Context, duration time.Duration, err error)

type middleware struct {
	inner gokv.Store

	randomSource64 rand.Source64

	traceCallbacks  []TraceCallback
	setCallbacks    []SetCallback
	getCallbacks    []GetCallback
	deleteCallbacks []DeleteCallback
	closeCallbacks  []CloseCallback

	setHitCounters    []Incrementer
	getHitCounters    []Incrementer
	getMissCounters   []Incrementer
	deleteHitCounters []Incrementer

	setObservers    []Observer
	getObservers    []Observer
	deleteObservers []Observer
}

func (m *middleware) fetchTraceID() TraceID {
	return TraceID(m.randomSource64.Uint64())
}

func (m *middleware) wrapContextWithTraceID(ctx context.Context) context.Context {
	traceID := m.fetchTraceID()

	return context.WithValue(ctx, traceIDKey, traceID)
}

func (m *middleware) Set(k string, v interface{}) (err error) {
	for _, counter := range m.setHitCounters {
		counter.Inc()
	}

	ctx := m.wrapContextWithTraceID(context.Background())

	for _, callback := range m.traceCallbacks {
		done := callback(ctx, "set", map[string]interface{}{
			"key":   k,
			"value": v,
		})

		defer done(ctx, err)
	}

	begin := time.Now()

	err = m.inner.Set(k, v)

	duration := time.Since(begin)

	for _, callback := range m.setCallbacks {
		callback(ctx, duration, k, v, err)
	}

	for _, observer := range m.setObservers {
		observer.Observe(duration.Seconds())
	}

	return err
}

func (m *middleware) Get(k string, v interface{}) (found bool, err error) {
	for _, counter := range m.getHitCounters {
		counter.Inc()
	}

	ctx := m.wrapContextWithTraceID(context.Background())

	for _, callback := range m.traceCallbacks {
		done := callback(ctx, "get", map[string]interface{}{
			"key":   k,
			"value": v,
		})

		defer done(ctx, err)
	}

	begin := time.Now()

	found, err = m.inner.Get(k, v)

	if !found {
		for _, counter := range m.getMissCounters {
			counter.Inc()
		}
	}

	duration := time.Since(begin)

	for _, callback := range m.getCallbacks {
		callback(ctx, duration, k, v, found, err)
	}

	for _, observer := range m.getObservers {
		observer.Observe(duration.Seconds())
	}

	return found, err
}

func (m *middleware) Delete(k string) (err error) {
	for _, counter := range m.deleteHitCounters {
		counter.Inc()
	}

	ctx := m.wrapContextWithTraceID(context.Background())

	for _, callback := range m.traceCallbacks {
		done := callback(ctx, "delete", map[string]interface{}{
			"key": k,
		})

		defer done(ctx, err)
	}

	begin := time.Now()

	err = m.inner.Delete(k)

	duration := time.Since(begin)

	for _, callback := range m.deleteCallbacks {
		callback(ctx, duration, k, err)
	}

	for _, observer := range m.deleteObservers {
		observer.Observe(duration.Seconds())
	}

	return err
}

func (m *middleware) Close() (err error) {
	ctx := m.wrapContextWithTraceID(context.Background())

	for _, callback := range m.traceCallbacks {
		done := callback(ctx, "close", nil)

		defer done(ctx, err)
	}

	begin := time.Now()

	err = m.inner.Close()

	duration := time.Since(begin)

	for _, callback := range m.closeCallbacks {
		callback(ctx, duration, err)
	}

	return err
}

// Option type.
type Option func(*middleware)

// WithRandomSource64 functional option to substitute the random io.Reader.
func WithRandomSource64(source rand.Source64) Option {
	return func(m *middleware) {
		m.randomSource64 = source
	}
}

// WithTraceCallback xxx.
func WithTraceCallback(callback TraceCallback) Option {
	return func(m *middleware) {
		m.traceCallbacks = append(m.traceCallbacks, callback)
	}
}

func WithSetCallback(callback SetCallback) Option {
	return func(m *middleware) {
		m.setCallbacks = append(m.setCallbacks, callback)
	}
}

func WithGetCallback(callback GetCallback) Option {
	return func(m *middleware) {
		m.getCallbacks = append(m.getCallbacks, callback)
	}
}

func WithDeleteCallback(callback DeleteCallback) Option {
	return func(m *middleware) {
		m.deleteCallbacks = append(m.deleteCallbacks, callback)
	}
}

func WithCloseCallback(callback CloseCallback) Option {
	return func(m *middleware) {
		m.closeCallbacks = append(m.closeCallbacks, callback)
	}
}

func Wrap(inner gokv.Store, opts ...Option) gokv.Store {
	if len(opts) == 0 {
		return inner
	}

	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)

	m := &middleware{
		inner:          inner,
		randomSource64: rand.New(rand.NewSource(rngSeed)),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
