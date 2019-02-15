// The MIT License (MIT)
//
// Copyright (c) 2018-2019 Cranky Kernel
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use, copy,
// modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package binance

import (
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/cryptoxscanner/db"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"strings"
	"sync"
	"time"
)

type StreamAggTrade = binanceapi.StreamAggTrade

type tradeStreamSubscriberQueue []binanceapi.StreamAggTrade

type TradeStream struct {
	subscribers map[chan binanceapi.StreamAggTrade]tradeStreamSubscriberQueue
	lock        sync.RWMutex
	cache       *db.GenericCache
}

func NewTradeStream() *TradeStream {
	tradeStream := &TradeStream{
		subscribers: map[chan binanceapi.StreamAggTrade]tradeStreamSubscriberQueue{},
	}
	cache, err := db.OpenGenericCache("binance-cache")
	if err != nil {
		log.WithError(err).Errorf("Failed to open generic cache for Binances trades.")
	} else {
		tradeStream.cache = cache
	}

	return tradeStream
}

func (b *TradeStream) Subscribe() chan binanceapi.StreamAggTrade {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan binanceapi.StreamAggTrade, 1024)
	b.subscribers[channel] = tradeStreamSubscriberQueue{}
	return channel
}

func (b *TradeStream) Unsubscribe(channel chan binanceapi.StreamAggTrade) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.subscribers, channel)
}

func (b *TradeStream) RestoreCache(cb func(*binanceapi.StreamAggTrade)) {
	rows, err := b.cache.QueryAgeLessThan("trade", 3600*2)
	if err != nil {
		log.WithError(err).Error("Failed to restore trades from database.")
	} else {
		trades := [][]byte{}
		for rows.Next() {
			var data []byte
			if err := rows.Scan(&data); err != nil {
				log.WithError(err).Error("Failed to scan row from database.")
				continue
			}
			trades = append(trades, data)
		}

		for _, data := range trades {
			aggTrade, err := b.DecodeTrade(data)
			if err != nil {
				log.WithError(err).
					WithField("data", string(data)).
					Error("Failed to decode trade from database.")
				continue
			}
			cb(aggTrade)
		}
	}
}

func (b *TradeStream) Run() {
	tradeChannel := make(chan *binanceapi.StreamAggTrade)

	go func() {
		for {
			// Get the symbols to subscribe to.
			var symbols []string
			for {
				var err error
				symbols, err = b.GetSymbols()
				if err != nil {
					log.Printf("binance: failed to get streams: %v", err)
					goto SymbolsTryAgain
				}
				if len(symbols) == 0 {
					log.Printf("binance: got 0 streams, trying again")
					goto SymbolsTryAgain
				}
				log.Printf("binance: got %d streams\n", len(symbols))
				break
			SymbolsTryAgain:
				time.Sleep(1 * time.Second)
			}

			combinedStreamBuilder := binanceapi.NewCombinedStreamBuilder()
			for _, symbol := range symbols {
				combinedStreamBuilder.SubscribeAggTrade(symbol)
			}

			// Connect.
			connect := func() *binanceapi.Stream {
				for {
					stream, err := combinedStreamBuilder.Connect()
					if err != nil {
						log.Errorf("Failed to connect to Binance trade streams: %v", err)
						time.Sleep(time.Second * 1)
						continue
					} else {
						return stream
					}
				}
			}

			log.Printf("binance: connecting to trade stream.")
			tradeStream := connect()

			// Read loop.
		ReadLoop:
			for {
				body, err := tradeStream.Next()
				if err != nil {
					log.Printf("binance: trade feed read error: %v\n", err)
					break ReadLoop
				}

				trade, err := b.DecodeTrade(body)
				if err != nil {
					log.Printf("binance: failed to decode trade feed: %v\n", err)
					goto ReadLoop
				}

				b.cache.AddItem(trade.Timestamp(), "trade", body)

				tradeChannel <- trade
			}

		}
	}()

	for {
		select {
		case trade := <-tradeChannel:
			b.Publish(trade)
		}
	}

	log.Printf("binance: trade feed exiting.\n")
}

func (b *TradeStream) Publish(trade *binanceapi.StreamAggTrade) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for channel, queue := range b.subscribers {

		// Process queued items.
		for len(queue) > 0 {
			next := queue[0]
			select {
			case channel <- next:
				queue = queue[1:]
			default:
				goto Next
			}
		}

		select {
		case channel <- *trade:
		default:
			queue = append(queue, *trade)
		}
	Next:
		b.subscribers[channel] = queue
	}
}

func (b *TradeStream) DecodeTrade(body []byte) (*binanceapi.StreamAggTrade, error) {
	streamEvent, err := binanceapi.DecodeCombinedStreamMessage(body)
	if err != nil {
		return nil, err
	}
	return streamEvent.AggTrade, nil
}

func (b *TradeStream) GetSymbols() ([]string, error) {
	prices, err := (&binanceapi.RestClient{}).GetPriceTickerAll()
	if err != nil {
		log.Errorf("Failed to get all prices for symbol list: %v", err)
	}
	symbols := []string{}
	for _, price := range prices {
		symbols = append(symbols, strings.ToLower(price.Symbol))
	}
	return symbols, nil
}
