// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external.
#include "grpc/grpc.h"
#include "grpcpp/channel.h"
#include "grpcpp/client_context.h"
#include "grpcpp/security/credentials.h"
#include "glog/logging.h"

/// internal.
#include "freighter/cpp/freighter.h"


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

/// @brief an internal method for reading the entire contents of certificate files
/// into a string.
inline std::string read_file(const std::string &path) {
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
    std::mutex mu;
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
        opts.pem_root_certs = priv::read_file(ca_path);
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
        opts.pem_root_certs = priv::read_file(ca_path);
        opts.pem_cert_chain = priv::read_file(cert_path);
        opts.pem_private_key = priv::read_file(key_path);
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
    std::shared_ptr<grpc::Channel> get_channel(const std::string &target) {
        std::lock_guard lock(this->mu);
        const auto it = this->channels.find(target);
        if (it != this->channels.end()) {
            auto channel = it->second;
            if (channel->GetState(true) == GRPC_CHANNEL_TRANSIENT_FAILURE)
                this->channels.erase(target);
            else return channel;
        }
        const grpc::ChannelArguments args;
        auto channel = grpc::CreateCustomChannel(target, this->credentials, args);
        this->channels[target] = channel;
        return channel;
    }
};


/// @brief An implementation of freighter::UnaryClient that uses GRPC as the backing transport. Safe to be shared between threads.
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
        this->mw.use(middleware);
    }

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<RS, xerrors::Error> send(
        const std::string &target,
        RQ &request
    ) override {
        freighter::Context ctx(
            priv::PROTOCOL,
            this->base_target.child(target).to_string(),
            freighter::UNARY
        );
        return mw.exec(ctx, this, request);
    }

    /// @brief the finalizer that executes the request.
    freighter::FinalizerReturn<RS> operator()(
        freighter::Context req_ctx,
        RQ &req
    ) override {
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
class Stream final :
        public freighter::Stream<RQ, RS>,
        freighter::Finalizer<nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS> > > {

    freighter::MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ,
        RS> > > mw;

    /// @brief the underlying grpc stream.
    std::unique_ptr<grpc::ClientReaderWriter<RQ, RS> > stream;
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
        const freighter::MiddlewareCollector<std::nullptr_t, std::unique_ptr<
            freighter::Stream<RQ, RS> > > &mw,
        freighter::Context &req_ctx,
        freighter::Context &res_ctx
    ) : mw(mw), stub(RPC::NewStub(ch)) {
        for (const auto &[k, v]: req_ctx.params)
            this->grpc_ctx.AddMetadata(k, v);
        this->stream = this->stub->Exec(&this->grpc_ctx);
        this->stream->WaitForInitialMetadata();
        for (const auto &[k, v]: this->grpc_ctx.GetServerInitialMetadata())
            res_ctx.set(k.data(), v.data());
    }

    /// @brief implements Stream::send.
    xerrors::Error send(RQ &request) override {
        if (this->stream->Write(request)) return xerrors::NIL;
        return freighter::STREAM_CLOSED;
    }

    /// @brief implements Stream::receive.
    std::pair<RS, xerrors::Error> receive() override {
        RS res;
        if (this->stream->Read(&res)) return {res, xerrors::NIL};
        const auto ctx = freighter::Context(priv::PROTOCOL, "", freighter::STREAM);
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

    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS> > > operator()(
        freighter::Context outbound,
        std::nullptr_t &_
    ) override {
        if (this->closed) return {outbound, this->close_err};
        const grpc::Status status = this->stream->Finish();
        this->closed = true;
        this->close_err = status.ok() ? freighter::EOF_ : priv::err_from_status(status);
        return {outbound, this->close_err, nullptr};
    }
};

/// @brief An implementation of freighter::StreamClient that uses GRPC as the backing
/// transport. Safe to be shared between threads.
/// @implements freighter::StreamClient
/// @see freighter::StreamClient
template<typename RQ, typename RS, typename RPC>
class StreamClient final : public freighter::StreamClient<RQ, RS>,
                           freighter::Finalizer<std::nullptr_t, std::unique_ptr<
                               freighter::Stream<RQ, RS> > > {
    /// GRPCPool to pool connections across clients.
    const std::shared_ptr<Pool> pool;
    /// Base target for all requests.
    const freighter::URL base_target;
    /// Middleware collector.
    freighter::MiddlewareCollector<
        std::nullptr_t,
        std::unique_ptr<freighter::Stream<RQ, RS> >
    > mw;
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
        this->mw.use(middleware);
    }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    /// NOTE: Sharing stream invocations is not thread safe.
    /// It is suggested to create one StreamClient and create a stream per thread.
    std::pair<std::unique_ptr<freighter::Stream<RQ, RS> >, xerrors::Error>
    stream(const std::string &target) override {
        auto ctx = freighter::Context(
            priv::PROTOCOL,
            this->base_target.child(target).to_string(),
            freighter::STREAM
        );
        auto v = nullptr;
        auto [stream, err] = this->mw.exec(ctx, this, v);
        return {std::move(stream), err};
    }

    /// @brief the finalizer that opens the stream.
    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS> > > operator()(
        freighter::Context req_ctx,
        std::nullptr_t &_
    ) override {
        auto channel = this->pool->get_channel(req_ctx.target);
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::STREAM
        );
        auto stream = std::make_unique<Stream<RQ, RS, RPC> >(
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

/// @brief Async implementation of the Stream class for GRPC
template<typename RQ, typename RS, typename RPC>
class AsyncStream final : public freighter::Stream<RQ, RS>,
                        freighter::Finalizer<nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {
private:
    std::unique_ptr<typename RPC::Stub> stub;
    grpc::ClientContext context_;
    std::unique_ptr<grpc::ClientAsyncReaderWriter<RQ, RS>> stream_;
    grpc::CompletionQueue cq_;
    freighter::MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> mw;
    bool is_connected_ = false;
    bool closed_ = false;
    xerrors::Error close_err = xerrors::NIL;
    bool writes_done_called = false;
    mutable std::atomic<void*> last_write_tag_ = nullptr;

    xerrors::Error wait_for_pending_write() {
        void* got_tag;
        bool ok = false;
        while (last_write_tag_.load() != nullptr) {
            if (!cq_.Next(&got_tag, &ok) || !ok) {
                return freighter::STREAM_CLOSED;
            }
            last_write_tag_.store(nullptr);
        }
        return xerrors::NIL;
    }

public:
    AsyncStream(
        std::shared_ptr<grpc::Channel> channel,
        const freighter::MiddlewareCollector<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>>& mw,
        freighter::Context& req_ctx,
        freighter::Context& res_ctx
    ) : stub(RPC::NewStub(channel)), mw(mw) {
        // Set up the context with metadata
        for (const auto& [k, v] : req_ctx.params) {
            context_.AddMetadata(k, v);
        }

        // Start the stream
        stream_ = stub->AsyncExec(&context_, &cq_, (void*)this);

        // Wait for stream to be ready
        void* got_tag;
        bool ok = false;
        if (cq_.Next(&got_tag, &ok) && ok) {
            is_connected_ = true;
            // Wait for initial metadata
            stream_->ReadInitialMetadata((void*)1);
            if (cq_.Next(&got_tag, &ok) && ok) {
                // Now it's safe to get initial metadata
                auto initial_metadata = context_.GetServerInitialMetadata();
                for (const auto& [k, v] : initial_metadata) {
                    res_ctx.set(k.data(), v.data());
                }
            }
        }
    }

    xerrors::Error send(RQ& request) override {
        if (!is_connected_ || closed_) return freighter::STREAM_CLOSED;

        auto err = wait_for_pending_write();
        if (err != xerrors::NIL) return err;

        void* current_tag = (void*)(&request);
        stream_->Write(request, current_tag);
        last_write_tag_.store(current_tag);

        return xerrors::NIL;
    }

    std::pair<RS, xerrors::Error> receive() override {
        if (!is_connected_ || closed_) {
            return {RS(), freighter::STREAM_CLOSED};
        }

        auto err = wait_for_pending_write();
        if (err != xerrors::NIL) {
            return {RS(), err};
        }

        RS response;
        void* tag = (void*)2;
        stream_->Read(&response, tag);

        void* got_tag;
        bool ok = false;
        if (!cq_.Next(&got_tag, &ok) || !ok) {
            const auto ctx = freighter::Context(priv::PROTOCOL, "", freighter::STREAM);
            auto v = nullptr;
            const auto err = this->mw.exec(ctx, this, v).second;
            return {response, err};
        }
        return {response, xerrors::NIL};
    }

    void close_send() override {
        if (writes_done_called) return;

        auto err = wait_for_pending_write();
        if (err != xerrors::NIL) return;

        void* tag = (void*)3;
        stream_->WritesDone(tag);
        
        void* got_tag;
        bool ok = false;
        cq_.Next(&got_tag, &ok);
        writes_done_called = true;
    }

    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>> operator()(
        freighter::Context outbound,
        std::nullptr_t& _
    ) override {
        if (closed_) return {outbound, close_err};
        
        grpc::Status status;
        void* tag = (void*)4;
        stream_->Finish(&status, tag);
        
        void* got_tag;
        bool ok = false;
        cq_.Next(&got_tag, &ok);
        
        closed_ = true;
        close_err = status.ok() ? freighter::EOF_ : priv::err_from_status(status);
        return {outbound, close_err, nullptr};
    }
};

/// @brief Async implementation of the StreamClient
template<typename RQ, typename RS, typename RPC>
class AsyncStreamClient final : public freighter::StreamClient<RQ, RS>,
                              freighter::Finalizer<std::nullptr_t, std::unique_ptr<freighter::Stream<RQ, RS>>> {
private:
    const std::shared_ptr<Pool> pool;
    const freighter::URL base_target;
    freighter::MiddlewareCollector<
        std::nullptr_t,
        std::unique_ptr<freighter::Stream<RQ, RS>>
    > mw;

public:
    AsyncStreamClient(
        const std::shared_ptr<Pool>& pool,
        const std::string& base_target
    ) : pool(pool), base_target(freighter::URL(base_target)) {}

    explicit AsyncStreamClient(const std::shared_ptr<Pool>& pool) : pool(pool) {}

    void use(std::shared_ptr<freighter::Middleware> middleware) override {
        this->mw.use(middleware);
    }

    std::pair<std::unique_ptr<freighter::Stream<RQ, RS>>, xerrors::Error>
    stream(const std::string& target) override {
        auto ctx = freighter::Context(
            priv::PROTOCOL,
            this->base_target.child(target).to_string(),
            freighter::STREAM
        );
        auto v = nullptr;
        auto [stream, err] = this->mw.exec(ctx, this, v);
        return {std::move(stream), err};
    }

    freighter::FinalizerReturn<std::unique_ptr<freighter::Stream<RQ, RS>>> operator()(
        freighter::Context req_ctx,
        std::nullptr_t& _
    ) override {
        auto channel = this->pool->get_channel(req_ctx.target);
        auto res_ctx = freighter::Context(
            req_ctx.protocol,
            req_ctx.target,
            freighter::STREAM
        );

        auto stream = std::make_unique<AsyncStream<RQ, RS, RPC>>(
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
