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

import {AfterViewInit, Component, OnDestroy, OnInit} from '@angular/core';
import {HttpClient, HttpParams} from '@angular/common/http';
import {ActivatedRoute, Router} from '@angular/router';
import {ScannerApiService, SymbolUpdate} from '../../scanner-api.service';
import {Subscription} from 'rxjs/Subscription';

import {BinanceApiService, Kline, KlineInterval} from '../../binance-api.service';
import {Chart} from "chart.js";

import * as Mousetrap from "mousetrap";
import * as $ from "jquery";

declare var TradingView: any;

interface Trade {
    price: number;
    quantity: number;
    timestamp: Date;
    buyerMaker: boolean;
}

class DepthChart {

    private element: HTMLCanvasElement = null;
    private ctx: CanvasRenderingContext2D = null;
    private chart: any = null;
    private labels: any[] = [];
    private bids: any[] = [];
    private asks: any[] = [];
    private options: any = {};

    constructor(elementId: string) {
        this.element = <HTMLCanvasElement>document.getElementById(elementId);
        this.ctx = this.element.getContext("2d");

        this.options = {
            type: 'line',
            data: {
                labels: this.labels,
                datasets: [
                    {
                        data: this.bids,
                        borderWidth: 1,
                        backgroundColor: "green",
                        lineTension: 0,
                    },
                    {
                        data: this.asks,
                        borderWidth: 1,
                        backgroundColor: "red",
                        lineTension: 0,
                    }
                ]
            },
            options: {
                legend: {
                    display: false,
                },
                tooltips: {
                    enabled: false,
                },
                responsive: true,
                elements: {
                    point: {
                        radius: 0,
                    }
                },
                scales: {
                    yAxes: [{
                        gridLines: {
                            drawBorder: false,
                            display: false,
                            tickMarkLength: 0,
                        },
                        ticks: {
                            display: false,
                            maxTicksLimit: 1,
                        }
                    }],
                    xAxes: [{
                        gridLines: {
                            drawBorder: false,
                            display: false,
                            tickMarkLength: 0,
                        },
                        ticks: {
                            display: false,
                            maxTicksLimit: 1,
                        }
                    }]
                },
            }
        };

        this.chart = new Chart(this.ctx, this.options);
    }

    public update(asks: any[], bids: any[]) {

        // If we're hidden, just return. Each update contains the whole order book
        // to a certain depth, its not incremental.
        if (document.hidden) {
            return;
        }


        // Reset.
        this.labels.length = 0;
        this.bids.length = 0;
        this.asks.length = 0;

        let bidSum = 0;
        for (let i = 0; i < bids.length; i++) {
            const price: number = +bids[i][0];
            const amount: number = +bids[i][1];
            bidSum += amount;
            this.asks.unshift(0);
            this.bids.unshift(bidSum);
            this.labels.unshift(price);
        }

        // Asks don't need reordering, process all at once.
        let askSum: number = 0;
        for (let i = 0, n = asks.length; i < n; i++) {
            const price: number = +asks[i][0];
            const amount: number = +asks[i][1];
            askSum += amount;
            this.bids.push(0);
            this.asks.push(askSum);
            this.labels.push(price);
        }

        let max = Math.max(bidSum, askSum);

        this.options.options.scales.yAxes[0].ticks.max = max;

        this.redraw();
    }

    public redraw() {
        this.chart.update();
    }
}

class PriceTickerChart {

    private labels: Date[] = [];
    private data: number[] = [];
    private element: HTMLCanvasElement;
    private ctx: CanvasRenderingContext2D;
    private chart: any;

    constructor(elementId: string) {
        this.element = <HTMLCanvasElement>document.getElementById(elementId);
        this.ctx = this.element.getContext("2d");
        this.chart = new Chart(this.ctx, {
            type: 'bar',
            data: {
                labels: this.labels,
                datasets: [{
                    data: this.data,
                    type: "line",
                    borderWidth: 1,
                    backgroundColor: "darkcyan",
                }]
            },
            options: {
                legend: {
                    display: false,
                },
                tooltips: {
                    enabled: false,
                },
                responsive: true,
                scales: {
                    yAxes: [{
                        ticks: {
                            maxTicksLimit: 2,
                        },
                    }],
                    xAxes: [{
                        type: "time",
                        time: {
                            unit: "second",
                            displayFormats: {
                                second: "h:mm:ss",
                            }
                        },
                        ticks: {
                            maxRotation: 0,
                        }
                    }]
                },
            }
        });
    }

    update(price: number, timestamp: Date) {
        this.data.push(price);
        this.labels.push(timestamp);
        while (this.data.length > 60) {
            this.data.shift();
            this.labels.shift();
        }
        this.redraw();
    }

    redraw() {
        if (!document.hidden) {
            this.chart.update();
        }
    }

    reset() {
        this.labels.length = 0;
        this.data.length = 0;
        this.chart.update();
    }
}

@Component({
    templateUrl: './symbol.component.html',
    styleUrls: ['./symbol.component.scss']
})
export class BinanceSymbolComponent implements OnInit, OnDestroy, AfterViewInit {

    /** The symbol with quote asset. */
    symbol: string = "BNBBTC";

    /** The base asset. */
    baseAsset: string = null;

    exchangeSymbol: string = "";

    private binanceStream: any = null;

    symbols: string[] = [];

    trades: Trade[] = [];

    lastPrice: number = null;

    private depthChart: DepthChart = null;

    maxTradeHistory: number = 20;

    private tokenfxFeed: Subscription = null;

    lastUpdate: SymbolUpdate = null;

    binanceState: string = "connecting";

    tokenFxState: string = "connecting";

    ATR: any = {};

    showTradingViewCharts: boolean = true;

    orderBook: OrderBookTracker = new OrderBookTracker();

    priceChart: PriceTickerChart = null;

    constructor(private http: HttpClient,
                private route: ActivatedRoute,
                private router: Router,
                private tokenfx: ScannerApiService,
                private binanceApi: BinanceApiService) {
    }

    ngOnDestroy() {
        document.removeEventListener("visibilitychange", this);
        this.reset();

        $("#symbolSelectMenu").off("show.bs.dropdown");

        Mousetrap.unbind("/");
    }

    ngAfterViewInit() {
        (<any>$("[data-toggle='tooltip']")).tooltip();
        (<any>$("th")).tooltip();
    }

    private reset() {

        if (this.binanceStream) {
            console.log("Closing Binance stream.");
            this.binanceStream.close();
            this.binanceStream.closeRequested = true;
            this.binanceStream = null;
        }

        if (this.tokenfxFeed) {
            console.log("Unsubscribing from TokenFX feed.");
            this.tokenfxFeed.unsubscribe();
        }

        if (this.priceChart) {
            this.priceChart.reset();
        }

        this.orderBook = new OrderBookTracker();

        this.ATR = {};
    }

    toggleSymbolDropdown() {
        if ($("#symbolSelectDropdownMenu").hasClass("show")) {
            $("#symbolSelectDropdownMenu").removeClass("show");
        } else {
            $("#symbolSelectDropdownMenu").addClass("show");
            setTimeout(() => {
                $("#symbolFilterInput").focus();
            }, 0);
        }
    }

    ngOnInit() {
        this.priceChart = new PriceTickerChart("priceTickerChart");

        Mousetrap.bind("/", () => {
            this.toggleSymbolDropdown();
        });

        document.addEventListener("visibilitychange", this);

        // Get all symbols.
        this.http.get("/api/1/binance/proxy/api/v3/ticker/price").subscribe((response: any[]) => {
            this.symbols = response.map((ticker) => {
                return ticker.symbol;
            }).filter((item) => {
                if (item == "123456") {
                    return false;
                }
                return true;
            }).sort();
        });

        this.route.params.subscribe((params) => {
            this.symbol = params.symbol.toUpperCase();
            this.exchangeSymbol = this.symbol.replace(/BTC$/, "_BTC")
                .replace(/ETH$/, "_ETC")
                .replace(/BNB$/, "_BNB")
                .replace(/USDT$/, "_USDT");
            document.title = this.symbol.toUpperCase();
            this.reset();
            this.init();
        });
    }

    handleEvent(event: Event) {
        switch (event.type) {
            case "visibilitychange":
                if (this.priceChart) {
                    this.priceChart.redraw();
                }
                this.depthChart.redraw();
                break;
            default:
                break;
        }
    }

    changeSymbol() {
        this.router.navigate(['/binance/chart', this.symbol]);
    }

    initOrderBook() {
        const depthUrl = "/api/1/binance/proxy/api/v1/depth";
        this.http.get(depthUrl, {
            params: new HttpParams()
                .append("symbol", this.symbol.toUpperCase())
                .append("limit", "1000")
        }).subscribe((response: any) => {
            this.orderBook.initialize(response);
        });
    }

    init() {
        this.baseAsset = this.symbol
            .replace(/BTC$/, "")
            .replace(/USDT$/, "")
            .replace(/BNB$/, "")
            .replace(/ETH$/, "");
        this.trades = [];

        this.depthChart = new DepthChart("newDepthChart");

        this.start();

        for (const interval of [KlineInterval.H1, KlineInterval.D1]) {
            this.binanceApi.getKlines({
                symbol: this.symbol,
                interval: interval,
                limit: 100,
            }).subscribe((klines) => {
                const atr = this.calculateATR(klines);
                this.ATR[interval] = atr[0];
            });
        }

        this.showTradingViewCharts = false;
        setTimeout(() => {
            this.showTradingViewCharts = true;
            setTimeout(() => {
                const tv_1m = new TradingView.widget(
                    {
                        "autosize": true,
                        "symbol": "BINANCE:" + this.symbol,
                        "interval": "1",
                        "timezone": "Etc/UTC",
                        "theme": "Dark",
                        "style": "1",
                        "locale": "en",
                        "toolbar_bg": "#f1f3f6",
                        "enable_publishing": false,
                        "withdateranges": true,
                        "show_popup_button": true,
                        "popup_width": "1000",
                        "popup_height": "650",
                        "container_id": "tradingview-1m",
                    }
                );
                const tv_5m = new TradingView.widget(
                    {
                        "autosize": true,
                        "symbol": "BINANCE:" + this.symbol,
                        "interval": "5",
                        "timezone": "Etc/UTC",
                        "theme": "Dark",
                        "style": "1",
                        "locale": "en",
                        "toolbar_bg": "#f1f3f6",
                        "enable_publishing": false,
                        "withdateranges": true,
                        "show_popup_button": true,
                        "popup_width": "1000",
                        "popup_height": "650",
                        "container_id": "tradingview-5m",
                    }
                );
            }, 0);
        }, 0);

    }

    // Calculate the ATR (Average True Range). A list of ATRs is returned,
    // with the first element being the most recent ATR.
    private calculateATR(klines: Kline[], period: number = 14): number[] {
        let atr = 0;
        const atrs: number[] = [];
        const n = klines.length;
        let prev = klines[0];
        for (let i = 0; i < n; i++) {
            const kline = klines[i];
            const tr0 = kline.high - kline.low;
            const tr1 = Math.abs(kline.high - prev.close);
            const tr2 = Math.abs(kline.low - prev.close);
            const tr = Math.max(tr0, tr1, tr2);
            atr = ((atr * (period - 1) + tr)) / (period);
            prev = kline;
            atrs.push(atr);
        }
        return atrs.reverse();
    }

    rawToTrade(raw): Trade {
        return {
            timestamp: new Date(raw.E),
            price: +raw.p,
            quantity: +raw.q,
            buyerMaker: raw.m,
        };
    }

    private addTrade(trade: Trade) {
        this.trades.unshift(trade);
        while (this.trades.length > this.maxTradeHistory) {
            this.trades.pop();
        }
        this.lastPrice = this.trades[0].price;
    }

    private start() {
        this.runTokenFxSocket();
        this.runBinanceSocket();
    }

    private runTokenFxSocket() {

        const reconnect = () => {
            setTimeout(() => {
                this.runTokenFxSocket();
            }, 1000);
        };

        this.tokenfxFeed = this.tokenfx.connectBinanceSymbol(this.symbol)
            .subscribe((message: SymbolUpdate) => {
                if (message === null) {
                    // Connected.
                    this.tokenFxState = "connected";
                    return;
                }
                if (message.symbol) {
                    this.lastUpdate = message;
                    this.lastPrice = message.close;
                    if (this.priceChart) {
                        this.priceChart.update(message.close, new Date(message.timestamp));
                    }
                }
            }, (error) => {
                // Error.
                console.log("tokenfx socket error: ");
                console.log(error);
                this.tokenFxState = "errored";
                reconnect();
            }, () => {
                // Closed.
                console.log("tokenfx socket closed.");
                this.tokenFxState = "closed";
                reconnect();
            });
    }

    private runBinanceSocket() {

        const reconnect = () => {
            setTimeout(() => {
                this.runBinanceSocket();
            }, 1000);
        };

        console.log("chart: connecting to binance stream.");

        const symbolLower = this.symbol.toLowerCase();
        const streams = [
            `${symbolLower}@aggTrade`,
            `${symbolLower}@depth`
        ];

        const url = `wss://stream.binance.com:9443/stream?streams=` +
            streams.join("/") + "/";

        const ws = new WebSocket(url);

        this.binanceStream = ws;

        this.binanceStream.onopen = (event) => {
            console.log("stream opened:");
            console.log(event);
            this.binanceState = "connected";
            this.initOrderBook();
        };

        this.binanceStream.onclose = (event) => {
            console.log("stream closed:");
            console.log(event);
            this.binanceState = "closed";
            if (!(<any>ws).closeRequested) {
                reconnect();
            }
        };

        this.binanceStream.onerror = (event) => {
            console.log("stream error:");
            console.log(event);
            this.binanceState = "error";
            if (!(<any>ws).closeRequested) {
                reconnect();
            }
        };

        this.binanceStream.onmessage = (message) => {
            const data = JSON.parse(message.data);
            if (data.stream.indexOf("@aggTrade") > -1) {
                const trade = this.rawToTrade(data.data);
                this.addTrade(trade);
            } else if (data.stream.indexOf("@depth") > -1) {
                this.orderBook.update(data.data);
                this.depthChart.update(this.orderBook.asks, this.orderBook.bids);
            } else {
                console.log("Unhandled Binance stream message type: " + data.stream);
            }

        };
    }
}

interface DepthUpdate {
    E: number;
    // First update ID in event.
    U: number;
    // Final update ID in event.
    u: number;
    b: any[];
    a: any[];
}

class OrderBookTracker {

    private lastUpdateId = 0;

    private initialized: boolean = false;

    private queue: DepthUpdate[] = [];

    bids: any[] = [];

    asks: any[] = [];

    private bidMap = {};

    private askMap = {};

    private displayDepth = 100;

    initialize(orderBook) {
        this.lastUpdateId = orderBook.lastUpdateId;

        for (const bid of orderBook.bids) {
            const p = +bid[0];
            const q = +bid[1];
            this.bidMap[p] = q;
        }

        for (const ask of orderBook.asks) {
            const p = +ask[0];
            const q = +ask[1];
            this.askMap[p] = q;
        }

        while (this.queue.length > 0) {
            const update = this.queue.shift();
            if (update.u > this.lastUpdateId) {
                this.processUpdate(update);
            }
        }

        this.render();

        this.initialized = true;
    }

    private render() {
        this.bids = Object.keys(this.bidMap).sort((a, b) => {
            return (+b) - (+a);
        }).map((key) => {
            return [+key, +this.bidMap[key]];
        }).slice(0, this.displayDepth);

        this.asks = Object.keys(this.askMap).sort((a, b) => {
            return (+a) - (+b);
        }).map((key) => {
            return [+key, +this.askMap[key]];
        }).slice(0, this.displayDepth);
    }

    processUpdate(update: DepthUpdate) {
        this.lastUpdateId = update.u;

        for (const bid of update.b) {
            const p = +bid[0];
            const q = +bid[1];
            if (+q === 0) {
                delete (this.bidMap[p]);
            } else {
                this.bidMap[p] = q;
            }
        }

        for (const ask of update.a) {
            const p = +ask[0];
            const q = +ask[1];
            if (+q === 0) {
                delete (this.askMap[p]);
            } else {
                this.askMap[p] = q;
            }
        }

        this.render();
    }

    update(update: DepthUpdate) {
        if (update.u <= this.lastUpdateId) {
            return;
        }
        if (!this.initialized) {
            this.queue.push(update);
        } else {
            this.processUpdate(update);
        }
    }

}
