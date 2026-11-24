#!/usr/bin/env bash
# raft-kv smoke test (Kubernetes дээр):
#   1) 3 pod босч, яг 1 leader сонгогдоно
#   2) leader-т өгөгдөл бичнэ
#   3) rolling restart хийнэ
#   4) дахин leader сонгогдож, бичсэн өгөгдөл хэвээр байна
set -euo pipefail

NS="${NS:-default}"
BASE_PORT=18000
KEY="smoke"
VALUE="survived-$(date +%s)"
PF_PIDS=()

stop_forwards() {
  for pid in "${PF_PIDS[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
  PF_PIDS=()
}
trap stop_forwards EXIT

start_forwards() {
  stop_forwards
  for i in 0 1 2; do
    kubectl -n "$NS" port-forward "pod/raft-kv-$i" "$((BASE_PORT + i)):8000" >/dev/null 2>&1 &
    PF_PIDS+=("$!")
  done
  sleep 3
}

# /status нь {"leader":true|false,"term":N} буцаадаг. Яг нэг leader гартал хүлээнэ.
wait_for_leader() {
  for _ in $(seq 1 40); do
    local leaders=()
    for i in 0 1 2; do
      if curl -fsS --max-time 2 "localhost:$((BASE_PORT + i))/status" 2>/dev/null | grep -q '"leader":true'; then
        leaders+=("$i")
      fi
    done
    if [ "${#leaders[@]}" -eq 1 ]; then
      echo "${leaders[0]}"
      return 0
    fi
    sleep 3
  done
  echo "FAIL: exactly one leader was not elected" >&2
  return 1
}

put_value() {
  local port=$1 code
  for _ in $(seq 1 10); do
    code=$(curl -s --max-time 5 -o /dev/null -w '%{http_code}' -X PUT --data "$VALUE" "localhost:$port/kv/$KEY" || true)
    if [ "$code" = "204" ]; then
      return 0
    fi
    sleep 2
  done
  echo "FAIL: PUT did not succeed (last status: $code)" >&2
  return 1
}

get_value() {
  local port=$1 out
  for _ in $(seq 1 10); do
    if out=$(curl -fsS --max-time 5 "localhost:$port/kv/$KEY" 2>/dev/null); then
      echo "$out"
      return 0
    fi
    sleep 2
  done
  echo "FAIL: GET did not succeed" >&2
  return 1
}

echo "==> waiting for StatefulSet"
kubectl -n "$NS" rollout status statefulset/raft-kv --timeout=240s

start_forwards
leader=$(wait_for_leader)
echo "==> leader elected: raft-kv-$leader"

put_value "$((BASE_PORT + leader))"
echo "==> wrote $KEY=$VALUE"

echo "==> rolling restart"
kubectl -n "$NS" rollout restart statefulset/raft-kv
kubectl -n "$NS" rollout status statefulset/raft-kv --timeout=300s

start_forwards
leader=$(wait_for_leader)
echo "==> leader after restart: raft-kv-$leader"

got=$(get_value "$((BASE_PORT + leader))")
if [ "$got" != "$VALUE" ]; then
  echo "FAIL: data lost. expected '$VALUE', got '$got'" >&2
  exit 1
fi
echo "PASS: cluster has one leader and data survived the rolling restart"
