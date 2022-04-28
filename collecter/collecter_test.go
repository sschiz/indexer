package collecter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sschiz/indexer/mock"
	"github.com/sschiz/indexer/stream"
	"github.com/sschiz/indexer/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamCollecter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockStream(ctrl)
	collecter := NewStreamCollecter([]stream.Stream{s})
	assert.Equal(t, &StreamCollecter{streams: []stream.Stream{s}}, collecter)
}

func TestStreamCollecter_Collect(t *testing.T) {
	t.Run("stream error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := errors.New("stream error")
		ctx := context.Background()

		s1 := mock.NewMockStream(ctrl)
		s2 := mock.NewMockStream(ctrl)
		s3 := mock.NewMockStream(ctrl)

		s1.EXPECT().Get(gomock.Any()).Return(&ticker.Price{}, nil)
		s2.EXPECT().Get(gomock.Any()).Return(nil, expected)
		s3.EXPECT().Get(gomock.Any()).Return(&ticker.Price{}, nil)

		collecter := NewStreamCollecter([]stream.Stream{s1, s2, s3})
		prices, err := collecter.Collect(ctx)
		require.Nil(t, prices)
		assert.ErrorIs(t, err, expected)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		expected := []*ticker.Price{
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

		s1 := mock.NewMockStream(ctrl)
		s2 := mock.NewMockStream(ctrl)
		s3 := mock.NewMockStream(ctrl)

		s1.EXPECT().Get(gomock.Any()).Return(expected[0], nil)
		s2.EXPECT().Get(gomock.Any()).Return(expected[1], nil)
		s3.EXPECT().Get(gomock.Any()).Return(expected[2], nil)

		collecter := NewStreamCollecter([]stream.Stream{s1, s2, s3})

		prices, err := collecter.Collect(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, prices)
	})
}
