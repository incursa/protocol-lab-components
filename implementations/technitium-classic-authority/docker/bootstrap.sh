#!/usr/bin/env bash
set -euo pipefail

if [[ "${PLAB_PLAN_ONLY:-false}" == true ]]; then
  echo "Technitium DNS Server 15.4"
  exit 0
fi

password="${DNS_SERVER_ADMIN_PASSWORD:-plab-local-only}"
export DNS_SERVER_ADMIN_PASSWORD="$password"
dotnet /opt/technitium/dns/DnsServerApp.dll /etc/dns &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' TERM INT EXIT

login=''
for _ in $(seq 1 120); do
  if login=$(curl -fsS -G http://127.0.0.1:5380/api/user/login \
      --data-urlencode user=admin \
      --data-urlencode "pass=$password" \
      --data-urlencode includeInfo=true 2>/dev/null); then
    break
  fi
  sleep .25
done

token=$(printf '%s' "$login" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
version=$(printf '%s' "$login" | sed -n 's/.*"version":"\([^"]*\)".*/\1/p')
[[ -n "$token" ]]
[[ "$version" == 15.4* ]]

settings=$(curl -fsS -G http://127.0.0.1:5380/api/settings/set \
  -H "Authorization: Bearer $token" \
  --data-urlencode qpmPrefixLimitsIPv4=false \
  --data-urlencode qpmPrefixLimitsIPv6=false)
printf '%s' "$settings" | grep -q '"status":"ok"'

zone=$(curl -fsS -G http://127.0.0.1:5380/api/zones/create \
  -H "Authorization: Bearer $token" \
  --data-urlencode zone=plab.test \
  --data-urlencode type=Primary)
printf '%s' "$zone" | grep -q '"status":"ok"'

record=$(curl -fsS -G http://127.0.0.1:5380/api/zones/records/add \
  -H "Authorization: Bearer $token" \
  --data-urlencode zone=plab.test \
  --data-urlencode domain=plab.test \
  --data-urlencode type=A \
  --data-urlencode ipAddress=192.0.2.1 \
  --data-urlencode ttl=0 \
  --data-urlencode overwrite=true)
printf '%s' "$record" | grep -q '"status":"ok"'

echo "Technitium DNS Server $version ready for classic authoritative DNS over UDP and TCP"
wait "$pid"
