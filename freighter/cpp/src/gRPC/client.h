#pragma once

/// Abstract class.
#include "src/freighter.h"

/// std.
#include <memory>
#include <string>
#include <utility>

/// grpc.
#include <grpc/grpc.h>
#include <grpcpp/grpcpp.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/security/credentials.h>

/// @brief freighter stream object.
template <typename response_t, typename request_t, typename err_t>
class gRPCStreamer : public Streamer<response_t, request_t, err_t>
{
public:
    /// @brief Ctor saves gRPC stream object to use under the hood.
    gRPCStreamer(std::unique_ptr<grpc::ClientReaderWriter<response_t, request_t>> stream_) 
    : stream(std::move(stream_)) {}

    /// @brief Streamer send.
    err_t send(request_t &request) override
    {
        // TODO: Expand on the returned statuses.
        return stream->Write(request) == true ? err_t() : err_t();
    }

    /// @brief Streamer read.
    std::pair<response_t, err_t> receive() override
    {
        response_t res;
        stream->Read(&res);
        err_t err = err_t();
        return {res, err};
    }

    /// @brief Closing streamer.
    err_t closeSend() override
    {
        stream->WritesDone();
        return err_t();
    }

private:
    std::unique_ptr<grpc::ClientReaderWriter<response_t, request_t>> stream;
};

/// @brief gRPC specific class
/// NOTE: stub_t comes from the generated protobuf file.
template <typename response_t, typename request_t, typename stream_t, typename err_t, typename rpc_t>
class gRPC : public Client<response_t, request_t, stream_t, err_t>
{ 
public:
    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<response_t, err_t> send(std::string target, request_t &request) override
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
        auto stat = stub->Unary(&context, request, &response);
        err_t status = stat;

        return {response, status};
    }

    /// @brief Interface for stream.
    /// @param target The server's IP. 
    /// @returns A stream object, which can be used to listen to the server.
    stream_t stream(std::string target) override
    {
        grpc::ClientContext context;

        // To abstract the interface, we construct the stub only if needed.
        if (!stub || target != last_target)
        {
            // TODO: Set up crypto context.
            auto channel = grpc::CreateChannel(target, grpc::InsecureChannelCredentials());
            stub = rpc_t::NewStub(channel);
        }
        return gRPCStreamer<response_t, request_t, err_t>(stub->Stream(&context));
    }
    
private:
    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;

    /// The last target used.
    std::string last_target;
};
