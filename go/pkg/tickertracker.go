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

package pkg

import (
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/cryptoxscanner/commonticker"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"math"
	"sync"
	"time"
)

type Aggregate struct {
	// The first moment of the period in this aggregate.
	Time time.Time

	Open  float64
	High  float64
	Low   float64
	Close float64

	// The 24 volume in the quote asset.
	QuoteVolume24 float64
}

type TickerMetrics struct {
	// Common metrics.
	PriceChangePercent  float64
	VolumeChangePercent float64
	High                float64
	Low                 float64
	Range               float64
	RangePercent        float64

	// Require trades.
	Vwap        float64
	TotalVolume float64
	NetVolume   float64
	BuyVolume   float64
	RSI         float64
}

type TickerTracker struct {
	Symbol     string
	Ticks      []*commonticker.CommonTicker
	Metrics    map[int]*TickerMetrics
	LastUpdate time.Time
	H24Metrics TickerMetrics

	// Trades, in Binance format.
	Trades []*binance.StreamAggTrade

	Aggs map[int][]Aggregate

	HaveVwap        bool
	HaveTotalVolume bool
	HaveNetVolume   bool
}

var Buckets []int

func init() {
	Buckets = []int{
		1,
		2,
		3,
		4,
		5,
		10,
		15,
		60,
	}
}

func NewTickerTracker(symbol string) *TickerTracker {
	tracker := TickerTracker{
		Symbol:  symbol,
		Ticks:   []*commonticker.CommonTicker{},
		Trades:  []*binance.StreamAggTrade{},
		Metrics: make(map[int]*TickerMetrics),
		Aggs:    make(map[int][]Aggregate),
	}

	for _, i := range Buckets {
		tracker.Metrics[i] = &TickerMetrics{}
	}

	return &tracker;
}

func (t *TickerTracker) LastTick() *commonticker.CommonTicker {
	if len(t.Ticks) == 0 {
		return nil
	}
	return t.Ticks[len(t.Ticks)-1]
}

func (t *TickerTracker) Recalculate() {
	t.CalculateTrades()
	t.CalculateTicks()

	for _, bucket := range Buckets {
		t.Metrics[bucket].RSI = t.CalculateRSI(t.Aggs[bucket])
	}
}

func (t *TickerTracker) CalculateRSI(aggs []Aggregate) (float64) {
	if aggs == nil {
		return 0
	}

	period := 14

	gains := float64(0)
	losses := float64(0)

	prev := aggs[0]

	for i, cp := range aggs {
		if i < period {
			if cp.Close < prev.Close {
				losses += (prev.Close - cp.Close)
			} else if cp.Close > prev.Close {
				gains += (cp.Close - prev.Close)
			}
			if i == period-1 {
				gains = gains / float64(period)
				losses = losses / float64(period)
			}
		} else {
			loss := float64(0)
			gain := float64(0)
			if cp.Close < prev.Close {
				loss = prev.Close - cp.Close
			} else if cp.Close > prev.Close {
				gain = cp.Close - prev.Close
			}
			losses = ((losses * 13) + loss) / 14
			gains = ((gains * 13) + gain) / 14
		}
		prev = cp
	}

	rs := gains / losses
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

func (t *TickerTracker) CalculateTicks() {
	last := t.LastTick()
	now := time.Now()
	count := len(t.Ticks)

	if count < 2 {
		return
	}

	low := last.LastPrice
	high := last.LastPrice

	for i := count - 2; i >= 0; i-- {
		tick := t.Ticks[i]
		age := now.Sub(tick.Timestamp)
		bucket := (int(age.Seconds()) / 60) + 1

		if tick.LastPrice < low {
			low = tick.LastPrice
		}
		if tick.LastPrice > high {
			high = tick.LastPrice
		}

		metrics := t.Metrics[bucket]
		if metrics == nil {
			continue
		}

		if tick.LastPrice > 0 {
			priceChange := last.LastPrice - tick.LastPrice
			priceChangePercent := Round3(priceChange / tick.LastPrice * 100)
			metrics.PriceChangePercent = priceChangePercent
		} else {
			metrics.PriceChangePercent = 0
		}

		// Volume rate of change (VROC).
		if tick.QuoteVolume > 0 {
			volumeChange := last.QuoteVolume - tick.QuoteVolume
			volumeChangePercent := Round3(volumeChange / tick.QuoteVolume * 100)
			metrics.VolumeChangePercent = volumeChangePercent
		} else {
			metrics.VolumeChangePercent = 0
		}

		metrics.High = high
		metrics.Low = low
		metrics.Range = Round8(high - low)
		if low > 0 {
			metrics.RangePercent = Round3(metrics.Range / low * 100)
		} else if high > 0 {
			// Low is 0, but high is not. Is that 100%?
			metrics.RangePercent = 100
		} else {
			metrics.RangePercent = 0
		}
	}

	// Some 24 hour metrics.
	t.H24Metrics.High = last.High
	t.H24Metrics.Low = last.Low
	t.H24Metrics.Range = Round8(last.High - last.Low)
	t.H24Metrics.RangePercent = Round3(t.H24Metrics.Range / last.Low * 100)
}

// Calculate values that depend on actual trades:
// - VWAP
// - Total volume
// - Net volume
func (t *TickerTracker) CalculateTrades() {
	count := len(t.Trades)
	if count < 1 {
		return
	}

	now := time.Now()

	t.HaveNetVolume = true
	t.HaveTotalVolume = true
	t.HaveVwap = true;
	vwapPrice := float64(0)
	vwapVolume := float64(0)
	buyVolume := float64(0)
	sellVolume := float64(0)

	for i := count - 1; i >= 0; i-- {
		trade := t.Trades[i]

		age := now.Sub(trade.Timestamp())

		if !trade.BuyerMaker {
			buyVolume += trade.QuoteQuantity()
		} else {
			sellVolume += trade.QuoteQuantity()
		}

		vwapVolume += trade.Quantity
		vwapPrice += trade.Quantity * trade.Price
		vwap := vwapPrice / vwapVolume

		totalVolume := buyVolume + sellVolume
		netVolume := buyVolume - sellVolume

		bucket := (int(age.Seconds()) / 60) + 1

		metrics := t.Metrics[bucket]
		if metrics == nil {
			continue
		}

		metrics.NetVolume = netVolume
		metrics.TotalVolume = totalVolume
		metrics.BuyVolume = buyVolume
		metrics.Vwap = vwap
	}

	t.PruneTrades(now)
}

func (t *TickerTracker) Update(ticker commonticker.CommonTicker) {
	t.LastUpdate = time.Now()
	t.Ticks = append(t.Ticks, &ticker)
	now := ticker.Timestamp
	for {
		first := t.Ticks[0]
		if now.Sub(first.Timestamp) > (time.Minute*60)+1 {
			t.Ticks = t.Ticks[1:]
		} else {
			break
		}
	}
}

func (t *TickerTracker) AddTrade(trade binance.StreamAggTrade) {
	if trade.Symbol == "" {
		log.Printf("error: not adding trade with empty symbol")
		return
	}

	if len(t.Trades) > 0 {
		lastTrade := t.Trades[len(t.Trades)-1]
		if trade.Timestamp().Before(lastTrade.Timestamp()) {
			log.Printf("error: received trade older than previous trade (symbol: %s)\n",
				t.Symbol)
		}
	}

	t.Trades = append(t.Trades, &trade)

	openTime := trade.Timestamp().Truncate(time.Minute)

	if t.Aggs[1] == nil {
		t.Aggs[1] = append(t.Aggs[1], Aggregate{
			Time:  openTime,
			Open:  trade.Price,
			Close: trade.Price,
			High:  trade.Price,
			Low:   trade.Price,
		})
	} else {
		aggs := t.Aggs[1]
		lastAgg := &aggs[len(aggs)-1]
		if lastAgg.Time == openTime {
			lastAgg.Close = trade.Price
			if trade.Price > lastAgg.High {
				lastAgg.High = trade.Price
			}
			if trade.Price < lastAgg.Low {
				lastAgg.Low = trade.Price
			}
		} else {
			nextTime := lastAgg.Time.Add(time.Minute)
			for {
				if nextTime.Before(openTime) {
					t.Aggs[1] = append(aggs, Aggregate{
						Time:  nextTime,
						Open:  lastAgg.Close,
						Close: lastAgg.Close,
						High:  lastAgg.Close,
						Low:   lastAgg.Close,
					})
					nextTime = nextTime.Add(time.Minute)
				} else {
					t.Aggs[1] = append(aggs, Aggregate{
						Time:  openTime,
						Open:  lastAgg.Close,
						Close: trade.Price,
						High:  trade.Price,
						Low:   trade.Price,
					})
					break
				}
			}
		}
	}
	m1Agg := t.Aggs[1][len(t.Aggs[1])-1]

	for _, interval := range Buckets[1:] {
		openTime := m1Agg.Time.Truncate(time.Minute * time.Duration(interval))
		aggs := t.Aggs[interval]
		if aggs == nil {
			aggs = append(aggs, Aggregate{
				Time:  openTime,
				Open:  m1Agg.Open,
				Close: m1Agg.Close,
				High:  m1Agg.High,
				Low:   m1Agg.Low,
			})
			t.Aggs[interval] = aggs
		} else {
			aggs := t.Aggs[interval]
			lastAgg := &aggs[len(aggs)-1]
			if lastAgg.Time == openTime {
				lastAgg.Close = m1Agg.Close
				if m1Agg.Close > lastAgg.High {
					lastAgg.High = m1Agg.Close
				}
				if m1Agg.Close < lastAgg.Low {
					lastAgg.Low = m1Agg.Close
				}
			} else {
				nextTime := lastAgg.Time.Add(time.Minute * time.Duration(interval))
				for {
					if nextTime.Before(openTime) {
						t.Aggs[interval] = append(aggs, Aggregate{
							Time:  nextTime,
							Open:  lastAgg.Close,
							Close: lastAgg.Close,
							High:  lastAgg.Close,
							Low:   lastAgg.Close,
						})
						nextTime = nextTime.Add(time.Minute * time.Duration(interval))
					} else {
						t.Aggs[interval] = append(aggs, Aggregate{
							Time:  openTime,
							Open:  lastAgg.Close,
							Close: m1Agg.Close,
							High:  m1Agg.High,
							Low:   m1Agg.Low,
						})
						break
					}
				}
			}
		}
	}
}

func (t *TickerTracker) PruneTrades(now time.Time) {
	chop := 0
	for i, trade := range t.Trades {
		age := now.Sub(trade.Timestamp())
		if age < time.Hour {
			break
		}
		chop = i + 1
	}
	if chop > 0 {
		t.Trades = t.Trades[chop:]
	}
}

type TickerTrackerMap struct {
	Trackers map[string]*TickerTracker
	lock     sync.RWMutex
}

func NewTickerTrackerMap() *TickerTrackerMap {
	return &TickerTrackerMap{
		Trackers: make(map[string]*TickerTracker),
	}
}

func (t *TickerTrackerMap) GetTracker(symbol string) *TickerTracker {
	if symbol == "" {
		log.Printf("GetTracker called with empty string symbol")
		return nil
	}
	t.lock.RLock()
	tracker := t.Trackers[symbol]
	if tracker != nil {
		t.lock.RUnlock()
		return tracker
	}
	t.lock.RUnlock()
	t.lock.Lock()
	defer t.lock.Unlock()
	t.Trackers[symbol] = NewTickerTracker(symbol)
	return t.Trackers[symbol]
}

func (t *TickerTrackerMap) GetLastForSymbol(symbol string) *commonticker.CommonTicker {
	if tracker, ok := t.Trackers[symbol]; ok {
		return tracker.LastTick()
	}
	return nil
}

func Round8(val float64) float64 {
	out := math.Round(val*100000000) / 100000000
	if math.IsInf(out, 0) {
		log.Printf("error: round8 output value IsInf\n")
		return 0
	}
	return out
}

func Round3(val float64) float64 {
	out := math.Round(val*1000) / 1000
	if math.IsInf(out, 0) {
		log.Printf("error: round3 output value IsInf\n")
		return 0
	}
	return out
}
