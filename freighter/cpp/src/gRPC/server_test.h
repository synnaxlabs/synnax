#pragma once

// NOTE: This class is only used for testing purposes.
#include <grpc/grpc.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/security/credentials.h>
#include <string>
#include "src/gRPC/protos/service.grpc.pb.h"

class myServiceImpl final : public test::messageService::Service 
{
public:
    grpc::Status Exec(grpc::ServerContext* context, const test::Message* request, test::Message* reply) override
    {
        std::string rep("Read request: ");
        reply->set_payload(rep + request->payload());
        return grpc::Status::OK;
    }
};