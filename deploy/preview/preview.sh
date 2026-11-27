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
#   preview up <pr> <image> <sample image> [human|none <time>]
#                             Starts or replaces the Core for a pull request, runs
#                             the sample image against it, and marks it as mark does.
#                             At the cap, it first removes a preview: ones not marked
#                             human before ones marked human, and within each, the
#                             one deployed or viewed longest ago.
#   preview mark <pr> human|none <time>
#                             Sets whether a pull request waits on a human review, as
#                             its labels showed at <time> (Unix seconds). A mark older
#                             than the saved one changes nothing.
#   preview upgrade           Gets the wildcard certificate, restarts the GitHub login,
#                             rewrites the site of each preview, starts Caddy or
#                             recreates it without its access log mount, and applies
#                             the CPU limit to the running previews. Memory limits
#                             apply from the next deploy.
#   preview down <pr>         Removes the Core for a pull request.
#   preview sweep             Removes every Core marked none and not deployed or
#                             viewed for the TTL, and renews the certificate when due.
#   preview notices           Prints the notices of removed previews, one per line:
#                             "<pr> <UTC time> evicted <pr that took its place>" or
#                             "<pr> <UTC time> expired".
#   preview ack [<pr> <UTC time>]...
#                             Deletes the notices that the pairs name.

set -euo pipefail

# shellcheck source=/dev/null
source "${PREVIEW_ENV:-/etc/synnax-preview.env}"
: "${PREVIEW_DOMAIN:?}" "${PREVIEW_SITES:?}" "${PREVIEW_TTL_DAYS:?}"
: "${PREVIEW_ZONE_ID:?}" "${PREVIEW_ACME:?}" "${PREVIEW_LOGIN_PARAMETERS:?}"
: "${PREVIEW_GITHUB_ORG:?}"
: "${PREVIEW_LICENSE_PARAMETER:?}" "${PREVIEW_REGION:?}"
: "${PREVIEW_MAX:?}" "${PREVIEW_CPUS:?}" "${PREVIEW_MEMORY:?}" "${PREVIEW_LOGS:?}"
: "${PREVIEW_GO_MEMORY:?}"

# One command at a time, so a sweep never removes a preview that up just replaced.
exec 9> "$PREVIEW_SITES/.lock"
flock 9

container() { echo "synnax-pr-$1"; }

# Each deploy pulls a new image, so the replaced one must go or the disk fills.
prune_images() { docker image prune --all --force > /dev/null; }

reload_caddy() { docker exec caddy caddy reload --config /etc/caddy/Caddyfile "$@"; }

# One wildcard certificate covers every preview, so a new preview never counts against
# the Let's Encrypt limit for the domain.
CERT=$PREVIEW_ACME/certificates/_.$PREVIEW_DOMAIN
LEGO=goacme/lego:v5.7.0
LEGO+=@sha256:9fe6151508c0e1ddc51a6d3e474353520f4b008782bf0c6d31676da72e6d4766

AUTH=auth.$PREVIEW_DOMAIN
OAUTH2_PROXY=quay.io/oauth2-proxy/oauth2-proxy:v7.15.6
OAUTH2_PROXY+=@sha256:c3415e8660af553f5e18f5d90db050614c02493c0ac6260ac59442f6d9b12048

parameter() {
    aws ssm get-parameter --region "$PREVIEW_REGION" --name "$1" --with-decryption \
        --query Parameter.Value --output text
}

# Starts the GitHub login that every preview site checks. Its cookie is on the parent
# domain, so one sign-in covers every preview.
start_login() {
    OAUTH2_PROXY_CLIENT_ID=$(parameter "$PREVIEW_LOGIN_PARAMETERS/client-id")
    OAUTH2_PROXY_CLIENT_SECRET=$(parameter "$PREVIEW_LOGIN_PARAMETERS/client-secret")
    OAUTH2_PROXY_COOKIE_SECRET=$(parameter "$PREVIEW_LOGIN_PARAMETERS/cookie-secret")
    export OAUTH2_PROXY_CLIENT_ID OAUTH2_PROXY_CLIENT_SECRET OAUTH2_PROXY_COOKIE_SECRET
    docker pull --quiet "$OAUTH2_PROXY" > /dev/null
    docker rm --force login > /dev/null
    docker run --detach --name login --network preview --restart unless-stopped \
        --env OAUTH2_PROXY_CLIENT_ID --env OAUTH2_PROXY_CLIENT_SECRET \
        --env OAUTH2_PROXY_COOKIE_SECRET "$OAUTH2_PROXY" --provider=github \
        --github-org="$PREVIEW_GITHUB_ORG" --email-domain='*' \
        --http-address=0.0.0.0:4180 --reverse-proxy --upstream=static://202 \
        --redirect-url="https://$AUTH/oauth2/callback" \
        --cookie-domain=".$PREVIEW_DOMAIN" --whitelist-domain=".$PREVIEW_DOMAIN" \
        > /dev/null
    cat > "$PREVIEW_SITES/auth.caddy" << SITE
$AUTH {
    tls $CERT.crt $CERT.key
    reverse_proxy login:4180
}
SITE
}

cert_hash() { sha256sum "$CERT.crt" 2> /dev/null || true; }

# Gets the certificate, or renews it when it is due. The host may send DNS only to the
# VPC resolver, so lego checks the challenge record there and not at the name servers
# of the zone. lego gets the role from the instance metadata, which only the host
# network can reach.
renew_cert() {
    docker run --rm --network host --volume "$PREVIEW_ACME:/acme" \
        --env AWS_REGION="$PREVIEW_REGION" \
        --env AWS_HOSTED_ZONE_ID="$PREVIEW_ZONE_ID" "$LEGO" run --accept-tos \
        --no-random-sleep --dns route53 --dns.resolvers 169.254.169.253:53 \
        --dns.propagation.disable-ans --domains "*.$PREVIEW_DOMAIN" --path /acme
}

# Prints "<pr> <deployed>" for each preview.
previews() {
    docker ps --all --filter label=synnax.preview \
        --format '{{.Label "synnax.preview"}} {{.Label "synnax.preview.deployed"}}'
}

mark_file() { echo "$PREVIEW_SITES/pr-$1.human"; }

marked_human() { [[ $(cut -d' ' -f2 "$(mark_file "$1")" 2> /dev/null) == human ]]; }

# A preview deployed before marks and access logs existed has neither, so nothing
# shows whether it waits on a review or gets views. The sweep keeps it until its next
# deploy writes a mark.
marked_none() { [[ $(cut -d' ' -f2 "$(mark_file "$1")" 2> /dev/null) == none ]]; }

# Notices stay on the host until a deploy job posts them and sends ack, so a run that
# is cancelled or fails before it posts them loses none.
NOTICES=$PREVIEW_SITES/notices
TIME='^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'

notice() { echo "$1 $(date -u +%Y-%m-%dT%H:%M:%SZ) $2" >> "$NOTICES"; }

# Prints the later of deployed time $2 and the last view of pull request $1. A
# preview's access log changes on each request, so its time is the last view.
active() {
    local log="$PREVIEW_LOGS/pr-$1.log"
    if [[ -f $log ]] && (($(date -r "$log" +%s) > $2)); then
        date -r "$log" +%s
    else
        echo "$2"
    fi
}

# Prints the pull requests of the previews other than $1, in the order up evicts them.
eviction_order() {
    local pr deployed human
    previews | while read -r pr deployed; do
        if [[ $pr == "$1" ]]; then continue; fi
        human=0
        if marked_human "$pr"; then human=1; fi
        echo "$pr $human $(active "$pr" "$deployed")"
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

# Routes the site of pull request $1 to its Core, behind the GitHub login.
write_site() {
    # A Caddy reload closes every proxied websocket unless delayed, so without the delay
    # one preview's deploy breaks the streams of all the others.
    cat > "$PREVIEW_SITES/pr-$1.caddy" << SITE
pr-$1.$PREVIEW_DOMAIN {
    tls $CERT.crt $CERT.key
    log {
        output file $PREVIEW_LOGS/pr-$1.log
    }
    forward_auth login:4180 {
        uri /oauth2/auth
        @signed_out status 401
        handle_response @signed_out {
            redir * https://$AUTH/oauth2/start?rd={scheme}://{host}{uri}
        }
    }
    reverse_proxy $(container "$1"):9090 {
        stream_close_delay 1h
    }
}
SITE
}

up() {
    local pr=$1 image=$2 sample=$3 name
    check_pr "$pr"
    mark "$pr" "${4:-none}" "${5:-0}"
    name=$(container "$pr")
    # An unlicensed Core rejects every request but login. The license and the image come
    # first, so a failure to get either leaves every running Core in place.
    SYNNAX_LICENSE_KEY=$(parameter "$PREVIEW_LICENSE_PARAMETER")
    export SYNNAX_LICENSE_KEY
    docker pull --quiet "$image"
    local order others=() other i
    # A plain assignment, so a failed listing stops the deploy before the cap.
    order=$(eviction_order "$pr")
    if [[ -n $order ]]; then
        while read -r other; do others+=("$other"); done <<< "$order"
    fi
    for ((i = 0; i <= ${#others[@]} - PREVIEW_MAX; i++)); do
        # Written first, so a failed removal still gets a notice.
        notice "${others[i]}" "evicted $pr"
        # No prune here: it would delete the image pulled above.
        remove "${others[i]}"
    done
    docker rm --force "$name" > /dev/null
    # In-memory storage: each deploy starts from an empty Core.
    docker run --detach --name "$name" --network preview --restart unless-stopped \
        --memory "$PREVIEW_MEMORY" --memory-swap "$PREVIEW_MEMORY" \
        --cpus "$PREVIEW_CPUS" --env SYNNAX_LICENSE_KEY \
        --env GOMEMLIMIT="$PREVIEW_GO_MEMORY" \
        --label synnax.preview="$pr" --label synnax.preview.deployed="$(date +%s)" \
        "$image" --listen 0.0.0.0:9090 --insecure --mem --no-driver > /dev/null
    prune_images
    for _ in $(seq 60); do
        if docker exec "$name" curl -fsS -o /dev/null -X POST \
            -H "Content-Type: application/json" -d '{}' \
            http://localhost:9090/api/v1/connectivity/check; then
            write_site "$pr"
            reload_caddy
            # The load takes about half a minute, so other commands need not wait on it.
            flock --unlock 9
            docker run --rm --network preview --memory "$PREVIEW_MEMORY" \
                --cpus "$PREVIEW_CPUS" "$sample" "http://$name:9090"
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
    local mounts preview pr
    renew_cert
    start_login
    # A running preview gets its site in the current form, so it needs no redeploy.
    previews | while read -r pr _; do write_site "$pr"; done
    # No Caddy container yet, on a new host, reads as no mounts.
    mounts=$(docker inspect -f '{{range .Mounts}}{{.Destination}} {{end}}' caddy \
        2> /dev/null || true)
    if [[ " $mounts " != *" $PREVIEW_LOGS "* ]]; then
        mkdir -p "$PREVIEW_LOGS"
        docker rm --force caddy > /dev/null
        docker run --detach --name caddy --network preview --restart unless-stopped \
            --publish 443:443 --volume /etc/caddy:/etc/caddy:ro \
            --volume "$PREVIEW_LOGS:$PREVIEW_LOGS" --volume caddy_data:/data \
            caddy:2.10 > /dev/null
    else
        reload_caddy --force
    fi
    # A running Core cannot get GOMEMLIMIT, so a lower memory limit could kill it.
    for preview in $(docker ps --all --quiet --filter label=synnax.preview); do
        docker update --cpus "$PREVIEW_CPUS" "$preview" > /dev/null
    done
}

sweep() {
    local cutoff pr deployed before
    cutoff=$(($(date +%s) - PREVIEW_TTL_DAYS * 86400))
    previews | while read -r pr deployed _; do
        if ! marked_none "$pr" || (($(active "$pr" "$deployed") >= cutoff)); then
            continue
        fi
        notice "$pr" expired
        down "$pr"
    done
    # A failed deploy can leave a pulled image that no container uses.
    prune_images
    before=$(cert_hash)
    renew_cert
    # Caddy loads a changed certificate file only on a forced reload.
    if [[ $(cert_hash) != "$before" ]]; then reload_caddy --force; fi
}

notices() { if [[ -f $NOTICES ]]; then cat "$NOTICES"; fi; }

ack() {
    local rest
    if (($# % 2)); then
        echo "usage: preview ack [<pr> <UTC time>]..." >&2
        exit 2
    fi
    while (($#)); do
        check_pr "$1"
        if [[ ! $2 =~ $TIME ]]; then
            echo "invalid notice time: $2" >&2
            exit 1
        fi
        if [[ -f $NOTICES ]]; then
            rest=$(awk -v pr="$1" -v at="$2" '!($1 == pr && $2 == at)' "$NOTICES")
            if [[ -n $rest ]]; then echo "$rest" > "$NOTICES"; else rm "$NOTICES"; fi
        fi
        shift 2
    done
}

case "${1:-}" in
    up) up "$2" "$3" "${4:-}" "${5:-}" "${6:-}" ;;
    mark) mark "$2" "${3:-}" "${4:-}" ;;
    upgrade) upgrade ;;
    down) down "$2" ;;
    sweep) sweep ;;
    notices) notices ;;
    ack) ack "${@:2}" ;;
    *)
        echo "usage: preview up <pr> <image> <sample image> [human|none <time>]" \
            "| mark <pr> human|none <time> | upgrade | down <pr> | sweep" \
            "| notices | ack [<pr> <UTC time>]..." >&2
        exit 2
        ;;
esac
