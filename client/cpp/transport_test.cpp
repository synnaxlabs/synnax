// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <future>
#include <thread>

#include "gtest/gtest.h"
#include <grpcpp/security/server_credentials.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include "client/cpp/transport.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/testutil/testutil.h"

#include "core/pkg/transport/grpc/connectivity/connectivity.grpc.pb.h"

namespace synnax::details {
/// @brief a connectivity service that answers no call until the call is cancelled.
class SilentConnectivity final
    : public grpc::connectivity::ConnectivityService::Service {
public:
    ::grpc::Status Exec(
        ::grpc::ServerContext *context,
        const google::protobuf::Empty *,
        grpc::connectivity::CheckResponse *
    ) override {
        while (!context->IsCancelled())
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        return ::grpc::Status::CANCELLED;
    }
};

/// @brief it should fail a unary request with TIMEOUT when the Core does not answer
/// before the transport timeout.
TEST(TransportTest, UnaryRequestTimesOut) {
    SilentConnectivity service;
    int port = 0;
    ::grpc::ServerBuilder builder;
    builder.AddListeningPort("127.0.0.1:0", ::grpc::InsecureServerCredentials(), &port);
    builder.RegisterService(&service);
    const auto server = builder.BuildAndStart();
    ASSERT_NE(server, nullptr);
    const Transport transport(
        static_cast<uint16_t>(port),
        "127.0.0.1",
        false,
        std::chrono::milliseconds(300)
    );
    auto sent = std::async(std::launch::async, [&transport] {
        google::protobuf::Empty req;
        return transport.connectivity_check->send("", req);
    });
    const auto status = sent.wait_for(std::chrono::seconds(10));
    // Cancels a call that the timeout did not end.
    server->Shutdown(std::chrono::system_clock::now());
    const auto err = sent.get().second;
    EXPECT_EQ(status, std::future_status::ready);
    ASSERT_OCCURRED_AS(err, freighter::TIMEOUT);
}
}
