// Copyright 2018 Cranky Kernel
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
