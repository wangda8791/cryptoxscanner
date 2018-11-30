# CryptoXScanner

This is the source code for my crypto exchange scanner. Currently
the other supported exchange is Binance.

## Download

Runnable builds can be downloaded from the GitLab CI system:
https://gitlab.com/crankykernel/cryptoxscanner/-/jobs/artifacts/master/browse?job=build

## Building

Before building _cryptoxscanner_ you must install Go and Node:
- Node 10+
- Go 1.11+
As Cgo is used, you will also need a gcc/clang installed.

Also, $GOAPTH/bin must be in your PATH.

1. From the top of the source tree run:

		make install-deps

	This command will:
	- In webapp into npm dependencies: `npm install`.
	- In the top level directory, install the Go dependencies.

2. In the top level directory run:

		make

   This will produce the *cryptoxscanner* binary in the current
   directory with the web application resources bundled into it.

## License

This code is licensed under GNU Affero Public License, see
`LICENSE.txt` for more details. However, HighCharts is also used which
falls under another license. Please make sure you comply with their
license terms before deploying this scanner yourself.
