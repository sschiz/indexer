package collecter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sschiz/indexer/stream"
	"github.com/sschiz/indexer/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamCollecter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := stream.NewMockStream(ctrl)
	collecter := NewStreamCollecter([]stream.Stream{s})
	assert.Equal(t, &StreamCollecter{streams: []stream.Stream{s}}, collecter)
}

func TestStreamCollecter_Collect(t *testing.T) {
	t.Run("stream error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := errors.New("stream error")
		ctx := context.Background()

		s1 := stream.NewMockStream(ctrl)
		s2 := stream.NewMockStream(ctrl)
		s3 := stream.NewMockStream(ctrl)

		s1.EXPECT().Get(gomock.Any()).Return(&ticker.TickerPrice{}, nil)
		s2.EXPECT().Get(gomock.Any()).Return(nil, expected)
		s3.EXPECT().Get(gomock.Any()).Return(&ticker.TickerPrice{}, nil)

		collecter := NewStreamCollecter([]stream.Stream{s1, s2, s3})
		prices, err := collecter.Collect(ctx)
		require.Nil(t, prices)
		assert.ErrorIs(t, err, expected)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		expected := []*ticker.TickerPrice{
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   time.Now(),
				Price:  "0.1",
			},
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   time.Now(),
				Price:  "0.2",
			},
			{
				Ticker: ticker.BTCUSDTicker,
				Time:   time.Now(),
				Price:  "0.3",
			},
		}

		s1 := stream.NewMockStream(ctrl)
		s2 := stream.NewMockStream(ctrl)
		s3 := stream.NewMockStream(ctrl)

		s1.EXPECT().Get(gomock.Any()).Return(expected[0], nil)
		s2.EXPECT().Get(gomock.Any()).Return(expected[1], nil)
		s3.EXPECT().Get(gomock.Any()).Return(expected[2], nil)

		collecter := NewStreamCollecter([]stream.Stream{s1, s2, s3})

		prices, err := collecter.Collect(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, prices)
	})
}
