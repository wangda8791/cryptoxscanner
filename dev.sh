#! /bin/sh

exec reflex -r '\.go$' -s -- sh -c "go build && ./cryptoxscanner server"
