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

#include "google/protobuf/empty.pb.h"
#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "core/pkg/transport/grpc/channel/channel.pb.h"

/// @brief loads a JSON string into a fresh Config via override.
synnax::Config load(const std::string &json) {
    synnax::Config cfg;
    auto parser = x::json::Parser(json);
    cfg.override(parser);
    return cfg;
}

/// @brief a fresh config is insecure.
TEST(ConfigSecure, InsecureByDefault) {
    const synnax::Config cfg;
    EXPECT_FALSE(cfg.secure);
}

/// @brief secure persists across a save/load.
TEST(ConfigPersist, SecureFlagTrue) {
    synnax::Config cfg;
    cfg.secure = true;
    const auto loaded = load(cfg.to_json().dump());
    EXPECT_TRUE(loaded.secure);
}

/// @brief insecure persists across a save/load.
TEST(ConfigPersist, SecureFlagFalse) {
    synnax::Config cfg;
    cfg.secure = false;
    const auto loaded = load(cfg.to_json().dump());
    EXPECT_FALSE(loaded.secure);
}

/// @brief every field survives the round-trip.
TEST(ConfigPersist, AllFields) {
    synnax::Config cfg;
    cfg.host = "example.com";
    cfg.port = 8080;
    cfg.username = "op";
    cfg.password = "op123";
    cfg.ca_cert_file = "/certs/ca.crt";
    cfg.client_cert_file = "/certs/client.crt";
    cfg.client_key_file = "/certs/client.key";
    cfg.secure = true;
    cfg.max_retries = 9;

    const auto loaded = load(cfg.to_json().dump());
    EXPECT_EQ(loaded.host, "example.com");
    EXPECT_EQ(loaded.port, 8080);
    EXPECT_EQ(loaded.username, "op");
    EXPECT_EQ(loaded.password, "op123");
    EXPECT_EQ(loaded.ca_cert_file, "/certs/ca.crt");
    EXPECT_EQ(loaded.client_cert_file, "/certs/client.crt");
    EXPECT_EQ(loaded.client_key_file, "/certs/client.key");
    EXPECT_TRUE(loaded.secure);
    EXPECT_EQ(loaded.max_retries, 9);
}

/// @brief a pre-flag config with only a CA loads secure.
TEST(ConfigOverride, LegacySecureViaCAFile) {
    const auto cfg = load(R"({"ca_cert_file":"/certs/ca.crt"})");
    EXPECT_TRUE(cfg.secure);
}

/// @brief a pre-flag config with only a client certificate and key loads secure.
TEST(ConfigOverride, LegacySecureViaClientCert) {
    const auto cfg = load(
        R"({"client_cert_file":"/c/client.crt","client_key_file":"/c/client.key"})"
    );
    EXPECT_TRUE(cfg.secure);
}

/// @brief a client certificate without its key does not imply secure.
TEST(ConfigOverride, LegacyClientCertWithoutKey) {
    const auto cfg = load(R"({"client_cert_file":"/certs/client.crt"})");
    EXPECT_FALSE(cfg.secure);
}

/// @brief an explicit secure flag wins over the certificate inference.
TEST(ConfigOverride, ExplicitFlagOverridesInference) {
    const auto cfg = load(R"({"ca_cert_file":"/certs/ca.crt","secure":false})");
    EXPECT_FALSE(cfg.secure);
}

/// @brief a pre-flag config with no CA loads insecure.
TEST(ConfigOverride, LegacyInsecure) {
    const auto cfg = load(R"({"username":"op"})");
    EXPECT_FALSE(cfg.secure);
}

/// @brief absent keys keep their defaults.
TEST(ConfigOverride, EmptyKeepsDefaults) {
    const auto cfg = load("{}");
    EXPECT_EQ(cfg.host, "localhost");
    EXPECT_EQ(cfg.port, 9090);
    EXPECT_FALSE(cfg.secure);
}

/// @brief present keys override, absent keys are untouched.
TEST(ConfigOverride, PartialDoesNotClobber) {
    const auto cfg = load(R"({"secure":true})");
    EXPECT_TRUE(cfg.secure);
    EXPECT_EQ(cfg.host, "localhost");
}

/// @brief an insecure config omits the cert-file lines.
TEST(ConfigStream, InsecureOmitsCertLines) {
    const synnax::Config cfg;
    std::ostringstream os;
    os << cfg;
    EXPECT_EQ(os.str().find("ca_cert_file"), std::string::npos);
}

/// @brief a secure config prints the cert-file lines.
TEST(ConfigStream, SecureIncludesCertLines) {
    synnax::Config cfg;
    cfg.secure = true;
    cfg.ca_cert_file = "/certs/ca.crt";
    std::ostringstream os;
    os << cfg;
    EXPECT_NE(os.str().find("ca_cert_file"), std::string::npos);
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

/// @brief the connectivity check skips client middleware so it can probe a
/// cluster before authenticating.
TEST(TransportMiddleware, ConnectivityCheckSkipsMiddleware) {
    synnax::details::Transport t(1, "localhost", "", "", "", false);
    auto mw = std::make_shared<CountingMiddleware>();
    t.use(mw);
    google::protobuf::Empty req;
    const auto err = t.connectivity_check->send("/connectivity/check", req).second;
    EXPECT_TRUE(err.matches(freighter::UNREACHABLE));
    EXPECT_EQ(mw->calls, 0);
}

/// @brief every other client runs middleware.
TEST(TransportMiddleware, OtherClientsRunMiddleware) {
    synnax::details::Transport t(1, "localhost", "", "", "", false);
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
    synnax::details::Transport t(1, "localhost", "", "", "", false);
    auto mw = std::make_shared<CountingMiddleware>();
    t.use(mw);
    grpc::channel::RetrieveRequest req;
    (void) t.chan_retrieve->send("/channel/retrieve", req);
    (void) t.chan_retrieve->send("/channel/retrieve", req);
    google::protobuf::Empty check_req;
    (void) t.connectivity_check->send("/connectivity/check", check_req);
    EXPECT_EQ(mw->calls, 2);
}

/// @brief the full client probes connectivity against an unreachable cluster
/// without needing credentials, mirroring the login flow.
TEST(SynnaxConnectivity, ProbeFailsAgainstUnreachableCluster) {
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
