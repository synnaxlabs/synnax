#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the clang-tidy on PATH for the --config=clang-tidy aspect. The aspect passes the
# file to lint right before the -- that starts the compiler arguments.

# A generated file follows its generator, so lint the generator instead.
previous=
for arg in "$@"; do
    if [ "$arg" = -- ]; then
        if head -n 12 "$previous" | grep -q "Code generated"; then exit 0; fi
        break
    fi
    previous=$arg
done

# The macOS toolchain passes Xcode path placeholders that only its own compiler wrapper
# expands.
if [ "$(uname)" = Darwin ]; then
    sdk_root=$(xcrun --show-sdk-path)
    developer_dir=$(xcode-select --print-path)
    args=()
    for arg in "$@"; do
        arg=${arg//__BAZEL_XCODE_SDKROOT__/$sdk_root}
        args+=("${arg//__BAZEL_XCODE_DEVELOPER_DIR__/$developer_dir}")
    done
    set -- "${args[@]}"
fi

exec clang-tidy "$@"
