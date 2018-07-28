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
	"fmt"
	"strings"
	"time"
	"sync"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"gitlab.com/crankykernel/cryptoxscanner/pkg/db"
)

type TradeStream struct {
	subscribers map[chan binance.StreamAggTrade]bool
	lock        sync.RWMutex
	cache       *db.GenericCache
}

func NewTradeStream() *TradeStream {
	tradeStream := &TradeStream{
		subscribers: map[chan binance.StreamAggTrade]bool{},
	}
	cache, err := db.OpenGenericCache("binance-cache")
	if err != nil {
		log.WithError(err).Errorf("Failed to open generic cache for Binances trades.")
	}
	tradeStream.cache = cache

	return tradeStream
}

func (b *TradeStream) Subscribe() chan binance.StreamAggTrade {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan binance.StreamAggTrade)
	b.subscribers[channel] = true
	return channel
}

func (b *TradeStream) Unsubscribe(channel chan binance.StreamAggTrade) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.subscribers, channel)
}

func (b *TradeStream) RestoreFromCache(channel chan *binance.StreamAggTrade) {
	i := int64(0)
	start := time.Now()
	first := time.Time{}
	last := time.Time{}

	rows, err := b.cache.QueryAgeLessThan("trade", 3600 * 2)
	if err != nil {
		log.WithError(err).Error("Failed to restore trades from database.")
	} else {
		count := 0
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

			if first.IsZero() {
				first = aggTrade.Timestamp()
			}
			last = aggTrade.Timestamp()

			channel <- aggTrade

			count += 1
		}
		log.Infof("Restored %d Binance trades from database.", count)
	}

	restoreDuration := time.Now().Sub(start)
	restoreRange := last.Sub(first)
	log.Printf("binance trades: restored %d trades in %v; range=%v\n",
		i, restoreDuration, restoreRange)

	channel <- nil
}

func (b *TradeStream) Run() {

	cacheChannel := make(chan *binance.StreamAggTrade)
	tradeChannel := make(chan *binance.StreamAggTrade)

	go b.RestoreFromCache(cacheChannel)

	go func() {
		for {
			// Get the streams to subscribe to.
			var streams []string
			for {
				var err error
				streams, err = b.GetStreams()
				if err != nil {
					log.Printf("binance: failed to get streams: %v", err)
					goto TryAgain
				}
				if len(streams) == 0 {
					log.Printf("binance: got 0 streams, trying again")
					goto TryAgain
				}
				log.Printf("binance: got %d streams\n", len(streams))
				break
			TryAgain:
				time.Sleep(1 * time.Second)
			}

			tradeStream := NewStreamClient("aggTrades", streams...)
			log.Printf("binance: connecting to trade stream.")
			tradeStream.Connect()

			// Read loop.
		ReadLoop:
			for {
				body, err := tradeStream.ReadNext()
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

	cacheDone := false
	tradeQueue := []*binance.StreamAggTrade{}
	for {
		select {
		case trade := <-cacheChannel:
			if trade == nil {
				cacheDone = true
			} else {
				if cacheDone {
					log.Printf("warning: got cached trade in state Cache done\n")
				}
				b.Publish(trade)
			}
		case trade := <-tradeChannel:
			if !cacheDone {
				// The Cache is still being processed. Queue.
				tradeQueue = append(tradeQueue, trade)
				continue
			}

			if len(tradeQueue) > 0 {
				log.Printf("binace trade stream: submitting %d queued trades\n",
					len(tradeQueue))
				for _, trade := range tradeQueue {
					b.Publish(trade)
				}
				tradeQueue = []*binance.StreamAggTrade{}
			}
			b.Publish(trade)
		}
	}

	log.Printf("binance: trade feed exiting.\n")
}

func (b *TradeStream) Publish(trade *binance.StreamAggTrade) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for subscriber := range b.subscribers {
		subscriber <- *trade
	}
}

func (b *TradeStream) DecodeTrade(body []byte) (*binance.StreamAggTrade, error) {
	streamEvent, err := binance.DecodeRawStreamMessage(body)
	if err != nil {
		return nil, err
	}
	return streamEvent.AggTrade, nil
}

func (b *TradeStream) GetStreams() ([]string, error) {
	symbols, err := binance.NewAnonymousClient().GetAllSymbols()
	if err != nil {
		return nil, nil
	}
	streams := []string{}
	for _, symbol := range symbols {
		streams = append(streams,
			fmt.Sprintf("%s@aggTrade", strings.ToLower(symbol)))
	}

	return streams, nil
}
