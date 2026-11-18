// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <atomic>
#include <cstdint>
#include <mutex>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <csignal>

#include <netinet/in.h>
#include <poll.h>
#include <sys/socket.h>
#include <unistd.h>
#endif

/// @brief forwards local TCP connections to the Core. cut drops every open
/// connection, as a short Core outage does. Later connections still go through.
class CorePipe {
#ifdef _WIN32
    using Socket = SOCKET;
    static constexpr Socket INVALID = INVALID_SOCKET;
    static constexpr int SHUT_BOTH = SD_BOTH;
    static void close_socket(const Socket s) { closesocket(s); }
#else
    using Socket = int;
    static constexpr Socket INVALID = -1;
    static constexpr int SHUT_BOTH = SHUT_RDWR;
    static void close_socket(const Socket s) { close(s); }
#endif
    static constexpr int BUFFER_SIZE = 16384;

    /// @brief readies the socket library for the life of the pipe.
    struct Startup {
        Startup() {
#ifdef _WIN32
            WSADATA wsa;
            EXPECT_EQ(WSAStartup(MAKEWORD(2, 2), &wsa), 0);
#else
            // A pump writes to a socket that the other side closed.
            signal(SIGPIPE, SIG_IGN);
#endif
        }

        ~Startup() {
#ifdef _WIN32
            WSACleanup();
#endif
        }

        Startup(const Startup &) = delete;
        Startup &operator=(const Startup &) = delete;
    };

    Startup startup;
    Socket listener;
    sockaddr_in core{};
    std::atomic<bool> open{true};
    std::atomic<bool> paused{false};
    std::mutex mu;
    std::vector<Socket> sockets;
    std::vector<std::thread> pumps;
    std::thread acceptor;

    static sockaddr_in loopback(const uint16_t port) {
        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_port = htons(port);
        addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
        return addr;
    }

    static void pump(const Socket from, const Socket to) {
        std::array<char, BUFFER_SIZE> buf{};
        while (true) {
            const auto n = recv(from, buf.data(), BUFFER_SIZE, 0);
            if (n <= 0) break;
            if (send(to, buf.data(), n, 0) != n) break;
        }
        shutdown(to, SHUT_BOTH);
    }

    /// @brief waits up to 50 ms for a connection on the listener.
    [[nodiscard]] bool readable() const {
        pollfd fd{};
        fd.fd = this->listener;
        fd.events = POLLIN;
#ifdef _WIN32
        return WSAPoll(&fd, 1, 50) > 0;
#else
        return poll(&fd, 1, 50) > 0;
#endif
    }

    void accept_loop() {
        while (this->open) {
            if (!this->readable()) continue;
            const Socket client = accept(this->listener, nullptr, nullptr);
            if (client == INVALID) continue;
            if (this->paused) {
                close_socket(client);
                continue;
            }
            const Socket upstream = socket(AF_INET, SOCK_STREAM, 0);
            const auto *addr = reinterpret_cast<const sockaddr *>(&this->core);
            if (connect(upstream, addr, sizeof(this->core)) != 0) {
                close_socket(client);
                close_socket(upstream);
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
        for (const auto s: this->sockets)
            shutdown(s, SHUT_BOTH);
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
        for (const auto s: this->sockets)
            close_socket(s);
        close_socket(this->listener);
    }
};
