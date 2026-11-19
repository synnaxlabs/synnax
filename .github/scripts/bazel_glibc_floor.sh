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
# The workspace and the Bazel output root mount at their host paths, so the bazel-bin
# symlinks stay valid on the host after the build. Each runner has its own output root.

set -euo pipefail

dockerfile=driver/linux.Dockerfile
image=synnax-driver-linux:$(sha256sum "$dockerfile" | cut -c1-12)
if ! docker image inspect "$image" > /dev/null 2>&1; then
    docker build --tag "$image" - < "$dockerfile"
fi
root="$HOME/.cache/bazel-glibc-floor/${RUNNER_NAME:-local}"
mkdir -p "$root"
# A cancelled job signals this script, not the container.
name="bazel-glibc-floor-$$"
trap 'docker kill "$name" > /dev/null 2>&1 || true' EXIT
command=$1
shift
# The image joins every action key, so the remote cache never serves a tool that a host
# build linked against a newer glibc.
docker run --rm --init --name "$name" --user "$(id -u):$(id -g)" \
    --env HOME="$root" --env USER=runner \
    --volume "$PWD:$PWD" --volume "$root:$root" --workdir "$PWD" "$image" \
    bazel --output_user_root="$root/bazel" "$command" --action_env=BUILD_IMAGE="$image" \
    --host_action_env=BUILD_IMAGE="$image" "$@"
