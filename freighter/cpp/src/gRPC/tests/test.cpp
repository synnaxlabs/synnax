/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "src/gRPC/protos/service.grpc.pb.h"
#include "src/gRPC/client.h"
#include "src/gRPC/tests/server_test.h"

/// std.
#include <thread>
#include <iostream>

/// If we don't need to use stream, we can just define an
/// internal stream_t that doesn't actually hold anything.
struct stream_t
{
};

/// @brief Used to awake server main thread when 


/// @brief Meant to be call within a thread. Simple
/// gRPC server. 
void server(std::string target, grpc::Status status, int expectedCalls)
{
  std::string server_address(target);
  myServiceImpl service;

  grpc::ServerBuilder builder;
  builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
  builder.RegisterService(&service);

  std::unique_ptr<grpc::Server> server(builder.BuildAndStart());
  service.setExpectedCalls(expectedCalls);
  service.setStatus(status);

  std::unique_lock<std::mutex> lck(mut);
  while (!end_session)
  {
    cond.wait(lck);
  }
  lck.unlock();
  server->Shutdown();
}

/// @brief Test to make sure message proto works as expected.
TEST(testGRPC, basicProto)
{
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}

TEST(testGRPC, testBasicUnary)
{
    std::string target("localhost:8080");
    std::thread s(server, target, grpc::Status::OK, 1);

    auto a = gRPC<test::Message, test::Message, stream_t, test::messageService>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto b = a.send(target, mes);
    ASSERT_EQ(b.payload(), "Read request: Sending to Server");
    ASSERT_TRUE(b.error().empty());

    s.join();
}

TEST(testGRPC, testFailedUnary)
{
    std::string target("localhost:8080");
    std::thread s(server, target, grpc::Status::CANCELLED, 1);

    std::string failure_msg("failed to connect to all addresses; last error: UNKNOWN: ipv4:127.0.0.1:8080: Failed to connect to remote host: Connection refused");

    auto a = gRPC<test::Message, test::Message, stream_t, test::messageService>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    auto b = a.send(target, mes);
    ASSERT_EQ(b.error(), failure_msg);

    s.join();
}



