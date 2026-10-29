#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p release
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=1.0.0" -o release/Pitchboard.exe ./cmd/pitchboard
echo "Built release/Pitchboard.exe"
