#!/bin/sh
set -e

mkdir -p "${MONITORING_LOGS}"

CONFIG_SRC="/etc/telegraf/telegraf.conf"
CONFIG_DST="/tmp/telegraf.conf"

cp "$CONFIG_SRC" "$CONFIG_DST"

sed -i "s|__PROMETHEUS_URLS__|${PROMETHEUS_URLS}|g" "$CONFIG_DST"

echo "Rendered prometheus urls:"
grep -n "urls =" "$CONFIG_DST"

exec /entrypoint.sh "$@" --config "$CONFIG_DST"