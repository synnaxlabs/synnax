// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <future>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include "freighter/cpp/freighter.h"
#include "freighter/cpp/grpc/grpc.h"
#include "freighter/cpp/grpc/testutil/freighter/cpp/grpc/testutil/service.grpc.pb.h"
#include "freighter/cpp/grpc/testutil/server.h"
#include "x/cpp/testutil/testutil.h"

namespace freighter::grpc {
/// Internal response type uses message.
using RQ = test::Message;
using RS = test::Message;
using UNARY_RPC = test::UnaryMessageService;
using STREAM_RPC = test::StreamMessageService;

const std::string base_target = "localhost:8080";

/// @brief it should set and get payload in a message proto.
TEST(testGRPC, basicProto) {
    auto m = test::Message();
    m.set_payload("Hello");
    ASSERT_EQ(m.payload(), "Hello");
}

/// @brief it should send a unary request and receive a response.
TEST(testGRPC, testBasicUnary) {
    std::thread s(testutil::server, base_target);
    testutil::wait_for_servers();
    const auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    const auto res = ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    testutil::stop_servers();
    s.join();
}

class myMiddleware : public PassthroughMiddleware {
public:
    bool ack = false;

    std::pair<Context, x::errors::Error>
    operator()(Context context, Next &next) override {
        context.set("test", "5");
        auto [outContext, exc] = next(context);
        auto a = outContext.get("test");
        if (a == "dog") { ack = true; }
        return {outContext, exc};
    }
};

/// @brief it should propagate metadata headers through middleware.
TEST(testGRPC, testMiddlewareInjection) {
    std::thread s(testutil::server, base_target);
    testutil::wait_for_servers();
    const auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    const auto mw = std::make_shared<myMiddleware>();
    client.use(mw);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto res = ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    testutil::stop_servers();
    s.join();
}

/// @brief it should return an unreachable error when server is not available.
TEST(testGRPC, testFailedUnary) {
    // Note that the easiest way to cause a failure
    // here is to simply not set up a server, so that
    // we don't get a response.
    const auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    ASSERT_OCCURRED_AS_P(client.send("", mes), UNREACHABLE);
}

/// @brief it should send messages to multiple targets.
TEST(testGRPC, testMultipleTargets) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(testutil::server, target_one);
    std::thread s2(testutil::server, target_two);
    testutil::wait_for_servers(2);
    auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool);
    auto mes_one = test::Message();
    mes_one.set_payload("Sending to Server One");
    auto res_one = ASSERT_NIL_P(client.send(target_one, mes_one));
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Server One");

    auto mes_two = test::Message();
    mes_two.set_payload("Sending to Server Two");
    auto res_two = ASSERT_NIL_P(client.send(target_two, mes_two));
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Server Two");

    testutil::stop_servers();
    s1.join();
    s2.join();
}

/// @brief it should send and receive a message over a stream.
TEST(testGRPC, testBasicStream) {
    std::string target("localhost:8080");
    std::thread s(testutil::server, target);
    testutil::wait_for_servers();

    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto streamer = ASSERT_NIL_P(client.stream(""));
    mes.set_payload("Sending to Streaming Server");
    ASSERT_NIL(streamer->send(mes));
    streamer->close_send();
    auto res = ASSERT_NIL_P(streamer->receive());
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");
    ASSERT_OCCURRED_AS_P(streamer->receive(), ERR_EOF);
    testutil::stop_servers();
    s.join();
}

/// @brief it should send messages using multiple stream objects to different targets.
TEST(testGRPC, testMultipleStreamObjects) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(testutil::server, target_one);
    std::thread s2(testutil::server, target_two);
    testutil::wait_for_servers(2);

    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool);
    auto mes_one = test::Message();
    auto mes_two = test::Message();

    auto streamer_one = ASSERT_NIL_P(client.stream(target_one));
    auto streamer_two = ASSERT_NIL_P(client.stream(target_two));
    mes_one.set_payload("Sending to Streaming Server from Streamer One");
    mes_two.set_payload("Sending to Streaming Server from Streamer Two");
    ASSERT_NIL(streamer_one->send(mes_one));
    streamer_one->close_send();
    ASSERT_NIL(streamer_two->send(mes_two));
    streamer_two->close_send();
    auto res_one = ASSERT_NIL_P(streamer_one->receive());
    auto res_two = ASSERT_NIL_P(streamer_two->receive());
    ASSERT_EQ(
        res_one.payload(),
        "Read request: Sending to Streaming Server from Streamer One"
    );
    ASSERT_EQ(
        res_two.payload(),
        "Read request: Sending to Streaming Server from Streamer Two"
    );
    ASSERT_OCCURRED_AS_P(streamer_one->receive(), ERR_EOF);
    ASSERT_OCCURRED_AS_P(streamer_two->receive(), ERR_EOF);

    testutil::stop_servers();
    s1.join();
    s2.join();
}

/// @brief it should send and receive multiple messages over a single stream.
TEST(testGRPC, testSendMultipleMessages) {
    std::string target("localhost:8080");
    std::thread s(testutil::server, target);
    testutil::wait_for_servers();

    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();
    auto mes_two = test::Message();

    auto streamer = ASSERT_NIL_P(client.stream(""));
    mes.set_payload("Sending to Streaming Server");
    streamer->send(mes);
    auto res = ASSERT_NIL_P(streamer->receive());
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");

    mes_two.set_payload("Sending New Message");
    streamer->send(mes_two);
    streamer->close_send();
    auto res_two = ASSERT_NIL_P(streamer->receive());
    ASSERT_EQ(res_two.payload(), "Read request: Sending New Message");

    ASSERT_OCCURRED_AS_P(streamer->receive(), ERR_EOF);

    testutil::stop_servers();
    s.join();
}

/// @brief it should return an error when stream server is not available.
TEST(testGRPC, testStreamError) {
    std::string target("localhost:8080");
    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto streamer = ASSERT_NIL_P(client.stream(target));
    ASSERT_OCCURRED_AS(streamer->send(mes), STREAM_CLOSED);

    ASSERT_OCCURRED_AS_P(streamer->receive(), UNREACHABLE);
}

void client_send(
    const int num,
    const std::shared_ptr<UnaryClient<RQ, RS, UNARY_RPC>> &client
) {
    auto mes = test::Message();
    mes.set_payload(std::to_string(num));
    auto res = ASSERT_NIL_P(client->send("", mes));
    ASSERT_EQ(res.payload(), "Read request: " + std::to_string(num));
}

constexpr int N_THREADS = 3;

/// @brief it should handle concurrent unary requests from multiple threads.
TEST(testGRPC, stressTestUnaryWithManyThreads) {
    std::thread s(testutil::server, base_target);
    testutil::wait_for_servers();
    auto pool = std::make_shared<Pool>();
    auto global_unary_client = std::make_shared<UnaryClient<RQ, RS, UNARY_RPC>>(
        pool,
        base_target
    );

    const auto mw = std::make_shared<myMiddleware>();
    global_unary_client->use(mw);
    std::vector<std::thread> threads;

    threads.reserve(N_THREADS);
    for (int i = 0; i < N_THREADS; i++)
        threads.emplace_back(client_send, i, global_unary_client);
    for (size_t i = 0; i < N_THREADS; i++) {
        threads[i].join();
    }
    testutil::stop_servers();
    s.join();
}

void stream_send(
    const int num,
    const std::shared_ptr<StreamClient<RQ, RS, STREAM_RPC>> &client
) {
    auto mes = test::Message();
    mes.set_payload(std::to_string(num));
    const auto stream = ASSERT_NIL_P(client->stream(""));
    ASSERT_NIL(stream->send(mes));
    const auto res = ASSERT_NIL_P(stream->receive());
    ASSERT_EQ(res.payload(), "Read request: " + std::to_string(num));
}

/// @brief it should handle concurrent stream requests from multiple threads.
TEST(testGRPC, stressTestStreamWithManyThreads) {
    std::thread s(testutil::server, base_target);
    testutil::wait_for_servers();

    auto pool = std::make_shared<Pool>();
    auto global_stream_client = std::make_shared<StreamClient<RQ, RS, STREAM_RPC>>(
        pool,
        base_target
    );

    const auto mw = std::make_shared<myMiddleware>();
    global_stream_client->use(mw);
    std::vector<std::thread> threads;

    threads.reserve(N_THREADS);
    for (int i = 0; i < N_THREADS; i++)
        threads.emplace_back(stream_send, i, global_stream_client);
    for (size_t i = 0; i < N_THREADS; i++)
        threads[i].join();
    testutil::stop_servers();
    s.join();
}

/// @brief it should reuse the same channel for requests to the same host.
TEST(testGRPC, testPoolChannelReuse) {
    std::string target("localhost:8080");
    std::thread s(testutil::server, target);
    testutil::wait_for_servers();

    auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool);

    // Send to first endpoint
    auto mes1 = test::Message();
    mes1.set_payload("First endpoint");
    const auto res1 = ASSERT_NIL_P(client.send(target + "/endpoint1", mes1));
    ASSERT_EQ(res1.payload(), "Read request: First endpoint");

    // Send to second endpoint with same host:port
    auto mes2 = test::Message();
    mes2.set_payload("Second endpoint");
    const auto res2 = ASSERT_NIL_P(client.send(target + "/endpoint2", mes2));
    ASSERT_EQ(res2.payload(), "Read request: Second endpoint");

    // Get the channel count from the pool's internal map
    size_t channel_count = pool->size();
    EXPECT_EQ(channel_count, 1)
        << "Pool should maintain only one channel for the same host:port";

    testutil::stop_servers();
    s.join();
}

/// @brief it should not crash when calling close_send on a dead connection.
TEST(testGRPC, testCloseSendOnDeadConnection) {
    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool, "localhost:9999");
    auto streamer = ASSERT_NIL_P(client.stream(""));
    auto mes = test::Message();
    mes.set_payload("test");
    streamer->send(mes);
    ASSERT_OCCURRED_AS_P(streamer->receive(), UNREACHABLE);
    streamer->close_send();
}

/// @brief it should safely call close_send multiple times.
TEST(testGRPC, testCloseSendIdempotent) {
    std::string target("localhost:8080");
    std::thread s(testutil::server, target);
    testutil::wait_for_servers();
    auto pool = std::make_shared<Pool>();
    auto client = StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto streamer = ASSERT_NIL_P(client.stream(""));
    streamer->close_send();
    streamer->close_send();
    testutil::stop_servers();
    s.join();
}

/// @brief a unary service that records how long each call has left before its
/// deadline. Each call waits for delay or until it is cancelled, and the first
/// `failures` calls fail as unavailable.
class DeadlineRecorder final : public test::UnaryMessageService::Service {
public:
    std::chrono::milliseconds delay{0};
    std::size_t failures = 0;
    std::mutex mu;
    std::vector<std::chrono::system_clock::duration> remaining;

    ::grpc::Status Exec(
        ::grpc::ServerContext *context,
        const test::Message *,
        test::Message *
    ) override {
        std::size_t call = 0;
        {
            std::lock_guard lock(this->mu);
            this->remaining.push_back(
                context->deadline() - std::chrono::system_clock::now()
            );
            call = this->remaining.size();
        }
        const auto end = std::chrono::steady_clock::now() + this->delay;
        while (!context->IsCancelled() && std::chrono::steady_clock::now() < end)
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        if (call <= this->failures)
            return {::grpc::StatusCode::UNAVAILABLE, "unavailable"};
        return ::grpc::Status::OK;
    }
};

/// @brief sends a request a second time when the first attempt fails.
class RetryOnce final : public PassthroughMiddleware {
public:
    std::pair<Context, x::errors::Error>
    operator()(const Context context, Next &next) override {
        auto [res_ctx, err] = next(context);
        if (!err) return {res_ctx, err};
        return next(context);
    }
};

/// @brief sets the deadline of each request to timeout from now.
class FixedDeadline final : public PassthroughMiddleware {
    std::chrono::milliseconds timeout;

public:
    explicit FixedDeadline(const std::chrono::milliseconds timeout): timeout(timeout) {}

    std::pair<Context, x::errors::Error>
    operator()(Context context, Next &next) override {
        context.deadline = std::chrono::system_clock::now() + this->timeout;
        return next(context);
    }
};

class DeadlineTest : public ::testing::Test {
protected:
    DeadlineRecorder service;
    std::unique_ptr<::grpc::Server> server;
    std::string target;

    void SetUp() override {
        int port = 0;
        ::grpc::ServerBuilder builder;
        builder.AddListeningPort(
            "127.0.0.1:0",
            ::grpc::InsecureServerCredentials(),
            &port
        );
        builder.RegisterService(&this->service);
        this->server = builder.BuildAndStart();
        ASSERT_NE(this->server, nullptr);
        this->target = "127.0.0.1:" + std::to_string(port);
    }

    void TearDown() override {
        this->server->Shutdown(std::chrono::system_clock::now());
    }
};

/// @brief it should fail a unary call with TIMEOUT when the server does not answer
/// before the unary timeout of the pool.
TEST_F(DeadlineTest, UnaryFailsAtPoolTimeout) {
    this->service.delay = std::chrono::seconds(10);
    const auto pool = std::make_shared<Pool>(
        false,
        Keepalive{},
        std::chrono::milliseconds(300)
    );
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, this->target);
    auto sent = std::async(std::launch::async, [&client] {
        auto mes = test::Message();
        return client.send("", mes);
    });
    const auto status = sent.wait_for(std::chrono::seconds(5));
    // Cancels a call that the timeout did not end.
    this->server->Shutdown(std::chrono::system_clock::now());
    const auto err = sent.get().second;
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS(err, TIMEOUT);
}

/// @brief it should give each attempt of a retried unary call the full unary timeout
/// of the pool.
TEST_F(DeadlineTest, RetryGetsFullPoolTimeout) {
    this->service.delay = std::chrono::seconds(1);
    this->service.failures = 1;
    const auto pool = std::make_shared<Pool>(
        false,
        Keepalive{},
        std::chrono::seconds(5)
    );
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, this->target);
    client.use(std::make_shared<RetryOnce>());
    auto mes = test::Message();
    ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(this->service.remaining.size(), 2);
    for (const auto &remaining: this->service.remaining) {
        EXPECT_GT(remaining, std::chrono::milliseconds(4500));
        EXPECT_LT(remaining, std::chrono::seconds(6));
    }
}

/// @brief it should use the deadline a middleware sets instead of the unary timeout
/// of the pool.
TEST_F(DeadlineTest, ContextDeadlineReplacesPoolTimeout) {
    const auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, this->target);
    client.use(std::make_shared<FixedDeadline>(std::chrono::seconds(2)));
    auto mes = test::Message();
    ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(this->service.remaining.size(), 1);
    EXPECT_LT(this->service.remaining[0], std::chrono::seconds(3));
    EXPECT_GT(this->service.remaining[0], std::chrono::seconds(1));
}
}
