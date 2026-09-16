// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "freighter/cpp/grpc/grpc.h"
#include "freighter/cpp/grpc/mock/freighter/cpp/grpc/mock/service.grpc.pb.h"
#include "freighter/cpp/grpc/mock/server.h"
#include "x/cpp/env/env.h"
#include "x/cpp/test/test.h"

namespace freighter::grpc {
using RQ = test::Message;
using RS = test::Message;
using UNARY_RPC = test::UnaryMessageService;

const std::string CERT_PATH = "freighter/cpp/grpc/mock/test_cert.pem";
const std::string KEY_PATH = "freighter/cpp/grpc/mock/test_key.pem";
const std::string TLS_TARGET = "localhost:8083";
const std::string PLAIN_TARGET = "localhost:8084";

test::Message message() {
    test::Message m;
    m.set_payload("Sending to Server");
    return m;
}

const std::string EXPECTED = "Read request: Sending to Server";

/// @brief a secure pool completes a request against a TLS server whose certificate is
/// in the configured root store.
TEST(testTLS, secureClientReachesTLSServer) {
    std::thread s(mock::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    mock::wait_for_servers();
    const auto pool = std::make_shared<Pool>(true);
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, TLS_TARGET);
    auto req = message();
    const auto res = ASSERT_NIL_P(client.send("", req));
    ASSERT_EQ(res.payload(), EXPECTED);
    mock::stop_servers();
    s.join();
}

/// @brief a plaintext pool cannot reach a TLS server.
TEST(testTLS, insecureClientFailsAgainstTLSServer) {
    std::thread s(mock::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    mock::wait_for_servers();
    const auto pool = std::make_shared<Pool>();
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, TLS_TARGET);
    auto req = message();
    ASSERT_OCCURRED_AS_P(client.send("", req), UNREACHABLE);
    mock::stop_servers();
    s.join();
}

/// @brief a secure pool cannot reach a plaintext server.
TEST(testTLS, secureClientFailsAgainstPlaintextServer) {
    std::thread s(mock::server, PLAIN_TARGET);
    mock::wait_for_servers();
    const auto pool = std::make_shared<Pool>(true);
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, PLAIN_TARGET);
    auto req = message();
    ASSERT_OCCURRED_AS_P(client.send("", req), UNREACHABLE);
    mock::stop_servers();
    s.join();
}
}

int main(int argc, char **argv) {
    // gRPC loads its root store once per process, so the override must be in place
    // before the first channel is created.
    x::env::set("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH", freighter::grpc::CERT_PATH);
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
