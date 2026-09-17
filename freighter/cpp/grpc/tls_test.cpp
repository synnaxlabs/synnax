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

/// A self-signed certificate that is its own CA.
const std::string CERT_PATH = "freighter/cpp/grpc/mock/test_cert.pem";
const std::string KEY_PATH = "freighter/cpp/grpc/mock/test_key.pem";
/// A leaf certificate signed by test_ca_cert.pem.
const std::string LEAF_CERT_PATH = "freighter/cpp/grpc/mock/test_leaf_cert.pem";
const std::string LEAF_KEY_PATH = "freighter/cpp/grpc/mock/test_leaf_key.pem";
const std::string CA_CERT_PATH = "freighter/cpp/grpc/mock/test_ca_cert.pem";
const std::string TLS_TARGET = "localhost:8083";
const std::string PLAIN_TARGET = "localhost:8084";

test::Message message() {
    test::Message m;
    m.set_payload("Sending to Server");
    return m;
}

const std::string EXPECTED = "Read request: Sending to Server";

/// @brief sends one request over a secure pool that trusts the given PEM file.
std::pair<RS, x::errors::Error> send_trusting(const std::string &roots_path) {
    x::env::set(priv::ROOTS_FILE_ENV, roots_path);
    const auto pool = std::make_shared<Pool>(true);
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, TLS_TARGET);
    auto req = message();
    return client.send("", req);
}

/// @brief a secure pool completes a request against a TLS server whose self-signed
/// certificate is the trust anchor.
TEST(testTLS, secureClientReachesTLSServer) {
    std::thread s(mock::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    mock::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    mock::stop_servers();
    s.join();
}

/// @brief a secure pool trusts a leaf certificate directly, without its issuing CA.
TEST(testTLS, leafCertificateIsTrustAnchor) {
    std::thread s(mock::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    mock::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(LEAF_CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    mock::stop_servers();
    s.join();
}

/// @brief a secure pool trusts a leaf certificate through its issuing CA.
TEST(testTLS, issuingCAIsTrustAnchor) {
    std::thread s(mock::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    mock::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(CA_CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    mock::stop_servers();
    s.join();
}

/// @brief a secure pool rejects a server whose certificate chains to no trust anchor.
TEST(testTLS, unrelatedTrustAnchorRejectsServer) {
    std::thread s(mock::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    mock::wait_for_servers();
    ASSERT_OCCURRED_AS_P(send_trusting(CERT_PATH), UNREACHABLE);
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
    x::env::set(priv::ROOTS_FILE_ENV, CERT_PATH);
    const auto pool = std::make_shared<Pool>(true);
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, PLAIN_TARGET);
    auto req = message();
    ASSERT_OCCURRED_AS_P(client.send("", req), UNREACHABLE);
    mock::stop_servers();
    s.join();
}
}
