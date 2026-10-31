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
#   preview up <pr> <image>   Starts or replaces the Core for a pull request.
#   preview down <pr>         Removes the Core for a pull request.
#   preview sweep             Removes every Core deployed more than the TTL ago.

set -euo pipefail

# shellcheck source=/dev/null
source "${PREVIEW_ENV:-/etc/synnax-preview.env}"
: "${PREVIEW_DOMAIN:?}" "${PREVIEW_SITES:?}" "${PREVIEW_TTL_DAYS:?}"
: "${PREVIEW_LICENSE_PARAMETER:?}" "${PREVIEW_REGION:?}"

# One command at a time, so a sweep never removes a preview that up just replaced.
exec 9> "$PREVIEW_SITES/.lock"
flock 9

container() { echo "synnax-pr-$1"; }

# Each deploy pulls a new image, so the replaced one must go or the disk fills.
prune_images() { docker image prune --all --force > /dev/null; }

reload_caddy() { docker exec caddy caddy reload --config /etc/caddy/Caddyfile; }

check_pr() {
    if [[ ! $1 =~ ^[0-9]+$ ]]; then
        echo "invalid pull request number: $1" >&2
        exit 1
    fi
}

up() {
    local pr=$1 image=$2 name
    check_pr "$pr"
    name=$(container "$pr")
    # An unlicensed Core rejects every request but login. Fetched first, so a failure
    # leaves the running Core in place.
    SYNNAX_LICENSE_KEY=$(aws ssm get-parameter --region "$PREVIEW_REGION" \
        --name "$PREVIEW_LICENSE_PARAMETER" --with-decryption \
        --query Parameter.Value --output text)
    export SYNNAX_LICENSE_KEY
    docker pull --quiet "$image"
    docker rm --force "$name" > /dev/null
    # In-memory storage: each deploy starts from an empty Core.
    docker run --detach --name "$name" --network preview --restart unless-stopped \
        --memory 2g --cpus 1 --env SYNNAX_LICENSE_KEY \
        --label synnax.preview="$pr" --label synnax.preview.deployed="$(date +%s)" \
        "$image" --listen 0.0.0.0:9090 --insecure --mem --no-driver > /dev/null
    prune_images
    for _ in $(seq 60); do
        if docker exec "$name" curl -fsS -o /dev/null -X POST \
            -H "Content-Type: application/json" -d '{}' \
            http://localhost:9090/api/v1/connectivity/check; then
            printf 'pr-%s.%s {\n\treverse_proxy %s:9090\n}\n' \
                "$pr" "$PREVIEW_DOMAIN" "$name" > "$PREVIEW_SITES/pr-$pr.caddy"
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
    rm -f "$PREVIEW_SITES/pr-$1.caddy"
    reload_caddy
    prune_images
}

sweep() {
    local cutoff pr deployed
    cutoff=$(($(date +%s) - PREVIEW_TTL_DAYS * 86400))
    docker ps --all --filter label=synnax.preview \
        --format '{{.Label "synnax.preview"}} {{.Label "synnax.preview.deployed"}}' \
        | while read -r pr deployed; do
            if ((deployed < cutoff)); then down "$pr"; fi
        done
    # A failed deploy can leave a pulled image that no container uses.
    prune_images
}

case "${1:-}" in
    up) up "$2" "$3" ;;
    down) down "$2" ;;
    sweep) sweep ;;
    *)
        echo "usage: preview up <pr> <image> | down <pr> | sweep" >&2
        exit 2
        ;;
esac
