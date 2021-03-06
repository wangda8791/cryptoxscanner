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
import {BinanceBaseCoins, ScannerApiService, SymbolUpdate,} from '../scanner-api.service';
import {Subscription} from 'rxjs/Subscription';
import {animate, state, style, transition, trigger} from '@angular/animations';
import * as $ from "jquery";
import * as toastr from "toastr";
import * as lodash from "lodash";

declare var localStorage: any;
declare var Notification: any;

const DEFAULT_BASE_COIN = "BTC";

interface MetaTicker extends SymbolUpdate {
    priceChangePercent: number;
    volumeChangePercent: number;
    priceChangePercent24: number;
}

enum AlertType {
    DROP = "drop",
    GAIN = "gain",
    VOLUME = "volume",
}

interface Alert {
    key: string;
    symbol: string;
    timestamp: Date;
    ticker: SymbolUpdate;
    trigger: string;
    type: AlertType;
    window: number;
}

@Component({
    templateUrl: './monitor.component.html',
    styleUrls: ['./monitor.component.scss'],
    animations: [
        trigger('bannerState', [
            state('void', style({
                opacity: 0,
            })),
            state('*', style({
                opacity: 1,
            })),
            transition('* => void', animate('500ms ease-out')),
            transition('void => *', animate('500ms ease-out')),
        ])
    ]
})
export class BinanceMonitorComponent implements OnInit, OnDestroy, AfterViewInit {

    private localStorageConfigKey = "binance.monitor.config";

    private tickers: SymbolUpdate[] = [];

    showAlertConfiguration: boolean = false;

    public exchange: string = "binance";
    public baseCoins = BinanceBaseCoins;
    public hasCharts: boolean = true;

    // Number of pairs monitored.
    pairCount: number = 0;

    gainers: MetaTicker[] = [];
    losers: MetaTicker[] = [];
    byVolume: MetaTicker[] = [];

    private tickerMap: { [key: string]: SymbolUpdate } = {};

    updateInterval = 1;

    banner = {
        show: true,
        className: "alert-info",
        message: "Connecting to API.",
    };

    config: any = {
        baseCoin: DEFAULT_BASE_COIN,

        blacklist: "",

        alerts: {
            desktopNotifications: false,
            sound: false,

            drop: {
                enabled: true,
                window: "15m",
                percent: 3,
                minVolume24: 150,
                minPrice: 0,
            },

            gain: {
                enabled: true,
                window: "15m",
                percent: 3,
                minVolume24: 150,
                minPrice: 0,
            },

            volume: {
                enabled: true,
                window: "15m",
                percent: 7,
                minVolume24: 150,
                minPrice: 0,
            },
        },

        gainers: {
            sortOrder: "desc",
            sortBy: "priceChangePercent",
            count: 10,
            range: "15m",
        },

        losers: {
            sortOrder: "asc",
            sortBy: "priceChangePercent",
            range: "15m",
        },

        volume: {
            sortOrder: "desc",
            sortBy: "volumeChangePercent",
            range: "15m",
        }
    };

    idleTime: number = 0;
    lastUpdate: number = 0;
    bullsEyeColor: string = "";

    private stream: Subscription;

    intervals: any = {};

    gainersTableConfig: any = {
        columns: [
            {
                title: "Symbol",
                field: "symbol",
            },
            {
                title: "Price %",
                field: "priceChangePercent",
            },
            {
                title: "24h %",
                field: "priceChangePercent24",
            },
            {
                title: "Vol %",
                field: "volumeChangePercent",
            },
            {
                title: "24h Vol",
                field: "volume",
            },
        ],
    };

    losersTableConfig = this.gainersTableConfig;

    volumeTableConfig: any = {
        columns: [
            {
                title: "Symbol",
                field: "symbol",
            },
            {
                title: "Price %",
                field: "priceChangePercent",
            },
            {
                title: "Vol %",
                field: "volumeChangePercent",
            },
            {
                title: "24h Vol",
                field: "volume",
            }
        ],
    };

    alertArray: Alert[] = [];

    private lastAlerts: { [key: string]: { [key: string]: Alert } } = {};

    private newAlerts: boolean = false;

    private lastUpdateTime: Date = null;

    constructor(protected tokenFxApi: ScannerApiService) {
        this.loadConfig();
    }

    protected getPageTitle() {
        return this.exchange.charAt(0).toUpperCase() + this.exchange.slice(1);
    }

    private saveToLocalStorage(key: string, value: any) {
        localStorage[key] = JSON.stringify(value);
    }

    private loadConfig() {
        if (localStorage[this.localStorageConfigKey]) {
            const config = JSON.parse(localStorage[this.localStorageConfigKey]);
            for (const key in this.config) {
                if (config[key] === undefined || config[key] === null) {
                    config[key] = this.config[key];
                }
            }
            lodash.merge(this.config, config);
        }
    }

    toggleDesktopNotifications() {
        if (!this.config.alerts.desktopNotifications) {
            Notification.requestPermission((permission) => {
                if (permission === "granted") {
                    const notification = new Notification("TokenFX", {
                        body: "Notifications Enabled!",
                        silent: false,
                    });
                    this.config.alerts.desktopNotifications = true;
                    this.saveConfig();
                } else if (permission === "denied") {
                    toastr.warning(
                        `Failed to enable desktop notifications: Permission ${permission}.`, null, {
                            progressBar: true,
                            closeButton: true,
                        });
                }
            });
        } else {
            this.config.alerts.desktopNotifications = false;
            this.saveConfig();
        }
    }

    sortBy(what: string, field: string) {
        let config: any = null;
        switch (what) {
            case "losers":
                config = this.config.losers;
                break;
            case "gainers":
                config = this.config.gainers;
                break;
            case "volume":
                config = this.config.volume;
                break;
            default:
                return;
        }

        console.log(`Sorting by ${what}${field}`);
        if (config.sortBy == field) {
            config.sortOrder = this.toggleSortOrder(config.sortOrder);
        } else {
            config.sortBy = field;
        }

        this.saveConfig();
        this.render();
    }

    toggleSortOrder(currentOrder: string): string {
        if (currentOrder == "asc") {
            return "desc";
        }
        return "asc";
    }

    saveConfig() {
        console.log("Saving config.");
        this.saveToLocalStorage(this.localStorageConfigKey, this.config);
    }

    inIframe: boolean = false;
    redirectLocation: string = location.href;

    ngOnInit() {
        document.title = `Crypto Monitor: ${this.getPageTitle()}`;

        // Attempt to break out of iframe...
        if (window != top && window.location !== window.parent.location) {
            this.inIframe = true;
            top.location.href = location.href;
            return;
        }

        this.startStream();

        this.intervals["lastUpdate"] = setInterval(() => {
            if (this.lastUpdate === 0) {
                return;
            }
            this.idleTime = (new Date().getTime() - this.lastUpdate) / 1000;
            if (this.idleTime < 2) {
                this.bullsEyeColor = "";
            } else if (this.idleTime < 5) {
                this.bullsEyeColor = "yellow";
            } else {
                this.bullsEyeColor = "red";
            }
        }, 1000);
    }

    ngOnDestroy() {
        if (this.stream) {
            this.stream.unsubscribe();
        }

        for (const key of Object.keys(this.intervals)) {
            clearInterval(this.intervals[key]);
        }
    }

    ngAfterViewInit() {
        $(function () {
            (<any>$('[data-toggle="popover"]')).popover();
        });
        $(function () {
            (<any>$('[data-toggle="tooltip"]')).tooltip();
        });

        (<any>$('#losersTablePopover')).popover({
            html: true,
            title: "Top Losers",
            content: `
            This table displays the top losers by price within the configured time window.
            <br/>
            <dl class="d-lg-inline">
            <dt>Price</dt>
            <dd>The price change in % for the configured time window.</dd>
            <dt>24h %</dt>
            <dd>The 24 hour price change %</dd>
            <dt>Vol %</dt>
            <dd>The % change of volume for the time window vs the previous 24 hour volume.</dd>
            <dt>Net Vol</dt>
            <dd>The net volume for the given time window. This is the total sell orders subtracted from the buy orders.</dd>
            </dl>
            `,
        });

        (<any>$('#gainersTablePopover')).popover({
            html: true,
            title: "Top Gainers",
            content: `
            This table displays the top gainers by price within the configured time window.
            <br/>
            <dl class="d-lg-inline">
            <dt>Price</dt>
            <dd>The price change in % for the configured time window.</dd>
            <dt>24h %</dt>
            <dd>The 24 hour price change %</dd>
            <dt>Vol %</dt>
            <dd>The % change of volume for the time window vs the previous 24 hour volume.</dd>
            <dt>Net Vol</dt>
            <dd>The net volume for the given time window. This is the total sell orders subtracted from the buy orders.</dd>
            </dl>
            `,
        });

    }

    protected connect() {
        return this.tokenFxApi.connectBinanceMonitor({
            updateInterval: +this.updateInterval,
        });
    }

    private startStream() {
        console.log(`Connecting to ${this.exchange} feed.`);
        this.stream = this.connect()
            .subscribe((tickers) => this.update(tickers),
                () => {
                    // Unfortunately not much is given for an error
                    // reason.
                    this.banner = {
                        show: true,
                        className: "alert-danger",
                        message: "WebSocket error. Reconnecting.",
                    };
                    setTimeout(() => {
                        this.startStream();
                    }, 1000);
                }, () => {
                    this.banner = {
                        show: true,
                        className: "alert-warning",
                        message: "WebSocket closed. Reconnecting.",
                    };
                    setTimeout(() => {
                        this.startStream();
                    }, 1000);
                });
    }

    changeUpdateInterval() {
        this.stream.unsubscribe();
        this.startStream();
    }

    private update(update: any) {

        this.lastUpdateTime = new Date();

        if (this.banner.show) {
            this.banner.className = "alert-success";
            this.banner.message = "Connected!";
            this.banner.show = true;
            setTimeout(() => {
                this.banner.show = false;
            }, 500);
        }

        if (update == null) {
            // Connect signal.
            return;
        }

        this.lastUpdate = new Date().getTime();

        const tickers: SymbolUpdate[] = update.tickers;

        const blacklist = this.config.blacklist.split(/[\s,]/)
            .filter((e) => {
                return e.length > 0;
            });

        // Map the tickers by symbol. Not all updates contain all tickers,
        // so this keeps a stable set of tickers for sorting on.
        for (let i = 0, n = tickers.length; i < n; i++) {
            const ticker = tickers[i];

            // Filter out coins not in the selected base pair.
            if (!ticker.symbol.endsWith(this.config.baseCoin)) {
                continue;
            }

            let skip = false;
            for (const symbol of blacklist) {
                if (symbol.toLowerCase() == ticker.symbol.toLowerCase()) {
                    skip = true;
                    break;
                }
            }
            if (skip) {
                delete (this.tickerMap[ticker.symbol]);
                continue;
            }

            this.tickerMap[ticker.symbol] = ticker;
        }

        // Count the number of pairs.
        this.pairCount = Object.keys(this.tickerMap).length;

        this.updateTopByVolume();

        this.render();
    }

    private render() {
        this.newAlerts = false;

        this.updateGainers();
        this.updateLosers();
        this.updateAlerts();

        if (this.newAlerts && this.config.alerts.sound) {
            try {
                const audio = new Audio();
                audio.src = "../../../assets/sonar.ogg";
                audio.load();
                audio.play().then(() => {}).catch(() => {});
            } catch (err) {
            }
        }
    }

    private addAlert(alert: Alert, msg = null) {

        if (this.lastAlerts[alert.type] === undefined) {
            this.lastAlerts[alert.type] = {};
        }

        if (this.lastAlerts[alert.type][alert.symbol]) {
            const lastAlert = this.lastAlerts[alert.type][alert.symbol];
            const ageMs = this.lastUpdateTime.getTime() - lastAlert.timestamp.getTime();
            if (ageMs < alert.window) {
                // The old alert of type/symbol happened less than the window
                // time of the new one. Ignore the new one.
                return;
            }
        }

        this.lastAlerts[alert.type][alert.symbol] = alert;

        this.alertArray.unshift(alert);

        if (this.config.alerts.desktopNotifications) {
            if (msg == null) {
                msg = `${alert.symbol}`;
            }
            const notification = new Notification("TokenFX Alert", {
                body: msg,
            });
        }

        // Remove any alerts over an hour old.
        const expireTime = 60 * 60 * 1000;
        while (this.alertArray.length > 1) {
            const last = this.alertArray[this.alertArray.length - 1];
            if (new Date().getTime() - last.timestamp.getTime() > expireTime) {
                this.alertArray.pop();
            } else {
                break;
            }
        }

        this.newAlerts = true;
    }

    removeAlert(index: number) {
        this.alertArray.splice(index, 1);
    }

    clearAlerts() {
        this.alertArray = [];
    }

    private updateGainers() {
        const range = this.config.gainers.range;

        const tickers: SymbolUpdate[] = Object.keys(this.tickerMap).map((key) => {
            return this.tickerMap[key];
        }).filter((ticker) => {
            if (ticker.price_change_pct[range] <= 0) {
                return false;
            }
            return true;
        }).sort((a, b) => {
            const diff = b.price_change_pct[range] - a.price_change_pct[range];
            return diff;
        });

        // Take the top 10 and convert to a MetaTicker.
        this.gainers = tickers.slice(0, 10).map((ticker: SymbolUpdate): MetaTicker => {
            const meta: MetaTicker = <MetaTicker>ticker;
            meta.priceChangePercent = ticker.price_change_pct[range];
            meta.volumeChangePercent = ticker.volume_change_pct[range];
            meta.priceChangePercent24 = ticker.price_change_pct["24h"];
            return meta;
        }).sort((a, b) => {
            return this.symbolTrackerSortFunc(
                a, b, this.config.gainers.sortBy, this.config.gainers.sortOrder);
        });
    }

    private updateLosers() {
        const range = this.config.losers.range;

        const tickers: SymbolUpdate[] = Object.keys(this.tickerMap).map((key) => {
            return this.tickerMap[key];
        }).filter((ticker) => {
            if (ticker.price_change_pct[range] >= 0) {
                return false;
            }
            return true;
        }).sort((a, b) => {
            const diff = a.price_change_pct[range] - b.price_change_pct[range];
            return diff;
        });

        // Take the top 10 and convert to a MetaTicker.
        this.losers = tickers.slice(0, 10).map((ticker: SymbolUpdate): MetaTicker => {
            const meta: MetaTicker = <MetaTicker>ticker;
            meta.priceChangePercent = ticker.price_change_pct[range];
            meta.volumeChangePercent = ticker.volume_change_pct[range];
            meta.priceChangePercent24 = ticker.price_change_pct["24h"];
            return meta;
        }).sort((a, b) => {
            return this.symbolTrackerSortFunc(
                a, b, this.config.losers.sortBy, this.config.losers.sortOrder);
        });
    }

    private updateTopByVolume() {
        const range = this.config.volume.range;

        const tickers = Object.keys(this.tickerMap).map((key) => {
            return this.tickerMap[key];
        }).sort((a, b) => {
            return b.volume_change_pct[range] - a.volume_change_pct[range];
        });

        this.byVolume = tickers.map((ticker: SymbolUpdate): MetaTicker => {
            const meta: MetaTicker = <MetaTicker>ticker;
            meta.volumeChangePercent = ticker.volume_change_pct[range];
            meta.priceChangePercent = ticker.price_change_pct[range];
            return meta;
        }).slice(0, 10).sort((a, b) => {
            return this.symbolTrackerSortFunc(
                a, b, this.config.volume.sortBy, this.config.volume.sortOrder);
        });
    }

    private updateAlerts() {

        const alertConfig = this.config.alerts;

        if (!(alertConfig.drop.enabled ||
            alertConfig.gain.enabled ||
            alertConfig.volume.enabled)) {
            return;
        }

        for (const key of Object.keys(this.tickerMap)) {
            const ticker = this.tickerMap[key];

            if (alertConfig.drop.enabled) {
                this.checkDropAlert(ticker);
            }

            if (alertConfig.gain.enabled) {
                this.checkGainAlert(ticker);
            }

            if (alertConfig.volume.enabled) {
                this.checkVolumeAlert(ticker);
            }
        }
    }

    private checkDropAlert(ticker: SymbolUpdate) {
        const config = this.config.alerts.drop;
        const priceChangePercent = ticker.price_change_pct[config.window];

        const minPrice = +config.minPrice;
        if (!isNaN(minPrice)) {
            if (ticker.close < minPrice) {
                return;
            }
        }

        if (priceChangePercent >= 0) {
            return;
        }

        if (Math.abs(priceChangePercent) < config.percent) {
            return;
        }

        if (ticker.volume < config.minVolume24) {
            return;
        }

        const alert: Alert = {
            key: `drop.${ticker.symbol}`,
            symbol: ticker.symbol,
            timestamp: this.lastUpdateTime,
            ticker: ticker,
            trigger: `Price ${priceChangePercent.toFixed(3)}%`,
            type: AlertType.DROP,
            window: intervalStringToMillis(config.window),
        };
        this.addAlert(alert, `${ticker.symbol}: Drop: ${priceChangePercent}%`);
    }

    private checkGainAlert(ticker: SymbolUpdate) {
        const config = this.config.alerts.gain;
        const priceChangePercent = ticker.price_change_pct[config.window];

        const minPrice = +config.minPrice;
        if (!isNaN(minPrice)) {
            if (ticker.close < minPrice) {
                return;
            }
        }

        if (priceChangePercent <= 0) {
            return;
        }

        if (Math.abs(priceChangePercent) < config.percent) {
            return;
        }

        if (ticker.volume < config.minVolume24) {
            return;
        }

        const alert: Alert = {
            key: `gain.${ticker.symbol}`,
            timestamp: this.lastUpdateTime,
            ticker: ticker,
            symbol: ticker.symbol,
            trigger: `Price +${priceChangePercent.toFixed(3)}%`,
            type: AlertType.GAIN,
            window: intervalStringToMillis(config.window),
        };
        this.addAlert(alert, `${ticker.symbol}: Gain: ${priceChangePercent}%`);
    }

    private checkVolumeAlert(ticker: SymbolUpdate) {
        const config = this.config.alerts.volume;
        const volumeChangePercent = ticker.volume_change_pct[config.window];

        const minPrice = +config.minPrice;
        if (!isNaN(minPrice)) {
            if (ticker.close < minPrice) {
                return;
            }
        }

        // Check the minimum 24 hour volume.
        if (ticker.volume < config.minVolume24) {
            return;
        }

        if (volumeChangePercent < config.percent) {
            return;
        }

        const alert: Alert = {
            key: `volume.${ticker.symbol}`,
            timestamp: this.lastUpdateTime,
            ticker: ticker,
            symbol: ticker.symbol,
            trigger: `Volume +${volumeChangePercent.toFixed(3)}%`,
            type: AlertType.VOLUME,
            window: intervalStringToMillis(config.window),
        };
        this.addAlert(alert, `${ticker.symbol}: Volume Increase: ${volumeChangePercent}%`);
    }

    private symbolTrackerSortFunc(a: MetaTicker, b: MetaTicker, field, order): number {
        const rev = order == "asc" ? 1 : -1;
        switch (field) {
            case "symbol":
                if (a.symbol < b.symbol) {
                    return -1 * rev;
                } else if (a.symbol > b.symbol) {
                    return 1 * rev;
                }
                return 0;
            default:
                return (a[field] - b[field]) * rev;
        }
    }

}

function intervalStringToMillis(interval: string): number {
    const parts = interval.match(/(\d+)(\w+)/);
    if (parts.length < 3) {
        return 0;
    }
    const val = +parts[1];
    const unit = parts[2];
    switch (unit) {
        case "m":
            return val * 60 * 1000;
        case "h":
            return val * 3600 * 1000;
        default:
            console.log(`error: unknown unit: ${unit}: ${interval}`);
            return 0;
    }
}
