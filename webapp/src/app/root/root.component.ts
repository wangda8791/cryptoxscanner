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

import {Component, OnInit} from '@angular/core';
import * as toastr from "toastr";
import {ScannerApiService} from '../scanner-api.service';

@Component({
    selector: 'app-root',
    templateUrl: './root.component.html',
})
export class RootComponent implements OnInit {

    constructor(private tokenFxApi: ScannerApiService) {
    }

    ngOnInit() {
        this.checkProtoVersion();
        setInterval(() => {
            this.checkProtoVersion();
        }, 60000);
    }

    private checkProtoVersion() {
        this.tokenFxApi.ping().subscribe((response) => {
            if (response.buildNumber != this.tokenFxApi.BUILD_NUMBER) {
                toastr.warning(`Service has been updated.
                    <a href="javascript:window.location.href=window.location.href"
                     type="button" class="btn btn-primary btn-block">Reload Now</a>`,
                    `Reload required`, {
                        progressBar: true,
                        timeOut: 15000,
                        onHidden: () => {
                            location.reload();
                        }
                    });
            }
        });
    }

    setTheme(name: string) {
        localStorage.setItem("theme", name);
        toastr.info(`<a href="javascript:window.location.href=window.location.href"
                     type="button" class="btn btn-primary btn-block">Reload to apply theme.</a>`);
    }
}
