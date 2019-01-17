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

import {Component, Input, OnChanges, OnInit} from '@angular/core';

@Component({
    selector: 'app-orderbook',
    templateUrl: './orderbook.component.html',
    styleUrls: ['./orderbook.component.scss']
})
export class OrderbookComponent implements OnInit, OnChanges {

    private displayCount = 20;

    @Input("bids") _bids: any[] = [];
    @Input("asks") _asks: any[] = [];

    averageBidAmount: number = 0;
    averageAskAmount: number = 0;

    bids: any[] = [];
    asks: any[] = [];

    ngOnInit() {
    }

    ngOnChanges() {
        this.bids = this._bids.slice(0, this.displayCount);
        this.asks = this._asks.slice(0, this.displayCount);

        let totalBids = 0;
        for (let i = 0, n = this.bids.length - 1; i < n; i++) {
            totalBids += +this.bids[i][1];
        }
        this.averageBidAmount = totalBids / this.bids.length;

        let totalAsks = 0;
        for (let i = 0, n = this.asks.length - 1; i < n; i++) {
            totalAsks += +this.asks[i][1];
        }
        this.averageAskAmount = totalAsks / this.asks.length;
    }

}
