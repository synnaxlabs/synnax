// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>
#include <string>
#include <thread>

#include "google/protobuf/empty.pb.h"
#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "freighter/cpp/grpc/testutil/freighter/cpp/grpc/testutil/service.grpc.pb.h"
#include "freighter/cpp/grpc/testutil/server.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/env/env.h"
#include "x/cpp/fs/fs.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "core/pkg/transport/grpc/channel/channel.pb.h"

/// @brief loads a JSON string into a fresh Config via override.
synnax::Config load(const std::string &json) {
    synnax::Config cfg;
    auto parser = x::json::Parser(json);
    cfg.override(parser);
    return cfg;
}

/// @brief a fresh config leaves secure unset and connects without TLS.
TEST(ConfigSecure, InsecureByDefault) {
    const synnax::Config cfg;
    EXPECT_EQ(cfg.secure, std::nullopt);
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_FALSE(tls.enabled);
}

/// @brief secure persists across a save/load.
TEST(ConfigPersist, SecureFlagTrue) {
    synnax::Config cfg;
    cfg.secure = true;
    const auto loaded = load(cfg.to_json().dump());
    EXPECT_EQ(loaded.secure, true);
}

/// @brief insecure persists across a save/load.
TEST(ConfigPersist, SecureFlagFalse) {
    synnax::Config cfg;
    cfg.secure = false;
    const auto loaded = load(cfg.to_json().dump());
    EXPECT_EQ(loaded.secure, false);
}

/// @brief an unset secure stays unset across a save/load.
TEST(ConfigPersist, SecureUnset) {
    const synnax::Config cfg;
    const auto loaded = load(cfg.to_json().dump());
    EXPECT_EQ(loaded.secure, std::nullopt);
}

/// @brief every field survives the round-trip.
TEST(ConfigPersist, AllFields) {
    synnax::Config cfg;
    cfg.host = "example.com";
    cfg.port = 8080;
    cfg.username = "op";
    cfg.password = "op123";
    cfg.secure = true;
    cfg.ca_cert = "/etc/synnax/ca.pem";
    cfg.max_retries = 9;
    cfg.api_key = "syk_key";

    const auto loaded = load(cfg.to_json().dump());
    EXPECT_EQ(loaded.host, "example.com");
    EXPECT_EQ(loaded.port, 8080);
    EXPECT_EQ(loaded.username, "op");
    EXPECT_EQ(loaded.password, "op123");
    EXPECT_EQ(loaded.secure, true);
    EXPECT_EQ(loaded.ca_cert, "/etc/synnax/ca.pem");
    EXPECT_EQ(loaded.max_retries, 9);
    EXPECT_EQ(loaded.api_key, "syk_key");
}

/// @brief a config without the secure flag leaves it unset.
TEST(ConfigOverride, AbsentFlagIsUnset) {
    const auto cfg = load(R"({"username":"op"})");
    EXPECT_EQ(cfg.secure, std::nullopt);
}

/// @brief a later override without the secure flag keeps the earlier value.
TEST(ConfigOverride, SecureSurvivesLaterOverride) {
    synnax::Config cfg;
    auto first = x::json::Parser(std::string(R"({"secure":true})"));
    cfg.override(first);
    auto second = x::json::Parser(std::string(R"({"host":"node1"})"));
    cfg.override(second);
    EXPECT_EQ(cfg.secure, true);
    EXPECT_EQ(cfg.host, "node1");
}

/// @brief a later override with an explicit secure flag replaces the earlier value.
TEST(ConfigOverride, LaterExplicitFlagWins) {
    synnax::Config cfg;
    auto first = x::json::Parser(std::string(R"({"secure":true})"));
    cfg.override(first);
    auto second = x::json::Parser(std::string(R"({"secure":false})"));
    cfg.override(second);
    EXPECT_EQ(cfg.secure, false);
}

/// @brief absent keys keep their defaults.
TEST(ConfigOverride, EmptyKeepsDefaults) {
    const auto cfg = load("{}");
    EXPECT_EQ(cfg.host, "localhost");
    EXPECT_EQ(cfg.port, 9090);
    EXPECT_EQ(cfg.secure, std::nullopt);
}

/// @brief present keys override, absent keys are untouched.
TEST(ConfigOverride, PartialDoesNotClobber) {
    const auto cfg = load(R"({"secure":true})");
    EXPECT_EQ(cfg.secure, true);
    EXPECT_EQ(cfg.host, "localhost");
}

/// @brief the secure field always prints.
TEST(ConfigStream, AlwaysShowsSecure) {
    const synnax::Config cfg;
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("secure"), std::string::npos);
}

/// @brief address joins host and port.
TEST(ConfigAddress, FormatsHostPort) {
    synnax::Config cfg;
    cfg.host = "node1";
    cfg.port = 9091;
    EXPECT_EQ(cfg.address(), "node1:9091");
}

/// @brief a middleware that counts how many times it runs.
class CountingMiddleware final : public freighter::PassthroughMiddleware {
public:
    int calls = 0;

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        this->calls++;
        return next(context);
    }
};

/// @brief the connectivity check skips client middleware so it can probe a Core before
/// authenticating.
TEST(TransportMiddleware, ConnectivityCheckSkipsMiddleware) {
    synnax::details::Transport t(
        1,
        "localhost",
        std::make_shared<freighter::grpc::Pool>()
    );
    auto mw = std::make_shared<CountingMiddleware>();
    t.use(mw);
    google::protobuf::Empty req;
    const auto err = t.connectivity_check->send("/connectivity/check", req).second;
    EXPECT_TRUE(err.matches(freighter::UNREACHABLE));
    EXPECT_EQ(mw->calls, 0);
}

/// @brief every other client runs middleware.
TEST(TransportMiddleware, OtherClientsRunMiddleware) {
    synnax::details::Transport t(
        1,
        "localhost",
        std::make_shared<freighter::grpc::Pool>()
    );
    auto mw = std::make_shared<CountingMiddleware>();
    t.use(mw);
    grpc::channel::RetrieveRequest req;
    const auto err = t.chan_retrieve->send("/channel/retrieve", req).second;
    EXPECT_TRUE(err.matches(freighter::UNREACHABLE));
    EXPECT_EQ(mw->calls, 1);
}

/// @brief middleware runs once per call, so repeated requests each pass
/// through it.
TEST(TransportMiddleware, MiddlewareRunsPerCall) {
    synnax::details::Transport t(
        1,
        "localhost",
        std::make_shared<freighter::grpc::Pool>()
    );
    auto mw = std::make_shared<CountingMiddleware>();
    t.use(mw);
    grpc::channel::RetrieveRequest req;
    (void) t.chan_retrieve->send("/channel/retrieve", req);
    (void) t.chan_retrieve->send("/channel/retrieve", req);
    google::protobuf::Empty check_req;
    (void) t.connectivity_check->send("/connectivity/check", check_req);
    EXPECT_EQ(mw->calls, 2);
}

/// @brief the full client probes connectivity against an unreachable Core without
/// needing credentials, mirroring the login flow.
TEST(SynnaxConnectivity, ProbeFailsAgainstUnreachableCore) {
    synnax::Config cfg;
    cfg.port = 1;
    cfg.username = "";
    cfg.password = "";
    const synnax::Synnax client(cfg);
    const auto state = client.connectivity->check();
    EXPECT_EQ(state.status, synnax::connection::Status::FAILED);
    EXPECT_TRUE(state.error.matches(freighter::UNREACHABLE));
    EXPECT_FALSE(state.message.empty());
}

namespace {
const std::string CA_CERT_PATH = "freighter/cpp/grpc/testutil/test_ca_cert.pem";
/// A leaf certificate signed by the CA at CA_CERT_PATH.
const std::string LEAF_CERT_PATH = "freighter/cpp/grpc/testutil/test_leaf_cert.pem";
const std::string LEAF_KEY_PATH = "freighter/cpp/grpc/testutil/test_leaf_key.pem";
const std::string TLS_TARGET = "localhost:8086";

/// @brief serves the Freighter test service over TLS with a certificate that the test
/// CA signs.
class ConfigTLS : public ::testing::Test {
protected:
    std::thread server;

    void SetUp() override {
        this->server = std::thread(
            freighter::grpc::testutil::tls_server,
            TLS_TARGET,
            LEAF_CERT_PATH,
            LEAF_KEY_PATH
        );
        freighter::grpc::testutil::wait_for_servers();
    }

    void TearDown() override {
        freighter::grpc::testutil::stop_servers();
        this->server.join();
    }

    /// @brief sends one request to the TLS server through the pool for cfg.
    static std::pair<test::Message, x::errors::Error> send(const synnax::Config &cfg) {
        auto client = freighter::grpc::
            UnaryClient<test::Message, test::Message, test::UnaryMessageService>(
                synnax::details::new_pool(cfg),
                TLS_TARGET
            );
        test::Message req;
        req.set_payload("ping");
        return client.send("", req);
    }
};
}

/// @brief a CA certificate with secure unset turns TLS on.
TEST_F(ConfigTLS, CACertAloneUsesTLS) {
    synnax::Config cfg;
    cfg.ca_cert = CA_CERT_PATH;
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_TRUE(tls.enabled);
    EXPECT_EQ(tls.ca_cert, CA_CERT_PATH);
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief a CA certificate with secure true uses TLS.
TEST_F(ConfigTLS, CACertWithSecureUsesTLS) {
    synnax::Config cfg;
    cfg.secure = true;
    cfg.ca_cert = CA_CERT_PATH;
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief a CA certificate with an explicit secure false is an error.
TEST_F(ConfigTLS, CACertWithSecureFalseIsError) {
    synnax::Config cfg;
    cfg.secure = false;
    cfg.ca_cert = CA_CERT_PATH;
    ASSERT_OCCURRED_AS_P(cfg.tls(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(send(cfg), x::errors::VALIDATION);
}

/// @brief SYNNAX_CA_CERT gives the CA certificate when ca_cert is empty.
TEST_F(ConfigTLS, CACertFromEnvironmentUsesTLS) {
    x::env::set("SYNNAX_CA_CERT", CA_CERT_PATH);
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_CA_CERT"); });
    const synnax::Config cfg;
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief SYNNAX_CA_CERT with an explicit secure false is an error.
TEST_F(ConfigTLS, CACertFromEnvironmentWithSecureFalseIsError) {
    x::env::set("SYNNAX_CA_CERT", CA_CERT_PATH);
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_CA_CERT"); });
    synnax::Config cfg;
    cfg.secure = false;
    ASSERT_OCCURRED_AS_P(send(cfg), x::errors::VALIDATION);
}

/// @brief a CA certificate that does not exist fails every call.
TEST_F(ConfigTLS, MissingCACertIsError) {
    synnax::Config cfg;
    cfg.ca_cert = "freighter/cpp/grpc/testutil/missing.pem";
    ASSERT_OCCURRED_AS_P(send(cfg), x::fs::NOT_FOUND);
}

/// @brief without a CA certificate, the client cannot reach a TLS Core.
TEST_F(ConfigTLS, NoCACertIsPlaintext) {
    const synnax::Config cfg;
    ASSERT_OCCURRED_AS_P(send(cfg), freighter::UNREACHABLE);
}
