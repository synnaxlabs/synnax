// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cerrno>
#include <system_error>
#include <tuple>

#include <netinet/in.h>
#include <netinet/tcp.h>
#include <sys/socket.h>

#include "driver/tcp/keepalive.h"

namespace driver::tcp::keepalive {
std::error_code enable(const int socket) {
    const std::array<std::tuple<int, int, int>, 4> options{{
        {SOL_SOCKET, SO_KEEPALIVE, 1},
        {IPPROTO_TCP, TCP_KEEPIDLE, static_cast<int>(IDLE.seconds())},
        {IPPROTO_TCP, TCP_KEEPINTVL, static_cast<int>(INTERVAL.seconds())},
        {IPPROTO_TCP, TCP_KEEPCNT, COUNT},
    }};
    for (const auto &[level, option, value]: options)
        if (::setsockopt(socket, level, option, &value, sizeof(value)) != 0)
            return {errno, std::generic_category()};
    return {};
}
}
