#!/usr/bin/env bash
set -euo pipefail

image=$1
directory=$(mktemp -d)
container=
cleanup() {
  if [[ -n "$container" ]]; then
    docker logs "$container"
    docker rm -f "$container" >/dev/null
  fi
  rm -rf "$directory"
}
trap cleanup EXIT
cat > "$directory/config.yaml" <<'YAML'
host: "0.0.0.0"
port: 8317
auth-dir: /tmp/empty-auth
api-keys:
  - fork-image-smoke-test
YAML
container=$(docker run -d -p 127.0.0.1:18317:8317 \
  -v "$directory/config.yaml:/CLIProxyAPI/config.yaml:ro" \
  "$image" ./CLIProxyAPI --local-model)
for attempt in $(seq 1 30); do
  if curl --silent --fail http://127.0.0.1:18317/ > "$directory/root.json"; then
    break
  fi
  sleep 1
done
curl --silent --show-error --fail http://127.0.0.1:18317/ > "$directory/root.json"
status=$(curl --silent --output /dev/null --write-out '%{http_code}' http://127.0.0.1:18317/v1/models)
test "$status" = 401
curl --silent --show-error --fail -H 'Authorization: Bearer fork-image-smoke-test' \
  http://127.0.0.1:18317/v1/models > "$directory/models.json"
python3 - "$directory/models.json" <<'PY'
import json
import sys

response = json.load(open(sys.argv[1]))
assert response["object"] == "list"
assert isinstance(response["data"], list)
PY
