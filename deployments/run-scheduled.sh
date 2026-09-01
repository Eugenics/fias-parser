#!/bin/sh
set -eu

load_type=${1:-}
lock_dir=/tmp/fias-parser-job.lock

if ! mkdir "$lock_dir" 2>/dev/null; then
    echo "$(date -Iseconds) another GAR load is running; $load_type load skipped"
    exit 0
fi
trap 'rmdir "$lock_dir"' EXIT INT TERM

echo "$(date -Iseconds) starting $load_type GAR load"
case "$load_type" in
    full)
        fias-parser 2
        fias-parser 0
        ;;
    delta)
        fias-parser 3
        fias-parser 1
        ;;
    *)
        echo "unknown load type: $load_type" >&2
        exit 2
        ;;
esac
echo "$(date -Iseconds) completed $load_type GAR load"
