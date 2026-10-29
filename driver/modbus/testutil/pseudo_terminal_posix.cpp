// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <functional>
#include <string>

#include <sys/select.h>
#include <termios.h>
#include <unistd.h>
#ifdef __APPLE__
#include <util.h>
#else
#include <pty.h>
#endif

#include "x/cpp/errors/errors.h"

#include "driver/modbus/testutil/pseudo_terminal.h"

namespace driver::modbus::testutil {
namespace {
bool readable(const int fd, const int timeout_us) {
    fd_set set;
    FD_ZERO(&set);
    FD_SET(fd, &set);
    timeval timeout{.tv_sec = 0, .tv_usec = timeout_us};
    return select(fd + 1, &set, nullptr, nullptr, &timeout) > 0;
}
}

x::errors::Error PseudoTerminal::open() {
    if (openpty(&this->controller_, &this->terminal_, nullptr, nullptr, nullptr) == -1)
        return x::errors::Error(
            "Failed to open a pseudo-terminal: " + std::string(strerror(errno))
        );
    termios tios{};
    tcgetattr(this->terminal_, &tios);
    cfmakeraw(&tios);
    tcsetattr(this->terminal_, TCSANOW, &tios);
    this->path_ = ttyname(this->terminal_);
    return x::errors::NIL;
}

void PseudoTerminal::close() {
    for (int *fd: {&this->controller_, &this->terminal_}) {
        if (*fd == -1) continue;
        ::close(*fd);
        *fd = -1;
    }
}

std::size_t
PseudoTerminal::read_frame(std::uint8_t *frame, const std::size_t size) const {
    if (!readable(this->controller_, 100000)) return 0;
    std::size_t length = 0;
    do {
        const auto n = read(this->controller_, frame + length, size - length);
        if (n <= 0) break;
        length += static_cast<std::size_t>(n);
    } while (length < size && readable(this->controller_, 5000));
    return length;
}

void PseudoTerminal::write_damaged(const std::function<void(int)> &write) const {
    int fds[2];
    if (pipe(fds) == -1) return;
    write(fds[1]);
    ::close(fds[1]);
    std::uint8_t frame[256];
    const auto n = read(fds[0], frame, sizeof(frame));
    ::close(fds[0]);
    if (n <= 0) return;
    frame[n - 1] ^= 0xFF;
    ::write(this->controller_, frame, static_cast<std::size_t>(n));
}
}
