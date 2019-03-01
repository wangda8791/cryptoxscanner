import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";

import * as $ from "jquery";
import {ActivatedRoute, Router} from "@angular/router";

@Component({
    templateUrl: './binance-newvolume.component.html',
    styleUrls: ['./binance-newvolume.component.scss']
})
export class BinanceNewvolumeComponent implements OnInit {

    tickers: any[] = [];

    min24HourVolume: number = 0;

    private refreshInterval = 6;

    interval = 5;

    constructor(private http: HttpClient,
                private router: Router,
                private route: ActivatedRoute) {
    }

    ngOnInit() {
        this.route.queryParams.subscribe((params) => {
            if (params.min24HourVolume) {
                this.min24HourVolume = +params.min24HourVolume;
            }
            this.refresh();
        });
        this.refreshLoop();
        (<any>$('[data-toggle="tooltip"]')).tooltip()
    }

    private refreshLoop() {
        this.refresh();
        setTimeout(() => this.refreshLoop(), this.refreshInterval * 1000);
    }

    updateConfig() {
        let params = Object.assign({}, this.route.snapshot.params);
        params["min24HourVolume"] = this.min24HourVolume;
        this.router.navigate(["."], {
            relativeTo: this.route,
            queryParams: params,
        });
    }

    columns = [
        {
            name: "Symbol",
            field: "symbol",
            format: "symbol",
        },
        {
            name: "Pings",
            field: "pings",
            format: "integer",
        },
        {
            name: "24h Vol",
            field: "volume",
            format: "number3",
        },
        {
            name: "1h Vol",
            field: "v60",
            format: "number3",
        },
        {
            name: "1h Net Vol",
            field: "nv60",
            format: "number3",
        },
        {
            name: "1h Trades",
            field: "t60",
            format: "integer",
        },
    ];

    private sortKey = "pings";

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

                    if (!ticker["nvh"]) {
                        continue;
                    }

                    if (ticker.vol < +this.min24HourVolume) {
                        continue;
                    }

                    let pings = 0;
                    let pingCount = 0;

                    let nvh = this.bucket(ticker["nvh"]);

                    for (let i = nvh.length - 1; i > 0; i--) {
                        if (nvh[i - 1] > nvh[i]) {
                            pings = pings + (60 - i);
                            pingCount += 1;
                        }
                    }

                    if (pings > 0) {
                        tickers.push({
                            symbol: key,
                            pings: pings / pingCount,
                            volume: ticker.vol,
                            nv60: ticker.nv60,
                            v60: ticker.v60,
                            t60: ticker.t60,
                        });
                    }
                }

                console.log("Sorting tickers.");

                this.tickers = tickers.sort((a, b) => {
                    return b[this.sortKey] - a[this.sortKey];
                });
            });
    }

    private bucket(vals: number[]): number[] {
        let output: number[] = [];
        let interval = +this.interval;
        for (let i = 0; i < vals.length; i++) {
            let parts = vals.slice(i, i + interval);
            let sum = 0;
            for (let i = 0; i < parts.length; i++) {
                sum += parts[i];
            }
            output.push(sum);
        }
        return output;
    }
}
