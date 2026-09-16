// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <fstream>
#include <iostream>
#include <mutex>
#include <sstream>
#include <string>

#include <grpc/grpc.h>
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include "freighter/cpp/grpc/mock/freighter/cpp/grpc/mock/service.grpc.pb.h"

namespace freighter::grpc::mock {
/// @brief Used to awake main thread when we are
/// done processing messages.
inline std::mutex mut;
inline std::condition_variable cond;
inline bool end_session = false;

/// @brief Implements .proto generated interface Unary.
class unaryServiceImpl final : public test::UnaryMessageService::Service {
public:
    /// @brief The implementation on the server side of unary communication.
    ::grpc::Status Exec(
        ::grpc::ServerContext *context,
        const test::Message *request,
        test::Message *reply
    ) override {
        // get the key 'test' from metadata
        const auto test = context->client_metadata().find("test");
        const std::string rep("Read request: ");
        // if the test value exists, set the reply key back to the same value.
        if (test != context->client_metadata().end()) {
            context->AddInitialMetadata("test", "dog");
        }
        reply->set_payload(rep + request->payload());
        return ::grpc::Status::OK;
    }
};

class myStreamServiceImpl final : public test::StreamMessageService::Service {
    /// @brief The implementation of the server side stream.
    ::grpc::Status Exec(
        ::grpc::ServerContext *context,
        ::grpc::ServerReaderWriter<test::Message, test::Message> *stream
    ) override {
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

        return ::grpc::Status::OK;
    }
};

/// @brief serves the unary and stream services on target with the given credentials
/// until stop_servers is called. Meant to run in its own thread.
inline void serve(
    const std::string &target,
    const std::shared_ptr<::grpc::ServerCredentials> &credentials
) {
    end_session = false;
    unaryServiceImpl u_service;
    myStreamServiceImpl s_service;

    ::grpc::ServerBuilder builder;
    builder.AddListeningPort(target, credentials);
    builder.RegisterService(&u_service);
    builder.RegisterService(&s_service);

    std::unique_ptr<::grpc::Server> server(builder.BuildAndStart());

    std::unique_lock<std::mutex> lck(mut);
    while (!end_session) {
        cond.wait(lck);
    }
    lck.unlock();
    server->Shutdown();
    end_session = false;
}

/// @brief serves in plaintext on target until stop_servers is called.
inline void server(const std::string &target) {
    serve(target, ::grpc::InsecureServerCredentials());
}

inline std::string read_file(const std::string &path) {
    std::ifstream file(path);
    std::stringstream buf;
    buf << file.rdbuf();
    return buf.str();
}

/// @brief serves over TLS on target with the PEM certificate and key at the given
/// paths until stop_servers is called.
inline void tls_server(
    const std::string &target,
    const std::string &cert_path,
    const std::string &key_path
) {
    ::grpc::SslServerCredentialsOptions opts;
    opts.pem_key_cert_pairs.push_back({read_file(key_path), read_file(cert_path)});
    serve(target, ::grpc::SslServerCredentials(opts));
}

/// @brief Abstraction of stopping servers.
inline void stop_servers() {
    end_session = true;
    cond.notify_all();
}
}
