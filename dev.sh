#! /bin/sh

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && make update-build && npm start) &

while true; do
    find */* -name \*.go | \
	entr -d -r sh -c "make cryptoxscanner && ./cryptoxscanner server"
done
