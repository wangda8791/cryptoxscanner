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

package server

import (
	"gitlab.com/crankykernel/cryptoxscanner/commonticker"
	"gitlab.com/crankykernel/cryptoxscanner/pkg"
	"gitlab.com/crankykernel/cryptoxscanner/pkg/binance"
	"time"
	"fmt"
	"sync"
	"runtime"
	"math"
	"gitlab.com/crankykernel/cryptoxscanner/log"
)

type BinanceRunner struct {
	trackers    *pkg.TickerTrackerMap
	websocket   *TickerWebSocketHandler
	subscribers map[string]map[chan interface{}]bool
	tickerStream *binance.TickerStream
}

func NewBinanceRunner() *BinanceRunner {
	feed := BinanceRunner{
		trackers: pkg.NewTickerTrackerMap(),
	}
	return &feed
}

func (b *BinanceRunner) Subscribe(symbol string) chan interface{} {
	channel := make(chan interface{})
	if b.subscribers == nil {
		b.subscribers = map[string]map[chan interface{}]bool{}
	}
	if b.subscribers[symbol] == nil {
		b.subscribers[symbol] = map[chan interface{}]bool{}
	}
	b.subscribers[symbol][channel] = true
	return channel
}

func (b *BinanceRunner) Unsubscribe(symbol string, channel chan interface{}) {
	if b.subscribers[symbol] != nil {
		if _, exists := b.subscribers[symbol][channel]; exists {
			delete(b.subscribers[symbol], channel)
		}
	}
}

func (b *BinanceRunner) Run() {
	lastUpdate := time.Now()

	// Create and start the trade stream.
	binanceTradeStream := binance.NewTradeStream()
	go binanceTradeStream.Run()

	// Subscribe to the trade stream. This will start queuing trades
	// until the cache is done loading.
	tradeChannel := binanceTradeStream.Subscribe()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		count := 0
		binanceTradeStream.RestoreCache(func(trade *binance.StreamAggTrade) {
			ticker := b.trackers.GetTracker(trade.Symbol)
			ticker.AddTrade(*trade)
			count += 1
		})
		log.Infof("Restored %d Binance trades from cache.", count)
		wg.Done()
	}()

	// Create, subscribe to and start the ticker stream.
	b.tickerStream = binance.NewTickerStream()
	binanceTickerChannel := b.tickerStream.Subscribe()
	go b.tickerStream.Run()

	// Restore the ticker stream cache.
	wg.Add(1)
	go func() {
		b.reloadStateFromRedis(b.trackers)
		wg.Done()
	}()

	// Wait for cache restores to complete.
	wg.Wait()

	go func() {
		tradeCount := 0
		lastTradeTime := time.Time{}
		for {
		ReadLoop:
			loopStartTime := time.Now()
			select {

			case trade := <-tradeChannel:
				ticker := b.trackers.GetTracker(trade.Symbol)
				ticker.AddTrade(trade)

				if trade.Timestamp().After(lastTradeTime) {
					lastTradeTime = trade.Timestamp()
				}

				tradeCount++

			case tickers := <-binanceTickerChannel:

				waitTime := time.Now().Sub(loopStartTime)
				if len(tickers) == 0 {
					goto ReadLoop
				}

				lastServerTickerTimestamp := time.Time{}
				for _, ticker := range tickers {
					if ticker.Timestamp.After(lastServerTickerTimestamp) {
						lastServerTickerTimestamp = ticker.Timestamp
					}
				}

				b.updateTrackers(b.trackers, tickers, true)

				// Create enhanced feed.
				message := []interface{}{}
				for key := range b.trackers.Trackers {
					tracker := b.trackers.Trackers[key]
					if tracker.LastUpdate.Before(lastUpdate) {
						continue
					}
					update := buildUpdateMessage(tracker)

					if tracker.HaveVwap {
						for i, k := range tracker.Metrics {
							update[fmt.Sprintf("vwap_%dm", i)] = pkg.Round8(k.Vwap)
						}
					}

					if tracker.HaveTotalVolume {
						for i, k := range tracker.Metrics {
							update[fmt.Sprintf("total_volume_%d", i)] = pkg.Round8(k.TotalVolume)
						}
					}

					if tracker.HaveNetVolume {
						for i, k := range tracker.Metrics {
							update[fmt.Sprintf("nv_%d", i)] = pkg.Round8(k.NetVolume);
						}
					}

					for i, k := range tracker.Metrics {
						if !math.IsNaN(k.RSI) {
							update[fmt.Sprintf("rsi_%d", i*60)] = pkg.Round8(k.RSI)
						}
					}

					message = append(message, update)

					for subscriber := range b.subscribers[key] {
						select {
						case subscriber <- update:
						default:
							log.Printf("warning: feed subscriber is blocked\n")
						}
					}
				}
				if err := b.websocket.Broadcast(&TickerStream{Tickers: &message,}); err != nil {
					log.Printf("error: broadcasting message: %v", err)
				}

				now := time.Now()
				lastUpdate = now;
				processingTime := now.Sub(loopStartTime) - waitTime
				lagTime := now.Sub(lastServerTickerTimestamp)
				tradeLag := now.Sub(lastTradeTime)

				log.Printf("binance: wait: %v; processing: %v; lag: %v; trades: %d; trade lag: %v",
					waitTime, processingTime, lagTime, tradeCount, tradeLag)
				tradeCount = 0
			}
		}
	}()
}

func (b *BinanceRunner) updateTrackers(trackers *pkg.TickerTrackerMap, tickers []commonticker.CommonTicker, recalculate bool) {
	channel := make(chan commonticker.CommonTicker)
	wg := sync.WaitGroup{}

	handler := func() {
		count := 0
		for {
			ticker := <-channel
			if ticker.Timestamp.IsZero() {
				break
			}
			count += 1
			tracker := trackers.GetTracker(ticker.Symbol)
			tracker.Update(ticker)
			if recalculate {
				tracker.Recalculate()
			}
		}
		wg.Done()
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go handler()
	}

	for _, ticker := range tickers {
		channel <- ticker
	}

	close(channel)
	wg.Wait()
}

func (b *BinanceRunner) reloadStateFromRedis(trackers *pkg.TickerTrackerMap) {
	log.Infof("Restoring Binance ticks from cache.")
	restoreCount := 0

	cachedTickers := b.tickerStream.LoadCache()
	for _, cachedTicker := range cachedTickers {
		b.updateTrackers(trackers, cachedTicker, false)
		restoreCount += 1
	}

	log.Infof("Restored %d Binance ticks from cache.", restoreCount)
}
