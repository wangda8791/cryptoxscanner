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
	"gitlab.com/crankykernel/cryptoxscanner/pkg"
	"time"
	"log"
	"gitlab.com/crankykernel/cryptotrader/binance"
)

type TickerStream struct {
	Cache *pkg.RedisInputCache
}

func NewTickerStream() *TickerStream {
	tickerStream := &TickerStream{}
	cache := pkg.NewRedisInputCache("binance")
	if err := cache.Ping(); err != nil {
		log.Printf("Redis not available. Tickers will not be cached.")
	} else {
		tickerStream.Cache = cache
	}

	return tickerStream
}

func (s *TickerStream) Run(channel chan []pkg.CommonTicker) {
	inChannel := make(chan *binance.CombinedStreamMessage)
	go NewStreamClient("binance.ticker", "!ticker@arr").Run(inChannel)
	for {
		streamMessage := <-inChannel
		if s.Cache != nil {
			s.CacheAdd(streamMessage.Bytes)
			s.PruneCache()
		}
		channel <- s.TransformTickers(streamMessage.Tickers)
	}
}

func (s *TickerStream) CacheAdd(body []byte) {
	s.Cache.RPush(body)
}

func (s *TickerStream) PruneCache() {
	for {
		next, err := s.Cache.GetFirst()
		if err != nil {
			log.Printf("error: binance ticker stream: failed to read from redis: %v\n", err)
			break
		}
		if time.Now().Sub(time.Unix(next.Timestamp, 0)) > time.Hour {
			s.Cache.LRemove()
		} else {
			break
		}
	}
}

func (s *TickerStream) TransformTickers(inTickers []binance.Stream24Ticker) []pkg.CommonTicker {
	tickers := []pkg.CommonTicker{}
	for _, rawTicker := range inTickers {
		tickers = append(tickers, pkg.CommonTickerFromBinanceTicker(rawTicker))
	}
	return tickers
}

func (s *TickerStream) DecodeTickers(buf []byte) ([]pkg.CommonTicker, error) {
	message, err := binance.DecodeRawStreamMessage(buf)
	if err != nil {
		return nil, err
	}

	tickers := []pkg.CommonTicker{}

	if len(message.Tickers) > 0 {
		for _, rawTicker := range message.Tickers {
			tickers = append(tickers,
				pkg.CommonTickerFromBinanceTicker(rawTicker))
		}
	}

	return tickers, nil
}
