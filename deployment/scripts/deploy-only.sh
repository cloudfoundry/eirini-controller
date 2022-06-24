#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT_DIR="$ROOT_DIR/deployment/scripts"

export KUBECONFIG
KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
KUBECONFIG=$(readlink -f "$KUBECONFIG")

export GOOGLE_APPLICATION_CREDENTIALS
GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-""}
if [[ -n $GOOGLE_APPLICATION_CREDENTIALS ]]; then
  GOOGLE_APPLICATION_CREDENTIALS=$(readlink -f "$GOOGLE_APPLICATION_CREDENTIALS")
fi

readonly SYSTEM_NAMESPACE=eirini-controller

readonly HELM_VALUES=${HELM_VALUES:-"$ROOT_DIR/deployment/helm/values.yaml"}

source "$SCRIPT_DIR/helpers/print.sh"

main() {
  print_disclaimer
  install_prometheus
  install_cert_manager
  install_eirini_controller "$@"
}

install_prometheus() {
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
  helm repo update
  helm upgrade prometheus \
    --install prometheus-community/prometheus \
    --namespace prometheus \
    --create-namespace \
    --wait
}

install_cert_manager() {
  kapp -y deploy -a cert-mgr -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.2/cert-manager.yaml
}

install_eirini_controller() {
  helm template eirini-controller "$ROOT_DIR/deployment/helm" \
    --namespace "$SYSTEM_NAMESPACE" \
    --values "$HELM_VALUES" \
    --values "$SCRIPT_DIR/assets/value-overrides.yaml" "$@" |
    kapp -y deploy -a eirini-controller -f -
}

main "$@"
