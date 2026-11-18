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

#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <netinet/ip.h>
#include <netinet/tcp.h>
#include <poll.h>
#include <sys/socket.h>
#include <unistd.h>

#include "driver/modbus/device/device.h"

namespace driver::modbus::device {
namespace {
using Clock = std::chrono::steady_clock;

/// @returns 0 once the connect of a non-blocking socket succeeds, or the error number:
/// EINTR if interrupted was set.
int await_connect(
    const int fd,
    const Clock::time_point deadline,
    const std::atomic<bool> &interrupted
) {
    pollfd pfd{.fd = fd, .events = POLLOUT, .revents = 0};
    const auto max_step = std::chrono::duration_cast<std::chrono::milliseconds>(
        WAIT_STEP.chrono()
    );
    while (!interrupted) {
        const auto left = std::chrono::ceil<std::chrono::milliseconds>(
            deadline - Clock::now()
        );
        if (left <= std::chrono::milliseconds::zero()) return ETIMEDOUT;
        const int rc = poll(
            &pfd,
            1,
            static_cast<int>(std::min(left, max_step).count())
        );
        if (rc == -1 && errno != EINTR) return errno;
        if (rc <= 0) continue;
        int err = 0;
        socklen_t len = sizeof(err);
        if (getsockopt(fd, SOL_SOCKET, SO_ERROR, &err, &len) == -1) return errno;
        return err == 0 ? 0 : ECONNREFUSED;
    }
    return EINTR;
}
}

int connect_tcp(
    modbus_t *ctx,
    const ConnectionConfig &cfg,
    const std::atomic<bool> &interrupted
) {
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
        const int fd = socket(addr->ai_family, addr->ai_socktype, addr->ai_protocol);
        if (fd == -1) {
            errnum = errno;
            continue;
        }
        fcntl(fd, F_SETFD, FD_CLOEXEC);
        fcntl(fd, F_SETFL, fcntl(fd, F_GETFL) | O_NONBLOCK);
        if (addr->ai_family == AF_INET) {
            const int nodelay = 1;
            setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &nodelay, sizeof(nodelay));
            const int tos = IPTOS_LOWDELAY;
            setsockopt(fd, IPPROTO_IP, IP_TOS, &tos, sizeof(tos));
        }
        const auto deadline = Clock::now() + cfg.timeout.chrono();
        if (::connect(fd, addr->ai_addr, addr->ai_addrlen) == 0)
            errnum = 0;
        else if (errno == EINPROGRESS)
            errnum = await_connect(fd, deadline, interrupted);
        else
            errnum = errno;
        if (errnum == 0) {
            modbus_set_socket(ctx, fd);
            break;
        }
        close(fd);
    }
    freeaddrinfo(addrs);
    return errnum;
}
}
