// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstddef>
#include <cstdlib>
#include <cstring>
#include <stdexcept>
#include <string>

#include "gtest/gtest.h"
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include "driver/daemon/daemon.h"

namespace driver::daemon {
namespace {
/// @brief stands in for the notification socket systemd provides to Type=notify
/// services: binds an abstract unix datagram socket and points NOTIFY_SOCKET at it
/// (the "@" prefix selects the abstract namespace, per sd_notify(3)). Abstract
/// sockets avoid filesystem paths, which can exceed sun_path's 108-byte limit under
/// Bazel test directories.
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

    /// @brief returns all datagrams received so far, newline-joined.
    std::string drain() const {
        std::string all;
        char buf[512];
        ssize_t n;
        while ((n = recv(this->fd, buf, sizeof(buf), MSG_DONTWAIT)) > 0) {
            all.append(buf, static_cast<size_t>(n));
            all.push_back('\n');
        }
        return all;
    }

private:
    /// @brief abstract socket name, without the leading NUL/"@".
    std::string name;
    int fd;
    bool bound = false;
};
}

/// @brief it should run the callback and report READY, watchdog liveness, and
/// STOPPING to the service manager.
TEST(Daemon, testRunNotifiesServiceManager) {
    const NotifySocket sock("notify");
    ASSERT_TRUE(sock.ok());
    std::atomic ran{false};
    Config config;
    config.watchdog_interval = 1;
    config.callback = [&](int, char **) { ran = true; };
    run(config, 0, nullptr);
    ASSERT_TRUE(ran.load());
    const auto msgs = sock.drain();
    EXPECT_NE(msgs.find("READY=1"), std::string::npos);
    EXPECT_NE(msgs.find("WATCHDOG=1"), std::string::npos);
    EXPECT_NE(msgs.find("STOPPING=1"), std::string::npos);
}

/// @brief it should report an error status when the callback throws, and still
/// return instead of propagating.
TEST(Daemon, testRunReportsCallbackError) {
    const NotifySocket sock("error");
    ASSERT_TRUE(sock.ok());
    Config config;
    config.watchdog_interval = 1;
    config.callback = [](int, char **) { throw std::runtime_error("boom"); };
    run(config, 0, nullptr);
    const auto msgs = sock.drain();
    EXPECT_NE(msgs.find("STATUS=Error: boom"), std::string::npos);
    EXPECT_NE(msgs.find("STOPPING=1"), std::string::npos);
}
}
