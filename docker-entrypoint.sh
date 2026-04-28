#!/bin/sh
set -eu

upload_dir="${NAVBOX_UPLOAD_DIR:-/app/data/uploads}"

mkdir -p "$upload_dir"

if [ "$(id -u)" = "0" ]; then
	chown -R navbox:navbox "$upload_dir"
	exec su-exec navbox "$@"
fi

exec "$@"
