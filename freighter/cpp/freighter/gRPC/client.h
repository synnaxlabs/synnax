// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Abstract class.
#include "freighter/freighter.h"

/// grpc.
#include <grpc/grpc.h>
#include <grpcpp/grpcpp.h>
#include <grpcpp/channel.h>
#include <grpcpp/client_context.h>
#include <grpcpp/create_channel.h>
#include <grpcpp/security/credentials.h>

// TODO: Create internal error types!!

freighter::Error errorFromGRPCStatus(grpc::Status status)
{
    if (status.ok())
        return freighter::NIL;
    if (status.error_code() == grpc::StatusCode::UNAVAILABLE)
        return freighter::Error(freighter::TYPE_UNREACHABLE, status.error_message());
    return freighter::Error(status.error_message());
}

class GRPCPool
{
private:
    // TODO: Is this the right way to store the channels?

    /// @brief A map of channels to targets.
    std::map<std::string, std::shared_ptr<grpc::Channel>> channels;

public:
    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target.
    std::shared_ptr<grpc::Channel> getChannel(const std::string &target)
    {
        if (channels.find(target) == channels.end())
            channels[target] = grpc::CreateChannel(target, grpc::InsecureChannelCredentials());
        return channels[target];
    }

    // TODO: Is this the right way to destruct the pool?

    ~GRPCPool()
    {
        for (auto &channel : channels)
            channel.second.reset();
    }
};

/// @brief freighter stream object.
template <typename response_t, typename request_t, typename rpc_t>
class GRPCStream : public freighter::Stream<response_t, request_t>
{
public:
    /// @brief Ctor saves GRPCUnaryClient stream object to use under the hood.
    explicit GRPCStream(std::shared_ptr<grpc::Channel> channel)
    {
        // Note that the streamer also sets up its own internal stub.
        stub = rpc_t::NewStub(channel);
        stream = stub->Exec(&context);
    }

    /// @brief Streamer send.
    freighter::Error send(request_t &request) override
    {
        if (stream->Write(request))
            return freighter::NIL;
        return freighter::STREAM_CLOSED;
    }

    /// @brief Streamer read.
    std::pair<response_t, freighter::Error> receive() override
    {
        response_t res;
        if (stream->Read(&res))
            return {res, freighter::NIL};
        if (err)
            return {res, err};
        grpc::Status stat = stream->Finish();
        if (stat.ok())
            return {res, freighter::EOF_};
        return {res, freighter::Error(stat.error_message())};
    }

    /// @brief Closing streamer.
    freighter::Error closeSend() override
    {
        stream->WritesDone();
        return freighter::NIL;
    }

private:
    freighter::Error err = freighter::NIL;

    /// The internal streaming type for GRPCUnaryClient.
    std::unique_ptr<grpc::ClientReaderWriter<response_t, request_t>> stream;

    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;

    /// Each streamer needs to manage its own context.
    grpc::ClientContext context;

    /// Last target managed.
    std::string last_target;
};

/// @brief An implementation of freighter::UnaryClient that uses GRPC as the backing transport.
/// @implements freighter::UnaryClient
/// @see freighter::UnaryClient
template <typename response_t, typename request_t, typename rpc_t>
class GRPCUnaryClient : public freighter::UnaryClient<response_t, request_t>,
                        freighter::Finalizer
{

public:
    GRPCUnaryClient(
        std::shared_ptr<GRPCPool> pool,
        const std::string &base_target) : pool(pool),
                                          base_target(freighter::URL(base_target))
    {
    }

    GRPCUnaryClient(std::shared_ptr<GRPCPool> pool) : pool(pool) {}

    // TODO: Can we make this not passthrough? I was having trouble compiling when inheriting from freighter::MiddlewareCollector.

    /// @brief Adds a middleware to the chain.
    /// @implements UnaryClient::use
    void use(std::unique_ptr<freighter::Middleware> middleware) override { mw.use(std::move(middleware)); }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<response_t, freighter::Error> send(const std::string &target, request_t &request) override
    {
        this->latest_request = request;
        freighter::Context ctx = freighter::Context("grpc", base_target.child(target).toString());
        auto [_, exc] = mw.exec(ctx, this);
        return {latest_response, exc};
    }

    /// @brief the finalizer that executes the request.
    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context outboundContext) override
    {
        // Set outbound metadata.
        grpc::ClientContext grpcContext;
        for (auto &param : outboundContext.params)
            grpcContext.AddMetadata(param.first, param.second);

        // Execute request.
        auto channel = pool->getChannel(outboundContext.target);
        stub = rpc_t::NewStub(channel);
        latest_response = response_t();
        auto stat = stub->Exec(&grpcContext, latest_request, &latest_response);
        if (!stat.ok())
            return {outboundContext, errorFromGRPCStatus(stat)};

        // Set inbound metadata.
        auto inboundContext = freighter::Context(outboundContext.protocol, outboundContext.target);
        for (auto &meta : grpcContext.GetServerInitialMetadata())
            inboundContext.set(std::string(meta.first.begin(), meta.first.end()),
                               std::string(meta.second.begin(), meta.second.end()));
        return {inboundContext, freighter::NIL};
    }

private:
    /// Middleware collector.
    freighter::MiddlewareCollector mw;

    /// GRPCPool to pool connections across clients.
    std::shared_ptr<GRPCPool> pool;

    /// Base target for all request.
    freighter::URL base_target;

    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;

    // TODO: This means our client is not thread safe. I'd like to see if there is a better way to do this.

    /// Latest request. Use to pass request to finalizer.
    request_t latest_request;

    /// Latest response. Use to pass response from finalizer.
    response_t latest_response;
};

/// @brief An implementation of freighter::StreamClient that uses GRPC as the backing transport.
/// @implements freighter::StreamClient
/// @see freighter::StreamClient
template <typename response_t, typename request_t, typename rpc_t>
class GRPCStreamClient : public freighter::StreamClient<response_t, request_t>,
                         freighter::PassthroughMiddleware
{
public:
    GRPCStreamClient(
        std::shared_ptr<GRPCPool> pool,
        const std::string &base_target) : pool(pool),
                                          base_target(freighter::URL(base_target))
    {
    }

    explicit GRPCStreamClient(std::shared_ptr<GRPCPool> pool) : pool(pool) {}

    // TODO: Can we make this not passthrough? I was having trouble compiling when inheriting from freighter::MiddlewareCollector.

    /// @brief Adds a middleware to the chain.
    /// @implements StreamClient::use
    void use(std::unique_ptr<freighter::Middleware> middleware) override { mw.use(std::move(middleware)); }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    std::pair<std::unique_ptr<freighter::Stream<response_t, request_t>>, freighter::Error>
    stream(const std::string &target) override
    {
        freighter::Context ctx = freighter::Context("grpc", base_target.child(target).toString());
        auto [_, exc] = mw.exec(ctx, this);
        auto p = std::unique_ptr<freighter::Stream<response_t, request_t>>(latest_stream);
        latest_stream = nullptr;
        return {std::move(p), exc};
    }

    /// @brief the finalizer that opens the stream.
    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context outboundContext) override
    {
        // Set outbound metadata.
        grpc::ClientContext grpcContext;
        for (auto &param : outboundContext.params)
            grpcContext.AddMetadata(param.first, param.second);

        auto channel = pool->getChannel(outboundContext.target);
        latest_stream = new GRPCStream<response_t, request_t, rpc_t>(channel);

        // Set inbound metadata.
        auto inboundContext = freighter::Context(outboundContext.protocol, outboundContext.target);
        for (auto &meta : grpcContext.GetServerTrailingMetadata())
            inboundContext.set(meta.first.data(), meta.second.data());

        return {inboundContext, freighter::NIL};
    }

private:
    /// GRPCPool to pool connections across clients.
    std::shared_ptr<GRPCPool> pool;

    /// Middleware collector.
    freighter::MiddlewareCollector mw;

    /// Base target for all requests.
    freighter::URL base_target;

    // TODO: This means our client is not thread safe. I'd like to see if there is a better way to do this.

    /// Latest stream. Use to pass stream from finalizer.
    freighter::Stream<response_t, request_t> *latest_stream{};
};