package stream

import (
	"context"
	"errors"

	"github.com/sschiz/indexer/ticker"
)

var ErrInvalidChannel = errors.New("invalid channel")

// Stream streams ticker price.
type Stream interface {
	Get(ctx context.Context) (*ticker.TickerPrice, error)
}

// ChanStream streams ticker price using channels.
type ChanStream struct {
	errors <-chan error
	ticker <-chan ticker.TickerPrice
}

// NewChanStream returns new ChanStream instance.
func NewChanStream(ticker <-chan ticker.TickerPrice, errors <-chan error) (*ChanStream, error) {
	if ticker == nil || errors == nil {
		return nil, ErrInvalidChannel
	}

	return &ChanStream{
		errors: errors,
		ticker: ticker,
	}, nil
}

// Get returns incoming TickerPrice.
func (s *ChanStream) Get(ctx context.Context) (*ticker.TickerPrice, error) {
	select {
	case price := <-s.ticker:
		return &price, nil
	case err := <-s.errors:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
