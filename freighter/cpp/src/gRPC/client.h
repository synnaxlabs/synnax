#pragma once

/// Abstract class.
#include "src/freighter.h"

/// std.
#include <memory>
#include <string>

/// grpc.
#include <grpc/grpc.h>
#include <grpcpp/grpcpp.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/security/credentials.h>

/// @brief gRPC specific class
/// NOTE: stub_t comes from the generated protobuf file.
template <typename response_t, typename request_t, typename stream_t, typename rpc_t>
class gRPC : public Client<response_t, request_t, stream_t>
{ 
public:
    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    response_t send(std::string target, request_t &request) override
    {
    grpc::ClientContext context;
    // To abstract the interface, we construct the stub only if needed.
    if (!stub || target != last_target)
    {
        // TODO: Set up crypto context.
        auto channel = grpc::CreateChannel(target, grpc::InsecureChannelCredentials());
        stub = rpc_t::NewStub(channel);
    }
    response_t response;
    auto stat = stub->Exec(&context, request, &response.payload);
    response.status = stat;
    
    return response;
    }

    /// @brief Interface for stream.
    stream_t stream(std::string target) override
    {
        return stream_t();
    }
    
    private:
    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;

    /// The last target used.
    std::string last_target;
};
