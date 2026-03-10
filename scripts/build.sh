#!/bin/bash

OUT=${1:-ce}
go build -o "$OUT" ./cmd/ce/main.go
