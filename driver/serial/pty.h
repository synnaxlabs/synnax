// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <cstdlib>
#include <span>
#include <string>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <poll.h>
#include <termios.h>
#include <unistd.h>

#include "client/cpp/serial/types.gen.h"

namespace driver::serial {
inline std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

inline std::string as_string(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}

/// @brief a pseudo-terminal pair. The port under test opens the terminal side, and the
/// test plays the device on the controller side.
class PtyTest : public ::testing::Test {
    int controller = -1;
    std::string terminal;

protected:
    void SetUp() override {
        this->controller = ::posix_openpt(O_RDWR | O_NOCTTY);
        ASSERT_GE(this->controller, 0);
        ASSERT_EQ(::grantpt(this->controller), 0);
        ASSERT_EQ(::unlockpt(this->controller), 0);
        this->terminal = ::ptsname(this->controller);
    }

    void TearDown() override { this->close_peer(); }

    [[nodiscard]] synnax::serial::Properties props() const {
        synnax::serial::Properties p;
        p.port = this->terminal;
        return p;
    }

    void peer_write(const std::string &data) const {
        ASSERT_EQ(
            ::write(this->controller, data.data(), data.size()),
            static_cast<ssize_t>(data.size())
        );
    }

    /// @brief reads until n bytes arrive or one second passes.
    [[nodiscard]] std::string peer_read(const std::size_t n) const {
        std::string out;
        pollfd pfd{.fd = this->controller, .events = POLLIN, .revents = 0};
        while (out.size() < n && ::poll(&pfd, 1, 1000) > 0) {
            char buf[256];
            const auto read = ::read(this->controller, buf, sizeof(buf));
            if (read <= 0) break;
            out.append(buf, static_cast<std::size_t>(read));
        }
        return out;
    }

    /// @returns the terminal's settings, read through a second descriptor.
    [[nodiscard]] ::termios termios() const {
        const int fd = ::open(this->terminal.c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK);
        EXPECT_GE(fd, 0);
        ::termios tio{};
        EXPECT_EQ(::tcgetattr(fd, &tio), 0);
        ::close(fd);
        return tio;
    }

    void close_peer() {
        if (this->controller >= 0) ::close(this->controller);
        this->controller = -1;
    }
};
}
