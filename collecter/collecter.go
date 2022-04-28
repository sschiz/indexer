package collecter

import (
	"context"

	"github.com/sschiz/indexer/stream"
	"github.com/sschiz/indexer/ticker"
	"golang.org/x/sync/errgroup"
)

// Collecter collects all ticker prices.
type Collecter interface {
	Collect(ctx context.Context) ([]*ticker.Price, error)
}

type StreamCollecter struct {
	streams []stream.Stream
}

// NewStreamCollecter returns new Collecter instance.
func NewStreamCollecter(streams []stream.Stream) *StreamCollecter {
	return &StreamCollecter{
		streams: streams,
	}
}

// Collect returns all data from streams.
func (c *StreamCollecter) Collect(ctx context.Context) ([]*ticker.Price, error) {
	g, ctx := errgroup.WithContext(ctx)
	prices := make([]*ticker.Price, len(c.streams))
	for i, s := range c.streams {
		i, s := i, s
		g.Go(func() error {
			price, err := s.Get(ctx)
			if err != nil {
				return err
			}

			prices[i] = price
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return prices, nil
}
