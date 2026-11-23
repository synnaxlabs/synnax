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

// clang-format off
// mstcpip.h must be included after winsock2.h.
#include <winsock2.h>
#include <mstcpip.h>
// clang-format on

#include "driver/udp/icmp.h"

namespace driver::udp::icmp {
std::error_code ignore(const asio::ip::udp::socket::native_handle_type socket) {
    for (const DWORD code: std::array<DWORD, 2>{SIO_UDP_CONNRESET, SIO_UDP_NETRESET}) {
        BOOL enabled = FALSE;
        DWORD returned = 0;
        if (::WSAIoctl(
                socket,
                code,
                &enabled,
                sizeof(enabled),
                nullptr,
                0,
                &returned,
                nullptr,
                nullptr
            ) == SOCKET_ERROR)
            return {::WSAGetLastError(), std::system_category()};
    }
    return {};
}
}
