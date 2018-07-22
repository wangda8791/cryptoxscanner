#! /bin/sh

#
# Requires reflex:
#    go get github.com/cespare/reflex
#

exec reflex -r '\.go$' -s -- sh -c "go build && ./cryptoxscanner server"
