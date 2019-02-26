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

package server

import (
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/cryptoxscanner/binance"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"runtime"
	"sync"
	"time"
)

type BinanceRunner struct {
	trackers          *TickerTrackerMap
	symbolSubscribers map[string]map[chan interface{}]bool
	subscribers       map[chan *TickerTrackerMap]bool
	tickerStream      *binance.TickerStream

	Cached    TickerTrackerMap
	CacheLock sync.RWMutex
}

func NewBinanceRunner() *BinanceRunner {
	feed := BinanceRunner{
		trackers:    NewTickerTrackerMap(),
		subscribers: map[chan *TickerTrackerMap]bool{},
	}
	return &feed
}

func (b *BinanceRunner) Subscribe() chan *TickerTrackerMap {
	channel := make(chan *TickerTrackerMap, 1)
	b.subscribers[channel] = true
	return channel
}

func (b *BinanceRunner) Unsubscribe(channel chan *TickerTrackerMap) {
	if _, exists := b.subscribers[channel]; exists {
		delete(b.subscribers, channel)
	}
}

func (b *BinanceRunner) SubscribeSymbol(symbol string) chan interface{} {
	channel := make(chan interface{})
	if b.symbolSubscribers == nil {
		b.symbolSubscribers = map[string]map[chan interface{}]bool{}
	}
	if b.symbolSubscribers[symbol] == nil {
		b.symbolSubscribers[symbol] = map[chan interface{}]bool{}
	}
	b.symbolSubscribers[symbol][channel] = true
	return channel
}

func (b *BinanceRunner) UnsubscribeSymbol(symbol string, channel chan interface{}) {
	if b.symbolSubscribers[symbol] != nil {
		if _, exists := b.symbolSubscribers[symbol][channel]; exists {
			delete(b.symbolSubscribers[symbol], channel)
		}
	}
}

func (b *BinanceRunner) Run() {
	// Create and start the trade stream.
	binanceTradeStream := binance.NewTradeStream()
	go binanceTradeStream.Run()

	// SubscribeSymbol to the trade stream. This will start queuing trades
	// until the cache is done loading.
	tradeChannel := binanceTradeStream.Subscribe()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		count := 0
		binanceTradeStream.RestoreCache(func(trade *binanceapi.StreamAggTrade) {
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
					if ticker.Timestamp().After(lastServerTickerTimestamp) {
						lastServerTickerTimestamp = ticker.Timestamp()
					}
				}

				b.updateTrackers(b.trackers, tickers, true)

				for key := range b.trackers.Trackers {
					count := len(b.symbolSubscribers[key])
					if count == 0 {
						continue
					}
					message := WsBuildCompleteEntry(b.trackers.Trackers[key])
					for subscriber := range b.symbolSubscribers[key] {
						subscriber <- message
					}
				}

				for subscriber := range b.subscribers {
					select {
					case subscriber <- b.trackers:
					default:
						log.Warnf("warning: failed to send trackers to subscriber")
					}
				}

				b.CacheLock.Lock()
				b.Cached = *b.trackers
				b.CacheLock.Unlock()

				now := time.Now()
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

func (b *BinanceRunner) GetCache() TickerTrackerMap {
	b.CacheLock.RLock()
	defer b.CacheLock.RUnlock()
	return b.Cached
}

func (b *BinanceRunner) updateTrackers(trackers *TickerTrackerMap, tickers []binanceapi.TickerStreamMessage, recalculate bool) {
	channel := make(chan binanceapi.TickerStreamMessage)
	wg := sync.WaitGroup{}

	handler := func() {
		count := 0
		for {
			ticker := <-channel
			if ticker.EventTime == 0 {
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

func (b *BinanceRunner) reloadStateFromRedis(trackers *TickerTrackerMap) {
	log.Infof("Restoring Binance ticks from cache.")
	restoreCount := 0

	cachedTickers := b.tickerStream.LoadCache()
	for _, cachedTicker := range cachedTickers {
		b.updateTrackers(trackers, cachedTicker, false)
		restoreCount += 1
	}

	log.Infof("Restored %d Binance ticks from cache.", restoreCount)
}
