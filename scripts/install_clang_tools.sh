#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -e

LLVM_VERSION=23

wget -O llvm.sh https://apt.llvm.org/llvm.sh
sudo bash llvm.sh "$LLVM_VERSION"
# clang-tools holds clang-apply-replacements, which scripts/clang_tidy_fix.sh runs.
sudo apt-get install -y "clang-format-${LLVM_VERSION}" "clang-tidy-${LLVM_VERSION}" \
    "clang-tools-${LLVM_VERSION}"
for tool in clang-format clang-tidy clang-apply-replacements; do
    sudo update-alternatives --install \
        "/usr/bin/${tool}" "$tool" "/usr/bin/${tool}-${LLVM_VERSION}" 100
    sudo update-alternatives --set "$tool" "/usr/bin/${tool}-${LLVM_VERSION}"
done
