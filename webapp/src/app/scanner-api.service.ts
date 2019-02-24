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

import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Observer} from "rxjs/Observer";
import 'rxjs/add/operator/map';
import {HttpClient} from '@angular/common/http';
import {environment} from '../environments/environment';

declare var window: Window;

export const BinanceBaseCoins: string[] = [
    "BTC",
    "ETH",
    "BNB",
    "USDT",
];

@Injectable()
export class ScannerApiService {

    public BUILD_NUMBER = environment.buildNumber;

    private protocol: string = "wss";

    private baseUrl: string;

    constructor(private http: HttpClient) {
        const location = window.location;
        switch (location.protocol) {
            case "https:":
                this.protocol = "wss";
                break;
            default:
                this.protocol = "ws";
        }

        this.baseUrl = `${this.protocol}://${location.host}`;

        console.log(`CryptoXScanner Client Build Number: ${this.BUILD_NUMBER}`);
    }

    public ping(): Observable<any> {
        return this.http.get("/api/1/ping");
    }

    public connect(url): Observable<SymbolUpdate[] | SymbolUpdate> {
        return new Observable(
                (obs: Observer<SymbolUpdate[]>) => {

                    const ws = new WebSocket(url);

                    // On connect send a null as a signal.
                    ws.onopen = () => {
                        obs.next(null);
                    };

                    const onmessage = (event) => {
                        obs.next(JSON.parse(event.data));
                    };

                    const onerror = (event) => {
                        obs.error(event);
                    };

                    const onclose = () => {
                        obs.complete();
                    };

                    ws.onmessage = onmessage;
                    ws.onerror = onerror;
                    ws.onclose = onclose;

                    return () => {
                        ws.close();
                    };
                });
    }

    public connectBinanceMonitor(options: any = {}): Observable<SymbolUpdate[] | SymbolUpdate> {
        let url = `${this.baseUrl}/ws/binance/monitor?`;
        if (options.updateInterval) {
            url = `${url}updateInterval=${options.updateInterval}&`;
        }
        return this.connect(url);
    }

    public connectBinanceLive(): Observable<SymbolUpdate[] | SymbolUpdate> {
        const url = `${this.baseUrl}/ws/binance/live`;
        return this.connect(url);
    }

    public connectBinanceSymbol(symbol: string): Observable<SymbolUpdate[] | SymbolUpdate> {
        const url = `${this.baseUrl}/ws/binance/symbol?symbol=${symbol}`;
        return this.connect(url);
    }

}

export interface SymbolUpdate {
    symbol: string;

    high: number;
    low: number;

    price_change_pct: {
        [key: string]: number;
    };

    volume_change_pct: {
        [key: string]: number;
    };

    net_volume_1?: number;
    net_volume_5?: number;
    net_volume_10?: number;
    net_volume_15?: number;
    net_volume_60?: number;

    total_volume_1?: number;
    total_volume_5?: number;
    total_volume_10?: number;
    total_volume_15?: number;
    total_volume_60?: number;

    vwap_1m?: number;
    vwap_2m?: number;
    vwap_3m?: number;
    vwap_4m?: number;
    vwap_5m?: number;
    vwap_10m?: number;
    vwap_15m?: number;
    vwap_60m?: number;

    bid: number;
    ask: number;
    close: number;
    timestamp: string;
    volume: number;

    rsi_60?: number;
}
