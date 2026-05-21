#!/bin/sh
set -e

CONFIG_SRC="/etc/telegraf/telegraf.conf"
CONFIG_DST="/tmp/telegraf.conf"

envsubst < "$CONFIG_SRC" > "$CONFIG_DST"

exec /entrypoint.sh "$@" --config "$CONFIG_DST"