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

import * as Mousetrap from "mousetrap";
import * as $ from "jquery";

declare var TradingView: any;
declare var Highcharts: any;

interface Trade {
    price: number;
    quantity: number;
    timestamp: Date;
    buyerMaker: boolean;
}

class PriceChart {

    public maxPoints: number = 60;

    private elementId: string = null;

    private chart: any = null;

    private data: any = [];

    constructor(elementId: string) {
        this.elementId = elementId;
        this.createChart();
    }

    createChart() {
        this.chart = Highcharts.chart(this.elementId, {
            title: {
                text: null,
            },
            yAxis: {
                labels: {
                    enabled: true,
                },
                title: null,
            },
            series: [
                {
                    name: "Price",
                    decimals: 8,
                    data: [],
                },
            ],
            rangeSelector: {
                enabled: false,
            },
            navigator: {
                enabled: false,
            },
            scrollbar: {
                enabled: false,
            },
            xAxis: {
                type: "datetime",
            },
            legend: {
                enabled: false,
            },
            time: {
                useUTC: false,
            },
            credits: {
                enabled: false,
            }
        });
    }

    public update(priceUpdate: { timestamp: Date, price: number }) {
        this.data.push([
            priceUpdate.timestamp.getTime(),
            priceUpdate.price,
        ]);

        const now = new Date().getTime();
        while (this.data.length > 0) {
            const age = now - this.data[0][0];
            if (age > 60000) {
                this.data.shift();
            } else {
                break;
            }
        }

        const series = this.chart.series[0];

        // Sometimes setData fails...
        try {
            series.setData(this.data.map((e) => {
                return e;
            }), false);
            this.redraw();
        } catch (err) {
            console.log("price chart: error setting data; recreating");
            console.log(err);
            this.createChart();
        }
    }

    public redraw() {
        if (document.hidden) {
            return;
        }
        this.chart.redraw();
    }

    public destroy() {
        try {
            this.chart.destroy();
            this.chart = null;
        } catch (err) {
        }
    }
}

class DepthChart {

    private chart: any = null;

    constructor(elementId: string) {
        this.chart = Highcharts.chart(elementId, {
            chart: {
                type: "area",
            },
            title: {
                text: null,
            },
            yAxis: {
                labels: {
                    enabled: false,
                },
                title: null,
            },
            xAxis: {
                labels: {
                    enabled: false,
                },
                title: null,
            },
            series: [
                {
                    name: "Asks",
                    data: this.askData,
                    color: "red",
                },
                {
                    name: "Bids",
                    data: this.bidData,
                    color: "green",
                }
            ],
            legend: {
                enabled: false,
            },
            credits: {
                enabled: false,
            }
        });
    }

    private askData = [];
    private bidData = [];

    public update(asks: any[], bids: any[]) {

        if (document.hidden) {
            return;
        }

        this.askData = [];

        let askTotal: number = 0;
        for (let i = 0, n = asks.length; i < n; i++) {
            const price: number = +asks[i][0];
            const amount: number = +asks[i][1];
            askTotal += amount;
            this.askData.push([price, askTotal]);
        }

        this.askData = this.askData.sort((a, b) => {
            return a[0] - b[0];
        });

        this.bidData = [];
        for (let i = 0, n = bids.length; i < n; i++) {
            const price: number = +bids[i][0];
            const amount: number = +bids[i][1];
            this.bidData.push([price, amount]);
        }

        this.bidData = this.bidData.sort((a, b) => {
            return a[0] - b[0];
        });

        let bidTotal: number = 0;
        for (let i = this.bidData.length - 1; i >= 0; i--) {
            bidTotal += this.bidData[i][1];
            this.bidData[i][1] = bidTotal;
        }

        this.chart.series[0].setData(this.askData, false, false);
        this.chart.series[1].setData(this.bidData, false, false);

        this.redraw();
    }

    public redraw() {
        this.chart.redraw(false);
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

    lastKline: Kline = null;

    lastPrice: number = null;

    private priceChart: PriceChart = null;

    private depthChart: DepthChart = null;

    maxTradeHistory: number = 20;

    private tokenfxFeed: Subscription = null;

    lastUpdate: SymbolUpdate = null;

    binanceState: string = "connecting";

    tokenFxState: string = "connecting";

    klineInterval: KlineInterval = KlineInterval.M1;

    private klinesReady: boolean = false;

    availableKlineIntervals: string[] = Object.keys(KlineInterval).map((key) => {
        return KlineInterval[key];
    });

    ATR: any = {};

    showTradingViewCharts: boolean = true;

    orderBook: OrderBookTracker = new OrderBookTracker();

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

        if (this.priceChart) {
            console.log("Destroying price chart.");
            this.priceChart.destroy();
            this.priceChart = null;
        }

        if (this.tokenfxFeed) {
            console.log("Unsubscribing from TokenFX feed.");
            this.tokenfxFeed.unsubscribe();
        }

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
                this.priceChart.redraw();
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

        this.priceChart = new PriceChart("priceChart");

        this.depthChart = new DepthChart("depthChart");

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

    changeInterval(interval: KlineInterval) {
        this.klinesReady = false;
        this.klineInterval = interval;
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
                    this.priceChart.update({
                        timestamp: new Date(message.timestamp),
                        price: message.close,
                    });
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
        }).slice(0, 30);

        this.asks = Object.keys(this.askMap).sort((a, b) => {
            return (+a) - (+b);
        }).map((key) => {
            return [+key, +this.askMap[key]];
        }).slice(0, 30);
    }

    processUpdate(update: DepthUpdate) {
        this.lastUpdateId = update.u;

        for (const bid of update.b) {
            const p = +bid[0];
            const q = +bid[1];
            if (+q === 0) {
                delete(this.bidMap[p]);
            } else {
                this.bidMap[p] = q;
            }
        }

        for (const ask of update.a) {
            const p = +ask[0];
            const q = +ask[1];
            if (+q === 0) {
                delete(this.askMap[p]);
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
