#! /bin/sh

#
# Requires reflex:
#    go get github.com/cespare/reflex
#

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && npm start) &
reflex -r '\.go$' -s -- sh -c "go build && ./cryptoxscanner server"
