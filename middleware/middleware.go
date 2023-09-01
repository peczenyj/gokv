package middleware

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/rand"
	"time"

	"github.com/philippgille/gokv"
)

var traceIDKey traceIDKeyType = "trace-id"

type traceIDKeyType string

// TraceID is a unique identity of a trace.
// "inspired" on "go.opentelemetry.io/otel/trace".TraceID
type TraceID [16]byte

var nilTraceID TraceID

// String returns the hex string representation form of a TraceID.
func (t TraceID) String() string {
	return hex.EncodeToString(t[:])
}

// GetTraceIDFromContext extract the trace id from context, if any
func GetTraceIDFromContext(ctx context.Context) (TraceID, bool) {
	v := ctx.Value(traceIDKey)
	if v != nil {
		return v.(TraceID), true
	}

	return nilTraceID, false
}

type TraceCallback func(ctx context.Context, method string, args map[string]interface{})

type SetCallback func(ctx context.Context, duration time.Duration, k string, v interface{}, err error)

type GetCallback func(ctx context.Context, duration time.Duration, k string, v interface{}, found bool, err error)

type DeleteCallback func(ctx context.Context, duration time.Duration, k string, err error)

type CloseCallback func(ctx context.Context, duration time.Duration, err error)

type middleware struct {
	inner gokv.Store

	randReader io.Reader

	traceCallbacks  []TraceCallback
	closeCallbacks  []CloseCallback
	deleteCallbacks []DeleteCallback
	getCallbacks    []GetCallback
	setCallbacks    []SetCallback
}

func (m *middleware) Set(k string, v interface{}) error {
	ctx := context.Background()

	var traceID TraceID
	_, _ = m.randReader.Read(traceID[:])

	ctx = context.WithValue(ctx, traceIDKey, traceID)

	for _, callback := range m.traceCallbacks {
		callback(ctx, "set", map[string]interface{}{
			"key":   k,
			"value": v,
		})
	}

	begin := time.Now()

	err := m.inner.Set(k, v)

	duration := time.Since(begin)

	for _, callback := range m.setCallbacks {
		callback(ctx, duration, k, v, err)
	}

	return err
}

func (m *middleware) Get(k string, v interface{}) (found bool, err error) {
	ctx := context.Background()

	var traceID TraceID
	_, _ = m.randReader.Read(traceID[:])

	ctx = context.WithValue(ctx, traceIDKey, traceID)

	for _, callback := range m.traceCallbacks {
		callback(ctx, "get", map[string]interface{}{
			"key":   k,
			"value": v,
		})
	}

	begin := time.Now()

	found, err = m.inner.Get(k, v)

	duration := time.Since(begin)

	for _, callback := range m.getCallbacks {
		callback(ctx, duration, k, v, found, err)
	}

	return found, err
}

func (m *middleware) Delete(k string) error {
	ctx := context.Background()

	var traceID TraceID
	_, _ = m.randReader.Read(traceID[:])

	ctx = context.WithValue(ctx, traceIDKey, traceID)

	for _, callback := range m.traceCallbacks {
		callback(ctx, "delete", map[string]interface{}{
			"key": k,
		})
	}

	begin := time.Now()

	err := m.inner.Delete(k)

	duration := time.Since(begin)

	for _, callback := range m.deleteCallbacks {
		callback(ctx, duration, k, err)
	}

	return err
}

func (m *middleware) Close() error {
	ctx := context.Background()

	var traceID TraceID
	_, _ = m.randReader.Read(traceID[:])

	ctx = context.WithValue(ctx, traceIDKey, traceID)

	for _, callback := range m.traceCallbacks {
		callback(ctx, "close", nil)
	}

	begin := time.Now()

	err := m.inner.Close()

	duration := time.Since(begin)

	for _, callback := range m.closeCallbacks {
		callback(ctx, duration, err)
	}

	return err
}

// Option type.
type Option func(*middleware)

// WithRandReader functional option to substitute the random io.Reader.
func WithRandReader(reader io.Reader) Option {
	return func(m *middleware) {
		m.randReader = reader
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
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)

	m := &middleware{
		inner:      inner,
		randReader: rand.New(rand.NewSource(rngSeed)),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
