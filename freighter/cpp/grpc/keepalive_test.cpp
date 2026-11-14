// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <future>
#include <memory>
#include <stdexcept>
#include <string>
#include <thread>

#include "gtest/gtest.h"
#include <arpa/inet.h>
#include <netinet/in.h>
#include <poll.h>
#include <sys/socket.h>
#include <unistd.h>

#include "freighter/cpp/freighter.h"
#include "freighter/cpp/grpc/grpc.h"
#include "freighter/cpp/grpc/testutil/freighter/cpp/grpc/testutil/service.grpc.pb.h"
#include "freighter/cpp/grpc/testutil/server.h"
#include "x/cpp/testutil/testutil.h"

namespace freighter::grpc {
/// @brief relays one TCP connection to an upstream port until drop() is called, then
/// discards all bytes in both directions while it keeps both sockets open, as a lost
/// network does.
class Blackhole {
    int listener = -1;
    int client = -1;
    int upstream = -1;
    std::atomic<bool> dropping = false;
    std::atomic<bool> closing = false;
    std::thread relay;

    void run(const std::uint16_t upstream_port) {
        pollfd listening{this->listener, POLLIN, 0};
        while (poll(&listening, 1, 50) <= 0)
            if (this->closing) return;
        this->client = accept(this->listener, nullptr, nullptr);
        this->upstream = socket(AF_INET, SOCK_STREAM, 0);
        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_port = htons(upstream_port);
        addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
        if (connect(
                this->upstream,
                reinterpret_cast<sockaddr *>(&addr),
                sizeof(addr)
            ) != 0)
            return;
        pollfd fds[2] = {{this->client, POLLIN, 0}, {this->upstream, POLLIN, 0}};
        char buf[16384];
        while (!this->closing) {
            if (poll(fds, 2, 50) <= 0) continue;
            for (int i = 0; i < 2; i++) {
                if ((fds[i].revents & POLLIN) == 0) continue;
                const auto n = recv(fds[i].fd, buf, sizeof(buf), 0);
                if (n <= 0) return;
                if (!this->dropping) send(fds[1 - i].fd, buf, n, 0);
            }
        }
    }

public:
    std::uint16_t port = 0;

    explicit Blackhole(const std::uint16_t upstream_port):
        listener(socket(AF_INET, SOCK_STREAM, 0)) {
        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
        socklen_t len = sizeof(addr);
        if (bind(this->listener, reinterpret_cast<sockaddr *>(&addr), len) != 0 ||
            getsockname(this->listener, reinterpret_cast<sockaddr *>(&addr), &len) !=
                0 ||
            listen(this->listener, 1) != 0)
            throw std::runtime_error("failed to open the relay listener");
        this->port = ntohs(addr.sin_port);
        this->relay = std::thread([this, upstream_port] { this->run(upstream_port); });
    }

    void drop() { this->dropping = true; }

    ~Blackhole() {
        this->closing = true;
        this->relay.join();
        for (const int fd: {this->listener, this->client, this->upstream})
            if (fd >= 0) close(fd);
    }

    Blackhole(const Blackhole &) = delete;
    Blackhole &operator=(const Blackhole &) = delete;
};

/// @brief it should fail a stream with UNREACHABLE when its server stops answering
/// pings.
TEST(testGRPC, testKeepaliveFailsStreamOnLostConnection) {
    std::thread s(testutil::server, "localhost:8087");
    testutil::wait_for_servers();
    auto hole = std::make_unique<Blackhole>(8087);
    auto pool = std::make_shared<Pool>(
        false,
        Keepalive{
            .time = std::chrono::milliseconds(500),
            .timeout = std::chrono::milliseconds(500),
        }
    );
    auto
        client = StreamClient<test::Message, test::Message, test::StreamMessageService>(
            pool,
            "127.0.0.1:" + std::to_string(hole->port)
        );
    auto stream = ASSERT_NIL_P(client.stream(""));
    test::Message msg;
    msg.set_payload("ping");
    ASSERT_NIL(stream->send(msg));
    ASSERT_NIL_P(stream->receive());
    // Lets the server answer the ping that the reply started, so only a keepalive
    // ping can find the lost connection.
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    hole->drop();
    auto received = std::async(std::launch::async, [&stream] {
        return stream->receive();
    });
    const auto status = received.wait_for(std::chrono::seconds(10));
    // Closing the relay ends a receive that keepalive did not end.
    hole.reset();
    const auto err = received.get().second;
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS(err, UNREACHABLE);
    testutil::stop_servers();
    s.join();
}

/// @brief it should fail a stream that sent nothing since it opened with UNREACHABLE
/// when its server stops answering pings.
TEST(testGRPC, testKeepaliveFailsReceiveOnlyStream) {
    std::thread s(testutil::server, "localhost:8088");
    testutil::wait_for_servers();
    auto hole = std::make_unique<Blackhole>(8088);
    auto pool = std::make_shared<Pool>(
        false,
        Keepalive{
            .time = std::chrono::milliseconds(200),
            .timeout = std::chrono::milliseconds(500),
        }
    );
    auto
        client = StreamClient<test::Message, test::Message, test::StreamMessageService>(
            pool,
            "127.0.0.1:" + std::to_string(hole->port)
        );
    auto stream = ASSERT_NIL_P(client.stream(""));
    // Lets several keepalive pings go out and come back before the drop.
    std::this_thread::sleep_for(std::chrono::milliseconds(1500));
    hole->drop();
    auto received = std::async(std::launch::async, [&stream] {
        return stream->receive();
    });
    const auto status = received.wait_for(std::chrono::seconds(10));
    hole.reset();
    const auto err = received.get().second;
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS(err, UNREACHABLE);
    testutil::stop_servers();
    s.join();
}
}
