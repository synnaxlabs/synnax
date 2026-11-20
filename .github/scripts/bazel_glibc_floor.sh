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
# The PPA's GCC is not pinned, and GCC rejects LTO objects from another version, so the
# action keys name the installed compiler as well as the Dockerfile.
gcc=$(docker run --rm "$image" dpkg-query --showformat='${Version}' --show gcc-13)
key="$image+gcc-13=$gcc"
root="$HOME/.cache/bazel-glibc-floor/${RUNNER_NAME:-local}"
mkdir -p "$root"
command=$1
shift
# A cancelled job signals this script, not the container, so the build runs in the
# background where a trap can stop the container at once.
name="bazel-glibc-floor-$$"
trap 'docker kill "$name" > /dev/null 2>&1 || true' INT TERM EXIT
# The image joins every action key, so the remote cache never serves a tool that a host
# build linked against a newer glibc. NI Linux Real-Time ships an older libstdc++ than
# GCC 13, so binaries link libstdc++ and libgcc statically. Bazel's toolchain names
# -lstdc++ itself, so -static-libstdc++ alone has no effect.
docker run --rm --init --name "$name" --user "$(id -u):$(id -g)" \
    --env HOME="$root" --env USER=runner \
    --volume "$PWD:$PWD" --volume "$root:$root" --workdir "$PWD" "$image" \
    bazel --output_user_root="$root/bazel" "$command" \
    --action_env=BUILD_IMAGE="$key" --host_action_env=BUILD_IMAGE="$key" \
    --repo_env=BAZEL_LINKLIBS=-l%:libstdc++.a:-lm --linkopt=-static-libgcc "$@" &
wait $!
