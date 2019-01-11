// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

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
