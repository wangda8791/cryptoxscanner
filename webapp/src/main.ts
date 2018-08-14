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

import {enableProdMode} from '@angular/core';
import {platformBrowserDynamic} from '@angular/platform-browser-dynamic';

import {AppModule} from './app/app.module';
import {environment} from './environments/environment';

import "popper.js";
import "bootstrap";

import * as toastr from "toastr";

declare function require(name: string);

function loadTheme() {
    const requireStyles = () => {
        const themeName = localStorage.getItem("theme");
        switch (themeName) {
            case "dark":
                return require("./styles/theme-dark.scss");
            default:
                return require("./styles/theme-default.scss");
        }
    };

    const styles = requireStyles();

    const node = <HTMLElement>document.createElement("style");
    node.id = "theme";
    node.innerHTML = styles;
    document.body.appendChild(node);
}

if (environment.production) {
    enableProdMode();
}

toastr.options.preventDuplicates = true;

loadTheme();

platformBrowserDynamic().bootstrapModule(AppModule)
        .catch(err => console.log(err));
