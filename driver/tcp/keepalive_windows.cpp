// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <system_error>
#include <tuple>

#include <winsock2.h>
#include <ws2tcpip.h>

#include "driver/tcp/keepalive.h"

namespace driver::tcp::keepalive {
std::error_code enable(const SOCKET socket) {
    const std::array<std::tuple<int, int, DWORD>, 4> options{{
        {SOL_SOCKET, SO_KEEPALIVE, TRUE},
        {IPPROTO_TCP, TCP_KEEPIDLE, static_cast<DWORD>(IDLE.seconds())},
        {IPPROTO_TCP, TCP_KEEPINTVL, static_cast<DWORD>(INTERVAL.seconds())},
        {IPPROTO_TCP, TCP_KEEPCNT, static_cast<DWORD>(COUNT)},
    }};
    for (const auto &[level, option, value]: options)
        if (::setsockopt(
                socket,
                level,
                option,
                reinterpret_cast<const char *>(&value),
                sizeof(value)
            ) != 0)
            return {::WSAGetLastError(), std::system_category()};
    return {};
}
}
