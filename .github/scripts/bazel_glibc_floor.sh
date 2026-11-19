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
# symlinks stay valid on the host after the build.

set -euo pipefail

dockerfile=driver/linux.Dockerfile
image=synnax-driver-linux:$(sha256sum "$dockerfile" | cut -c1-12)
if ! docker image inspect "$image" > /dev/null 2>&1; then
    docker build --tag "$image" - < "$dockerfile"
fi
root="$HOME/.cache/bazel-glibc-floor"
mkdir -p "$root"
docker run --rm --user "$(id -u):$(id -g)" --env HOME="$root" --env USER=runner \
    --volume "$PWD:$PWD" --volume "$root:$root" --workdir "$PWD" "$image" \
    bazel --output_user_root="$root/bazel" "$@"
