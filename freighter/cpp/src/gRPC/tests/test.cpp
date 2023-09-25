/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "src/gRPC/protos/service.grpc.pb.h"
#include "src/gRPC/client.h"
#include "src/gRPC/tests/server.h"

/// std.
#include <thread>
#include <iostream>

/// If we don't need to use stream, we can just define an
/// internal stream_t that doesn't actually hold anything.
struct stream_t
{
};

/// Internal response type uses message.
struct response_t
{
    test::Message payload;
    grpc::Status status;
};

using request_t = test::Message;
using rpc_t = test::messageService;

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

    auto client = gRPC<response_t, request_t, stream_t, rpc_t>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto res = client.send(target, mes);
    ASSERT_EQ(res.payload.payload(), "Read request: Sending to Server");
    ASSERT_TRUE(res.status.ok());

    stopServer();
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
    auto client = gRPC<response_t, request_t, stream_t, rpc_t>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto res = client.send(target, mes);
    ASSERT_EQ(res.payload.payload(), "");
    ASSERT_EQ(res.status.error_message(), failure_msg);
}

TEST(testGRPC, testMultipleTargets)
{
    std::string target_one("localhost:8080");
    std::string target_two("localhost:8081");
    std::thread s1(server, target_one);
    std::thread s2(server, target_two);

    auto client = gRPC<response_t, request_t, stream_t, rpc_t>();
    auto mes_one = test::Message();
    mes_one.set_payload("Sending to Server One");
    auto res_one = client.send(target_one, mes_one);
    ASSERT_EQ(res_one.payload.payload(), "Read request: Sending to Server One");

    auto mes_two = test::Message();
    mes_two.set_payload("Sending to Server Two");
    auto res_two = client.send(target_two, mes_two);
    ASSERT_EQ(res_two.payload.payload(), "Read request: Sending to Server Two");

    stopServers();
    s1.join();
    s2.join();
}



