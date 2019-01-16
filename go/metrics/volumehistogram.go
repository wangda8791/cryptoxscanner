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

package metrics

import (
	"gitlab.com/crankykernel/cryptoxscanner/binance"
	"time"
)

const Buckets = 60

type VolumeHistogramCalculator struct {
	// Trade count.
	TradeCount [Buckets]uint64
	// Sell trade count.
	SellTradeCount [Buckets]uint64
	// Buy trade count.
	BuyTradeCount [Buckets]uint64
	// Volume in quote currency.
	Volume [Buckets]float64
	// Sell volume in quote currency.
	SellVolume [Buckets]float64
	// Buy volume in quote currency.
	BuyVolume [Buckets]float64
}

func NewVolumeHistogramCalculator() VolumeHistogramCalculator {
	return VolumeHistogramCalculator{}
}

func (v *VolumeHistogramCalculator) AddTrade(trade *binance.StreamAggTrade) {
	now := time.Now()
	age := now.Sub(trade.Timestamp())
	bucket := int(age.Truncate(time.Minute).Minutes())
	if bucket < Buckets {
		v.TradeCount[bucket] += 1
		v.Volume[bucket] += trade.QuoteQuantity()

		if trade.BuyerMaker {
			v.SellVolume[bucket] += trade.QuoteQuantity()
			v.SellTradeCount[bucket] += 1
		} else {
			v.BuyVolume[bucket] += trade.QuoteQuantity()
			v.BuyTradeCount[bucket] += 1
		}
	}
}
