// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"
#include "grpc/grpc.h"

#include "grpcpp/channel.h"
#include "grpcpp/client_context.h"
#include "grpcpp/security/credentials.h"

/// internal.
#include "freighter/cpp/freighter.h"
#include "x/cpp/fs/fs.h"

namespace priv {
const std::string PROTOCOL = "grpc";
const std::string ERROR_KEY = "error";

/// @brief converts a grpc::Status to a xerrors::Error.
inline xerrors::Error err_from_status(const grpc::Status &status) {
    if (status.ok()) return xerrors::NIL;
    if (status.error_code() == grpc::StatusCode::UNAVAILABLE)
        return {freighter::UNREACHABLE.type, status.error_message()};
    return xerrors::Error(status.error_message());
}
}

namespace fgrpc {
class Pool {
    std::mutex mu;
    /// @brief A map of channels to targets.
    std::unordered_map<std::string, std::shared_ptr<grpc::Channel>> channels{};
    /// @brief GRPC credentials to provide when connecting to a target.
    std::shared_ptr<grpc::ChannelCredentials>
        credentials = grpc::InsecureChannelCredentials();

public:
    Pool() = default;

    /// @brief returns the number of channels in the pool.
    size_t size() {
        std::lock_guard lock(this->mu);
        return this->channels.size();
    }

    /// @brief Instantiates the GRPC pool to use TLS encryption where the CA
    /// certificate is located at the provided path.
    explicit Pool(const std::string &ca_path) {
        grpc::SslCredentialsOptions opts;
        auto [pem_root_certs, err] = fs::read_file(ca_path);
        if (err)
            LOG(ERROR) << "Failed to read CA certificate from " << ca_path << ": "
                       << err.message();
        opts.pem_root_certs = pem_root_certs;
        credentials = SslCredentials(opts);
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
        bool secure = false;
        if (!ca_path.empty()) {
            auto [pem_root_certs, err] = fs::read_file(ca_path);
            if (err) { LOG(ERROR) << "Failed to read CA certificate: " << err; }
            opts.pem_root_certs = pem_root_certs;
            secure = true;
        }
        if (!cert_path.empty() && !key_path.empty()) {
            auto [pem_cert_chain, err] = fs::read_file(cert_path);
            if (err) LOG(ERROR) << "Failed to read client certificate: " << err;
            opts.pem_cert_chain = pem_cert_chain;
            auto [pem_private_key, pem_priv_key_err] = fs::read_file(key_path);
            if (pem_priv_key_err)
                LOG(ERROR) << "Failed to read client private key from " << err;
            opts.pem_private_key = pem_private_key;
            secure = true;
        }
        if (secure) credentials = SslCredentials(opts);
    }

    /// @brief instantiates a GRPC pool with the provided credentials.
    explicit Pool(const std::shared_ptr<grpc::ChannelCredentials> &credentials):
        credentials(credentials) {}

    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target.
    std::shared_ptr<grpc::Channel> get_channel(const freighter::URL &target) {
        std::lock_guard lock(this->mu);
        const auto host_addr = target.host_address();
        const auto it = this->channels.find(host_addr);
        if (it != this->channels.end()) {
            auto channel = it->second;
            if (channel->GetState(true) == GRPC_CHANNEL_TRANSIENT_FAILURE)
                this->channels.erase(host_addr);
            else
                return channel;
        }
        const grpc::ChannelArguments args;
        auto channel = CreateCustomChannel(host_addr, this->credentials, args);
        this->channels[host_addr] = channel;
        return channel;
    }
};

/// @brief An implementation of freighter::UnaryClient that uses GRPC as the backing
/// transport. Safe to be shared between threads.
/// @implements freighter::UnaryClient
/// @see freighter::UnaryClient
template<typename RQ, typename RS, typename RPC>
class UnaryClient final : public freighter::UnaryClient<RQ, RS>,
                          freighter::Finalizer<RQ, RS> {
    /// Middleware collector.
    freighter::MiddlewareCollector<RQ, RS> mw;
    /// GRPCPool to pool connections across clients.
    const std::shared_ptr<Pool> pool;
    /// Base target for all requests.
    const freighter::URL base_target;

public:
    UnaryClient(const std::shared_ptr<Pool> &pool, const std::string &base_target):
        pool(pool), base_target(freighter::URL(base_target)) {}

    explicit UnaryClient(const std::shared_ptr<Pool> &pool): pool(pool) {}

    /// @brief Adds a middleware to the chain.
    /// @implements UnaryClient::use
    void use(const std::shared_ptr<freighter::Middleware> middleware) override {
        this->mw.use(middleware);
    }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<RS, xerrors::Error>
    send(const std::string &target, RQ &request) override {
        freighter::Context ctx(
            priv::PROTOCOL,
            this->base_target.child(target),
            freighter::UNARY
        );
        return mw.exec(ctx, this, request);
    }

    /// @brief the finalizer that executes the request.
    freighter::FinalizerReturn<RS>
    operator()(freighter::Context req_ctx, RQ &req) override {
        // Set outbound metadata.
        grpc::ClientContext grpc_ctx;
        for (const auto &[k, v]: req_ctx.params)
            grpc_ctx.AddMetadata(k, v);

        // Execute request.
        auto channel = this->pool->get_channel(req_ctx.target);
        auto stub = RPC::NewStub(channel);
        auto res = RS();

        const auto stat = stub->Exec(&grpc_ctx, req, &res);
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::UNARY
        );
        if (!stat.ok()) return {res_ctx, priv::err_from_status(stat), res};

        // Set inbound metadata.
        for (const auto &[k, v]: grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
        return {res_ctx, xerrors::NIL, res};
    }
};

/// @brief freighter stream object.
template<typename RQ, typename RS, typename RPC>
class Stream final
    : public freighter::Stream<RQ, RS>,
      freighter::Finalizer<nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {
    freighter::
        MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>>
            mw;

    /// @brief the underlying grpc stream.
    std::unique_ptr<grpc::ClientReaderWriter<RQ, RS>> stream;
    /// GRPC requires us to keep these around so the stream doesn't die.
    grpc::ClientContext grpc_ctx;
    /// @brief the RPC stub used to instantiate the connection.
    const std::unique_ptr<typename RPC::Stub> stub;

    /// @brief set to true when the stream is closed.
    bool closed = false;
    /// @brief the error that the stream closed with.
    xerrors::Error close_err = xerrors::NIL;
    /// @brief set to true when writes_done is called.
    bool writes_done_called = false;

public:
    Stream(
        std::shared_ptr<grpc::Channel> ch,
        const freighter::MiddlewareCollector<
            std::nullptr_t,
            std::unique_ptr<freighter::Stream<RQ, RS>>> &mw,
        freighter::Context &req_ctx,
        freighter::Context &res_ctx
    ):
        mw(mw), stub(RPC::NewStub(ch)) {
        for (const auto &[k, v]: req_ctx.params)
            this->grpc_ctx.AddMetadata(k, v);
        this->stream = this->stub->Exec(&this->grpc_ctx);
        this->stream->WaitForInitialMetadata();
        for (const auto &[k, v]: this->grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
    }

    /// @brief implements Stream::send.
    xerrors::Error send(RQ &request) const override {
        if (this->stream->Write(request)) return xerrors::NIL;
        return freighter::STREAM_CLOSED;
    }

    /// @brief implements Stream::receive.
    std::pair<RS, xerrors::Error> receive() override {
        RS res;
        if (this->stream->Read(&res)) return {res, xerrors::NIL};
        const auto ctx = freighter::Context(
            priv::PROTOCOL,
            freighter::URL(),
            freighter::STREAM
        );
        auto v = nullptr;
        const auto err = this->mw.exec(ctx, this, v).second;
        return {res, err};
    }

    /// @brief implements Stream::close_send.
    void close_send() override {
        if (this->writes_done_called) return;
        this->stream->WritesDone();
        this->writes_done_called = true;
    }

    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>>
    operator()(freighter::Context outbound, std::nullptr_t &_) override {
        if (this->closed) return {outbound, this->close_err};
        const grpc::Status status = this->stream->Finish();
        this->closed = true;
        this->close_err = status.ok() ? freighter::EOF_ERR
                                      : priv::err_from_status(status);
        return {outbound, this->close_err, nullptr};
    }
};

/// @brief An implementation of freighter::StreamClient that uses GRPC as the
/// backing transport. Safe to be shared between threads.
/// @implements freighter::StreamClient
/// @see freighter::StreamClient
template<typename RQ, typename RS, typename RPC>
class StreamClient final
    : public freighter::StreamClient<RQ, RS>,
      freighter::Finalizer<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {
    /// GRPCPool to pool connections across clients.
    const std::shared_ptr<Pool> pool;
    /// Base target for all requests.
    const freighter::URL base_target;
    /// Middleware collector.
    freighter::
        MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>>
            mw;

public:
    StreamClient(const std::shared_ptr<Pool> &pool, const std::string &base_target):
        pool(pool), base_target(freighter::URL(base_target)) {}

    explicit StreamClient(const std::shared_ptr<Pool> &pool): pool(pool) {}

    /// @brief Adds a middleware to the chain.
    /// @implements StreamClient::use
    void use(std::shared_ptr<freighter::Middleware> middleware) override {
        this->mw.use(middleware);
    }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    /// NOTE: Sharing stream invocations is not thread safe.
    /// It is suggested to create one StreamClient and create a stream per thread.
    std::pair<std::unique_ptr<freighter::Stream<RQ, RS>>, xerrors::Error>
    stream(const std::string &target) override {
        auto ctx = freighter::Context(
            priv::PROTOCOL,
            this->base_target.child(target),
            freighter::STREAM
        );
        auto v = nullptr;
        auto [stream, err] = this->mw.exec(ctx, this, v);
        return {std::move(stream), err};
    }

    /// @brief the finalizer that opens the stream.
    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>>
    operator()(freighter::Context req_ctx, std::nullptr_t &_) override {
        auto channel = this->pool->get_channel(req_ctx.target);
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::STREAM
        );
        auto stream = std::make_unique<Stream<RQ, RS, RPC>>(
            channel,
            this->mw,
            req_ctx,
            res_ctx
        );
        if (res_ctx.has(priv::ERROR_KEY))
            return {res_ctx, xerrors::Error(res_ctx.get(priv::ERROR_KEY))};
        return {res_ctx, xerrors::NIL, std::move(stream)};
    }
};
}
