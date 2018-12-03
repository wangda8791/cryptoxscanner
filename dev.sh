#! /bin/sh

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && make update-build && npm start) &

while true; do
    find go/* -name \*.go | \
	entr -d -r sh -c "cd go && make && ./cryptoxscanner server"
done
