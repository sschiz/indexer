package indexer

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sschiz/indexer/mock"
	"github.com/sschiz/indexer/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

type testEnv struct {
	collecter *mock.MockCollecter
	idxer     *Indexer
	ctrl      *gomock.Controller
}

func tearUp(t *testing.T) *testEnv {
	ctrl := gomock.NewController(t)

	clctr := mock.NewMockCollecter(ctrl)
	handler := Handler(func(tp ticker.TickerPrice) { t.Log(tp) })

	idxer, err := NewIndexer(clctr, handler, time.Minute)
	require.NoError(t, err)

	return &testEnv{
		collecter: clctr,
		idxer:     idxer,
		ctrl:      ctrl,
	}
}

func tearDown(env *testEnv) {
	env.ctrl.Finish()
}

func TestNewIndexer(t *testing.T) {
	t.Run("invalid collecter", func(t *testing.T) {
		got, err := NewIndexer(
			nil,
			func(tp ticker.TickerPrice) { t.Log(tp) },
			time.Minute,
		)

		require.Nil(t, got)
		assert.ErrorIs(t, err, ErrInvalidCollecter)
	})

	t.Run("invalid handler", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clctr := mock.NewMockCollecter(ctrl)

		got, err := NewIndexer(
			clctr,
			nil,
			time.Minute,
		)

		require.Nil(t, got)
		assert.ErrorIs(t, err, ErrInvalidHandler)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clctr := mock.NewMockCollecter(ctrl)
		handler := Handler(func(tp ticker.TickerPrice) { t.Log(tp) })

		got, err := NewIndexer(
			clctr,
			handler,
			time.Minute,
		)

		require.NoError(t, err)

		expected := Indexer{
			avgs:      make(map[ticker.Ticker]*avg),
			handle:    handler,
			collecter: clctr,
			err:       nil,
			started:   atomic.NewBool(false),
			done:      make(chan struct{}),
			interval:  time.Minute,
		}

		assert.Condition(t, func() (success bool) {
			return got.done != nil &&
				assert.ObjectsAreEqualValues(
					expected.avgs,
					got.avgs,
				) &&
				assert.ObjectsAreEqual(
					expected.collecter,
					got.collecter,
				) &&
				assert.ObjectsAreEqual(
					expected.err,
					got.err,
				) &&
				assert.ObjectsAreEqual(
					expected.interval,
					got.interval,
				) &&
				assert.ObjectsAreEqual(
					expected.started,
					got.started,
				)
		})
	})
}

func TestIndexer_Stop(t *testing.T) {
	t.Run("already stopped", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		err := env.idxer.Stop(context.Background())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		env.idxer.Stop(ctx)
		assert.NoError(t, err)
	})

	t.Run("stopped successfully", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		env.idxer.started.Store(true)

		err := env.idxer.Stop(context.Background())
		require.NoError(t, err)

		assert.Equal(t, <-env.idxer.done, struct{}{})
	})

	t.Run("canceled context", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		env.idxer.started.Store(true)
		env.idxer.done = make(chan struct{})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := env.idxer.Stop(ctx)

		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestIndexer_Start(t *testing.T) {
	t.Run("already started", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		called := false
		env.idxer.handle = func(_ ticker.TickerPrice) {
			called = true
		}

		env.idxer.started.Store(true)

		env.idxer.Start(context.Background())

		assert.False(t, called)
	})

	t.Run("started successfully", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		env.idxer.Start(ctx)

		assert.True(t, env.idxer.started.Load())
	})
}

func TestIndexer_Err(t *testing.T) {
	expectedErr := errors.New("any error")

	t.Run("error is not nil", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		env.idxer.err = expectedErr

		assert.ErrorIs(t, env.idxer.Err(), expectedErr)
	})

	t.Run("error is nil", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		assert.NoError(t, env.idxer.Err())
	})
}

func TestIndexer_start(t *testing.T) {
	t.Run("canceled context", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		env.idxer.start(ctx)

		assert.ErrorIs(t, env.idxer.err, context.Canceled)
		assert.False(t, env.idxer.started.Load())
	})
	t.Run("stopped", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		env.idxer.done <- struct{}{}
		env.idxer.start(ctx)

		assert.NoError(t, env.idxer.err)
		assert.False(t, env.idxer.started.Load())
	})
	t.Run("collecter error", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		env.idxer.interval = time.Second

		expectedErr := errors.New("any error")
		env.collecter.EXPECT().Collect(ctx).Return(nil, expectedErr)

		env.idxer.start(ctx)

		assert.ErrorIs(t, env.idxer.err, expectedErr)
		assert.False(t, env.idxer.started.Load())
	})
}

func TestIndexer_index(t *testing.T) {
	t.Run("collecter error", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		expectedErr := errors.New("any error")
		env.collecter.EXPECT().Collect(ctx).Return(nil, expectedErr)

		err := env.idxer.index(ctx, time.Now())

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("invalid price", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		env.collecter.EXPECT().Collect(ctx).Return([]*ticker.TickerPrice{
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   time.Now(),
				Price:  "invalid num",
			},
		}, nil)

		err := env.idxer.index(ctx, time.Now())

		assert.ErrorIs(t, err, strconv.ErrSyntax)
	})

	t.Run("indexed successfully", func(t *testing.T) {
		env := tearUp(t)
		defer tearDown(env)

		now := time.Now().UTC()

		var got []ticker.TickerPrice
		env.idxer.handle = func(tp ticker.TickerPrice) {
			got = append(got, tp)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		env.collecter.EXPECT().Collect(ctx).Return([]*ticker.TickerPrice{
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   now,
				Price:  "2",
			},
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   now,
				Price:  "2",
			},
		}, nil)

		err := env.idxer.index(ctx, now)

		require.NoError(t, err)

		expected := []ticker.TickerPrice{
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   now,
				Price:  "2",
			},
		}
		assert.ElementsMatch(t, expected, got)
	})
}

func Test_avg_Add(t *testing.T) {
	tests := []struct {
		name string
		args []float64
		want avg
	}{
		{
			name: "positive",
			args: []float64{1, 2, 3, 4},
			want: avg{
				sum: 10,
				num: 4,
			},
		},
		{
			name: "negative",
			args: []float64{-1, -2, -3, -4},
			want: avg{
				sum: 10,
				num: 4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &avg{}
			for _, v := range tt.args {
				a.Add(v)
			}

			assert.Equal(t, tt.want, *a)
		})
	}
}

func Test_avg_Average(t *testing.T) {
	tests := []struct {
		name string
		args []float64
		want float64
	}{
		{
			name: "positive",
			args: []float64{2, 4, 8, 10},
			want: 6,
		},
		{
			name: "negative",
			args: []float64{-2, -4, -8, -10},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &avg{}
			for _, v := range tt.args {
				a.Add(v)
			}

			assert.Equal(t, tt.want, a.Average())
		})
	}
}
