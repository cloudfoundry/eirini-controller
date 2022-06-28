#!/bin/bash

set -euo pipefail

BASEDIR="$(cd "$(dirname "$0")"/.. && pwd)"
readonly BASEDIR

main() {
  run_tests "$@"
}

run_tests() {
  pushd "$BASEDIR" >/dev/null || exit 1
  go run github.com/onsi/ginkgo/v2/ginkgo -p -r --keep-going --skip-package=tests --randomize-all --randomize-suites "$@"
  popd >/dev/null || exit 1
}

main "$@"
