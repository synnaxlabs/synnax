#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Manages the preview Cores on this host.
#
#   preview up <pr> <image> [human]
#                             Starts or replaces the Core for a pull request. "human"
#                             marks a pull request that waits on a human review. At
#                             the cap, it first removes a preview and prints "evicted
#                             <pr>": unmarked ones before marked ones, and within each,
#                             the one deployed or viewed longest ago.
#   preview down <pr>         Removes the Core for a pull request.
#   preview sweep             Removes every Core deployed more than the TTL ago.

set -euo pipefail

# shellcheck source=/dev/null
source "${PREVIEW_ENV:-/etc/synnax-preview.env}"
: "${PREVIEW_DOMAIN:?}" "${PREVIEW_SITES:?}" "${PREVIEW_TTL_DAYS:?}"
: "${PREVIEW_LICENSE_PARAMETER:?}" "${PREVIEW_REGION:?}"
: "${PREVIEW_MAX:?}" "${PREVIEW_CPUS:?}" "${PREVIEW_MEMORY:?}" "${PREVIEW_LOGS:?}"

# One command at a time, so a sweep never removes a preview that up just replaced.
exec 9> "$PREVIEW_SITES/.lock"
flock 9

container() { echo "synnax-pr-$1"; }

reload_caddy() { docker exec caddy caddy reload --config /etc/caddy/Caddyfile; }

# Prints "<pr> <deployed> <human>" for each preview.
previews() {
    local format='{{.Label "synnax.preview"}} {{.Label "synnax.preview.deployed"}}'
    format+=' {{.Label "synnax.preview.human"}}'
    docker ps --all --filter label=synnax.preview --format "$format"
}

# Prints the pull requests of the previews other than $1, in the order up evicts them.
# A preview's access log changes on each request, so its time is the last view.
eviction_order() {
    local pr deployed human active log
    previews | while read -r pr deployed human; do
        if [[ $pr == "$1" ]]; then continue; fi
        active=$deployed
        log="$PREVIEW_LOGS/pr-$pr.log"
        if [[ -f $log ]] && (($(date -r "$log" +%s) > active)); then
            active=$(date -r "$log" +%s)
        fi
        echo "$pr ${human:-0} $active"
    done | sort -k2,2n -k3,3n | cut -d' ' -f1
}

check_pr() {
    if [[ ! $1 =~ ^[0-9]+$ ]]; then
        echo "invalid pull request number: $1" >&2
        exit 1
    fi
}

up() {
    local pr=$1 image=$2 human=0 name
    check_pr "$pr"
    case "${3:-}" in
        "") ;;
        human) human=1 ;;
        *)
            echo "invalid review mark: $3" >&2
            exit 1
            ;;
    esac
    name=$(container "$pr")
    # An unlicensed Core rejects every request but login. The license and the image come
    # first, so a failure to get either leaves every running Core in place.
    SYNNAX_LICENSE_KEY=$(aws ssm get-parameter --region "$PREVIEW_REGION" \
        --name "$PREVIEW_LICENSE_PARAMETER" --with-decryption \
        --query Parameter.Value --output text)
    export SYNNAX_LICENSE_KEY
    docker pull --quiet "$image"
    local others=() other i
    while read -r other; do others+=("$other"); done < <(eviction_order "$pr")
    for ((i = 0; i <= ${#others[@]} - PREVIEW_MAX; i++)); do
        down "${others[i]}"
        echo "evicted ${others[i]}"
    done
    docker rm --force "$name" > /dev/null
    # In-memory storage: each deploy starts from an empty Core.
    docker run --detach --name "$name" --network preview --restart unless-stopped \
        --memory "$PREVIEW_MEMORY" --cpus "$PREVIEW_CPUS" --env SYNNAX_LICENSE_KEY \
        --label synnax.preview="$pr" --label synnax.preview.deployed="$(date +%s)" \
        --label synnax.preview.human="$human" \
        "$image" --listen 0.0.0.0:9090 --insecure --mem --no-driver > /dev/null
    for _ in $(seq 60); do
        if docker exec "$name" curl -fsS -o /dev/null -X POST \
            -H "Content-Type: application/json" -d '{}' \
            http://localhost:9090/api/v1/connectivity/check; then
            cat > "$PREVIEW_SITES/pr-$pr.caddy" << SITE
pr-$pr.$PREVIEW_DOMAIN {
    log {
        output file $PREVIEW_LOGS/pr-$pr.log
    }
    reverse_proxy $name:9090
}
SITE
            reload_caddy
            return
        fi
        sleep 2
    done
    docker logs --tail 100 "$name" >&2
    echo "Core for pull request $pr did not start" >&2
    exit 1
}

down() {
    check_pr "$1"
    docker rm --force "$(container "$1")" > /dev/null
    rm -f "$PREVIEW_SITES/pr-$1.caddy" "$PREVIEW_LOGS/pr-$1.log"
    reload_caddy
}

sweep() {
    local cutoff pr deployed
    cutoff=$(($(date +%s) - PREVIEW_TTL_DAYS * 86400))
    previews | while read -r pr deployed _; do
        if ((deployed < cutoff)); then down "$pr"; fi
    done
    docker image prune --all --force --filter "until=$((PREVIEW_TTL_DAYS * 24))h"
}

case "${1:-}" in
    up) up "$2" "$3" "${4:-}" ;;
    down) down "$2" ;;
    sweep) sweep ;;
    *)
        echo "usage: preview up <pr> <image> [human] | down <pr> | sweep" >&2
        exit 2
        ;;
esac
