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

/// @brief a pseudo-terminal pair that plays a device. The port under test opens the
/// terminal side, and the test plays the device on the controller side.
class Pty {
    int controller = -1;
    std::string terminal;

public:
    Pty() = default;
    Pty(const Pty &) = delete;
    Pty &operator=(const Pty &) = delete;
    ~Pty() { this->close(); }

    /// @brief plugs in a new device on a new pseudo-terminal.
    void open() {
        this->controller = ::posix_openpt(O_RDWR | O_NOCTTY);
        ASSERT_GE(this->controller, 0);
        ASSERT_EQ(::grantpt(this->controller), 0);
        ASSERT_EQ(::unlockpt(this->controller), 0);
        this->terminal = ::ptsname(this->controller);
    }

    /// @returns the path of the terminal side.
    [[nodiscard]] const std::string &path() const { return this->terminal; }

    void write(const std::string &data) const {
        ASSERT_EQ(
            ::write(this->controller, data.data(), data.size()),
            static_cast<ssize_t>(data.size())
        );
    }

    /// @brief reads until n bytes arrive or one second passes.
    [[nodiscard]] std::string read(const std::size_t n) const {
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

    /// @returns the terminal's settings, read through the controller because the port
    /// keeps other programs from opening the terminal.
    [[nodiscard]] ::termios termios() const {
        ::termios tio{};
        EXPECT_EQ(::tcgetattr(this->controller, &tio), 0);
        return tio;
    }

    /// @brief unplugs the device. The terminal side hangs up.
    void close() {
        if (this->controller >= 0) ::close(this->controller);
        this->controller = -1;
    }
};

/// @brief a fixture with a pseudo-terminal that plays the device.
class PtyTest : public ::testing::Test {
    Pty pty;

protected:
    void SetUp() override { this->open_peer(); }

    void TearDown() override { this->close_peer(); }

    /// @brief plugs in a new device on a new pseudo-terminal.
    void open_peer() { this->pty.open(); }

    [[nodiscard]] synnax::serial::Properties props() const {
        synnax::serial::Properties p;
        p.port = this->pty.path();
        return p;
    }

    void peer_write(const std::string &data) const { this->pty.write(data); }

    /// @brief reads until n bytes arrive or one second passes.
    [[nodiscard]] std::string peer_read(const std::size_t n) const {
        return this->pty.read(n);
    }

    [[nodiscard]] ::termios termios() const { return this->pty.termios(); }

    void close_peer() { this->pty.close(); }
};
}
