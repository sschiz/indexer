# Indexer

Indexer is a simple library that calculates price index.

## Entities

The library has three main entities:
* Stream
* Collecter
* Indexer

### Stream

Stream is an interface that streams price.

The library has ChanStream that is implementation of the Stream. ChanStream has error and price channels. The error channel serves to get errors. The price channel serves to get price on the basis of which the index will be calculated in the future.

### Collecter

Collecter is an interface that collects a prices from a streams.

The library has StreamCollecter that collects prices from [Stream](#stream). When the Collect method is called, StreamCollecter receives the data of all the Streams that were passed in the constructor.

### Indexer

Indexer is at the head of the corner. It is engaged in the management of all data and the calculation of indexes. In its constructor, you need to specify the [Collecter](#collecter), Handler and the interval with which data will be collected.
Handler is a payload in which the user specifies exactly what to do with the calculated index.

## Example

```go
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

```

## Docs
See https://pkg.go.dev/github.com/sschiz/indexer
