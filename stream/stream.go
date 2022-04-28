package stream

import (
	"context"
	"errors"

	"github.com/sschiz/indexer/ticker"
)

var ErrInvalidChannel = errors.New("invalid channel")

// Stream streams ticker price.
type Stream interface {
	Get(ctx context.Context) (*ticker.Price, error)
}

// ChanStream streams ticker price using channels.
type ChanStream struct {
	errors <-chan error
	ticker <-chan ticker.Price
}

// NewChanStream returns new ChanStream instance.
func NewChanStream(t <-chan ticker.Price, errs <-chan error) (*ChanStream, error) {
	if t == nil || errs == nil {
		return nil, ErrInvalidChannel
	}

	return &ChanStream{
		errors: errs,
		ticker: t,
	}, nil
}

// Get returns incoming TickerPrice.
func (s *ChanStream) Get(ctx context.Context) (*ticker.Price, error) {
	select {
	case price := <-s.ticker:
		return &price, nil
	case err := <-s.errors:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
