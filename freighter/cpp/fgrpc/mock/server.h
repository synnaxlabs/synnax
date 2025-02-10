// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <grpc/grpc.h>
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>
#include <string>
#include <iostream>
#include <mutex>
#include <condition_variable>
#include "freighter/cpp/fgrpc/mock/freighter/cpp/fgrpc/mock/service.grpc.pb.h"

/// @brief Used to awake main thread when we are
/// done processing messages.
std::mutex mut;
std::condition_variable cond;
bool end_session = false;

/// @brief Implements .proto generated interface Unary.
class unaryServiceImpl final : public test::UnaryMessageService::Service {
public:
    /// @brief The implementation on the server side of unary communication.
    grpc::Status Exec(grpc::ServerContext *context, const test::Message *request,
                      test::Message *reply) override {
        // get the key 'test' from metadata
        auto test = context->client_metadata().find("test");
        std::string rep("Read request: ");
        // if the test value exists, set the reply key back to the same value.
        if (test != context->client_metadata().end()) {
            context->AddInitialMetadata("test", "dog");
        }
        reply->set_payload(rep + request->payload());
        return grpc::Status::OK;
    }

private:
};

class myStreamServiceImpl final : public test::StreamMessageService::Service {
    /// @brief The implementation of the server side stream.
    grpc::Status
    Exec(grpc::ServerContext *context,
         grpc::ServerReaderWriter<test::Message, test::Message> *stream) override {
        // Send initial metadata
        context->AddInitialMetadata("test", "dog");
        stream->SendInitialMetadata();
        test::Message request;
        while (stream->Read(&request)) {
            std::unique_lock<std::mutex> lock(mut);
            test::Message res;
            std::string rep("Read request: ");
            res.set_payload(rep + request.payload());
            stream->Write(res);
        }

        return grpc::Status::OK;
    }
};


/// @brief Meant to be call within a thread. Simple
/// GRPCUnaryClient server.
void server(std::string target) {
    end_session = false;
    std::string server_address(target);
    unaryServiceImpl u_service;
    myStreamServiceImpl s_service;

    grpc::ServerBuilder builder;
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&u_service);
    builder.RegisterService(&s_service);

    std::unique_ptr<grpc::Server> server(builder.BuildAndStart());

    std::unique_lock<std::mutex> lck(mut);
    while (!end_session) {
        cond.wait(lck);
    }
    lck.unlock();
    server->Shutdown();
    end_session = false;
}

/// @brief Abstraction of stopping servers.
void stopServers() {
    end_session = true;
    cond.notify_all();
}
