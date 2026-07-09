#!/bin/sh

set -eu

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  printf 'gofmt required for:\n%s\n' "$unformatted" >&2
  exit 1
fi

go vet ./...
go test ./... -count=1
go test -race ./... -count=1
go build ./...
go run ./cmd/gnosis validate
