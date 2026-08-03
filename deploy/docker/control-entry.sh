#!/usr/bin/env sh
# control-entry.sh - chown the data volume (root) then drop to the
# non-root fairwave user. The Go binary is static; sh + coreutils suffice.
set -e

if [ "$(id -u)" = "0" ]; then
    mkdir -p /var/lib/fairwave 2>/dev/null || true
    chown -R fairwave:fairwave /var/lib/fairwave 2>/dev/null || true
    exec su-exec fairwave /usr/local/bin/fairwave-control "$@"
fi
exec /usr/local/bin/fairwave-control "$@"
