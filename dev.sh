#! /bin/sh

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && make update-build && npm start) &

while true; do
    find go/* -name \*.go | grep -v packr.go | \
	entr -d -r sh -c "(cd go && make) && ./go/cryptoxscanner server"
done
