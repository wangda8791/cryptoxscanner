import {Component, OnInit} from '@angular/core';
import {HttpClient} from "@angular/common/http";

@Component({
    selector: 'app-binance-volume',
    templateUrl: './binance-volume.component.html',
    styleUrls: ['./binance-volume.component.scss']
})
export class BinanceVolumeComponent implements OnInit {

    tickers: any[] = [];

    constructor(private http: HttpClient) {
    }

    ngOnInit() {
        this.refresh();
    }

    private refresh() {
        console.log("Refreshing...");
        this.http.get("/api/1/binance/volume")
            .subscribe((response: any) => {
                let tickers: any[] = [];
                for (let key of Object.keys(response.data)) {
                    if (!key.endsWith("BTC")) {
                        continue;
                    }
                    const ticker = response.data[key];
                    const pings = this.calculatePings(key, response.data[key].nvh);
                    tickers.push({
                        symbol: key,
                        pings: pings,
                        volume: ticker.vol,
                        nv60: ticker.nv60,
                        v60: ticker.v60,
                    });
                }
                this.tickers = tickers.sort((a, b) => {
                    return b.pings - a.pings;
                });
            });
        setTimeout(() => this.refresh(), 6000);
    }

    private calculatePings(key: string, nvh: number[]): number {
        let pings = 0;
        if (nvh == null || nvh.length == 0) {
            console.log(`${key}: ${nvh}`);
            return 0;
        }
        for (let i = nvh.length - 1; i > 0; i--) {
            if (nvh[i - 1] > nvh[i]) {
                pings += 1;
            }
        }
        return pings;
    }

}
