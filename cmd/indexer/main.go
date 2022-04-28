package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/sschiz/indexer"
	"github.com/sschiz/indexer/collecter"
	"github.com/sschiz/indexer/stream"
	"github.com/sschiz/indexer/ticker"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var src source
	priceCh, errCh := src.SubscribePriceStream(ticker.BTCUSDTicker)

	s, err := stream.NewChanStream(priceCh, errCh)
	if err != nil {
		panic(err)
	}

	col := collecter.NewStreamCollecter([]stream.Stream{s})

	idxer, err := indexer.NewIndexer(col, func(p ticker.Price) {
		fmt.Printf("ticker = %s\ntimestamp = %d\nindex = %s\n\n\n",
			p.Ticker, p.Time.Unix(), p.Price)
	}, time.Second)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	idxer.Start(ctx)

	time.Sleep(10 * time.Second)

	idxer.Stop(ctx)

	if err = idxer.Err(); err != nil {
		panic(err)
	}
}

type source struct{}

const (
	min = 1
	max = 1000
)

func (s *source) SubscribePriceStream(t ticker.Ticker) (<-chan ticker.Price, <-chan error) {
	errs := make(chan error)
	prices := make(chan ticker.Price)

	go func() {
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()

		for t := range tick.C {
			randomDecimal := min + rand.Float64()*(max-min)

			prices <- ticker.Price{
				Ticker: ticker.BTCUSDTicker,
				Time:   t,
				Price:  strconv.FormatFloat(randomDecimal, 'f', -1, 64),
			}
		}
	}()

	return prices, errs
}
