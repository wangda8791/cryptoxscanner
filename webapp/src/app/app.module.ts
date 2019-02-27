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

import {BrowserModule} from '@angular/platform-browser';
import {Component, NgModule, OnInit} from '@angular/core';
import {BinanceMonitorComponent,} from './monitor/monitor.component';
import {ScannerApiService} from './scanner-api.service';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {RouterModule, Routes} from '@angular/router';
import {RootComponent} from './root/root.component';
import {AppThSortableComponent, AppUpDownDirective, BinanceLiveComponent,} from './live/live.component';
import {BrowserAnimationsModule} from "@angular/platform-browser/animations";
import {HomeComponent} from './home/home.component';
import {HttpClientModule} from '@angular/common/http';
import {OrderbookComponent} from './binance/symbol/orderbook/orderbook.component';
import {BinanceSymbolComponent} from './binance/symbol/symbol.component';
import {SymbolFilterPipe} from './symbol-filter.pipe';
import {BinanceApiService} from './binance-api.service';
import {DoubleScrollModule} from 'mindgaze-doublescroll';
import {BaseassetPipe} from './baseasset.pipe';
import {ExchangesymbolPipe} from './exchangesymbol.pipe';
import {HodlooLinkPipe} from './hodloo-link.pipe';
import {BinanceNetvolumezComponent} from './binance-netvolumez/binance-netvolumez.component';
import {BinanceNewvolumeComponent} from "./binance-newvolume/binance-newvolume.component";

@Component({
    template: ``,
})
export class KuCoinMonitorRedirectComponent implements OnInit {
    ngOnInit(): void {
        window.location.href = "https://kucoin.cryptoxscanner.com/kucoin/monitor";
    }
}

@Component({
    template: ``,
})
export class KuCoinLiveRedirectComponent implements OnInit {
    ngOnInit(): void {
        window.location.href = "https://kucoin.cryptoxscanner.com/kucoin/live";
    }
}

const appRoutes: Routes = [

    // Binance.
    {
        path: "binance/monitor",
        component: BinanceMonitorComponent,
        pathMatch: "prefix",
    },
    {
        path: "binance/live",
        component: BinanceLiveComponent,
        pathMatch: "prefix",
    },
    {
        path: "binance/screener",
        pathMatch: "prefix",
        redirectTo: "binance/live",
    },
    {
        path: "binance/chart",
        pathMatch: "prefix",
        redirectTo: "binance/symbol",
    },
    {
        path: "binance/symbol/:symbol",
        pathMatch: "prefix",
        component: BinanceSymbolComponent,
    },

    {
        path: "binance/netvolumez",
        pathMatch: "prefix",
        component: BinanceNetvolumezComponent,
    },

    {
        path: "binance/newvolume",
        pathMatch: "prefix",
        component: BinanceNewvolumeComponent,
    },

    {
        path: "kucoin/monitor",
        pathMatch: "prefix",
        component: KuCoinMonitorRedirectComponent,
    },
    {
        path: "kucoin/live",
        pathMatch: "prefix",
        component: KuCoinLiveRedirectComponent,
    },

    {
        path: '', component: HomeComponent, pathMatch: "prefix",
    }
];

@NgModule({
    declarations: [
        BinanceMonitorComponent,
        RootComponent,
        BinanceLiveComponent,
        AppThSortableComponent,
        AppUpDownDirective,
        HomeComponent,
        BinanceSymbolComponent,
        OrderbookComponent,
        SymbolFilterPipe,
        BaseassetPipe,
        ExchangesymbolPipe,
        HodlooLinkPipe,

        KuCoinMonitorRedirectComponent,
        KuCoinLiveRedirectComponent,
        BinanceNetvolumezComponent,
        BinanceNewvolumeComponent,
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        FormsModule,
        ReactiveFormsModule,
        HttpClientModule,
        RouterModule.forRoot(
            appRoutes, {useHash: false},
        ),
        DoubleScrollModule,
    ],
    providers: [
        ScannerApiService,
        BinanceApiService,
    ],
    bootstrap: [RootComponent]
})
export class AppModule {
}
