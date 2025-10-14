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

/// Internal response type uses message.
using RQ = test::Message;
using RS = test::Message;
using UNARY_RPC = test::UnaryMessageService;
using STREAM_RPC = test::StreamMessageService;

auto base_target = "localhost:8080";

/// @brief Test to make sure message proto works as expected.
TEST(testGRPC, basicProto) {
    auto m = test::Message();
    m.set_payload("Hello");
    ASSERT_EQ(m.payload(), "Hello");
}

///// @brief Test the basic unary interface on success.
TEST(testGRPC, testBasicUnary) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_TRUE(err.ok());
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stopServers();
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

///// @brief Test that the basic unary interface propagates metadata headers through
///// middleware.
TEST(testGRPC, testMiddlewareInjection) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mw = std::make_shared<myMiddleware>();
    client.use(mw);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stopServers();
    s.join();
}

///// @brief Test the basic unary interface on failure.
TEST(testGRPC, testFailedUnary) {
    // Note that the easiest way to cause a failure
    // here is to simply not set up a server, so that
    // we don't get a response.
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_EQ(res.payload(), "");
    ASSERT_TRUE(err.matches(freighter::UNREACHABLE));
}

///// @brief Test sending a message to multiple targets.
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
    auto [res_one, err] = client.send(target_one, mes_one);
    ASSERT_FALSE(err);
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Server One");

    auto mes_two = test::Message();
    mes_two.set_payload("Sending to Server Two");
    auto [res_two, err2] = client.send(target_two, mes_two);
    ASSERT_FALSE(err2);
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Server Two");

    stopServers();
    s1.join();
    s2.join();
}

///// @brief Test sending and receiving one message.
TEST(testGRPC, testBasicStream) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto [streamer, err] = client.stream("");
    ASSERT_FALSE(err);
    mes.set_payload("Sending to Streaming Server");
    err = streamer->send(mes);
    ASSERT_FALSE(err) << err.message();
    streamer->close_send();
    auto [res, err2] = streamer->receive();
    ASSERT_FALSE(err2) << err2.message();
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");
    auto [_, err3] = streamer->receive();
    ASSERT_TRUE(err3.type == freighter::EOF_ERR.type) << err3.message();
    stopServers();
    s.join();
}

///// @brief Test making and sending with multiple stream objects.
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

    auto [streamer_one, err_one] = client.stream(target_one);
    auto [streamer_two, err_two] = client.stream(target_two);
    ASSERT_FALSE(err_one);
    ASSERT_FALSE(err_two);
    mes_one.set_payload("Sending to Streaming Server from Streamer One");
    mes_two.set_payload("Sending to Streaming Server from Streamer Two");
    ASSERT_FALSE(streamer_one->send(mes_one));
    streamer_one->close_send();
    ASSERT_FALSE(streamer_two->send(mes_two));
    streamer_two->close_send();
    auto [res_one, err_one2] = streamer_one->receive();
    auto [res_two, err_two2] = streamer_two->receive();
    ASSERT_EQ(
        res_one.payload(),
        "Read request: Sending to Streaming Server from Streamer One"
    );
    ASSERT_EQ(
        res_two.payload(),
        "Read request: Sending to Streaming Server from Streamer Two"
    );
    auto err_one3 = streamer_one->receive().second;
    auto err_two3 = streamer_two->receive().second;
    ASSERT_TRUE(err_one3.type == freighter::EOF_ERR.type);
    ASSERT_TRUE(err_two3.type == freighter::EOF_ERR.type);

    stopServers();
    s1.join();
    s2.join();
}

///// @brief Test sending and receiving one message.
TEST(testGRPC, testSendMultipleMessages) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();
    auto mes_two = test::Message();

    auto [streamer, exc] = client.stream("");
    ASSERT_FALSE(exc) << exc;
    mes.set_payload("Sending to Streaming Server");
    streamer->send(mes);
    auto [res, err2] = streamer->receive();
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");

    mes_two.set_payload("Sending New Message");
    streamer->send(mes_two);
    streamer->close_send();
    auto [res_two, err_two2] = streamer->receive();
    ASSERT_FALSE(err_two2) << err_two2;
    ASSERT_EQ(res_two.payload(), "Read request: Sending New Message");

    auto [_, err3] = streamer->receive();
    ASSERT_TRUE(err3.type == freighter::EOF_ERR.type) << err3;

    stopServers();
    s.join();
}

///// @brief Test sending and receiving one message.
TEST(testGRPC, testStreamError) {
    std::string target("localhost:8080");
    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::StreamClient<RQ, RS, STREAM_RPC>(pool, base_target);
    auto mes = test::Message();

    auto [streamer, exc] = client.stream(target);
    ASSERT_FALSE(exc);
    xerrors::Error err = streamer->send(mes);
    ASSERT_FALSE(err.ok());

    auto [res, err2] = streamer->receive();
    ASSERT_FALSE(err2.ok());
}

void client_send(
    int num,
    std::shared_ptr<fgrpc::UnaryClient<RQ, RS, UNARY_RPC>> client
) {
    auto mes = test::Message();
    mes.set_payload(std::to_string(num));
    auto [res, err] = client->send("", mes);
    ASSERT_TRUE(err.ok());
    ASSERT_EQ(res.payload(), "Read request: " + std::to_string(num));
}

const int N_THREADS = 3;

///// @brief Test that we can send many messages with the same client and don't have any
/// errors.
TEST(testGRPC, stressTestUnaryWithManyThreads) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    auto pool = std::make_shared<fgrpc::Pool>();
    auto global_unary_client = std::make_shared<fgrpc::UnaryClient<RQ, RS, UNARY_RPC>>(
        pool,
        base_target
    );

    auto mw = std::make_shared<myMiddleware>();
    global_unary_client->use(mw);
    std::vector<std::thread> threads;

    // Time to boil all the cores.
    for (int i = 0; i < N_THREADS; i++) {
        threads.emplace_back(client_send, i, global_unary_client);
    }
    for (size_t i = 0; i < N_THREADS; i++) {
        threads[i].join();
    }
    stopServers();
    s.join();
}

void stream_send(
    int num,
    std::shared_ptr<fgrpc::StreamClient<RQ, RS, STREAM_RPC>> client
) {
    auto mes = test::Message();
    mes.set_payload(std::to_string(num));
    auto [stream, err] = client->stream("");
    ASSERT_TRUE(err.ok());
    err = stream->send(mes);
    ASSERT_TRUE(err.ok());
    auto [res, err_] = stream->receive();
    ASSERT_TRUE(err_.ok());
    ASSERT_EQ(res.payload(), "Read request: " + std::to_string(num));
}

///// @brief Test that we can send many messages with the same client and different
/// stream invocations.
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

    // Time to boil all the cores.
    for (int i = 0; i < N_THREADS; i++) {
        threads.emplace_back(stream_send, i, global_stream_client);
    }
    for (size_t i = 0; i < N_THREADS; i++) {
        threads[i].join();
    }
    stopServers();
    s.join();
}
TEST(testGRPC, testPoolChannelReuse) {
    std::string target("localhost:8080");
    std::thread s(server, target);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    auto pool = std::make_shared<fgrpc::Pool>();
    auto client = fgrpc::UnaryClient<RQ, RS, UNARY_RPC>(pool);

    // Send to first endpoint
    auto mes1 = test::Message();
    mes1.set_payload("First endpoint");
    auto [res1, err1] = client.send(target + "/endpoint1", mes1);
    ASSERT_FALSE(err1);

    // Send to second endpoint with same host:port
    auto mes2 = test::Message();
    mes2.set_payload("Second endpoint");
    auto [res2, err2] = client.send(target + "/endpoint2", mes2);
    ASSERT_FALSE(err2);

    // Get the channel count from the pool's internal map
    size_t channel_count = pool->size();
    EXPECT_EQ(channel_count, 1)
        << "Pool should maintain only one channel for the same host:port";

    stopServers();
    s.join();
}
