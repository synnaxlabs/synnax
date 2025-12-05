// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "freighter/cpp/fgrpc/fgrpc.h"
#include "freighter/cpp/fgrpc/mock/freighter/cpp/fgrpc/mock/service.grpc.pb.h"
#include "freighter/cpp/fgrpc/mock/server.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/xtest/xtest.h"

/// Internal response type uses message.
using RQ = test::Message;
using RS = test::Message;
using UNARY_RPC = test::UnaryMessageService;
using STREAM_RPC = test::StreamMessageService;

auto base_target = "localhost:8080";

/// @brief it should set and get payload in a message proto.
TEST(testGRPC, basicProto) {
    auto m = test::Message();
    m.set_payload("Hello");
    ASSERT_EQ(m.payload(), "Hello");
}

/// @brief it should send a unary request and receive a response.
TEST(testGRPC, testBasicUnary) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    const auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    const auto res = ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stop_servers();
    s.join();
}

class myMiddleware : public freighter::PassthroughMiddleware {
public:
    bool ack = false;

    std::pair<freighter::Context, xerrors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        context.set("test", "5");
        auto [outContext, exc] = next(context);
        auto a = outContext.get("test");
        if (a == "dog") { ack = true; }
        return {outContext, exc};
    }
};

/// @brief it should propagate metadata headers through middleware.
TEST(testGRPC, testMiddlewareInjection) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    const auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    const auto mw = std::make_shared<myMiddleware>();
    client.use(mw);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto res = ASSERT_NIL_P(client.send("", mes));
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stop_servers();
    s.join();
}

/// @brief it should return an unreachable error when server is not available.
TEST(testGRPC, testFailedUnary) {
    // Note that the easiest way to cause a failure
    // here is to simply not set up a server, so that
    // we don't get a response.
    const auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    ASSERT_OCCURRED_AS_P(client.send("", mes), freighter::UNREACHABLE);
}

/// @brief it should send messages to multiple targets.
TEST(testGRPC, testMultipleTargets) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool);
    auto mes_one = test::Message();
    mes_one.set_payload("Sending to Server One");
    auto res_one = ASSERT_NIL_P(client.send(target_one, mes_one));
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Server One");

    auto mes_two = test::Message();
    mes_two.set_payload("Sending to Server Two");
    auto res_two = ASSERT_NIL_P(client.send(target_two, mes_two));
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Server Two");

    stop_servers();
    s1.join();
    s2.join();
}

/// @brief it should send and receive a message over a stream.
TEST(testGRPC, testBasicStream) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto streamer = ASSERT_NIL_P(client.stream(""));
    mes.set_payload("Sending to Streaming Server");
    ASSERT_NIL(streamer->send(mes));
    streamer->close_send();
    auto res = ASSERT_NIL_P(streamer->receive());
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");
    ASSERT_OCCURRED_AS_P(streamer->receive(), freighter::EOF_ERR);
    stop_servers();
    s.join();
}

/// @brief it should send messages using multiple stream objects to different targets.
TEST(testGRPC, testMultipleStreamObjects) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool);
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
    ASSERT_OCCURRED_AS_P(streamer_one->receive(), freighter::EOF_ERR);
    ASSERT_OCCURRED_AS_P(streamer_two->receive(), freighter::EOF_ERR);

    stop_servers();
    s1.join();
    s2.join();
}

/// @brief it should send and receive multiple messages over a single stream.
TEST(testGRPC, testSendMultipleMessages) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
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

    ASSERT_OCCURRED_AS_P(streamer->receive(), freighter::EOF_ERR);

    stop_servers();
    s.join();
}

/// @brief it should return an unreachable error when stream server is not available.
TEST(testGRPC, testStreamError) {
    std::string target("localhost:8080");
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto streamer = ASSERT_NIL_P(client.stream(target));
    ASSERT_OCCURRED_AS(streamer->send(mes), freighter::UNREACHABLE);

    ASSERT_OCCURRED_AS_P(streamer->receive(), freighter::UNREACHABLE);
}

void client_send(
    const int num,
    const std::shared_ptr<fgrpc::UnaryClient<RQ, RS, UNARY_RPC>> &client
) {
    auto mes = test::Message();
    mes.set_payload(std::to_string(num));
    auto [res, err] = client->send("", mes);
    ASSERT_TRUE(err.ok());
    ASSERT_EQ(res.payload(), "Read request: " + std::to_string(num));
}

constexpr int N_THREADS = 3;

/// @brief it should handle concurrent unary requests from multiple threads.
TEST(testGRPC, stressTestUnaryWithManyThreads) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    auto pool = std::make_shared<fgrpc::Pool>();
    auto global_unary_client = std::make_shared<fgrpc::UnaryClient<RQ, RS, UNARY_RPC>>(
        pool,
        base_target
    );

    const auto mw = std::make_shared<myMiddleware>();
    global_unary_client->use(mw);
    std::vector<std::thread> threads;

    for (int i = 0; i < N_THREADS; i++)
        threads.emplace_back(client_send, i, global_unary_client);
    for (size_t i = 0; i < N_THREADS; i++) {
        threads[i].join();
    }
    stop_servers();
    s.join();
}

void stream_send(
    const int num,
    const std::shared_ptr<fgrpc::StreamClient<RQ, RS, STREAM_RPC>> &client
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
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto global_stream_client = std::make_shared<
        fgrpc::StreamClient<RQ, RS, STREAM_RPC>>(pool, base_target);

    const auto mw = std::make_shared<myMiddleware>();
    global_stream_client->use(mw);
    std::vector<std::thread> threads;

    for (int i = 0; i < N_THREADS; i++)
        threads.emplace_back(stream_send, i, global_stream_client);
    for (size_t i = 0; i < N_THREADS; i++)
        threads[i].join();
    stop_servers();
    s.join();
}

/// @brief it should reuse the same channel for requests to the same host.
TEST(testGRPC, testPoolChannelReuse) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool);

    // Send to first endpoint
    auto mes1 = test::Message();
    mes1.set_payload("First endpoint");
    auto res1 = ASSERT_NIL_P(client.send(target + "/endpoint1", mes1));

    // Send to second endpoint with same host:port
    auto mes2 = test::Message();
    mes2.set_payload("Second endpoint");
    auto res2 = ASSERT_NIL_P(client.send(target + "/endpoint2", mes2));

    // Get the channel count from the pool's internal map
    size_t channel_count = pool->size();
    EXPECT_EQ(channel_count, 1)
        << "Pool should maintain only one channel for the same host:port";

    stop_servers();
    s.join();
}
