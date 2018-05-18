import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'hodlooLink'
})
export class HodlooLinkPipe implements PipeTransform {

    transform(value: string, exchange: string): any {
        const symbol = value
                .replace(/BTC$/, "-BTC")
                .replace(/ETH$/, "-ETH")
                .replace(/USDT$/, "-USDT")
                .replace(/BNB$/, "-BNB")
                .toLowerCase();
        return `https://qft.hodloo.com/#/binance:${symbol}`;
    }

}
