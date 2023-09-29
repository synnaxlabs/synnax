/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "freighter/gRPC/protos/service.grpc.pb.h"
#include "freighter/gRPC/client.h"
#include "freighter/gRPC/tests/server.h"

/// std.
#include <thread>
#include <iostream>

/// Internal response type uses message.
using response_t = test::Message;
using request_t = test::Message;
using rpc_t = test::messageService;
using err_t = grpc::Status;
using stream_t = gRPCStreamer<response_t, request_t, err_t, rpc_t>;

/// @brief Test to make sure message proto works as expected.
TEST(testGRPC, basicProto)
{
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}

/// @brief Test the basic unary interface on success.
TEST(testGRPC, testBasicUnary)
{
    std::string target("localhost:8080");
    std::thread s(server, target);

    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send(target, mes);
    ASSERT_EQ(res.payload(), "Read request: Sending to Server");
    ASSERT_TRUE(err.ok());

    stopServers();
    s.join();
}

/// @brief Test the basic unary interface on failure.
TEST(testGRPC, testFailedUnary)
{
    // Note that the easiest way to cause a failure
    // here is to simply not set up a server, so that
    // we don't get a response.
    std::string target("localhost:8080");
    std::string failure_msg("failed to connect to all addresses; last error: UNKNOWN: ipv4:127.0.0.1:8080: Failed to connect to remote host: Connection refused");
    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto [res, err] = client.send(target, mes);
    ASSERT_EQ(res.payload(), "");
    ASSERT_EQ(err.error_message(), failure_msg);
}

/// @brief Test sending a message to multiple targets.
TEST(testGRPC, testMultipleTargets)
{
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);

    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
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
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testBasicStream)
{
    std::string target("localhost:8080");
    std::thread s(server, target);

    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes = test::Message();

    auto streamer = client.stream(target);
    mes.set_payload("Sending to Streaming Server");
    auto err = streamer.send(mes);
    streamer.closeSend();
    auto [res, err2] = streamer.receive();
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");

    stopServers();
    s.join();
}

/// @brief Test making and sending with multiple stream objects.
TEST(testGRPC, testMultipleStreamObjects)
{
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);

    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes_one = test::Message();
    auto mes_two = test::Message();

    auto streamer_one = client.stream(target_one);
    auto streamer_two = client.stream(target_two);
    mes_one.set_payload("Sending to Streaming Server from Streamer One");
    mes_two.set_payload("Sending to Streaming Server from Streamer Two");
    auto err_one = streamer_one.send(mes_one);
    streamer_one.closeSend();
    auto err_two = streamer_two.send(mes_two);
    streamer_two.closeSend();
    auto [res_one, err_one2] = streamer_one.receive();
    auto [res_two, err_two2] = streamer_two.receive();
    ASSERT_EQ(res_one.payload(), "Read request: Sending to Streaming Server from Streamer One");
    ASSERT_EQ(res_two.payload(), "Read request: Sending to Streaming Server from Streamer Two");

    stopServers();
    s1.join();
    s2.join();
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testSendMultipleMessages)
{
    std::string target("localhost:8080");
    std::thread s(server, target);

    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes = test::Message();
    auto mes_two = test::Message();

    auto streamer = client.stream(target);
    mes.set_payload("Sending to Streaming Server");
    streamer.send(mes);
    auto [res, err2] = streamer.receive();
    ASSERT_EQ(res.payload(), "Read request: Sending to Streaming Server");

    mes_two.set_payload("Sending New Message");
    streamer.send(mes_two);
    streamer.closeSend();
    auto [res_two, err_two2] = streamer.receive();
    ASSERT_EQ(res_two.payload(), "Read request: Sending New Message");

    stopServers();
    s.join();
}

/// @brief Test sending and receiving one message.
TEST(testGRPC, testStreamError)
{
    std::string target("localhost:8080");
    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();
    auto mes = test::Message();

    auto streamer = client.stream(target);
    grpc::Status err = streamer.send(mes);
    ASSERT_FALSE(err.ok());

    auto [res, err2] = streamer.receive();
    ASSERT_FALSE(err2.ok());
}



