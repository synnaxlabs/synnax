// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstddef>
#include <cstdlib>

#include "grpc/grpc.h"
#include "grpcpp/channel.h"
#include "grpcpp/client_context.h"
#include "grpcpp/security/credentials.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/fs/fs.h"

namespace freighter::grpc {
namespace priv {
const std::string PROTOCOL = "grpc";
const std::string ERROR_KEY = "error";
/// @brief the gRPC environment variable naming a PEM file of trust anchors.
const std::string ROOTS_FILE_ENV = "GRPC_DEFAULT_SSL_ROOTS_FILE_PATH";

/// @brief converts a ::grpc::Status to a x::errors::Error.
inline x::errors::Error err_from_status(const ::grpc::Status &status) {
    if (status.ok()) return x::errors::NIL;
    if (status.error_code() == ::grpc::StatusCode::UNAVAILABLE)
        return {UNREACHABLE.type, status.error_message()};
    if (status.error_code() == ::grpc::StatusCode::DEADLINE_EXCEEDED)
        return {TIMEOUT.type, status.error_message()};
    return x::errors::Error(status.error_message());
}
}

/// @brief builds TLS credentials that verify the server against the trust anchors in a
/// PEM file. A leaf certificate can be its own anchor.
/// @param ca_cert the path to the PEM file. When empty, the credentials use the file
/// named by GRPC_DEFAULT_SSL_ROOTS_FILE_PATH, or the system trust store when that is
/// unset.
/// @returns the credentials, or an error when the file at ca_cert cannot be read or
/// holds no certificates.
inline std::pair<std::shared_ptr<::grpc::ChannelCredentials>, x::errors::Error>
ssl_credentials(const std::string &ca_cert) {
    // Both files go in as explicit roots, because gRPC allows a leaf certificate as
    // the anchor only there, not in the store it builds from the variable.
    ::grpc::SslCredentialsOptions opts;
    if (!ca_cert.empty()) {
        auto [roots, err] = x::fs::read_file(ca_cert);
        if (err) return {nullptr, err};
        // gRPC reads empty roots as the system trust store.
        if (roots.find_first_not_of(" \t\r\n") == std::string::npos)
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "the CA certificate file " + ca_cert + " is empty"
                )
            };
        opts.pem_root_certs = std::move(roots);
    } else if (
        const char *path = std::getenv(priv::ROOTS_FILE_ENV.c_str()); path != nullptr
    )
        if (auto [roots, err] = x::fs::read_file(path); !err)
            opts.pem_root_certs = std::move(roots);
    return {::grpc::SslCredentials(opts), x::errors::NIL};
}

/// @brief how a channel finds a server that stops answering while calls run.
struct Keepalive {
    /// @brief how long the channel gets no bytes from the server before it pings. A
    /// Core closes a connection that pings more often than every five seconds.
    std::chrono::milliseconds time = std::chrono::seconds(10);
    /// @brief how long the channel waits for the answer to any ping, a keepalive ping
    /// or a flow control ping, before it closes the connection, which fails its calls
    /// with UNREACHABLE. With time, it finds a silent network loss in about 30 s. A
    /// ping waits behind queued data, so a slow uplink needs this margin.
    std::chrono::milliseconds timeout = std::chrono::seconds(20);
};

/// @brief the default Pool::unary_timeout.
const std::chrono::milliseconds UNARY_TIMEOUT = std::chrono::seconds(60);

class Pool {
    std::mutex mu;
    /// @brief A map of channels to targets.
    std::unordered_map<std::string, std::shared_ptr<::grpc::Channel>> channels;
    /// @brief GRPC credentials to provide when connecting to a target.
    std::shared_ptr<::grpc::ChannelCredentials> credentials;
    /// @brief the error that every call fails with, when the pool has no credentials.
    x::errors::Error err;
    Keepalive keepalive;

public:
    /// @brief how long a unary call waits for its response when its context sets no
    /// deadline. Each attempt of a retried call gets the full time.
    const std::chrono::milliseconds unary_timeout = UNARY_TIMEOUT;

    /// @brief instantiates a gRPC pool that connects with the provided credentials.
    explicit Pool(
        const std::shared_ptr<::grpc::ChannelCredentials> &credentials =
            ::grpc::InsecureChannelCredentials(),
        const Keepalive keepalive = {},
        const std::chrono::milliseconds unary_timeout = UNARY_TIMEOUT
    ):
        credentials(credentials), keepalive(keepalive), unary_timeout(unary_timeout) {}

    /// @brief instantiates a gRPC pool that fails every call with err. Use it when
    /// building the credentials fails.
    explicit Pool(const x::errors::Error &err): err(err) {}

    /// @brief returns the number of channels in the pool.
    size_t size() {
        std::lock_guard lock(this->mu);
        return this->channels.size();
    }

    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target, or the error the pool was built with.
    std::pair<std::shared_ptr<::grpc::Channel>, x::errors::Error>
    get_channel(const x::url::URL &target) {
        if (this->err) return {nullptr, this->err};
        std::lock_guard lock(this->mu);
        const auto host_addr = target.host_address();
        const auto it = this->channels.find(host_addr);
        if (it != this->channels.end()) {
            auto channel = it->second;
            if (channel->GetState(true) == GRPC_CHANNEL_TRANSIENT_FAILURE)
                this->channels.erase(host_addr);
            else
                return {channel, x::errors::NIL};
        }
        ::grpc::ChannelArguments args;
        args.SetInt(
            GRPC_ARG_KEEPALIVE_TIME_MS,
            static_cast<int>(this->keepalive.time.count())
        );
        args.SetInt(
            GRPC_ARG_KEEPALIVE_TIMEOUT_MS,
            static_cast<int>(this->keepalive.timeout.count())
        );
        // gRPC times out a keepalive ping with its ping timeout, which has no public
        // name.
        args.SetInt(
            "grpc.http2.ping_timeout_ms",
            static_cast<int>(this->keepalive.timeout.count())
        );
        // A stream that only receives sends no data, so its keepalive pings must not
        // wait for a budget of pings without data.
        args.SetInt(GRPC_ARG_HTTP2_MAX_PINGS_WITHOUT_DATA, 0);
        auto channel = CreateCustomChannel(host_addr, this->credentials, args);
        this->channels[host_addr] = channel;
        return {channel, x::errors::NIL};
    }
};

/// @brief An implementation of UnaryClient that uses gRPC as the backing transport.
/// Safe to be shared between threads.
/// @implements UnaryClient
/// @see UnaryClient
template<typename RQ, typename RS, typename RPC>
class UnaryClient final : public freighter::UnaryClient<RQ, RS>, Finalizer<RQ, RS> {
    /// Middleware collector.
    MiddlewareCollector<RQ, RS> mw;
    /// Pool to pool connections across clients.
    const std::shared_ptr<Pool> pool;
    /// Base target for all requests.
    const x::url::URL base_target;

public:
    UnaryClient(const std::shared_ptr<Pool> &pool, const std::string &base_target):
        pool(pool), base_target(x::url::URL(base_target)) {}

    explicit UnaryClient(const std::shared_ptr<Pool> &pool): pool(pool) {}

    /// @brief Adds a middleware to the chain.
    /// @implements UnaryClient::use
    void use(const std::shared_ptr<Middleware> middleware) override {
        this->mw.use(middleware);
    }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<RS, x::errors::Error>
    send(const std::string &target, RQ &request) override {
        Context ctx(priv::PROTOCOL, this->base_target.child(target), UNARY);
        return mw.exec(ctx, this, request);
    }

    /// @brief the finalizer that executes the request.
    FinalizerReturn<RS> operator()(Context req_ctx, RQ &req) override {
        // Set outbound metadata.
        ::grpc::ClientContext grpc_ctx;
        for (const auto &[k, v]: req_ctx.params)
            grpc_ctx.AddMetadata(k, v);
        grpc_ctx.set_deadline(req_ctx.deadline.value_or(
            std::chrono::system_clock::now() + this->pool->unary_timeout
        ));

        auto res = RS();
        auto res_ctx = Context(req_ctx.protocol, req_ctx.target, UNARY);
        auto [channel, ch_err] = this->pool->get_channel(req_ctx.target);
        if (ch_err) return {res_ctx, ch_err, res};
        auto stub = RPC::NewStub(channel);

        const auto stat = stub->Exec(&grpc_ctx, req, &res);
        if (!stat.ok()) return {res_ctx, priv::err_from_status(stat), res};

        // Set inbound metadata.
        for (const auto &[k, v]: grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
        return {res_ctx, x::errors::NIL, res};
    }
};

/// @brief Freighter stream object.
template<typename RQ, typename RS, typename RPC>
class Stream final
    : public freighter::Stream<RQ, RS>,
      Finalizer<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {

    MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> mw;

    /// @brief the underlying gRPC stream.
    std::unique_ptr<::grpc::ClientReaderWriter<RQ, RS>> stream;
    /// gRPC requires us to keep the context around so the stream doesn't die.
    ::grpc::ClientContext grpc_ctx;
    /// @brief the RPC stub used to instantiate the connection.
    const std::unique_ptr<typename RPC::Stub> stub;
    /// @brief stored to check connection state.
    std::shared_ptr<::grpc::Channel> channel;

    /// @brief set to true when the stream is closed.
    bool closed = false;
    /// @brief the error that the stream closed with.
    x::errors::Error close_err = x::errors::NIL;
    /// @brief set to true when writes_done is called.
    bool writes_done_called = false;

public:
    Stream(
        std::shared_ptr<::grpc::Channel> ch,
        const MiddlewareCollector<
            std::nullptr_t,
            std::unique_ptr<freighter::Stream<RQ, RS>>> &mw,
        Context &req_ctx,
        Context &res_ctx
    ):
        mw(mw), stub(RPC::NewStub(ch)), channel(ch) {
        for (const auto &[k, v]: req_ctx.params)
            this->grpc_ctx.AddMetadata(k, v);
        this->stream = this->stub->Exec(&this->grpc_ctx);
        this->stream->WaitForInitialMetadata();
        for (const auto &[k, v]: this->grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
    }

    /// @brief implements Stream::send.
    x::errors::Error send(RQ &request) const override {
        if (this->stream->Write(request)) return x::errors::NIL;
        return STREAM_CLOSED;
    }

    /// @brief implements Stream::receive.
    std::pair<RS, x::errors::Error> receive() override {
        RS res;
        if (this->stream->Read(&res)) return {res, x::errors::NIL};
        const auto ctx = Context(priv::PROTOCOL, x::url::URL(), STREAM);
        auto v = nullptr;
        const auto err = this->mw.exec(ctx, this, v).second;
        return {res, err};
    }

    /// @brief implements Stream::close_send.
    void close_send() override {
        if (this->writes_done_called) return;
        this->writes_done_called = true;
        auto state = this->channel->GetState(false);
        if (state == GRPC_CHANNEL_SHUTDOWN || state == GRPC_CHANNEL_TRANSIENT_FAILURE)
            return;
        this->stream->WritesDone();
    }

    FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>>
    operator()(Context outbound, std::nullptr_t &_) override {
        if (this->closed) return {outbound, this->close_err};
        const ::grpc::Status status = this->stream->Finish();
        this->closed = true;
        this->close_err = status.ok() ? ERR_EOF : priv::err_from_status(status);
        return {outbound, this->close_err, nullptr};
    }
};

/// @brief An implementation of StreamClient that uses gRPC as the backing transport.
/// Safe to be shared between threads.
/// @implements StreamClient
/// @see StreamClient
template<typename RQ, typename RS, typename RPC>
class StreamClient final
    : public freighter::StreamClient<RQ, RS>,
      Finalizer<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {
    /// Pool to pool connections across clients.
    const std::shared_ptr<Pool> pool;
    /// Base target for all requests.
    const x::url::URL base_target;
    /// Middleware collector.

    MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> mw;

public:
    StreamClient(const std::shared_ptr<Pool> &pool, const std::string &base_target):
        pool(pool), base_target(x::url::URL(base_target)) {}

    explicit StreamClient(const std::shared_ptr<Pool> &pool): pool(pool) {}

    /// @brief Adds a middleware to the chain.
    /// @implements StreamClient::use
    void use(std::shared_ptr<Middleware> middleware) override {
        this->mw.use(middleware);
    }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    /// NOTE: Sharing stream invocations is not thread safe. It is suggested to create
    /// one StreamClient and create a stream per thread.
    std::pair<std::unique_ptr<freighter::Stream<RQ, RS>>, x::errors::Error>
    stream(const std::string &target) override {
        auto ctx = Context(priv::PROTOCOL, this->base_target.child(target), STREAM);
        auto v = nullptr;
        auto [stream, err] = this->mw.exec(ctx, this, v);
        return {std::move(stream), err};
    }

    /// @brief the finalizer that opens the stream.
    FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>>
    operator()(Context req_ctx, std::nullptr_t &_) override {
        auto res_ctx = Context(req_ctx.protocol, req_ctx.target, STREAM);
        auto [channel, ch_err] = this->pool->get_channel(req_ctx.target);
        if (ch_err) return {res_ctx, ch_err};
        auto stream = std::make_unique<Stream<RQ, RS, RPC>>(
            channel,
            this->mw,
            req_ctx,
            res_ctx
        );
        if (res_ctx.has(priv::ERROR_KEY))
            return {res_ctx, x::errors::Error(res_ctx.get(priv::ERROR_KEY))};
        return {res_ctx, x::errors::NIL, std::move(stream)};
    }
};
}
