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

package kucoin

import (
	"gitlab.com/crankykernel/cryptotrader/kucoin"
	"gitlab.com/crankykernel/cryptoxscanner/pkg"
	"encoding/json"
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"gitlab.com/crankykernel/cryptoxscanner/pkg/db"
	"time"
)

type TickerStream struct {
	client   *kucoin.Client
	cache    *db.GenericCache
}

func NewTickerStream() (*TickerStream) {
	cache, err := db.OpenGenericCache("kucoin")
	if err != nil {
		log.WithError(err).Errorf("Failed to open cache: kucoin.")
	}
	return &TickerStream{
		client:   kucoin.NewAnonymousClient(),
		cache:    cache,
	}
}

func (t *TickerStream) GetTickers() ([]pkg.CommonTicker, error) {
	response, err := t.client.GetTick()
	if err != nil {
		return nil, err
	}
	t.Cache(response)
	return t.toCommonTicker(response), nil
}

func (t *TickerStream) toCommonTicker(tickers *kucoin.TickResponse) []pkg.CommonTicker {
	common := []pkg.CommonTicker{}
	for _, entry := range tickers.Entries {
		ticker := pkg.CommonTickerFromKuCoinTicker(entry)
		common = append(common, ticker)
	}
	return common
}

func (t *TickerStream) Cache(tickers *kucoin.TickResponse) {
	t.cache.AddItem(tickers.GetTimestamp(), "tickers", []byte(tickers.Raw))
}

func (k *TickerStream) ReplayCache(cb func(tickers []pkg.CommonTicker)) {
	start := time.Now()
	count := 0
	rows, err := k.cache.QueryAgeLessThan("tickers", 3600)
	if err != nil {
		log.WithError(err).Errorf("Failed to load cached tickers for KuCoin.")
		return
	}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			log.WithError(err).Errorf("Failed to scan row.")
			continue
		}

		var response kucoin.TickResponse
		if err := json.Unmarshal(data, &response); err != nil {
			log.WithError(err).Errorf("Failed to unmarshal cached KuCoin ticker.")
			continue
		}
		cb(k.toCommonTicker(&response))

		count += 1
	}

	log.WithFields(log.Fields{
		"duration": time.Now().Sub(start),
		"count": count,
	}).Infof("KuCoin ticker cache reload complete.")
}
