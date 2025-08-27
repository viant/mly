#!/bin/bash

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
source $script_dir/goenv_cgo.lib.sh

goenv_cgo::push /usr/local/lib /opt/homebrew/lib

"$@"

goenv_cgo::pop