#!/usr/bin/env bash
set -euo pipefail
exec java -jar "${JETTY_HOME}/start.jar" "jetty.http.port=${PLAB_HTTP_PORT:-8080}"
