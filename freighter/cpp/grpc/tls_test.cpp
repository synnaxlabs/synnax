// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "freighter/cpp/grpc/grpc.h"
#include "freighter/cpp/grpc/testutil/freighter/cpp/grpc/testutil/service.grpc.pb.h"
#include "freighter/cpp/grpc/testutil/server.h"
#include "x/cpp/env/env.h"
#include "x/cpp/testutil/testutil.h"

namespace freighter::grpc {
using RQ = test::Message;
using RS = test::Message;
using UNARY_RPC = test::UnaryMessageService;

/// A self-signed certificate that is its own CA.
const std::string CERT_PATH = "freighter/cpp/grpc/testutil/test_cert.pem";
const std::string KEY_PATH = "freighter/cpp/grpc/testutil/test_key.pem";
/// A leaf certificate signed by test_ca_cert.pem.
const std::string LEAF_CERT_PATH = "freighter/cpp/grpc/testutil/test_leaf_cert.pem";
const std::string LEAF_KEY_PATH = "freighter/cpp/grpc/testutil/test_leaf_key.pem";
const std::string CA_CERT_PATH = "freighter/cpp/grpc/testutil/test_ca_cert.pem";
const std::string TLS_TARGET = "localhost:8083";
const std::string PLAIN_TARGET = "localhost:8084";

test::Message message() {
    test::Message m;
    m.set_payload("Sending to Server");
    return m;
}

const std::string EXPECTED = "Read request: Sending to Server";

/// @brief sends one request to target over a pool with the given credentials.
std::pair<RS, x::errors::Error> send_with(
    const std::shared_ptr<::grpc::ChannelCredentials> &creds,
    const std::string &target = TLS_TARGET
) {
    const auto pool = std::make_shared<Pool>(creds);
    auto client = UnaryClient<RQ, RS, UNARY_RPC>(pool, target);
    auto req = message();
    return client.send("", req);
}

/// @brief sends one request over a secure pool that trusts the given PEM file.
std::pair<RS, x::errors::Error> send_trusting(const std::string &ca_cert) {
    const auto creds = ASSERT_NIL_P(ssl_credentials(ca_cert));
    return send_with(creds);
}

/// @brief a secure pool completes a request against a TLS server whose self-signed
/// certificate is the trust anchor.
TEST(testTLS, secureClientReachesTLSServer) {
    std::thread s(testutil::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    testutil::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    testutil::stop_servers();
    s.join();
}

/// @brief a secure pool trusts a leaf certificate directly, without its issuing CA.
TEST(testTLS, leafCertificateIsTrustAnchor) {
    std::thread s(testutil::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    testutil::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(LEAF_CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    testutil::stop_servers();
    s.join();
}

/// @brief a secure pool trusts a leaf certificate through its issuing CA.
TEST(testTLS, issuingCAIsTrustAnchor) {
    std::thread s(testutil::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    testutil::wait_for_servers();
    const auto res = ASSERT_NIL_P(send_trusting(CA_CERT_PATH));
    ASSERT_EQ(res.payload(), EXPECTED);
    testutil::stop_servers();
    s.join();
}

/// @brief a secure pool rejects a server whose certificate chains to no trust anchor.
TEST(testTLS, unrelatedTrustAnchorRejectsServer) {
    std::thread s(testutil::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    testutil::wait_for_servers();
    ASSERT_OCCURRED_AS_P(send_trusting(CERT_PATH), UNREACHABLE);
    testutil::stop_servers();
    s.join();
}

/// @brief a plaintext pool cannot reach a TLS server.
TEST(testTLS, insecureClientFailsAgainstTLSServer) {
    std::thread s(testutil::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    testutil::wait_for_servers();
    ASSERT_OCCURRED_AS_P(send_with(::grpc::InsecureChannelCredentials()), UNREACHABLE);
    testutil::stop_servers();
    s.join();
}

/// @brief a secure pool cannot reach a plaintext server.
TEST(testTLS, secureClientFailsAgainstPlaintextServer) {
    std::thread s(testutil::server, PLAIN_TARGET);
    testutil::wait_for_servers();
    const auto creds = ASSERT_NIL_P(ssl_credentials(CERT_PATH));
    ASSERT_OCCURRED_AS_P(send_with(creds, PLAIN_TARGET), UNREACHABLE);
    testutil::stop_servers();
    s.join();
}

/// @brief with no CA, the credentials trust the file that the gRPC variable names.
TEST(testTLS, rootsFileVariableIsTrustAnchor) {
    std::thread s(testutil::tls_server, TLS_TARGET, CERT_PATH, KEY_PATH);
    testutil::wait_for_servers();
    x::env::set(priv::ROOTS_FILE_ENV, CERT_PATH);
    const auto creds = ASSERT_NIL_P(ssl_credentials(""));
    x::env::unset(priv::ROOTS_FILE_ENV);
    const auto res = ASSERT_NIL_P(send_with(creds));
    ASSERT_EQ(res.payload(), EXPECTED);
    testutil::stop_servers();
    s.join();
}

/// @brief the CA argument wins over the file that the gRPC variable names.
TEST(testTLS, caCertWinsOverRootsFileVariable) {
    std::thread s(testutil::tls_server, TLS_TARGET, LEAF_CERT_PATH, LEAF_KEY_PATH);
    testutil::wait_for_servers();
    x::env::set(priv::ROOTS_FILE_ENV, CERT_PATH);
    const auto creds = ASSERT_NIL_P(ssl_credentials(CA_CERT_PATH));
    x::env::unset(priv::ROOTS_FILE_ENV);
    const auto res = ASSERT_NIL_P(send_with(creds));
    ASSERT_EQ(res.payload(), EXPECTED);
    testutil::stop_servers();
    s.join();
}

/// @brief a CA file that does not exist is an error.
TEST(testTLS, missingCACertIsError) {
    ASSERT_OCCURRED_AS_P(
        ssl_credentials("freighter/cpp/grpc/testutil/missing.pem"),
        x::fs::NOT_FOUND
    );
}

/// @brief a CA file with no certificates is an error, not the system trust store.
TEST(testTLS, emptyCACertIsError) {
    const auto dir = std::filesystem::temp_directory_path();
    for (const auto &[name, contents]: std::vector<std::pair<std::string, std::string>>{
             {"empty.pem", ""},
             {"blank.pem", " \n\t\r\n"},
         }) {
        const auto path = (dir / name).string();
        std::ofstream(path) << contents;
        const auto [creds, err] = ssl_credentials(path);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        EXPECT_NE(err.data.find(path), std::string::npos);
        EXPECT_EQ(creds, nullptr);
    }
}

/// @brief a pool built from an error fails unary calls and streams with it.
TEST(testTLS, errorPoolFailsEveryCall) {
    const auto err = x::errors::Error(x::errors::VALIDATION, "no credentials");
    const auto pool = std::make_shared<Pool>(err);
    auto unary = UnaryClient<RQ, RS, UNARY_RPC>(pool, TLS_TARGET);
    auto req = message();
    ASSERT_OCCURRED_AS_P(unary.send("", req), x::errors::VALIDATION);
    auto stream = StreamClient<RQ, RS, test::StreamMessageService>(pool, TLS_TARGET);
    ASSERT_OCCURRED_AS_P(stream.stream(""), x::errors::VALIDATION);
    ASSERT_EQ(pool->size(), 0);
}
}
