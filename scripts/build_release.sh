#!/bin/bash

script_path="$(realpath "${BASH_SOURCE[0]}")"
script_path="$(dirname "$script_path")"

cd "$script_path" || exit "Failed cd"
rm spieven* -f
go build -tags "handshake_check,version_release" -ldflags="-s -w" -trimpath .. || exit "Failed to build spieven"
echo "Built spieven version $(./spieven version)"
realpath spieven
