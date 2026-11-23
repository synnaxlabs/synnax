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
#include <memory>
#include <mutex>
#include <set>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#ifdef _WIN32
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
/// connection, as a short Core outage does. Later connections still go through. reset
/// resets every open gRPC stream, as a Core or proxy fault does. The connections stay
/// up.
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
    static constexpr size_t FRAME_HEADER_SIZE = 9;
    static constexpr uint8_t DATA_FRAME = 0x0;
    static constexpr uint8_t HEADERS_FRAME = 0x1;
    static constexpr uint8_t RST_STREAM_FRAME = 0x3;
    static constexpr uint8_t END_STREAM_FLAG = 0x1;
    static constexpr uint8_t INTERNAL_ERROR = 0x2;

    static bool send_all(const Socket to, const uint8_t *data, size_t size) {
        while (size > 0) {
            const auto n = send(
                to,
                reinterpret_cast<const char *>(data),
                static_cast<int>(size),
                0
            );
            if (n <= 0) return false;
            data += n;
            size -= n;
        }
        return true;
    }

    /// @brief the HTTP/2 frames from the Core to one client, and the streams that the
    /// Core has answered and not yet ended.
    struct Connection {
        Socket client;
        std::mutex mu;
        std::set<uint32_t> open;

        explicit Connection(const Socket client): client(client) {}

        /// @brief sends one whole frame to the client.
        bool forward(const uint8_t *frame, const size_t size) {
            std::lock_guard lock{this->mu};
            const uint8_t type = frame[3];
            const uint8_t flags = frame[4];
            const uint32_t id = (frame[5] & 0x7Fu) << 24 | frame[6] << 16 |
                                frame[7] << 8 | frame[8];
            if (id != 0) {
                if (type == HEADERS_FRAME) this->open.insert(id);
                if ((type == HEADERS_FRAME || type == DATA_FRAME) &&
                    (flags & END_STREAM_FLAG) != 0)
                    this->open.erase(id);
                if (type == RST_STREAM_FRAME) this->open.erase(id);
            }
            return send_all(this->client, frame, size);
        }

        /// @brief sends the client an RST_STREAM with INTERNAL_ERROR for each open
        /// stream.
        void reset() {
            std::lock_guard lock{this->mu};
            for (const auto id: this->open) {
                const std::array<uint8_t, 13> frame{
                    0,
                    0,
                    4,
                    RST_STREAM_FRAME,
                    0,
                    static_cast<uint8_t>(id >> 24),
                    static_cast<uint8_t>(id >> 16),
                    static_cast<uint8_t>(id >> 8),
                    static_cast<uint8_t>(id),
                    0,
                    0,
                    0,
                    INTERNAL_ERROR,
                };
                send_all(this->client, frame.data(), frame.size());
            }
            this->open.clear();
        }
    };

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
    std::vector<std::shared_ptr<Connection>> connections;
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

    /// @brief forwards the frames from the Core to the client one whole frame at a
    /// time, so a reset never lands inside a frame.
    static void pump_frames(const Socket from, const std::shared_ptr<Connection> conn) {
        std::array<uint8_t, BUFFER_SIZE> buf{};
        std::vector<uint8_t> pending;
        while (true) {
            const auto
                n = recv(from, reinterpret_cast<char *>(buf.data()), BUFFER_SIZE, 0);
            if (n <= 0) break;
            pending.insert(pending.end(), buf.begin(), buf.begin() + n);
            size_t off = 0;
            bool sent = true;
            while (sent && pending.size() - off >= FRAME_HEADER_SIZE) {
                const auto *frame = pending.data() + off;
                const size_t size = FRAME_HEADER_SIZE +
                                    (frame[0] << 16 | frame[1] << 8 | frame[2]);
                if (pending.size() - off < size) break;
                sent = conn->forward(frame, size);
                off += size;
            }
            if (!sent) break;
            pending.erase(pending.begin(), pending.begin() + off);
        }
        shutdown(conn->client, SHUT_BOTH);
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
            auto conn = std::make_shared<Connection>(client);
            this->connections.push_back(conn);
            this->pumps.emplace_back(pump, client, upstream);
            this->pumps.emplace_back(pump_frames, upstream, conn);
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

    /// @brief resets every gRPC stream that the Core has answered and not yet ended.
    void reset() {
        std::lock_guard lock{this->mu};
        for (const auto &conn: this->connections)
            conn->reset();
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
