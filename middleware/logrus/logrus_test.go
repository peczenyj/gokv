package logrus_test

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/middleware"
	middleware_logrus "github.com/philippgille/gokv/middleware/logrus"
)

func TestMiddlewareLogger(t *testing.T) {
	t.Parallel()

	var rngSeed int64 = 1
	var nopTraceID middleware.TraceID

	_, _ = rand.New(rand.NewSource(rngSeed)).Read(nopTraceID[:])

	testcases := []struct {
		label   string
		opts    []middleware_logrus.Option
		prepare func(*storeMock)
		test    func(*testing.T, gokv.Store)
		verify  func(*testing.T, *test.Hook)
	}{
		{
			label: "should log when get with success",
			prepare: func(s *storeMock) {
				s.On("Get", "foo", mock.AnythingOfType("*int")).Run(func(args mock.Arguments) {
					valuePtr := args.Get(1).(*int)
					*valuePtr = 1
				}).Return(true, nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				var v int
				found, err := s.Get("foo", &v)
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, v, 1)
			},
			verify: Verify(1,
				logrus.DebugLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "get",
					"key":       "foo",
					"found":     true,
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when get with success but not found",
			prepare: func(s *storeMock) {
				s.On("Get", "foo", mock.AnythingOfType("*int")).Return(false, nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				var v int
				found, err := s.Get("foo", &v)
				assert.NoError(t, err)
				assert.False(t, found)
				assert.Equal(t, v, 0)
			},
			verify: Verify(1,
				logrus.DebugLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "get",
					"key":       "foo",
					"found":     false,
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when get with failure",
			prepare: func(s *storeMock) {
				s.On("Get", "foo", mock.AnythingOfType("*int")).Return(false, errors.New("ops"))
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				var v int
				found, err := s.Get("foo", &v)
				assert.EqualError(t, err, "ops")
				assert.False(t, found)
				assert.Equal(t, v, 0)
			},
			verify: Verify(1,
				logrus.WarnLevel,
				"gokv.Store return error",
				logrus.Fields{
					"operation": "get",
					"key":       "foo",
					"found":     false,
					"trace-id":  nopTraceID,
				},
				"ops",
			),
		},
		{
			label: "should log when get with success",
			prepare: func(s *storeMock) {
				s.On("Get", "foo", mock.AnythingOfType("*int")).Run(func(args mock.Arguments) {
					valuePtr := args.Get(1).(*int)
					*valuePtr = 1
				}).Return(true, nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				var v int
				found, err := s.Get("foo", &v)
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, v, 1)
			},
			verify: Verify(1,
				logrus.DebugLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "get",
					"key":       "foo",
					"found":     true,
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when Set with success",
			prepare: func(s *storeMock) {
				s.On("Set", "foo", 1).Return(nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				err := s.Set("foo", 1)
				assert.NoError(t, err)
			},
			verify: Verify(1,
				logrus.DebugLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "set",
					"key":       "foo",
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when Set with failure",
			prepare: func(s *storeMock) {
				s.On("Set", "foo", 1).Return(errors.New("ops"))
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				err := s.Set("foo", 1)
				assert.EqualError(t, err, "ops")
			},
			verify: Verify(1,
				logrus.WarnLevel,
				"gokv.Store return error",
				logrus.Fields{
					"operation": "set",
					"key":       "foo",
					"trace-id":  nopTraceID,
				},
				"ops",
			),
		},
		{
			label: "should log when Delete with success",
			prepare: func(s *storeMock) {
				s.On("Delete", "foo").Return(nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				err := s.Delete("foo")
				assert.NoError(t, err)
			},
			verify: Verify(1,
				logrus.DebugLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "delete",
					"key":       "foo",
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when Delete with failure",
			prepare: func(s *storeMock) {
				s.On("Delete", "foo").Return(errors.New("ops"))
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				err := s.Delete("foo")
				assert.EqualError(t, err, "ops")
			},
			verify: Verify(1,
				logrus.WarnLevel,
				"gokv.Store return error",
				logrus.Fields{
					"operation": "delete",
					"key":       "foo",
					"trace-id":  nopTraceID,
				},
				"ops",
			),
		},
		{
			label: "should log when close with success",
			opts:  []middleware_logrus.Option{middleware_logrus.WithMessageNormalLevel(logrus.InfoLevel)},
			prepare: func(s *storeMock) {
				s.On("Close").Return(nil)
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				assert.NoError(t, s.Close())
			},
			verify: Verify(1,
				logrus.InfoLevel,
				"gokv.Store return success",
				logrus.Fields{
					"operation": "close",
					"trace-id":  nopTraceID,
				},
				"",
			),
		},
		{
			label: "should log when close with failure",
			opts:  []middleware_logrus.Option{middleware_logrus.WithMessageErrorLevel(logrus.ErrorLevel)},
			prepare: func(s *storeMock) {
				s.On("Close").Return(errors.New("ops"))
			},
			test: func(t *testing.T, s gokv.Store) {
				t.Helper()

				assert.EqualError(t, s.Close(), "ops")
			},
			verify: Verify(1,
				logrus.ErrorLevel,
				"gokv.Store return error",
				logrus.Fields{
					"operation": "close",
					"trace-id":  nopTraceID,
				},
				"ops",
			),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()
			store := newStoreMock(t)

			tc.prepare(store)

			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			randReader := rand.New(rand.NewSource(rngSeed))
			opts := []middleware_logrus.Option{
				middleware_logrus.WithBaseMiddlewareOption(middleware.WithRandReader(randReader)),
			}

			opts = append(opts, tc.opts...)

			storeLogger := middleware_logrus.WithLogrus(store, logger, opts...)

			tc.test(t, storeLogger)

			tc.verify(t, hook)
		})
	}
}

func Verify(n int,
	level logrus.Level,
	logMsg string,
	fields logrus.Fields,
	errMsg string,
) func(t *testing.T, hook *test.Hook) {
	return func(t *testing.T, hook *test.Hook) {
		t.Helper()

		require.Len(t, hook.Entries, n)

		entry := hook.LastEntry()

		assert.Equal(t, level, entry.Level)
		assert.Equal(t, logMsg, entry.Message)
		for k, v := range fields {
			assert.Equalf(t, v, entry.Data[k], "checking field %q", k)
		}

		_, durationExists := entry.Data["duration"]
		assert.True(t, durationExists)

		if errMsg != "" {
			val, ok := entry.Data[logrus.ErrorKey]
			require.True(t, ok)

			err, ok := val.(error)
			require.True(t, ok)

			assert.EqualError(t, err, errMsg)
		} else {
			_, ok := entry.Data[logrus.ErrorKey]
			require.False(t, ok)
		}

	}
}

type storeMock struct {
	mock.Mock
}

func (m *storeMock) Set(k string, v interface{}) error {
	args := m.Called(k, v)

	return args.Error(0)
}

func (m *storeMock) Get(k string, v interface{}) (bool, error) {
	args := m.Called(k, v)

	return args.Bool(0), args.Error(1)
}

func (m *storeMock) Delete(k string) error {
	args := m.Called(k)

	return args.Error(0)
}

func (m *storeMock) Close() error {
	args := m.Called()

	return args.Error(0)
}

func newStoreMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *storeMock {
	mock := &storeMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
