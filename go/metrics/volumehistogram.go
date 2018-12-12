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
