#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the pinned clang-tidy for the --config=clang-tidy aspect. The aspect passes the
# file to lint right before the -- that starts the compiler arguments. Bazel writes the
# path of the binary in place of @CLANG_TIDY@.

# A generated file follows its generator, so lint the generator instead.
previous=
for arg in "$@"; do
    if [ "$arg" = -- ]; then
        if head -n 12 "$previous" | grep -q "Code generated"; then exit 0; fi
        break
    fi
    previous=$arg
done

# Bazel passes third-party and generated include paths as user paths, so every check
# runs over their headers before clang-tidy drops the findings. As system paths, the
# checks skip them, which halves the run time.
third_party='^(external|vendor|bazel-out)/'
os=$(uname)
gcc_install_dir=
args=()
while [ "$#" -gt 0 ]; do
    case $1 in
        # On Linux, the aspect passes the GCC built-in include paths with the C paths
        # first, which breaks the #include_next in libstdc++. clang-tidy finds the same
        # paths in the correct order from the GCC install.
        -isystem)
            if [ "$os" = Linux ] && [[ $2 == /* ]]; then
                if [[ $2 =~ ^(/usr/lib/gcc/[^/]+/[^/]+)/include$ ]]; then
                    gcc_install_dir=${BASH_REMATCH[1]}
                fi
            else
                args+=("$1" "$2")
            fi
            shift 2
            ;;
        -iquote | -I)
            if [[ $2 =~ $third_party ]]; then
                args+=(-isystem "$2")
            else
                args+=("$1" "$2")
            fi
            shift 2
            ;;
        -I*)
            if [[ ${1#-I} =~ $third_party ]]; then
                args+=(-isystem "${1#-I}")
            else
                args+=("$1")
            fi
            shift
            ;;
        *)
            args+=("$1")
            shift
            ;;
    esac
done
if [ -n "$gcc_install_dir" ]; then args+=("--gcc-install-dir=$gcc_install_dir"); fi
set -- "${args[@]}"

# The macOS toolchain passes Xcode path placeholders that only its own compiler wrapper
# expands.
if [ "$os" = Darwin ]; then
    sdk_root=$(xcrun --show-sdk-path)
    developer_dir=$(xcode-select --print-path)
    args=()
    for arg in "$@"; do
        arg=${arg//__BAZEL_XCODE_SDKROOT__/$sdk_root}
        args+=("${arg//__BAZEL_XCODE_DEVELOPER_DIR__/$developer_dir}")
    done
    set -- "${args[@]}"
fi

exec @CLANG_TIDY@ "$@"
