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

/// std.
#include <mutex>

freighter::Error errorFromGRPCStatus(grpc::Status status)
{
    if (status.ok())
        return freighter::NIL;
    if (status.error_code() == grpc::StatusCode::UNAVAILABLE)
        return {freighter::TYPE_UNREACHABLE, status.error_message()};
    return {status.error_message()};
}

std::string readFile(const std::string &path) {
    std::string data;
    FILE *f = fopen(path.c_str(), "r");
    if (f == nullptr)
        throw std::runtime_error("failed to open " + path);
    char buf[1024];
    for (;;) {
        ssize_t n = fread(buf, 1, sizeof(buf), f);
        if (n <= 0)
            break;
        data.append(buf, n);
    }
    if (ferror(f)) {
        throw std::runtime_error("failed to read " + path);
    }
    fclose(f);
    return data;
}

class GRPCPool
{
private:
    /// @brief A map of channels to targets.
    std::unordered_map<std::string, std::shared_ptr<grpc::Channel>> channels;

    /// @brief GRPC credentials to provide when connecting to a target.
    std::shared_ptr<grpc::ChannelCredentials> credentials = grpc::InsecureChannelCredentials();

public:
    GRPCPool() = default;


    /// @brief Instantiates the GRPC pool to use TLS encryption where the CA certificate
    /// is located at the provided path.
    explicit GRPCPool(const std::string &ca_path)
    {
        grpc::SslCredentialsOptions opts;
        opts.pem_root_certs = readFile(ca_path);
        credentials = grpc::SslCredentials(opts);
    }

    /// @brief instantiates the GRPC pool to use TLS encryption and authentication
    /// where the CA certificate, client certificate, and client key are located at
    /// the provided paths.
    GRPCPool(
        const std::string &ca_path,
        const std::string &cert_path,
        const std::string &key_path)
    {
        grpc::SslCredentialsOptions opts;
        opts.pem_root_certs = readFile(ca_path);
        opts.pem_cert_chain = readFile(cert_path);
        opts.pem_private_key = readFile(key_path);
        credentials = grpc::SslCredentials(opts);
    }

    /// @brief instantiates a GRPC pool with the provided credentials.
    GRPCPool(std::shared_ptr<grpc::ChannelCredentials> credentials) : credentials(credentials) {}

    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target.
    std::shared_ptr<grpc::Channel> getChannel(const std::string &target)
    {
        if (channels.find(target) == channels.end()) {
            channels[target] = grpc::CreateChannel(target, credentials);
        }
        return channels[target];
    }
};

/// @brief freighter stream object.
template <typename response_t, typename request_t, typename rpc_t>
class GRPCStream : public freighter::Stream<response_t, request_t>
{
public:
    /// Each streamer needs to manage its own context.
    grpc::ClientContext context;

    /// @brief Ctor saves GRPCUnaryClient stream object to use under the hood.
    explicit GRPCStream(std::shared_ptr<grpc::Channel> channel, freighter::Context &ctx)
    {
        // Note that the streamer also sets up its own internal stub.
        stub = rpc_t::NewStub(channel);
        for (auto &param : ctx.params)
            context.AddMetadata(param.first, param.second);
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
    std::unique_ptr<grpc::ClientReaderWriter<request_t, response_t>> stream;

    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;



    /// Last target managed.
    std::string last_target;
};

/// @brief An implementation of freighter::UnaryClient that uses GRPC as the backing transport. Safe to be shared between threads.
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

    /// @brief Adds a middleware to the chain.
    /// @implements UnaryClient::use
    void use(std::shared_ptr<freighter::Middleware> middleware) override { mw.use(middleware); }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<response_t, freighter::Error> send(const std::string &target, request_t &request) override
    {
        freighter::Context ctx = freighter::Context("grpc", base_target.child(target).toString());

        // Set this context's id to largest id.
        mut.lock();
        ctx.id = ++largest_id;
        latest_requests_and_responses[ctx.id].first = request;
        mut.unlock();

        auto [_, exc] = mw.exec(ctx, this);

        // Clean up container.
        mut.lock();
        auto latest_response = latest_requests_and_responses[ctx.id].second;
        latest_requests_and_responses.erase(ctx.id);
        mut.unlock();

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
        auto stub = rpc_t::NewStub(channel);
        auto latest_response = response_t();

        // Retrieve latest request with lock held.
        mut.lock();
        auto latest_request = latest_requests_and_responses[outboundContext.id].first;
        mut.unlock();

        auto stat = stub->Exec(&grpcContext, latest_request, &latest_response);
        if (!stat.ok())
            return {outboundContext, errorFromGRPCStatus(stat)};

        // If stat is ok, we can set response.
        mut.lock();
        latest_requests_and_responses[outboundContext.id].second = latest_response;
        mut.unlock();

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

    /// Used to map from context id to request/ response pair.
    std::unordered_map<int, std::pair<request_t, response_t>> latest_requests_and_responses;

    /// Used to keep track of the largest id.
    int largest_id = 0;

    /// For thread safety.
    std::mutex mut;
};

/// @brief An implementation of freighter::StreamClient that uses GRPC as the backing transport. Safe to be shared between threads.
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

    /// @brief Adds a middleware to the chain.
    /// @implements StreamClient::use
    void use(std::shared_ptr<freighter::Middleware> middleware) override { mw.use(middleware); }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    /// NOTE: Sharing stream invocations is not thread safe.
    /// It is suggested to create one StreamClient and create a stream per thread.
    std::pair<std::unique_ptr<freighter::Stream<response_t, request_t>>, freighter::Error>
    stream(const std::string &target) override
    {
        // Requires lock to do this or else DNS resolver gets overloaded.
        mut.lock();
        freighter::Context ctx = freighter::Context("grpc", base_target.child(target).toString());

        // Get context id quickly.
        ctx.id = ++largest_id;
        mut.unlock();

        // Mut is unlocked for expensive exec function call.
        auto [_, exc] = mw.exec(ctx, this);

        // Lock again to read data in latest_streams.
        mut.lock();
        auto latest_stream = std::move(latest_streams[ctx.id]);
        latest_streams.erase(ctx.id);
        mut.unlock();

        return {std::move(latest_stream), exc};
    }

    /// @brief the finalizer that opens the stream.
    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context outboundContext) override
    {
        auto channel = pool->getChannel(outboundContext.target);
        auto latest_stream = std::make_unique<GRPCStream<response_t, request_t, rpc_t>>(channel, outboundContext);
        // Set inbound metadata.
        auto inboundContext = freighter::Context(outboundContext.protocol, outboundContext.target);
        for (auto &meta : latest_stream->context.GetServerTrailingMetadata())
            inboundContext.set(meta.first.data(), meta.second.data());

        mut.lock();
        latest_streams[outboundContext.id] = std::move(latest_stream);
        mut.unlock();

        return {inboundContext, freighter::NIL};
    }

private:
    /// GRPCPool to pool connections across clients.
    std::shared_ptr<GRPCPool> pool;

    /// Middleware collector.
    freighter::MiddlewareCollector mw;

    /// Base target for all requests.
    freighter::URL base_target;

    /// Map from context instances to latest streams.
    std::unordered_map<int, std::unique_ptr<freighter::Stream<response_t, request_t>>> latest_streams;

    /// Largest id a context has at the moment.
    int largest_id = 0;

    /// Lock for mutual exclusion.
    std::mutex mut;
};