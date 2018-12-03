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

type StreamAggTrade = binance.StreamAggTrade

type tradeStreamSubscriberQueue []binance.StreamAggTrade

type TradeStream struct {
	subscribers map[chan binance.StreamAggTrade]tradeStreamSubscriberQueue
	lock        sync.RWMutex
	cache       *db.GenericCache
}

func NewTradeStream() *TradeStream {
	tradeStream := &TradeStream{
		subscribers: map[chan binance.StreamAggTrade]tradeStreamSubscriberQueue{},
	}
	cache, err := db.OpenGenericCache("binance-cache")
	if err != nil {
		log.WithError(err).Errorf("Failed to open generic cache for Binances trades.")
	} else {
		tradeStream.cache = cache
	}

	return tradeStream
}

func (b *TradeStream) Subscribe() chan binance.StreamAggTrade {
	b.lock.Lock()
	defer b.lock.Unlock()
	channel := make(chan binance.StreamAggTrade, 1024)
	b.subscribers[channel] = tradeStreamSubscriberQueue{}
	return channel
}

func (b *TradeStream) Unsubscribe(channel chan binance.StreamAggTrade) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.subscribers, channel)
}

func (b *TradeStream) RestoreCache(cb func(*binance.StreamAggTrade)) {
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
	tradeChannel := make(chan *binance.StreamAggTrade)

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

	for {
		select {
		case trade := <-tradeChannel:
			b.Publish(trade)
		}
	}

	log.Printf("binance: trade feed exiting.\n")
}

func (b *TradeStream) Publish(trade *binance.StreamAggTrade) {
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
				goto Next;
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
