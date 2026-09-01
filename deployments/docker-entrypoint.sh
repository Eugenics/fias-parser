#!/bin/sh
set -eu

validate_schedule() {
    name=$1
    value=$2
    fields=$(printf '%s\n' "$value" | awk '{print NF}')
    case "$value" in
        *[!0-9*,/?\ -]*)
            echo "$name contains unsupported characters: $value" >&2
            exit 1
            ;;
    esac
    if [ "$fields" -ne 5 ]; then
        echo "$name must contain exactly five cron fields: $value" >&2
        exit 1
    fi
}

if [ "${1:-}" = "crond" ]; then
    FULL_SCHEDULE=${FULL_SCHEDULE:-0 2 * * 0}
    DELTA_SCHEDULE=${DELTA_SCHEDULE:-0 3 * * *}
    validate_schedule FULL_SCHEDULE "$FULL_SCHEDULE"
    validate_schedule DELTA_SCHEDULE "$DELTA_SCHEDULE"

    {
        printf '%s /usr/local/bin/run-scheduled.sh full >> /proc/1/fd/1 2>> /proc/1/fd/2\n' "$FULL_SCHEDULE"
        printf '%s /usr/local/bin/run-scheduled.sh delta >> /proc/1/fd/1 2>> /proc/1/fd/2\n' "$DELTA_SCHEDULE"
    } > /etc/crontabs/root

    echo "Scheduled full load:  $FULL_SCHEDULE"
    echo "Scheduled delta load: $DELTA_SCHEDULE"

    if [ "${RUN_ON_STARTUP:-false}" = "true" ]; then
        /usr/local/bin/run-scheduled.sh "${STARTUP_LOAD_TYPE:-delta}"
    fi
fi

exec "$@"
