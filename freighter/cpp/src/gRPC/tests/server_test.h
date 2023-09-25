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

class myServiceImpl final : public test::messageService::Service 
{
public:
    grpc::Status Exec(grpc::ServerContext* context, const test::Message* request, test::Message* reply) override
    {
        std::string rep("Read request: ");
        reply->set_payload(rep + request->payload());

        currentCalls++;
        if (currentCalls == expectedCalls_)
        {
            std::unique_lock<std::mutex> lck(mut);
            end_session = true;
            cond.notify_all();
            lck.unlock();
        }
        return status_;
    }

    /// @brief Need this to be able to terminate 
    void setExpectedCalls(int expectedCalls) 
    {
        expectedCalls_ = expectedCalls;
    }

    /// @brief Sets the status we want to send back. 
    void setStatus(grpc::Status status)
    {
        status_ = status;
    }

private:
    int expectedCalls_;
    int currentCalls = 0;
    grpc::Status status_ = grpc::Status::OK;
};