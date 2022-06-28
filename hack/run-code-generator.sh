#!/bin/bash

set -euo pipefail

EIRINI_CONTROLLER_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")

cleanup() {
  rm -rf "$EIRINI_CONTROLLER_ROOT/code.cloudfoundry.org"
}

trap cleanup EXIT

rm -rf "$EIRINI_CONTROLLER_ROOT/pkg/generated"

go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  object \
  object:headerFile="$EIRINI_CONTROLLER_ROOT/hack/boilerplate.go.txt" \
  paths="$EIRINI_CONTROLLER_ROOT/pkg/apis/eirini/v1"

go run k8s.io/code-generator/cmd/client-gen \
  --clientset-name versioned \
  --input-base "" \
  --input "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1" \
  --go-header-file "${EIRINI_CONTROLLER_ROOT}/hack/boilerplate.go.txt" \
  --output-base "$(dirname "${BASH_SOURCE[0]}")/.." \
  --output-package "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset" \
  "$EIRINI_CONTROLLER_ROOT/pkg/apis/eirini/v1"

cp -R "$EIRINI_CONTROLLER_ROOT"/code.cloudfoundry.org/eirini-controller/pkg/* "$EIRINI_CONTROLLER_ROOT"/pkg/
