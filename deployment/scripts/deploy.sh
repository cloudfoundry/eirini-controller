#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT_DIR="$ROOT_DIR/deployment/scripts"

"$SCRIPT_DIR"/build.sh
"$SCRIPT_DIR"/deploy-only.sh "$@"
