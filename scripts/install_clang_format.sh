#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

LLVM_VERSION=22

wget -O llvm.sh https://apt.llvm.org/llvm.sh
sudo bash llvm.sh "$LLVM_VERSION"
sudo apt-get install -y "clang-format-${LLVM_VERSION}"
sudo update-alternatives --install /usr/bin/clang-format clang-format "/usr/bin/clang-format-${LLVM_VERSION}" 100
sudo update-alternatives --set clang-format "/usr/bin/clang-format-${LLVM_VERSION}"
