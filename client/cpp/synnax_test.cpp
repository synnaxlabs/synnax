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

/// @brief it should leave secure unset and connect without TLS by default.
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

/// @brief it should keep an unset secure unset across a save and load.
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

/// @brief it should leave secure unset when the config has no secure flag.
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

/// @brief it should replace both secure and ca_cert when a later source sets only
/// the CA certificate.
TEST(ConfigOverride, CACertReplacesEarlierSecure) {
    synnax::Config cfg;
    auto saved = x::json::Parser(std::string(R"({"secure":false})"));
    cfg.override(saved);
    auto later = x::json::Parser(std::string(R"({"ca_cert":"/etc/synnax/ca.pem"})"));
    cfg.override(later);
    EXPECT_EQ(cfg.secure, std::nullopt);
    EXPECT_EQ(cfg.ca_cert, "/etc/synnax/ca.pem");
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_TRUE(tls.enabled);
}

/// @brief it should drop an earlier CA certificate when a later source sets only
/// secure.
TEST(ConfigOverride, SecureReplacesEarlierCACert) {
    synnax::Config cfg;
    auto saved = x::json::Parser(std::string(R"({"ca_cert":"/etc/synnax/ca.pem"})"));
    cfg.override(saved);
    auto later = x::json::Parser(std::string(R"({"secure":false})"));
    cfg.override(later);
    EXPECT_EQ(cfg.secure, false);
    EXPECT_EQ(cfg.ca_cert, "");
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_FALSE(tls.enabled);
}

/// @brief it should keep the earlier pair when a later source sets neither field.
TEST(ConfigOverride, PairSurvivesLaterOverride) {
    synnax::Config cfg;
    auto saved = x::json::Parser(
        std::string(R"({"secure":true,"ca_cert":"/etc/synnax/ca.pem"})")
    );
    cfg.override(saved);
    auto later = x::json::Parser(std::string(R"({"host":"node1"})"));
    cfg.override(later);
    EXPECT_EQ(cfg.secure, true);
    EXPECT_EQ(cfg.ca_cert, "/etc/synnax/ca.pem");
}

/// @brief it should fail when one source sets secure false and a CA certificate.
TEST(ConfigOverride, SecureFalseWithCACertInOneSourceIsError) {
    const auto cfg = load(R"({"secure":false,"ca_cert":"/etc/synnax/ca.pem"})");
    ASSERT_OCCURRED_AS_P(cfg.tls(), x::errors::VALIDATION);
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
        x::env::unset("SYNNAX_CA_CERT");
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

/// @brief it should turn TLS on for a CA certificate with secure unset.
TEST_F(ConfigTLS, CACertAloneUsesTLS) {
    synnax::Config cfg;
    cfg.ca_cert = CA_CERT_PATH;
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_TRUE(tls.enabled);
    EXPECT_EQ(tls.ca_cert, CA_CERT_PATH);
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief it should use TLS for a CA certificate with secure true.
TEST_F(ConfigTLS, CACertWithSecureUsesTLS) {
    synnax::Config cfg;
    cfg.secure = true;
    cfg.ca_cert = CA_CERT_PATH;
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief it should fail every call for a CA certificate with secure false.
TEST_F(ConfigTLS, CACertWithSecureFalseIsError) {
    synnax::Config cfg;
    cfg.secure = false;
    cfg.ca_cert = CA_CERT_PATH;
    ASSERT_OCCURRED_AS_P(cfg.tls(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(send(cfg), x::errors::VALIDATION);
}

/// @brief it should read SYNNAX_CA_CERT when secure and ca_cert are unset.
TEST_F(ConfigTLS, CACertFromEnvironmentUsesTLS) {
    x::env::set("SYNNAX_CA_CERT", CA_CERT_PATH);
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_CA_CERT"); });
    const synnax::Config cfg;
    const auto res = ASSERT_NIL_P(send(cfg));
    EXPECT_EQ(res.payload(), "Read request: ping");
}

/// @brief it should resolve TLS from the fields alone, without the environment.
TEST(ConfigTLSPure, IgnoresEnvironment) {
    x::env::set("SYNNAX_CA_CERT", "/etc/synnax/ca.pem");
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_CA_CERT"); });
    const synnax::Config cfg;
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_FALSE(tls.enabled);
}

/// @brief it should take the CA certificate from the environment only when neither
/// secure nor ca_cert is set.
TEST(ConfigOverrideTLSDefault, AppliesOnlyWhenPairIsUnset) {
    x::env::set("TEST_TLS_DEFAULT_CA_CERT", "/etc/synnax/env.pem");
    x::defer::defer unset_env([] { x::env::unset("TEST_TLS_DEFAULT_CA_CERT"); });
    x::env::Parser env("TEST_TLS_DEFAULT");
    synnax::Config unset;
    unset.override_tls_default(env);
    EXPECT_EQ(unset.ca_cert, "/etc/synnax/env.pem");
    synnax::Config secure;
    secure.secure = false;
    secure.override_tls_default(env);
    EXPECT_EQ(secure.ca_cert, "");
    synnax::Config ca;
    ca.ca_cert = "/etc/synnax/ca.pem";
    ca.override_tls_default(env);
    EXPECT_EQ(ca.ca_cert, "/etc/synnax/ca.pem");
    ASSERT_NIL(env.error());
}

/// @brief it should ignore SYNNAX_CA_CERT when the config sets secure.
TEST_F(ConfigTLS, SecureFromConfigIgnoresEnvironmentCACert) {
    x::env::set("SYNNAX_CA_CERT", CA_CERT_PATH);
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_CA_CERT"); });
    synnax::Config cfg;
    cfg.secure = false;
    const auto tls = ASSERT_NIL_P(cfg.tls());
    EXPECT_FALSE(tls.enabled);
    ASSERT_OCCURRED_AS_P(send(cfg), freighter::UNREACHABLE);
}

/// @brief it should fail every call when the CA certificate does not exist.
TEST_F(ConfigTLS, MissingCACertIsError) {
    synnax::Config cfg;
    cfg.ca_cert = "freighter/cpp/grpc/testutil/missing.pem";
    ASSERT_OCCURRED_AS_P(send(cfg), x::fs::NOT_FOUND);
}

/// @brief it should connect without TLS when no CA certificate is set.
TEST_F(ConfigTLS, NoCACertIsPlaintext) {
    const synnax::Config cfg;
    ASSERT_OCCURRED_AS_P(send(cfg), freighter::UNREACHABLE);
}
