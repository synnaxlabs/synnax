// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <system_error>

// clang-format off
// mstcpip.h must be included after winsock2.h.
#include <winsock2.h>
#include <mstcpip.h>
// clang-format on

#include "driver/tcp/keepalive.h"

namespace driver::tcp::keepalive {
std::error_code enable(const asio::ip::tcp::socket::native_handle_type socket) {
    tcp_keepalive values{
        .onoff = 1,
        .keepalivetime = static_cast<ULONG>(IDLE.milliseconds()),
        .keepaliveinterval = static_cast<ULONG>(INTERVAL.milliseconds()),
    };
    DWORD returned = 0;
    if (::WSAIoctl(
            socket,
            SIO_KEEPALIVE_VALS,
            &values,
            sizeof(values),
            nullptr,
            0,
            &returned,
            nullptr,
            nullptr
        ) == SOCKET_ERROR)
        return {::WSAGetLastError(), std::system_category()};
    return {};
}
}
