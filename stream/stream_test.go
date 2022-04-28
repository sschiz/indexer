package stream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sschiz/indexer/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChanStream(t *testing.T) {
	t.Run("nillable", func(t *testing.T) {
		errs := make(chan error)
		tick := make(chan ticker.Price)

		stream, err := NewChanStream(nil, errs)
		require.Nil(t, stream)
		assert.ErrorIs(t, err, ErrInvalidChannel)

		stream, err = NewChanStream(tick, nil)
		require.Nil(t, stream)
		assert.ErrorIs(t, err, ErrInvalidChannel)

		stream, err = NewChanStream(nil, nil)
		require.Nil(t, stream)
		assert.ErrorIs(t, err, ErrInvalidChannel)
	})

	t.Run("success", func(t *testing.T) {
		errs := make(chan error)
		tick := make(chan ticker.Price)

		stream, err := NewChanStream(tick, errs)
		require.NoError(t, err)
		assert.Equal(t, &ChanStream{errors: errs, ticker: tick}, stream)
	})
}

func TestChanStream_Get(t *testing.T) {
	t.Run("TickerPrice returned", func(t *testing.T) {
		now := time.Now()

		errs := make(chan error)
		tick := make(chan ticker.Price, 1)

		tick <- ticker.Price{
			Ticker: ticker.BTCUSDTicker,
			Time:   now,
			Price:  "test",
		}

		stream, err := NewChanStream(tick, errs)
		require.NoError(t, err)
		require.Equal(t, &ChanStream{errors: errs, ticker: tick}, stream)

		price, err := stream.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(
			t,
			&ticker.Price{
				Ticker: ticker.BTCUSDTicker,
				Time:   now,
				Price:  "test",
			},
			price,
		)
	})

	t.Run("error returned", func(t *testing.T) {
		errs := make(chan error, 1)
		tick := make(chan ticker.Price)

		expected := errors.New("any error")
		errs <- expected

		stream, err := NewChanStream(tick, errs)
		require.NoError(t, err)
		require.Equal(t, &ChanStream{errors: errs, ticker: tick}, stream)

		price, err := stream.Get(context.Background())
		require.Nil(t, price)
		assert.ErrorIs(t, err, expected)
	})

	t.Run("DeadlineExceeded returned", func(t *testing.T) {
		errs := make(chan error)
		tick := make(chan ticker.Price)

		stream, err := NewChanStream(tick, errs)
		require.NoError(t, err)
		require.Equal(t, &ChanStream{errors: errs, ticker: tick}, stream)

		ctx, cancel := context.WithDeadline(context.Background(), time.Now())
		defer cancel()

		price, err := stream.Get(ctx)
		require.Nil(t, price)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
