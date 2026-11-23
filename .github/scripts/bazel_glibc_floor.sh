#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs Bazel in driver/linux.Dockerfile, so Linux binaries need glibc 2.35 at most.
# Usage: bazel_glibc_floor.sh <bazel command> [args...]
#
# DRIVER_BUILDER_IMAGE names the image of the Dockerfile, pinned by digest, and GH_TOKEN
# pulls it from GitHub Container Registry.
#
# The workspace and the Bazel output root mount at their host paths, so the bazel-bin
# symlinks stay valid on the host after the build. Each runner has its own output root.

set -euo pipefail

image=${DRIVER_BUILDER_IMAGE:?}
command=$1
shift
root="$HOME/.cache/bazel-glibc-floor/${RUNNER_NAME:-local}"
mkdir -p "$root"
# Each job logs in to its own config, so a job that ends never logs out another.
DOCKER_CONFIG=$(mktemp -d)
export DOCKER_CONFIG
# A cancelled job signals this script, not the container, so the build runs in the
# background where a trap can stop the container at once.
name="bazel-glibc-floor-$$"
cleanup() {
    docker kill "$name" > /dev/null 2>&1 || true
    rm -rf "$DOCKER_CONFIG"
}
trap cleanup INT TERM EXIT
docker login ghcr.io --username "${GITHUB_ACTOR:?}" --password-stdin <<< "${GH_TOKEN:?}"
docker pull --quiet "$image"
# The image joins every action key, so the remote cache never serves a tool that a host
# build linked against a newer glibc, or an LTO object from another GCC. NI Linux
# Real-Time ships an older libstdc++ than GCC 13, so binaries link libstdc++ and libgcc
# statically. Bazel's toolchain names -lstdc++ itself, so -static-libstdc++ alone has
# no effect. An -O3 compile of a Driver file takes up to 2 GB, so eight at once fill the
# 16 GB of an Ubuntu build bot. Bazel rates a compile at a fixed small amount of memory,
# so only --jobs limits them.
docker run --rm --init --name "$name" --user "$(id -u):$(id -g)" \
    --env HOME="$root" --env USER=runner \
    --volume "$PWD:$PWD" --volume "$root:$root" --workdir "$PWD" "$image" \
    bazel --output_user_root="$root/bazel" "$command" --jobs=4 \
    --action_env=BUILD_IMAGE="$image" --host_action_env=BUILD_IMAGE="$image" \
    --repo_env=BAZEL_LINKLIBS=-l%:libstdc++.a:-lm --linkopt=-static-libgcc "$@" &
wait $!
