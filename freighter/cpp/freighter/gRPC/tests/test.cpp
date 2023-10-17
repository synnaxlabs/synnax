// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "freighter/gRPC/protos/service.grpc.pb.h"
#include "freighter/gRPC/client.h"
#include "freighter/gRPC/tests/server.h"

/// std.
#include <thread>

/// Internal response type uses message.
using response_t = test::Message;
using request_t = test::Message;
using unary_rpc_t = test::UnaryMessageService;
using stream_rpc_t = test::StreamMessageService;

auto base_target = "localhost:8080";

/// @brief Test to make sure message proto works as expected.
TEST(testGRPC, basicProto) {
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}

/// @brief Test the basic unary interface on success.
TEST(testGRPC, testBasicUnary) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    auto pool = new GRPCPool();
    auto client = GRPCUnaryClient<response_t, request_t, unary_rpc_t>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_TRUE(err.ok());
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stopServers();
    s.join();
    delete pool;
}

class myMiddleware : public Freighter::PassthroughMiddleware {
public:
    bool ack = false;

    std::pair<Freighter::Context, Freighter::Error> operator()(Freighter::Context context) override {
        context.set("test", "5");
        auto [outContext, exc] =  Freighter::PassthroughMiddleware::operator()(context);
        auto a = outContext.get("test");
        if (a == "dog") {
            ack = true;
        }
        return {outContext, exc};
    }
};

/// @brief Test that the basic unary interface propagates metadata headers through
/// middleware.
TEST(testGRPC, testMiddlewareInjection) {
    std::thread s(server, base_target);
    // Sleep for 100 ms to make sure server is up.
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    auto pool = new GRPCPool();
    auto client = GRPCUnaryClient<response_t, request_t, unary_rpc_t>(pool, base_target);
    auto mw = new myMiddleware();
    client.use(mw);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    stopServers();
    s.join();
    delete pool;
}

/// @brief Test the basic unary interface on failure.
TEST(testGRPC, testFailedUnary) {
    // Note that the easiest way to cause a failure
    // here is to simply not set up a server, so that
    // we don't get a response.
    std::string failure_msg(
            "failed to connect to all addresses; last error: UNKNOWN: ipv4:127.0.0.1:8080: Failed to connect to remote host: Connection refused");
    auto pool = new GRPCPool();
    auto client = GRPCUnaryClient<response_t, request_t, unary_rpc_t>(pool, base_target);
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send("", mes);
    ASSERT_EQ(res.payload(), "");
    ASSERT_EQ(err.message(), failure_msg);
    delete pool;
}

/// @brief Test sending a message to multiple targets.
TEST(testGRPC, testMultipleTargets) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);

    auto pool = new GRPCPool();
    auto client = GRPCUnaryClient<response_t, request_t, unary_rpc_t>(pool, base_target);
    auto mes_one = test::Message();
    mes_one.set_payload("Sending to Server One");
    auto [res_one, _] = client.send(target_one, mes_one);
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Server One");

    auto mes_two = test::Message();
    mes_two.set_payload("Sending to Server Two");
    auto [res_two, __] = client.send(target_two, mes_two);
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Server Two");

    stopServers();
    s1.join();
    s2.join();
    delete pool;
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testBasicStream) {
    std::string target("localhost:8080");
    std::thread s(server, target);

    auto pool = new GRPCPool();
    auto client = GRPCStreamClient<response_t, request_t, stream_rpc_t>(pool, base_target);
    auto mes = test::Message();

    auto [streamer ,err] = client.stream(target);
    ASSERT_FALSE(err);
    mes.set_payload("Sending to Streaming Server");
    err = streamer->send(mes);
    ASSERT_FALSE(err);
    ASSERT_FALSE(streamer->closeSend());
    auto [res, err2] = streamer->receive();
    ASSERT_FALSE(err2);
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");
    stopServers();
    s.join();
    delete pool;
}

/// @brief Test making and sending with multiple stream objects.
TEST(testGRPC, testMultipleStreamObjects) {
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);

    auto pool = new GRPCPool();
    auto client = GRPCStreamClient<response_t, request_t, stream_rpc_t>(pool, base_target);
    auto mes_one = test::Message();
    auto mes_two = test::Message();

    auto [streamer_one , err_one] = client.stream(target_one);
    auto [streamer_two , err_two] = client.stream(target_two);
    ASSERT_FALSE(err_one);
    ASSERT_FALSE(err_two);
    mes_one.set_payload("Sending to Streaming Server from Streamer One");
    mes_two.set_payload("Sending to Streaming Server from Streamer Two");
    ASSERT_FALSE(streamer_one->send(mes_one));
    streamer_one->closeSend();
    ASSERT_FALSE(streamer_two->send(mes_two));
    streamer_two->closeSend();
    auto [res_one, err_one2] = streamer_one->receive();
    auto [res_two, err_two2] = streamer_two->receive();
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Streaming Server from Streamer One");
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Streaming Server from Streamer Two");

    stopServers();
    s1.join();
    s2.join();
    delete pool;
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testSendMultipleMessages) {
    std::string target("localhost:8080");
    std::thread s(server, target);

    auto pool = new GRPCPool();
    auto client = GRPCStreamClient<response_t, request_t, stream_rpc_t>(pool, base_target);
    auto mes = test::Message();
    auto mes_two = test::Message();

    auto [streamer, exc] = client.stream(target);
    ASSERT_FALSE(exc);
    mes.set_payload("Sending to Streaming Server");
    streamer->send(mes);
    auto [res, err2] = streamer->receive();
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");

    mes_two.set_payload("Sending New Message");
    streamer->send(mes_two);
    streamer->closeSend();
    auto [res_two, err_two2] = streamer->receive();
    ASSERT_EQ(res_two.payload(), "Read request: Sending New Message");

    stopServers();
    s.join();
    delete pool;
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testStreamError) {
    std::string target("localhost:8080");
    auto pool = new GRPCPool();
    auto client = GRPCStreamClient<response_t, request_t, stream_rpc_t>(pool, base_target);
    auto mes = test::Message();

    auto [streamer ,exc]= client.stream(target);
    ASSERT_FALSE(exc);
    Freighter::Error err = streamer->send(mes);
    ASSERT_FALSE(err.ok());

    auto [res, err2] = streamer->receive();
    ASSERT_FALSE(err2.ok());
    delete pool;
}
