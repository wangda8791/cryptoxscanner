import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";

import * as $ from "jquery";

@Component({
    templateUrl: './binance-netvolumez.component.html',
    styleUrls: ['./binance-netvolumez.component.scss']
})
export class BinanceNetvolumezComponent implements OnInit {

    tickers: any[] = [];

    interval: number = 1;

    availableIntervals = [1, 2, 3, 5];

    min24HourVolume: number = 0;

    pingBuyVolumeChange: number = 0;

    private refreshInterval = 60;

    constructor(private http: HttpClient) {
    }

    ngOnInit() {
        this.refreshLoop();
        (<any>$('[data-toggle="tooltip"]')).tooltip()
    }

    private refreshLoop() {
        this.refresh();
        setTimeout(() => this.refreshLoop(), this.refreshInterval * 1000);
    }

    pingMap = {};

    refresh() {
        console.log("Refreshing...");
        this.http.get("/api/1/binance/volume")
            .subscribe((response: any) => {
                let tickers: any[] = [];
                for (let key of Object.keys(response.data)) {
                    if (!key.endsWith("BTC")) {
                        continue;
                    }
                    const ticker = response.data[key];

                    if (ticker.vol < +this.min24HourVolume) {
                        continue;
                    }

                    let pongs = 0;

                    let timestamp = null;

                    if (!ticker.v24h) {
                        console.log("No 24 hour volume histogram.");
                        continue;
                    }

                    if (!ticker.nvh) {
                        console.log("No net volume histogram.");
                        continue;
                    }

                    if (!ticker.vh) {
                        console.log("No volume histogram.");
                        continue;
                    }

                    for (let i = 0; i < ticker.vh.length; i++) {
                        let nv1p = ticker.nvh[i] / ticker.v24h[i] * 100;
                        if (nv1p > 0.3) {
                            pongs = pongs + 1;
                            if (timestamp == null) {
                                timestamp = new Date().getTime() - (i * 60000);
                            }
                        }
                    }
                    if (this.pingMap[key] === undefined) {
                        this.pingMap[key] = pongs;
                    } else {
                        this.pingMap[key] = pongs;
                    }

                    if (pongs > 0) {
                        tickers.push({
                            symbol: key,
                            pings: pongs,
                            timestamp: timestamp,
                            volume: ticker.vol,
                            nv60: ticker.nv60,
                            v60: ticker.v60,
                            rsi15: ticker.rsi15,
                        });
                    }

                    continue;
                }

                console.log("Sorting tickers.");
                this.tickers = tickers.sort((a, b) => {
                    return b.timestamp - a.timestamp;
                });
            });
    }

    private bucket(nvh: number[]): number[] {
        let output: number[] = [];
        let interval = +this.interval;
        for (let i = 0; i < nvh.length; i++) {
            let part = nvh.slice(i, i + interval);
            if (part.length < interval) {
            } else {
                let sum = 0;
                for (let i = 0; i < part.length; i++) {
                    sum += part[i];
                }
                output.push(sum);
            }
        }
        return output;
    }
}
