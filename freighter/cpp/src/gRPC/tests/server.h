/// @brief This file abstracts the server side for gRPC testing.

#pragma once

// NOTE: This class is only used for testing purposes.
#include <grpc/grpc.h>
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>
#include <string>
#include <iostream>
#include <mutex>
#include <condition_variable>
#include "src/gRPC/protos/service.grpc.pb.h"

/// @brief Used to awake main thread when we are 
/// done processing messages.
std::mutex mut;
std::condition_variable cond;
bool end_session = false;

/// @brief Implements .proto generated interface Unary.
class myServiceImpl final : public test::messageService::Service 
{
public:
    grpc::Status Unary(grpc::ServerContext* context, const test::Message* request, test::Message* reply) override
    {
        std::string rep("Read request: ");
        reply->set_payload(rep + request->payload());
        return grpc::Status::OK;
    }
};

/// @brief Meant to be call within a thread. Simple
/// gRPC server. 
void server(std::string target)
{
  end_session = false;
  std::string server_address(target);
  myServiceImpl service;

  grpc::ServerBuilder builder;
  builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
  builder.RegisterService(&service);

  std::unique_ptr<grpc::Server> server(builder.BuildAndStart());

  std::unique_lock<std::mutex> lck(mut);
  while (!end_session)
  {
    cond.wait(lck);
  }
  lck.unlock();
  server->Shutdown();
  end_session = false;
}

/// @brief Abstraction of stopping servers.
void stopServers()
{
    end_session = true;
    cond.notify_all();
}