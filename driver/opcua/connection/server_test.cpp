// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/connection/server.h"

namespace driver::opcua::connection {
namespace srv = ::synnax::opcua::server;

/// @brief it should connect anonymously without security, with the timeouts in ms.
TEST(FromServer, Anonymous) {
    const srv::Connection conn{
        .endpoint = "opc.tcp://plc:4840",
        .client_timeout = x::telem::SECOND * 2,
    };
    const auto cfg = ASSERT_NIL_P(from_server(conn));
    ASSERT_EQ(cfg.endpoint, "opc.tcp://plc:4840");
    ASSERT_TRUE(cfg.username.empty());
    ASSERT_TRUE(cfg.password_secret.is_nil());
    ASSERT_EQ(cfg.security_mode, "None");
    ASSERT_EQ(cfg.security_policy, "None");
    ASSERT_EQ(cfg.client_timeout_ms, 2000);
    ASSERT_EQ(cfg.secure_channel_lifetime_ms, 600000);
    ASSERT_EQ(cfg.session_timeout_ms, 1200000);
}

/// @brief it should log in with the user name and the secret of the password.
TEST(FromServer, Username) {
    const auto password = x::uuid::create();
    const srv::Connection conn{
        .endpoint = "opc.tcp://plc:4840",
        .auth = srv::UsernameAuth{.username = "op", .password = password},
    };
    const auto cfg = ASSERT_NIL_P(from_server(conn));
    ASSERT_EQ(cfg.username, "op");
    ASSERT_EQ(cfg.password_secret, password);
}

/// @brief it should sign with the client certificate and trust the pinned one.
TEST(FromServer, SignPinned) {
    const auto client = x::uuid::create();
    const auto pinned = x::uuid::create();
    srv::SignSecurity security;
    security.policy = srv::SECURITY_POLICY_BASIC_256_SHA_256;
    security.client_certificate = client;
    security.server_trust = srv::PinnedServerTrust{.certificate = pinned};
    const srv::Connection conn{.endpoint = "opc.tcp://plc:4840", .security = security};
    const auto cfg = ASSERT_NIL_P(from_server(conn));
    ASSERT_EQ(cfg.security_mode, "Sign");
    ASSERT_EQ(cfg.security_policy, "Basic256Sha256");
    ASSERT_EQ(cfg.client_certificate_item, client);
    ASSERT_EQ(cfg.server_certificate_item, pinned);
    ASSERT_FALSE(cfg.certificate_validation_disabled);
}

/// @brief it should encrypt, trust the first certificate, and present the Driver
/// certificate.
TEST(FromServer, SignAndEncryptFirstUse) {
    srv::SignAndEncryptSecurity security;
    security.policy = srv::SECURITY_POLICY_AES_256_SHA_256_RSA_PSS;
    const srv::Connection conn{.endpoint = "opc.tcp://plc:4840", .security = security};
    const auto cfg = ASSERT_NIL_P(from_server(conn));
    ASSERT_EQ(cfg.security_mode, "SignAndEncrypt");
    ASSERT_EQ(cfg.security_policy, "Aes256_Sha256_RsaPss");
    ASSERT_TRUE(cfg.client_certificate_item.is_nil());
    ASSERT_TRUE(cfg.server_certificate_item.is_nil());
    ASSERT_FALSE(cfg.certificate_validation_disabled);
}

/// @brief it should accept any server certificate when validation is disabled.
TEST(FromServer, ValidationDisabled) {
    srv::SignSecurity security;
    security.policy = srv::SECURITY_POLICY_BASIC_256_SHA_256;
    security.server_trust = srv::ValidationDisabledServerTrust{};
    const srv::Connection conn{.endpoint = "opc.tcp://plc:4840", .security = security};
    const auto cfg = ASSERT_NIL_P(from_server(conn));
    ASSERT_TRUE(cfg.certificate_validation_disabled);
}

/// @brief it should refuse a certificate login.
TEST(FromServer, CertificateLogin) {
    const srv::Connection conn{
        .endpoint = "opc.tcp://plc:4840",
        .auth = srv::CertificateAuth{.certificate = x::uuid::create()},
    };
    ASSERT_OCCURRED_AS_P(from_server(conn), x::errors::VALIDATION);
}
}
