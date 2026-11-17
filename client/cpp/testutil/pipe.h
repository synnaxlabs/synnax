// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#ifndef _WIN32
#include <array>
#include <atomic>
#include <csignal>
#include <mutex>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include <netinet/in.h>
#include <poll.h>
#include <sys/socket.h>
#include <unistd.h>

/// @brief forwards local TCP connections to the Core. cut drops every open
/// connection, as a short Core outage does. Later connections still go through.
class CorePipe {
    int listener = -1;
    sockaddr_in core{};
    std::atomic<bool> open{true};
    std::atomic<bool> paused{false};
    std::mutex mu;
    std::vector<int> sockets;
    std::vector<std::thread> pumps;
    std::thread acceptor;

    static sockaddr_in loopback(const uint16_t port) {
        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_port = htons(port);
        addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
        return addr;
    }

    static void pump(const int from, const int to) {
        std::array<char, 16384> buf{};
        ssize_t n = 0;
        while ((n = recv(from, buf.data(), buf.size(), 0)) > 0)
            if (send(to, buf.data(), n, 0) != n) break;
        shutdown(to, SHUT_RDWR);
    }

    void accept_loop() {
        while (this->open) {
            pollfd fd{.fd = this->listener, .events = POLLIN, .revents = 0};
            if (poll(&fd, 1, 50) <= 0) continue;
            const int client = accept(this->listener, nullptr, nullptr);
            if (client < 0) continue;
            if (this->paused) {
                close(client);
                continue;
            }
            const int upstream = socket(AF_INET, SOCK_STREAM, 0);
            const auto *addr = reinterpret_cast<sockaddr *>(&this->core);
            if (connect(upstream, addr, sizeof(this->core)) != 0) {
                close(client);
                close(upstream);
                continue;
            }
            std::lock_guard lock{this->mu};
            this->sockets.insert(this->sockets.end(), {client, upstream});
            this->pumps.emplace_back(pump, client, upstream);
            this->pumps.emplace_back(pump, upstream, client);
        }
    }

public:
    uint16_t port = 0;

    explicit CorePipe(const uint16_t core_port):
        listener(socket(AF_INET, SOCK_STREAM, 0)), core(loopback(core_port)) {
        // A pump writes to a socket that the other side closed.
        signal(SIGPIPE, SIG_IGN);
        auto addr = loopback(0);
        socklen_t len = sizeof(addr);
        auto *sock_addr = reinterpret_cast<sockaddr *>(&addr);
        EXPECT_EQ(bind(this->listener, sock_addr, len), 0);
        EXPECT_EQ(listen(this->listener, 16), 0);
        EXPECT_EQ(getsockname(this->listener, sock_addr, &len), 0);
        this->port = ntohs(addr.sin_port);
        this->acceptor = std::thread([this] { this->accept_loop(); });
    }

    CorePipe(const CorePipe &) = delete;
    CorePipe &operator=(const CorePipe &) = delete;

    void cut() {
        std::lock_guard lock{this->mu};
        for (const int s: this->sockets)
            shutdown(s, SHUT_RDWR);
    }

    /// @brief cuts every connection and refuses new ones until resume.
    void pause() {
        this->paused = true;
        this->cut();
    }

    void resume() { this->paused = false; }

    ~CorePipe() {
        this->open = false;
        this->acceptor.join();
        this->cut();
        for (auto &t: this->pumps)
            t.join();
        for (const int s: this->sockets)
            close(s);
        close(this->listener);
    }
};
#endif
