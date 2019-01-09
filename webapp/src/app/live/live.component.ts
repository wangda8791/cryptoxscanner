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

import {Component, Directive, ElementRef, Input, OnDestroy, OnInit} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {BinanceBaseCoins, ScannerApiService, SymbolUpdate} from '../scanner-api.service';
import {animate, state, style, transition, trigger} from "@angular/animations";
import {Observable} from 'rxjs/Observable';
import * as lodash from "lodash";
import {FormControl} from "@angular/forms";
import {ActivatedRoute, Router} from "@angular/router";

declare var localStorage: any;

interface Banner {
    show: boolean;
    className: string;
    message: string;
}

@Component({
    selector: "[app-th-sortable]",
    template: `
      <!-- @formatter:off -->
      <span style="cursor: pointer;"><ng-content></ng-content><span *ngIf="sortBy===name">
          <span *ngIf="sortOrder==='desc'"><i class="fas fa-caret-down"></i></span>
          <span *ngIf="sortOrder==='asc'"><i class="fas fa-caret-up"></i></span>
        </span>
      </span>
    `,
})
export class AppThSortableComponent {
    @Input() name;
    @Input() sortBy;
    @Input() sortOrder;
}

interface ColumnConfig {
    title: string;
    name: string;
    display: boolean;
    type: string;
    format?: string;
    routerLink?: string;
    fn?: any;
    updown?: boolean;
}

@Component({
    templateUrl: './live.component.html',
    styleUrls: ['./live.component.scss'],
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
    ],
})
export class BinanceLiveComponent implements OnInit, OnDestroy {

    public exchange: string = "binance";

    public baseTokens: string[] = BinanceBaseCoins;

    private configKey: string = "binance.live.config";

    hasCharts: boolean = true;

    showMoreFilters: boolean = false;

    config: any = {
        base: "BTC",
        sortBy: "price_change_pct15",
        sortOrder: "desc",
        maxPrice: null,
        minPrice: null,
        max24Change: null,
        min24Change: null,
        filter: null,
        count: 25,
        visibleColumns: {},
        watching: {},

        filters: {
            maxRsi60: null,
            minVol24: null,
            maxVol24: null,
        },

        blacklist: "",
        whitelist: "",
    };

    private stream: Subscription = null;

    public stream$: Observable<any>;

    // A map of all the tickers keyed by symbol. The update message includes
    // a list of tickers, and not all may be present (for example, no update
    // for a symbol). So we want to save the previous tickers to prevent
    // symbols from disappearing from the display.
    private tickerMap: any = {};

    // The sorted and filtered tickers to be displayed on the screen.
    //tickers: SymbolUpdate[] = [];
    tickers: any[] = [];

    banner: Banner = {
        show: true,
        className: "alert-info",
        message: "Connecting to API.",
    };

    private lastUpdate: number = 0;

    private idleInterval: any = null;

    idleTime: number = 0;

    columns: ColumnConfig[] = [];

    visibleColumns: ColumnConfig[] = [];

    // The index of the row the user is currently hovering over.
    private activeRow: number = null;

    blacklistForm = new FormControl('');

    whitelistForm = new FormControl('');

    constructor(public tokenFxApi: ScannerApiService,
                private route: ActivatedRoute,
                private router: Router) {
    }

    private initHeaders() {
        this.columns = [
            {
                title: "Last",
                name: "close",
                type: "number",
                format: ".8",
                display: true,
            },
            {
                title: "Bid",
                name: "bid",
                type: "number",
                format: ".8",
                display: false,
            },
            {
                title: "Ask",
                name: "ask",
                type: "number",
                format: ".8",
                display: false,
            },
            {
                title: "Spread",
                name: "spread",
                type: "percent",
                format: ".3-3",
                display: false,
            },
            {
                title: "24h High",
                name: "high",
                type: "number",
                format: ".8",
                display: false,
            },
            {
                title: "24h Low",
                name: "low",
                type: "number",
                format: ".8",
                display: false,
            },
            {
                title: "24h %",
                name: "price_change_pct_24h",
                type: "percent-number",
                display: true,
            },
            {
                title: "24h Vol",
                name: "volume",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "1m %",
                name: "price_change_pct_1m",
                type: "percent-number",
                display: true,
            },
            {
                title: "5m %",
                name: "price_change_pct_5m",
                type: "percent-number",
                display: true,
            },
            {
                title: "10m %",
                name: "price_change_pct_10m",
                type: "percent-number",
                display: true,
            },
            {
                title: "15m %",
                name: "price_change_pct_15m",
                type: "percent-number",
                display: true,
            },
            {
                title: "60m %",
                name: "price_change_pct_1h",
                type: "percent-number",
                display: true,
            },
            {
                title: "1m Vol %",
                name: "volume_change_pct_1m",
                type: "percent-number",
                display: true,
            },
            {
                title: "2m Vol %",
                name: "volume_change_pct_2m",
                type: "percent-number",
                display: true,
            },
            {
                title: "3m Vol %",
                name: "volume_change_pct_3m",
                type: "percent-number",
                display: true,
            },
            {
                title: "5m Vol %",
                name: "volume_change_pct_5m",
                type: "percent-number",
                display: true,
            },
            {
                title: "10m Vol %",
                name: "volume_change_pct_10m",
                type: "percent-number",
                display: true,
            },
            {
                title: "15m Vol %",
                name: "volume_change_pct_15m",
                type: "percent-number",
                display: true,
            },
            {
                title: "60m Vol %",
                name: "volume_change_pct_1h",
                type: "percent-number",
                display: true,
            },
        ];

        for (const i of [1, 2, 3, 5, 10, 15, 60]) {
            this.columns.push({
                title: `${i}mL`,
                name: `l_${i}`,
                type: "number",
                format: ".8-8",
                display: false,
            });
            this.columns.push({
                title: `${i}mH`,
                name: `h_${i}`,
                type: "number",
                format: ".8-8",
                display: false,
            });
            this.columns.push({
                title: `${i}mR%`,
                name: `rp_${i}`,
                type: "percent-number",
                display: true,
            });
        }

        this.columns.push(...[
            {
                title: "1mNV",
                name: "nv_1",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "2mNV",
                name: "nv_2",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "3mNV",
                name: "nv_3",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "4mNV",
                name: "nv_4",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "5mNV",
                name: "nv_5",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "10mNV",
                name: "nv_10",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "15mNV",
                name: "nv_15",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "60mNV",
                name: "nv_60",
                type: "number",
                format: ".2-2",
                display: true,
                updown: true,
            },
            {
                title: "1m Vol",
                name: "total_volume_1",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "5m Vol",
                name: "total_volume_5",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "10m Vol",
                name: "total_volume_10",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "15m Vol",
                name: "total_volume_15",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "60m Vol",
                name: "total_volume_60",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "RSI 1m",
                name: "rsi_60",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "RSI 3m",
                name: "rsi_180",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "RSI 5m",
                name: "rsi_300",
                type: "number",
                format: ".2-2",
                display: true,
            },
            {
                title: "RSI 15m",
                name: "rsi_900",
                type: "number",
                format: ".2-2",
                display: true,
            }
        ]);

        // Buy volumes.
        for (const r of ["1", "2", "3", "5", "15", "60"]) {
            const entry: any = {
                title: `${r}mBV`,
                name: `bv_${r}`,
                type: "number",
                format: ".2-2",
                display: true,
            };
            this.columns.push(entry);
        }

        // Sell volumes.
        for (const r of ["1", "2", "3", "5", "15", "60"]) {
            const entry: any = {
                title: `${r}mSV`,
                name: `sv_${r}`,
                type: "number",
                format: ".2-2",
                display: true,
            };
            this.columns.push(entry);
        }

        this.restoreConfig();
    }

    updateBlacklist() {
        this.updateQueryParams();
        document.getElementById("blacklistInput").blur();
    }

    clearBlacklist() {
        this.blacklistForm.setValue(undefined);
        this.updateBlacklist();
    }

    updateWhitelist() {
        this.updateQueryParams();
        document.getElementById("whitelistInput").blur();
    }

    clearWhitelist() {
        this.whitelistForm.setValue("");
        this.updateWhitelist();
    }

    private previousQueryParams = {};

    private updateQueryParams() {
        let queryParams = Object.assign({}, this.previousQueryParams);

        if (this.whitelistForm.value) {
            queryParams["whitelist"] = this.whitelistForm.value;
        } else {
            delete queryParams["whitelist"];
        }

        if (this.blacklistForm.value) {
            queryParams["blacklist"] = this.blacklistForm.value;
        } else {
            delete queryParams["blacklist"];
        }

        this.router.navigate(["/binance/live"], {
            queryParams: queryParams,
        });
    }

    showDefaultColumns() {
        for (const col of this.columns) {
            this.config.visibleColumns[col.name] = col.display;
        }
        this.saveConfig();
    }

    deselectAllColumns() {
        for (const col of this.columns) {
            this.config.visibleColumns[col.name] = false;
        }
        this.saveConfig();
    }

    selectAllColumns() {
        for (const col of this.columns) {
            this.config.visibleColumns[col.name] = true;
        }
        this.saveConfig();
    }

    restoreConfig() {
        if (localStorage[this.configKey]) {
            try {
                const config = JSON.parse(localStorage[this.configKey]);
                lodash.merge(config, this.config);
                if (!config.visibleColumns) {
                    this.showDefaultColumns();
                } else {
                    for (const col of this.columns) {
                        if (!(col.name in config.visibleColumns)) {
                            config.visibleColumns[col.name] = col.display;
                        }
                    }
                }
                this.config = config;
                return;
            } catch (err) {
            }
        }
        this.showDefaultColumns();
    }

    saveConfig() {
        localStorage[this.configKey] = JSON.stringify(this.config);
    }

    ngOnInit() {
        this.route.queryParams.subscribe(params => {
            this.previousQueryParams = params;

            this.blacklistForm.setValue(params.blacklist);
            this.config.blacklist = this.blacklistForm.value;

            this.whitelistForm.setValue(params.whitelist);
            this.config.whitelist = this.whitelistForm.value;

            if (params.showFilters !== undefined) {
                this.showMoreFilters = true;
            }
        });
        this.initHeaders();
        this.startUpdates();
        this.idleInterval = setInterval(() => {
            if (this.lastUpdate === 0) {
                return;
            }
            this.idleTime = (new Date().getTime() - this.lastUpdate) / 1000;
        }, 1000);
    }

    ngOnDestroy() {
        if (this.stream) {
            this.stream.unsubscribe();
        }
        clearInterval(this.idleInterval);
    }

    protected connect() {
        return this.tokenFxApi.connectBinanceLive();
    }

    private startUpdates() {
        if (this.stream) {
            this.stream.unsubscribe();
        }

        this.stream = this.connect().subscribe(
            (update: any) => {
                if (this.banner.show) {
                    console.log("Updating banner.");
                    this.banner = {
                        show: true,
                        className: "alert-success",
                        message: "Connected!",
                    };
                    setTimeout(() => {
                        this.banner.show = false;
                    }, 1000);
                }

                if (update == null) {
                    // Connect signal.
                    return;
                }

                const tickers: SymbolUpdate[] = update.tickers || update;

                // Put the tickers into a map.
                for (let i = 0, n = tickers.length; i < n; i++) {
                    this.flattenTicker(tickers[i]);
                    this.tickerMap[tickers[i].symbol] = tickers[i];
                }

                this.render();
            },
            (error) => {
                this.banner = {
                    show: true,
                    className: "alert-warning",
                    message: "WebSocket error! Reconnecting.",
                };
                console.log("websocket error:");
                console.log(error);
                setTimeout(() => {
                    this.startUpdates();
                }, 1000);
            },
            () => {
                this.banner = {
                    show: true,
                    className: "alert-warning",
                    message: "WebSocket closed!",
                };
                console.log("websocket closed. Reconnecting.");
                setTimeout(() => {
                    this.startUpdates();
                });
            });
    }

    /**
     * Convert the string v into a number. Null is returned if the string
     * is not a number.
     */
    private asNumber(input: string): number {
        if (input == null || input === "") {
            return null;
        }
        const value: number = +input;
        if (!isNaN(value)) {
            return value;
        }
        return null;
    }

    render() {
        this.lastUpdate = new Date().getTime();

        const volumePairs = [
            ["bv_1", "sv_1"],
            ["bv_2", "sv_2"],
            ["bv_3", "sv_3"],
            ["bv_5", "sv_5"],
            ["bv_15", "sv_15"],
            ["bv_60", "sv_60"],
        ];

        // If active row is non-null the user is hovering on a row. Record
        // the index and the symbol.
        let activeSymbol: string = null;
        const activeRow = this.activeRow;
        if (activeRow != null) {
            try {
                activeSymbol = this.tickers[this.activeRow].symbol;
            } catch (e) {
                activeSymbol = null;
            }
        }
        let activeTicker: SymbolUpdate = null;

        let tickers: SymbolUpdate[] = Object.keys(this.tickerMap).map(key => {
            return this.tickerMap[key];
        });

        const maxPrice = this.asNumber(this.config.maxPrice);
        const minPrice = this.asNumber(this.config.minPrice);
        const max24Change = this.asNumber(this.config.max24Change);
        const min24Change = this.asNumber(this.config.min24Change);
        const maxRsi60 = this.asNumber(this.config.filters.maxRsi60);

        const blacklist = new Blacklist(this.config.blacklist);
        const whitelist = new Whitelist(this.config.whitelist);

        tickers = tickers.filter((ticker) => {

            if (blacklist.match(ticker.symbol)) {
                return false;
            }

            if (!whitelist.match(ticker.symbol)) {
                return false;
            }

            if (!this.filterBase(ticker)) {
                return false;
            }

            // If this is the symbol that is being hovered, pluck it out. It
            // will be insert back in at the same position later.
            if (activeSymbol != null && ticker.symbol == activeSymbol) {
                activeTicker = ticker;
                return false;
            }

            if (max24Change != null) {
                if (ticker.price_change_pct["24h"] > max24Change) {
                    return false;
                }
            }

            if (min24Change != null) {
                if (ticker.price_change_pct["24h"] < min24Change) {
                    return false;
                }
            }

            if (maxPrice) {
                if (ticker.close > maxPrice) {
                    return false;
                }
            }

            if (minPrice) {
                if (ticker.close < minPrice) {
                    return false;
                }
            }

            if (maxRsi60) {
                if (ticker.rsi_60 && ticker.rsi_60 > maxRsi60) {
                    return false;
                }
            }

            if (this.config.filters.maxVol24) {
                if (ticker.volume > this.config.filters.maxVol24) {
                    return false;
                }
            }

            if (this.config.filters.minVol24) {
                if (ticker.volume < this.config.filters.minVol24) {
                    return false;
                }
            }

            if (this.config.filter != null && this.config.filter != "") {
                if (ticker.symbol.indexOf(this.config.filter.toUpperCase()) < 0) {
                    return false;
                }
            }

            return true;
        });

        tickers = tickers.sort((a, b) => this.sortTickers(a, b));

        for (let i = 0, n = tickers.length; i < n; i++) {
            if (this.config.watching[tickers[i].symbol]) {
                const ticker = tickers[i];
                tickers.splice(i, 1);
                tickers.unshift(ticker);
            }
        }

        tickers = tickers.slice(0, this.config.count);

        // If we plucked out a ticker, re-insert it here.
        if (activeRow != null && activeTicker) {
            tickers.splice(activeRow, 0, activeTicker);
        }

        this.visibleColumns = this.columns.filter((col) => {
            return this.config.visibleColumns[col.name];
        });

        this.tickers = tickers.map((ticker) => {
            const new_ticker = {};
            for (const key of Object.keys(ticker)) {
                new_ticker[key] = {
                    value: ticker[key],
                    background_color: "",
                };

                if (this.config.sortBy == key) {
                    new_ticker[key].background_color = "gainsboro";
                }

            }

            // Colourize the buy volume based on if its greater than or less
            // than the sell volume for the same time period.
            for (const pair of volumePairs) {
                if (ticker[pair[0]] > ticker[pair[1]]) {
                    new_ticker[pair[0]].background_color = "lightgreen";
                } else if (ticker[pair[0]] < ticker[pair[1]]) {
                    new_ticker[pair[0]].background_color = "orange";
                }
            }

            return new_ticker;
        });
    }

    private flattenTicker(ticker: SymbolUpdate) {
        for (const key of Object.keys(ticker.price_change_pct)) {
            ticker[`price_change_pct_${key}`] =
                ticker.price_change_pct[key];
        }
        for (const key of Object.keys(ticker.volume_change_pct)) {
            ticker[`volume_change_pct_${key}`] =
                ticker.volume_change_pct[key];
        }

        ticker["spread"] = (ticker.ask - ticker.bid) / ticker.bid;
    }

    /**
     * Base coin filter. Returns true if the symbol ends in the base coin.
     */
    filterBase(ticker: SymbolUpdate): boolean {
        return ticker.symbol.endsWith(this.config.base);
    }

    sortTickers(a, b: SymbolUpdate): number {
        switch (this.config.sortBy) {
            case "symbol":
                switch (this.config.sortOrder) {
                    case "desc":
                        return b.symbol.localeCompare(a.symbol);
                    default:
                        return a.symbol.localeCompare(b.symbol);
                }
            default:
                // By default sort as numbers.
                if (this.config.sortOrder == "asc") {
                    return a[this.config.sortBy] - b[this.config.sortBy];
                }
                return b[this.config.sortBy] - a[this.config.sortBy];
        }
    }

    sortBy(column: string) {
        if (this.config.sortBy == column) {
            this.toggleSortOrder();
        } else {
            this.config.sortBy = column;
        }
        this.render();
    }

    private toggleSortOrder() {
        if (this.config.sortOrder == "asc") {
            this.config.sortOrder = "desc";
        } else {
            this.config.sortOrder = "asc";
        }
    }

    trackBy(index, item) {
        return item.symbol;
    }

    mouseEnter(index: number) {
        this.activeRow = index;
    }
}

@Directive({
    selector: "[appUpDown]",
})
export class AppUpDownDirective {

    constructor(el: ElementRef) {
        el.nativeElement.style.color = "green";
    }

}

class Blacklist {
    private entries: string[];

    constructor(private blacklistString = "") {
        this.entries = blacklistString.split(/[\s,]/)
            .filter((e) => {
                return e.length > 0;
            });
    }

    match(item: string): boolean {
        for (const entry of this.entries) {
            if (entry.toLowerCase() === item.toLowerCase()) {
                return true;
            }
        }
        return false;
    }
}

class Whitelist {
    private entries: string[];

    constructor(private list = "") {
        this.entries = list.split(/[\s,]/)
            .filter((e) => {
                return e.length > 0;
            });
    }

    match(item: string): boolean {
        if (this.entries.length == 0) {
            return true;
        }
        for (const entry of this.entries) {
            if (entry.toLowerCase() === item.toLowerCase()) {
                return true;
            }
        }
        return false;
    }
}