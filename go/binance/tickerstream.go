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
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/cryptoxscanner/db"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"sync"
	"time"
)

type StreamTicker24 = binance.StreamTicker24

type TickerStream struct {
	subscribers map[chan []binance.StreamTicker24][][]binance.StreamTicker24
	cache       *db.GenericCache
	lock        sync.RWMutex
}

func NewTickerStream() *TickerStream {
	tickerStream := &TickerStream{
		subscribers: map[chan []binance.StreamTicker24][][]binance.StreamTicker24{},
	}
	cache, err := db.OpenGenericCache("binance-cache")
	if err != nil {
		log.WithError(err).Errorf("Failed to open generic cache for Binance tickers.")
	} else {
		tickerStream.cache = cache
	}

	return tickerStream
}

func (b *TickerStream) Subscribe() chan []binance.StreamTicker24 {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan []binance.StreamTicker24)
	b.subscribers[channel] = [][]binance.StreamTicker24{}
	return channel
}

func (b *TickerStream) Unsubscribe(channel chan []binance.StreamTicker24) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.subscribers, channel)
}

func (b *TickerStream) Publish(tickers []binance.StreamTicker24) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for channel, queue := range b.subscribers {
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
		case channel <- tickers:
		default:
			queue = append(queue, tickers)
		}
	Next:
		b.subscribers[channel] = queue
	}
}

func (s *TickerStream) Run() {
	inChannel := make(chan *binance.CombinedStreamMessage)
	go NewStreamClient("binance.ticker", "!ticker@arr").Run(inChannel)
	for {
		streamMessage := <-inChannel
		s.CacheAdd(streamMessage.Bytes)
		s.Publish(streamMessage.Tickers)
	}
}

func (s *TickerStream) CacheAdd(body []byte) {
	s.cache.AddItem(time.Now(), "ticker", body)
}

func (s *TickerStream) DecodeTickers(buf []byte) ([]binance.StreamTicker24, error) {
	message, err := binance.DecodeStreamMessage(buf)
	if err != nil {
		return nil, err
	}

	return message.Tickers, nil
}

func (b *TickerStream) LoadCache() [][]binance.StreamTicker24 {
	tickers := [][]binance.StreamTicker24{}

	rows, err := b.cache.QueryAgeLessThan("ticker", 3600)
	if err != nil {
		log.WithError(err).Errorf("Failed to query ticker cache.")
	} else {
		entries := [][]byte{}
		for rows.Next() {
			var data []byte
			if err := rows.Scan(&data); err != nil {
				log.WithError(err).Errorf("Failed to scan row.")
				continue
			}
			entries = append(entries, data)
		}

		for _, ticker := range entries {
			decoded, err := b.DecodeTickers(ticker)
			if err != nil {
				log.WithError(err).Errorf("Failed to decode Binance ticker.")
				continue
			}
			if len(decoded) == 0 {
				log.Warnf("Decoded Binance ticker contains 0 items.")
				continue
			}
			tickers = append(tickers, decoded)
		}
	}

	return tickers
}
