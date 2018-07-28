#! /bin/sh

#
# Requires reflex:
#    go get github.com/cespare/reflex
#

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && npm start) &
reflex -r '\.go$' -s -- sh -c "CGO_ENABLED=1 go build --tags json1 && ./cryptoxscanner server"
