// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstddef>
#include <cstdlib>
#include <cstring>
#include <stdexcept>
#include <string>
#include <thread>

#include "gtest/gtest.h"
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include "driver/daemon/daemon.h"

namespace driver::daemon {
namespace {
/// @brief stands in for the notification socket systemd provides to Type=notify
/// services: binds an abstract unix datagram socket and points NOTIFY_SOCKET at it (the
/// "@" prefix selects the abstract namespace, per sd_notify(3)). Abstract sockets avoid
/// filesystem paths, which can exceed sun_path's 108-byte limit under Bazel test
/// directories.
class NotifySocket {
public:
    explicit NotifySocket(const std::string &suffix):
        name("synnax-daemon-test-" + std::to_string(getpid()) + "-" + suffix),
        fd(socket(AF_UNIX, SOCK_DGRAM, 0)) {
        sockaddr_un addr{};
        addr.sun_family = AF_UNIX;
        std::memcpy(addr.sun_path + 1, this->name.c_str(), this->name.size());
        const auto len = static_cast<socklen_t>(
            offsetof(sockaddr_un, sun_path) + 1 + this->name.size()
        );
        this->bound = this->fd >= 0 &&
                      bind(this->fd, reinterpret_cast<sockaddr *>(&addr), len) == 0;
        setenv("NOTIFY_SOCKET", ("@" + this->name).c_str(), 1);
    }

    ~NotifySocket() {
        unsetenv("NOTIFY_SOCKET");
        if (this->fd >= 0) close(this->fd);
    }

    bool ok() const { return this->bound; }

    /// @brief appends any pending datagrams to the received buffer, newline-joined, and
    /// returns everything received so far.
    const std::string &drain() {
        char buf[512];
        ssize_t n;
        while ((n = recv(this->fd, buf, sizeof(buf), MSG_DONTWAIT)) > 0) {
            this->received.append(buf, static_cast<size_t>(n));
            this->received.push_back('\n');
        }
        return this->received;
    }

    /// @brief drains until the received buffer contains needle, failing after 5s.
    bool wait_for(const std::string &needle) {
        const auto deadline = std::chrono::steady_clock::now() +
                              std::chrono::seconds(5);
        while (std::chrono::steady_clock::now() < deadline) {
            if (this->drain().find(needle) != std::string::npos) return true;
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        }
        return false;
    }

private:
    /// @brief abstract socket name, without the leading NUL/"@".
    std::string name;
    int fd;
    bool bound = false;
    /// @brief accumulated datagrams, so waiting for one message never discards another.
    std::string received;
};
}

/// @brief it should run the callback and report READY, watchdog liveness, and STOPPING
/// to the service manager.
TEST(Daemon, testRunNotifiesServiceManager) {
    NotifySocket sock("notify");
    ASSERT_TRUE(sock.ok());
    Config config;
    config.watchdog_interval = 1;
    // The watchdog thread stops pinging once the callback returns, so observe a ping
    // before returning: an instantly-returning callback can win the race against the
    // watchdog thread's first iteration, leaving no WATCHDOG=1 on the socket.
    config.callback = [&] { EXPECT_TRUE(sock.wait_for("WATCHDOG=1")); };
    run(config);
    const auto &msgs = sock.drain();
    EXPECT_NE(msgs.find("READY=1"), std::string::npos);
    EXPECT_NE(msgs.find("STOPPING=1"), std::string::npos);
}

/// @brief it should report an error status when the callback throws, and still return
/// instead of propagating.
TEST(Daemon, testRunReportsCallbackError) {
    NotifySocket sock("error");
    ASSERT_TRUE(sock.ok());
    Config config;
    config.watchdog_interval = 1;
    config.callback = [] { throw std::runtime_error("boom"); };
    run(config);
    const auto &msgs = sock.drain();
    EXPECT_NE(msgs.find("STATUS=Error: boom"), std::string::npos);
    EXPECT_NE(msgs.find("STOPPING=1"), std::string::npos);
}
}
