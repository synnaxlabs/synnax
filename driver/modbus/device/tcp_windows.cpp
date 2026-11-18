// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <string>

#include <winsock2.h>
#include <ws2tcpip.h>

#include "driver/modbus/device/device.h"

namespace driver::modbus::device {
namespace {
using Clock = std::chrono::steady_clock;

/// @returns 0 once the connect of a non-blocking socket succeeds, or the error number:
/// EINTR if interrupted was set.
int await_connect(
    const SOCKET s,
    const Clock::time_point deadline,
    const std::atomic<bool> &interrupted
) {
    const auto max_step = std::chrono::duration_cast<std::chrono::microseconds>(
        WAIT_STEP.chrono()
    );
    while (!interrupted) {
        const auto left = std::chrono::ceil<std::chrono::microseconds>(
            deadline - Clock::now()
        );
        if (left <= std::chrono::microseconds::zero()) return ETIMEDOUT;
        fd_set connected;
        fd_set failed;
        FD_ZERO(&connected);
        FD_ZERO(&failed);
        FD_SET(s, &connected);
        FD_SET(s, &failed);
        timeval step{0, static_cast<long>(std::min(left, max_step).count())};
        // Windows reports a failed connect in the exception set, not the write set.
        const int rc = select(0, nullptr, &connected, &failed, &step);
        if (rc == SOCKET_ERROR) return ECONNREFUSED;
        if (rc == 0) continue;
        return FD_ISSET(s, &connected) ? 0 : ECONNREFUSED;
    }
    return EINTR;
}
}

int connect_tcp(
    modbus_t *ctx,
    const ConnectionConfig &cfg,
    const std::atomic<bool> &interrupted
) {
    WSADATA wsa;
    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) return EIO;
    addrinfo hints{};
    hints.ai_flags = AI_ADDRCONFIG;
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    addrinfo *addrs = nullptr;
    const auto port = std::to_string(cfg.port);
    if (getaddrinfo(cfg.host.c_str(), port.c_str(), &hints, &addrs) != 0)
        return ECONNREFUSED;
    int errnum = ECONNREFUSED;
    for (const auto *addr = addrs; addr != nullptr && errnum != EINTR;
         addr = addr->ai_next) {
        const SOCKET s = socket(addr->ai_family, addr->ai_socktype, addr->ai_protocol);
        if (s == INVALID_SOCKET) continue;
        u_long nonblocking = 1;
        ioctlsocket(s, FIONBIO, &nonblocking);
        if (addr->ai_family == AF_INET) {
            const int nodelay = 1;
            setsockopt(
                s,
                IPPROTO_TCP,
                TCP_NODELAY,
                reinterpret_cast<const char *>(&nodelay),
                sizeof(nodelay)
            );
        }
        const auto deadline = Clock::now() + cfg.timeout.chrono();
        if (::connect(s, addr->ai_addr, static_cast<int>(addr->ai_addrlen)) == 0)
            errnum = 0;
        else if (WSAGetLastError() == WSAEWOULDBLOCK)
            errnum = await_connect(s, deadline, interrupted);
        else
            errnum = ECONNREFUSED;
        if (errnum == 0) {
            modbus_set_socket(ctx, static_cast<int>(s));
            break;
        }
        closesocket(s);
    }
    freeaddrinfo(addrs);
    return errnum;
}
}
