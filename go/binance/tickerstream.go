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
	"encoding/json"
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/cryptoxscanner/db"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"sync"
	"time"
)

type TickerStream struct {
	subscribers map[chan []binanceapi.TickerStreamMessage][][]binanceapi.TickerStreamMessage
	cache       *db.GenericCache
	lock        sync.RWMutex
}

func NewTickerStream() *TickerStream {
	tickerStream := &TickerStream{
		subscribers: map[chan []binanceapi.TickerStreamMessage][][]binanceapi.TickerStreamMessage{},
	}
	cache, err := db.OpenGenericCache("binance-cache")
	if err != nil {
		log.WithError(err).Errorf("Failed to open generic cache for Binance tickers.")
	} else {
		tickerStream.cache = cache
	}

	return tickerStream
}

func (b *TickerStream) Subscribe() chan []binanceapi.TickerStreamMessage {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan []binanceapi.TickerStreamMessage)
	b.subscribers[channel] = [][]binanceapi.TickerStreamMessage{}
	return channel
}

func (b *TickerStream) Unsubscribe(channel chan []binanceapi.TickerStreamMessage) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.subscribers, channel)
}

func (b *TickerStream) Publish(tickers []binanceapi.TickerStreamMessage) {
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
	go func() {
	Reconnect:
		allTickerStream, err := binanceapi.OpenAllMarketTickerStream()
		if err != nil {
			log.Errorf("Failed to open all market ticker stream: %v", err)
			time.Sleep(1 * time.Second)
			goto Reconnect
		}
		for {
			body, err := allTickerStream.Next()
			if err != nil {
				log.Errorf("Failed to read next message from ticker stream: %v", err)
				goto Reconnect
			}
			var tickers []binanceapi.TickerStreamMessage
			if err := json.Unmarshal(body, &tickers); err != nil {
				log.Errorf("Failed to decode ticker stream: %v", err)
			} else {
				s.CacheAdd(body)
				s.Publish(tickers)
			}
		}
	}()
}

func (s *TickerStream) CacheAdd(body []byte) {
	s.cache.AddItem(time.Now(), "ticker", body)
}

func (s *TickerStream) DecodeTickers(buf []byte) ([]binanceapi.TickerStreamMessage, error) {
	message, err := binanceapi.DecodeAllMarketTickerStream(buf)
	if err != nil {
		message, err := binanceapi.DecodeCombinedStreamMessage(buf)
		if err != nil {
			return nil, err
		}
		return message.Tickers, nil
	}
	return message, nil
}

func (b *TickerStream) LoadCache() [][]binanceapi.TickerStreamMessage {
	tickers := [][]binanceapi.TickerStreamMessage{}

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
