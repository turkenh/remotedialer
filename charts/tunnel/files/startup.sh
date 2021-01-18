#!/usr/bin/env sh
set -euo pipefail

debug_arg=""
if [ "${DEBUG}" == "true" ]; then
  debug_arg="--debug"
fi

if [ -n "${CLIENT_ID}" ]; then
  /app/tunnel-client --id="${CLIENT_ID}" --connect="${CLIENT_CONNECT_ADDR}" "${debug_arg}"
else
  if [ "${REPLICA_COUNT}" == "1" ]; then
    /app/tunnel-server --listen=":8080" --token="top-secret" "${debug_arg}"
  elif [ "${PODNAME}" == "tunnel-server-0" ]; then
    /app/tunnel-server --listen=":8080" --token="top-secret" --id=0 --peers-config-file=/script/peers-config.yaml "${debug_arg}"
  elif [ "${PODNAME}" == "tunnel-server-1" ]; then
    /app/tunnel-server --listen=":8080" --token="top-secret" --id=1 --peers-config-file=/script/peers-config.yaml "${debug_arg}"
  elif [ "${PODNAME}" == "tunnel-server-2" ]; then
    /app/tunnel-server --listen=":8080" --token="top-secret" --id=2 --peers-config-file=/script/peers-config.yaml "${debug_arg}"
  else
    echo "unknown pod name $PODNAME"
    exit 1
  fi
fi
