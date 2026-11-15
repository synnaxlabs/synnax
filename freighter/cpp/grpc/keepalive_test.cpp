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
using namespace std::chrono_literals;

/// @brief relays one TCP connection to an upstream port. After drop(), it discards all
/// bytes but keeps both sockets open, as a lost network does.
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

/// @brief serves the test services behind a Blackhole, with a pool that pings soon.
class KeepaliveTest : public ::testing::Test {
protected:
    std::thread server;
    std::unique_ptr<Blackhole> hole;
    std::shared_ptr<Pool> pool;
    std::string target;

    void serve(const std::uint16_t port, const Keepalive keepalive) {
        this->server = std::thread(
            testutil::server,
            "localhost:" + std::to_string(port)
        );
        testutil::wait_for_servers();
        this->hole = std::make_unique<Blackhole>(port);
        this->pool = std::make_shared<Pool>(false, keepalive);
        this->target = "127.0.0.1:" + std::to_string(this->hole->port);
    }

    void TearDown() override {
        this->hole.reset();
        if (!this->server.joinable()) return;
        testutil::stop_servers();
        this->server.join();
    }
};

/// @brief it should fail a stream with UNREACHABLE when its server stops answering
/// pings.
TEST_F(KeepaliveTest, FailsStreamOnLostConnection) {
    this->serve(8087, {.time = 500ms, .timeout = 500ms});
    StreamClient<test::Message, test::Message, test::StreamMessageService> client(
        this->pool,
        this->target
    );
    auto stream = ASSERT_NIL_P(client.stream(""));
    test::Message msg;
    msg.set_payload("ping");
    ASSERT_NIL(stream->send(msg));
    ASSERT_NIL_P(stream->receive());
    // Lets the server answer the ping that the reply started. Any ping that finds the
    // drop fails the stream, so the sleep picks that ping, not the result.
    std::this_thread::sleep_for(200ms);
    this->hole->drop();
    auto received = std::async(std::launch::async, [&stream] {
        return stream->receive();
    });
    const auto status = received.wait_for(10s);
    // Closing the relay ends a receive that keepalive did not end.
    this->hole.reset();
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS_P(received.get(), UNREACHABLE);
}

/// @brief it should fail a stream that sent nothing since it opened with UNREACHABLE
/// when its server stops answering pings.
TEST_F(KeepaliveTest, FailsReceiveOnlyStream) {
    this->serve(8088, {.time = 200ms, .timeout = 500ms});
    StreamClient<test::Message, test::Message, test::StreamMessageService> client(
        this->pool,
        this->target
    );
    auto stream = ASSERT_NIL_P(client.stream(""));
    // Lets keepalive pings go out and come back before the drop. The sleep picks the
    // ping that finds the drop, not the result.
    std::this_thread::sleep_for(1500ms);
    this->hole->drop();
    auto received = std::async(std::launch::async, [&stream] {
        return stream->receive();
    });
    const auto status = received.wait_for(10s);
    this->hole.reset();
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS_P(received.get(), UNREACHABLE);
}

/// @brief it should fail a unary call with UNREACHABLE when its server stops answering
/// pings.
TEST_F(KeepaliveTest, FailsUnaryOnLostConnection) {
    this->serve(8089, {.time = 200ms, .timeout = 500ms});
    UnaryClient<test::Message, test::Message, test::UnaryMessageService> client(
        this->pool,
        this->target
    );
    test::Message msg;
    msg.set_payload("ping");
    ASSERT_NIL_P(client.send("", msg));
    this->hole->drop();
    auto sent = std::async(std::launch::async, [&client, &msg] {
        return client.send("", msg);
    });
    const auto status = sent.wait_for(10s);
    this->hole.reset();
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS_P(sent.get(), UNREACHABLE);
}
}
