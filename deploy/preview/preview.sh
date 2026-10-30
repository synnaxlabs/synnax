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
#   preview up <pr> <image> [human|none <time>]
#                             Starts or replaces the Core for a pull request, and
#                             marks it as mark does. At the cap, it first removes a
#                             preview and prints "evicted <pr> <UTC time>": unmarked
#                             ones before marked ones, and within each, the one
#                             deployed or viewed longest ago.
#   preview mark <pr> human|none <time>
#                             Sets whether a pull request waits on a human review, as
#                             its labels showed at <time> (Unix seconds). A mark older
#                             than the saved one changes nothing.
#   preview upgrade           Starts Caddy, or recreates it without its access log
#                             mount, and applies the CPU and memory limits to the
#                             running previews.
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

# Each deploy pulls a new image, so the replaced one must go or the disk fills.
prune_images() { docker image prune --all --force > /dev/null; }

reload_caddy() { docker exec caddy caddy reload --config /etc/caddy/Caddyfile; }

# Prints "<pr> <deployed>" for each preview.
previews() {
    docker ps --all --filter label=synnax.preview \
        --format '{{.Label "synnax.preview"}} {{.Label "synnax.preview.deployed"}}'
}

mark_file() { echo "$PREVIEW_SITES/pr-$1.human"; }

# Prints the pull requests of the previews other than $1, in the order up evicts them.
# A preview's access log changes on each request, so its time is the last view.
eviction_order() {
    local pr deployed human active log
    previews | while read -r pr deployed; do
        if [[ $pr == "$1" ]]; then continue; fi
        human=0
        if [[ $(cut -d' ' -f2 "$(mark_file "$pr")" 2> /dev/null) == human ]]; then
            human=1
        fi
        active=$deployed
        log="$PREVIEW_LOGS/pr-$pr.log"
        if [[ -f $log ]] && (($(date -r "$log" +%s) > active)); then
            active=$(date -r "$log" +%s)
        fi
        echo "$pr $human $active"
    done | sort -k2,2n -k3,3n | cut -d' ' -f1
}

check_pr() {
    if [[ ! $1 =~ ^[0-9]+$ ]]; then
        echo "invalid pull request number: $1" >&2
        exit 1
    fi
}

mark() {
    local file saved
    check_pr "$1"
    if [[ ! $2 =~ ^(human|none)$ || ! $3 =~ ^[0-9]+$ ]]; then
        echo "invalid review mark: $2 $3" >&2
        exit 1
    fi
    file=$(mark_file "$1")
    saved=$(cut -d' ' -f1 "$file" 2> /dev/null || echo 0)
    if (($3 >= saved)); then echo "$3 $2" > "$file"; fi
}

up() {
    local pr=$1 image=$2 name
    check_pr "$pr"
    mark "$pr" "${3:-none}" "${4:-0}"
    name=$(container "$pr")
    # An unlicensed Core rejects every request but login. The license and the image come
    # first, so a failure to get either leaves every running Core in place.
    SYNNAX_LICENSE_KEY=$(aws ssm get-parameter --region "$PREVIEW_REGION" \
        --name "$PREVIEW_LICENSE_PARAMETER" --with-decryption \
        --query Parameter.Value --output text)
    export SYNNAX_LICENSE_KEY
    docker pull --quiet "$image"
    local order others=() other i
    # A plain assignment, so a failed listing stops the deploy before the cap.
    order=$(eviction_order "$pr")
    if [[ -n $order ]]; then
        while read -r other; do others+=("$other"); done <<< "$order"
    fi
    for ((i = 0; i <= ${#others[@]} - PREVIEW_MAX; i++)); do
        # No prune here: it would delete the image pulled above.
        remove "${others[i]}"
        echo "evicted ${others[i]} $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    done
    docker rm --force "$name" > /dev/null
    # In-memory storage: each deploy starts from an empty Core.
    docker run --detach --name "$name" --network preview --restart unless-stopped \
        --memory "$PREVIEW_MEMORY" --memory-swap "$PREVIEW_MEMORY" \
        --cpus "$PREVIEW_CPUS" --env SYNNAX_LICENSE_KEY \
        --label synnax.preview="$pr" --label synnax.preview.deployed="$(date +%s)" \
        "$image" --listen 0.0.0.0:9090 --insecure --mem --no-driver > /dev/null
    prune_images
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

remove() {
    docker rm --force "$(container "$1")" > /dev/null
    rm -f "$PREVIEW_SITES/pr-$1.caddy" "$PREVIEW_LOGS/pr-$1.log" "$(mark_file "$1")"
    reload_caddy
}

down() {
    check_pr "$1"
    remove "$1"
    prune_images
}

upgrade() {
    local mounts preview
    # No Caddy container yet, on a new host, reads as no mounts.
    mounts=$(docker inspect -f '{{range .Mounts}}{{.Destination}} {{end}}' caddy \
        2> /dev/null || true)
    if [[ " $mounts " != *" $PREVIEW_LOGS "* ]]; then
        mkdir -p "$PREVIEW_LOGS"
        docker rm --force caddy > /dev/null
        docker run --detach --name caddy --network preview --restart unless-stopped \
            --publish 80:80 --publish 443:443 --volume /etc/caddy:/etc/caddy:ro \
            --volume "$PREVIEW_LOGS:$PREVIEW_LOGS" --volume caddy_data:/data \
            caddy:2.10 > /dev/null
    fi
    for preview in $(docker ps --all --quiet --filter label=synnax.preview); do
        docker update --cpus "$PREVIEW_CPUS" --memory "$PREVIEW_MEMORY" \
            --memory-swap "$PREVIEW_MEMORY" "$preview" > /dev/null
    done
}

sweep() {
    local cutoff pr deployed
    cutoff=$(($(date +%s) - PREVIEW_TTL_DAYS * 86400))
    previews | while read -r pr deployed _; do
        if ((deployed < cutoff)); then down "$pr"; fi
    done
    # A failed deploy can leave a pulled image that no container uses.
    prune_images
}

case "${1:-}" in
    up) up "$2" "$3" "${4:-}" "${5:-}" ;;
    mark) mark "$2" "${3:-}" "${4:-}" ;;
    upgrade) upgrade ;;
    down) down "$2" ;;
    sweep) sweep ;;
    *)
        echo "usage: preview up <pr> <image> [human|none <time>]" \
            "| mark <pr> human|none <time> | upgrade | down <pr> | sweep" >&2
        exit 2
        ;;
esac
