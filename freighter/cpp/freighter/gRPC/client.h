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

// TODO: Create internal error types!!

class GRPCPool {
private:
    /// @brief A map of channels to targets.
    std::map<std::string, std::shared_ptr<grpc::Channel>> channels;
public:
    /// @brief Get a channel for a given target.
    /// @param target The target to connect to.
    /// @returns A channel to the target.
    std::shared_ptr<grpc::Channel> getChannel(const std::string &target) {
        if (channels.find(target) == channels.end())
            channels[target] = grpc::CreateChannel(target, grpc::InsecureChannelCredentials());
        return channels[target];
    }

    ~GRPCPool() {
        for (auto &channel: channels) {
            channel.second.reset();
        }
    }
};

/// @brief freighter stream object.
template<typename response_t, typename request_t, typename err_t, typename rpc_t>
class GRPCStream : public Freighter::Stream<response_t, request_t, err_t> {
public:

    /// @brief Ctor saves GRPCUnaryClient stream object to use under the hood.
    explicit GRPCStream(std::shared_ptr<grpc::Channel> channel) {
        // Note that the streamer also sets up its own internal stub.
        stub = rpc_t::NewStub(channel);
        stream = stub->Streamer(&context);
    }

    /// @brief Streamer send.
    err_t send(request_t &request) override {
        // TODO: Expand on the returned statuses.
        if (stream->Write(request)) {
            return grpc::Status::OK;
        };
        return grpc::Status::CANCELLED;
    }

    /// @brief Streamer read.
    std::pair<response_t, err_t> receive() override {
        response_t res;
        if (stream->Read(&res)) {
            return {res, grpc::Status::OK};
        };
        return {res, grpc::Status::CANCELLED};
    }

    /// @brief Closing streamer.
    err_t closeSend() override {
        if (stream->WritesDone()) {
            return grpc::Status();
        }

        return grpc::Status::CANCELLED;
    }

private:
    /// The internal streaming type for GRPCUnaryClient.
    std::unique_ptr<grpc::ClientReaderWriter<response_t, request_t>> stream;

    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;

    /// Each streamer needs to manage its own context.
    grpc::ClientContext context;

    /// Last target managed.
    std::string last_target;
};

/// @brief GRPCUnaryClient specific class
template<typename response_t, typename request_t, typename err_t, typename rpc_t>
class GRPCUnaryClient :
        public Freighter::UnaryClient<response_t, request_t, err_t>,
        Freighter::BaseMiddleware {
    request_t latest_request;
    response_t latest_response;
    err_t latest_err;
public:
    GRPCUnaryClient(
            GRPCPool *pool,
            const std::string &base_target
    ) : pool(pool),
        base_target(Freighter::URL(base_target)) {
    }

    void use(Freighter::Middleware *middleware) override {
        mw.use(middleware);
    }


    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    std::pair<response_t, err_t> send(const std::string &target, request_t &request) override {
        Freighter::Context ctx = Freighter::Context("grpc", base_target.child(target).toString());
        auto [_, exc] = mw.exec(ctx, this);
        if (exc != nullptr) throw *exc;
        return {latest_response, latest_err};
    }

    virtual std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context &context) {
        grpc::ClientContext ctx;
        auto target = context.target;
        auto channel = pool->getChannel(base_target.child(target).toString());
        stub = rpc_t::NewStub(channel);
        auto stat = stub->Unary(&ctx, latest_request, &latest_response);
        latest_err = stat;
        return {context, nullptr};
    }


private:
    Freighter::MiddlewareCollector mw;

    /// GRPCPool to pool connections across clients.
    GRPCPool *pool;

    /// base target.
    Freighter::URL base_target;

    /// Stub to manage connection.
    std::unique_ptr<typename rpc_t::Stub> stub;
};

template<typename response_t, typename request_t, typename err_t, typename rpc_t>
class GRPCStreamClient :
        public Freighter::StreamClient<response_t, request_t, err_t>,
        Freighter::BaseMiddleware {
private:
    Freighter::MiddlewareCollector mw;
    GRPCPool *pool;
    Freighter::URL base_target;
    GRPCStream<response_t, request_t, err_t, rpc_t> *latest_stream;
public:
    GRPCStreamClient(
            GRPCPool *pool,
            const std::string &base_target
    ) : pool(pool),
        base_target(Freighter::URL(base_target)) {
    }

    void use(Freighter::Middleware *middleware) override {
        mw.use(middleware);
    }

    /// @brief Interface for stream.
    /// @param target The server's IP.
    /// @returns A stream object, which can be used to listen to the server.
    Freighter::Stream<response_t, request_t, err_t> *stream(const std::string &target) override {
        Freighter::Context ctx = Freighter::Context("grpc", base_target.child(target).toString());
        auto [_, exc] = mw.exec(ctx, this);
        if (exc != nullptr) throw *exc;
        return latest_stream;
    }

    std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context &context) {
        auto target = context.target;
        auto channel = pool->getChannel(base_target.child(target).toString());
        latest_stream = new GRPCStream<response_t, request_t, err_t, rpc_t>(channel);
        return {context, nullptr};
    }
};