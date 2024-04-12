// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include "freighter/cpp/freighter/freighter.h"
#include "grpc/grpc.h"
#include "grpcpp/grpcpp.h"
#include "grpcpp/channel.h"
#include "grpcpp/client_context.h"
#include "grpcpp/create_channel.h"
#include "grpcpp/security/credentials.h"


namespace priv {
const std::string PROTOCOL = "grpc";
/// @brief converts a grpc::Status to a freighter::Error.
inline freighter::Error errFromStatus(const grpc::Status &status) {
    if (status.ok()) return freighter::NIL;
    if (status.error_code() == grpc::StatusCode::UNAVAILABLE)
        return {freighter::UNREACHABLE.type, status.error_message()};
    return freighter::Error(status.error_message());
}

/// @brief an internal method for reading the entire contents of certificate files
/// into a string.
inline std::string readFile(const std::string &path) {
    std::string data;
    FILE *f = fopen(path.c_str(), "r");
    if (f == nullptr)
        throw std::runtime_error("failed to open " + path);
    char buf[1024];
    for (;;) {
        const size_t n = fread(buf, 1, sizeof(buf), f);
        if (n <= 0) break;
        data.append(buf, n);
    }
    if (ferror(f)) {
        throw std::runtime_error("failed to read " + path);
    }
    fclose(f);
    return data;
}
}

namespace fgrpc {
class Pool {
    /// @brief A map of channels to targets.
    std::unordered_map<std::string, std::shared_ptr<grpc::Channel> > channels{};

    /// @brief GRPC credentials to provide when connecting to a target.
    std::shared_ptr<grpc::ChannelCredentials> credentials =
            grpc::InsecureChannelCredentials();

public:
    Pool() = default;

    /// @brief Instantiates the GRPC pool to use TLS encryption where the CA certificate
    /// is located at the provided path.
    explicit Pool(const std::string &ca_path) {
        grpc::SslCredentialsOptions opts;
        opts.pem_root_certs = priv::readFile(ca_path);
        credentials = grpc::SslCredentials(opts);
    }

    /// @brief instantiates the GRPC pool to use TLS encryption and authentication
    /// where the CA certificate, client certificate, and client key are located at
    /// the provided paths.
    Pool(
        const std::string &ca_path,
        const std::string &cert_path,
        const std::string &key_path
    ) {
        grpc::SslCredentialsOptions opts;
        opts.pem_root_certs = priv::readFile(ca_path);
        opts.pem_cert_chain = priv::readFile(cert_path);
        opts.pem_private_key = priv::readFile(key_path);
        credentials = grpc::SslCredentials(opts);
    }

    /// @brief instantiates a GRPC pool with the provided credentials.
    explicit Pool(
        const std::shared_ptr<grpc::ChannelCredentials> &credentials
    ) : credentials(credentials) {
    }

    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target.
    std::shared_ptr<grpc::Channel> getChannel(const std::string &target) {
        if (channels.find(target) == channels.end())
            channels[target] = grpc::CreateChannel(target, credentials);
        return channels[target];
    }
};


/// @brief An implementation of freighter::UnaryClient that uses GRPC as the backing transport. Safe to be shared between threads.
/// @implements freighter::UnaryClient
/// @see freighter::UnaryClient
template<typename RQ, typename RS, typename RPC>
class UnaryClient final : public freighter::UnaryClient<RQ, RS>,
                          freighter::Finalizer {
public:
    UnaryClient(
        const std::shared_ptr<Pool> &pool,
        const std::string &base_target
    ) : pool(pool),
        base_target(freighter::URL(base_target)) {
    }

    explicit UnaryClient(const std::shared_ptr<Pool> &pool) : pool(pool) {
    }

    /// @brief Adds a middleware to the chain.
    /// @implements UnaryClient::use
    void use(const std::shared_ptr<freighter::Middleware> middleware) override {
        mw.use(middleware);
    }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<RS, freighter::Error> send(
        const std::string &target,
        RQ &request
    ) override {
        freighter::Context ctx(
            priv::PROTOCOL,
            base_target.child(target).toString(),
            freighter::UNARY
        );

        // Set this context's id to largest id.
        mut.lock();
        ctx.id = ++largest_id;
        req_res_buf[ctx.id].first = request;
        mut.unlock();

        auto err = mw.exec(ctx, this).second;

        // Clean up container.
        mut.lock();
        auto latest_response = req_res_buf[ctx.id].second;
        req_res_buf.erase(ctx.id);
        mut.unlock();

        return {latest_response, err};
    }

    /// @brief the finalizer that executes the request.
    std::pair<freighter::Context, freighter::Error> operator()(
        freighter::Context req_ctx
    ) override {
        // Set outbound metadata.
        grpc::ClientContext grpc_ctx;
        for (const auto &[k, v]: req_ctx.params)
            grpc_ctx.AddMetadata(k, v);

        // Execute request.
        auto channel = pool->getChannel(req_ctx.target);
        auto stub = RPC::NewStub(channel);
        auto res = RS();

        // Retrieve latest request with lock held.
        mut.lock();
        auto req = req_res_buf[req_ctx.id].first;
        mut.unlock();

        const auto stat = stub->Exec(&grpc_ctx, req, &res);
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::UNARY
        );
        if (!stat.ok()) return {res_ctx, priv::errFromStatus(stat)};

        // If stat is ok, we can set response.
        mut.lock();
        req_res_buf[req_ctx.id].second = res;
        mut.unlock();

        // Set inbound metadata.
        for (const auto &[k, v]: grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
        return {res_ctx, freighter::NIL};
    }

private:
    /// Middleware collector.
    freighter::MiddlewareCollector mw;

    /// GRPCPool to pool connections across clients.
    std::shared_ptr<Pool> pool;

    /// Base target for all request.
    freighter::URL base_target;

    /// Used to map from context id to request/ response pair.
    std::unordered_map<int, std::pair<RQ, RS> >
    req_res_buf;

    /// Used to keep track of the largest id.
    int largest_id = 0;

    /// For thread safety.
    std::mutex mut;
};

/// @brief freighter stream object.
template<typename RQ, typename RS, typename RPC>
class Stream final : public freighter::Stream<RQ, RS>, freighter::Finalizer {
public:
    /// @brief Ctor saves GRPCUnaryClient stream object to use under the hood.
    Stream(
        std::shared_ptr<grpc::Channel> ch,
        const freighter::MiddlewareCollector &mw,
        freighter::Context &req_ctx,
        freighter::Context &res_ctx
    ): mw(mw) {
        stub = RPC::NewStub(ch);
        for (const auto &[k, v]: req_ctx.params)
            grpc_ctx.AddMetadata(k, v);
        stream = stub->Exec(&grpc_ctx);
        stream->WaitForInitialMetadata();
        for (const auto &[k, v]: grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
    }

    /// @brief Streamer send.
    freighter::Error send(RQ &request) const override {
        if (stream->Write(request)) return freighter::NIL;
        return freighter::STREAM_CLOSED;
    }

    /// @brief Streamer read.
    std::pair<RS, freighter::Error> receive() override {
        RS res;
        if (stream->Read(&res)) return {res, freighter::NIL};
        const auto ctx = freighter::Context("grpc", "", freighter::STREAM);
        const auto err = mw.exec(ctx, this).second;
        return {res, err};
    }

    /// @brief Closing streamer.
    void closeSend() override {
        if (writes_done_called) return;
        stream->WritesDone();
        writes_done_called = true;
    }

    std::pair<freighter::Context, freighter::Error> operator()(
        freighter::Context outbound
    ) override {
        if (closed) return {outbound, close_err};
        const grpc::Status status = stream->Finish();
        closed = true;
        close_err = status.ok() ? freighter::EOF_ : priv::errFromStatus(status);
        return {outbound, close_err};
    }

private:
    freighter::MiddlewareCollector mw;

    std::unique_ptr<grpc::ClientReaderWriter<RQ, RS> > stream;
    /// For god knows what reason, GRPC requries us to keep these around so
    /// the stream doesn't die.
    grpc::ClientContext grpc_ctx{};
    std::unique_ptr<typename RPC::Stub> stub;

    bool closed = false;
    freighter::Error close_err = freighter::NIL;
    bool writes_done_called = false;
};

/// @brief An implementation of freighter::StreamClient that uses GRPC as the backing
/// transport. Safe to be shared between threads.
/// @implements freighter::StreamClient
/// @see freighter::StreamClient
template<typename RQ, typename RS, typename RPC>
class StreamClient final : public freighter::StreamClient<RQ, RS>,
                           freighter::PassthroughMiddleware {
public:
    StreamClient(
        const std::shared_ptr<Pool> &pool,
        const std::string &base_target
    ) : pool(pool),
        base_target(freighter::URL(base_target)) {
    }

    explicit StreamClient(const std::shared_ptr<Pool> &pool) : pool(pool) {
    }

    /// @brief Adds a middleware to the chain.
    /// @implements StreamClient::use
    void use(std::shared_ptr<freighter::Middleware> middleware) override {
        mw.use(middleware);
    }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    /// NOTE: Sharing stream invocations is not thread safe.
    /// It is suggested to create one StreamClient and create a stream per thread.
    std::pair<std::unique_ptr<freighter::Stream<RQ, RS> >,
        freighter::Error>
    stream(const std::string &target) override {
        mut.lock();
        auto ctx = freighter::Context(
            "grpc",
            base_target.child(target).toString(),
            freighter::STREAM
        );
        ctx.id = ++largest_id;
        mut.unlock();

        auto err = mw.exec(ctx, this).second;

        mut.lock();
        auto latest_stream = std::move(latest_streams[ctx.id]);
        latest_streams.erase(ctx.id);
        mut.unlock();
        return {std::move(latest_stream), err};
    }

    /// @brief the finalizer that opens the stream.
    std::pair<freighter::Context, freighter::Error> operator()(
        freighter::Context req_ctx
    ) override {
        auto channel = pool->getChannel(req_ctx.target);
        grpc::ClientContext grpcContext;
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::STREAM
        );
        auto latest_stream = std::make_unique<Stream<RQ, RS, RPC> >(
            channel,
            mw,
            req_ctx,
            res_ctx
        );
        mut.lock();
        latest_streams[req_ctx.id] = std::move(latest_stream);
        mut.unlock();
        if (res_ctx.has("error"))
            return {res_ctx, freighter::Error(res_ctx.get("error"))};
        return {res_ctx, freighter::NIL};
    }

private:
    /// GRPCPool to pool connections across clients.
    std::shared_ptr<Pool> pool;

    /// Middleware collector.
    freighter::MiddlewareCollector mw;

    /// Base target for all requests.
    freighter::URL base_target;

    /// Map from context instances to latest streams.
    std::unordered_map<
        int,
        std::unique_ptr<freighter::Stream<RQ, RS> >
    > latest_streams{};

    /// Largest id a context has at the moment.
    int largest_id = 0;

    /// Lock for mutual exclusion.
    std::mutex mut;
};
}
