#!/bin/bash

script_path="$(realpath "${BASH_SOURCE[0]}")"
script_path="$(dirname "$script_path")"

cd "$script_path" || exit "Failed cd"
rm spieven* -f

go build -tags release -ldflags="-s -w" -trimpath .. || exit "Failed to build spieven"
version="$(./spieven version)" || exit "Failed to check spieven version"
zipname=spieven-$version.zip
zip $zipname spieven >/dev/null || exit "Failed to create zip"

echo "Built spieven version $version"
realpath spieven || exit "Path error"
realpath $zipname || exit "Path error"
