# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# The Linux and NI Linux Real-Time Drivers build here. Ubuntu 22.04 ships glibc 2.35, the
# oldest glibc the Driver supports (NI Linux Real-Time 23.8 and later). GCC 13 comes from
# the toolchain PPA, because the Driver needs C++23.
FROM ubuntu:22.04@sha256:9abcaffc92a5279fff3a6a41635c0800ea33e528425fb389658806476e7defde

ARG BAZELISK_VERSION=1.30.0
ARG BAZELISK_SHA256=a248ae2b2b62b5baebb748f6fba7203bff1a27e5d01b55399a8406744cb539e8

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates gnupg \
        software-properties-common \
    && add-apt-repository -y ppa:ubuntu-toolchain-r/test \
    && apt-get update \
    && apt-get install -y --no-install-recommends autoconf automake curl gcc-13 g++-13 \
        git libsystemd-dev libtool make python3 unzip xz-utils \
    && rm -rf /var/lib/apt/lists/* \
    && update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-13 100 \
        --slave /usr/bin/g++ g++ /usr/bin/g++-13 \
    && curl -fsSL -o /usr/local/bin/bazel \
        "https://github.com/bazelbuild/bazelisk/releases/download/v${BAZELISK_VERSION}/bazelisk-linux-amd64" \
    && echo "${BAZELISK_SHA256}  /usr/local/bin/bazel" | sha256sum -c - \
    && chmod +x /usr/local/bin/bazel
