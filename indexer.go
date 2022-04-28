package indexer

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/sschiz/indexer/collecter"
	"github.com/sschiz/indexer/ticker"
	"go.uber.org/atomic"
)

type Handler func(ticker.Price)

var (
	ErrInvalidHandler   = errors.New("invalid handler")
	ErrInvalidCollecter = errors.New("invalid collecter")
)

// Indexer streaming price indexer.
type Indexer struct {
	mu   sync.Mutex
	avgs map[ticker.Ticker]*avg

	handle   Handler
	interval time.Duration

	collecter collecter.Collecter
	err       error // last error from collecter

	started *atomic.Bool
	done    chan struct{}
}

// NewIndexer returns new Indexer instance.
// Handle is called for each indexed TickerPrice.
// Interval is a period during which indexing will be carried out.
func NewIndexer(clctr collecter.Collecter, handle Handler, interval time.Duration) (*Indexer, error) {
	if handle == nil {
		return nil, ErrInvalidHandler
	}

	if clctr == nil {
		return nil, ErrInvalidCollecter
	}

	return &Indexer{
		collecter: clctr,
		done:      make(chan struct{}, 1),
		started:   atomic.NewBool(false),
		avgs:      make(map[ticker.Ticker]*avg),
		handle:    handle,
		interval:  interval,
	}, nil
}

// Stop stops Indexer.
func (i *Indexer) Stop(ctx context.Context) error {
	if !i.started.Load() {
		return nil
	}

	select {
	case i.done <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start starts Indexer.
func (i *Indexer) Start(ctx context.Context) {
	if i.started.Load() {
		return
	}

	go i.start(ctx)

	i.started.Store(true)
}

// Err returns error
// if any of that returned while collecting.
func (i *Indexer) Err() error {
	return i.err
}

func (i *Indexer) start(ctx context.Context) {
	t := time.NewTicker(i.interval)
	defer t.Stop()
	defer i.started.Store(false)

	errs := make(chan error)

	for {
		select {
		case t := <-t.C:
			go func() {
				err := i.index(ctx, t)
				if err != nil {
					errs <- err
					return
				}
			}()
		case <-i.done:
			return
		case <-ctx.Done():
			if i.err == nil {
				i.err = ctx.Err()
			}
			return
		case err := <-errs:
			i.err = err
			return
		}
	}
}

const bitSize = 64

func (i *Indexer) index(ctx context.Context, t time.Time) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	prices, err := i.collecter.Collect(ctx)
	if err != nil {
		return err
	}

	for _, price := range prices {
		p, err := strconv.ParseFloat(price.Price, bitSize)
		if err != nil {
			return err
		}

		if _, ok := i.avgs[price.Ticker]; !ok {
			i.avgs[price.Ticker] = &avg{
				num: 1,
				sum: p,
			}
			continue
		}

		i.avgs[price.Ticker] = i.avgs[price.Ticker].Add(p)
	}

	for k, v := range i.avgs {
		i.handle(ticker.Price{
			Ticker: k,
			Time:   t,
			Price:  strconv.FormatFloat(v.Average(), 'f', -1, bitSize),
		})
	}

	return nil
}

type avg struct {
	sum float64
	num float64
}

func (a *avg) Add(b float64) *avg {
	a.sum += math.Abs(b)
	a.num++

	return a
}

func (a avg) Average() float64 {
	return a.sum / a.num
}
