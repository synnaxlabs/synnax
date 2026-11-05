// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <mutex>

#include "absl/log/globals.h"
#include "absl/log/log_entry.h"
#include "absl/log/log_sink.h"
#include "absl/log/log_sink_registry.h"
#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::connection {
/// @brief enables VLOG(1) and captures the text of every log line while in scope.
class CaptureSink final : public absl::LogSink {
    int prev_vlog_level;
    std::mutex mu;
    std::vector<std::string> lines;

public:
    CaptureSink(): prev_vlog_level(absl::SetGlobalVLogLevel(1)) {
        absl::AddLogSink(this);
    }

    ~CaptureSink() override {
        absl::RemoveLogSink(this);
        absl::SetGlobalVLogLevel(this->prev_vlog_level);
    }

    void Send(const absl::LogEntry &entry) override {
        std::lock_guard lock(this->mu);
        this->lines.emplace_back(entry.text_message());
    }

    [[nodiscard]] bool contains(const std::string &substr) {
        std::lock_guard lock(this->mu);
        return std::ranges::any_of(this->lines, [&](const std::string &line) {
            return line.contains(substr);
        });
    }
};

/// @brief it should establish basic connection and read node value.
TEST(ConnectionTest, testBasicConn) {
    UA_Variant float_val;
    UA_Variant_init(&float_val);
    UA_Float float_data = 5.0f;
    UA_Variant_setScalarCopy(&float_val, &float_data, &UA_TYPES[UA_TYPES_FLOAT]);

    testutil::TestNode
        node(1, "test", &UA_TYPES[UA_TYPES_FLOAT], float_val, "Test Float Node");

    testutil::ServerConfig server_cfg;
    server_cfg.test_nodes = {node};
    server_cfg.port = 4849;

    testutil::Server server(server_cfg);
    server.start();

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4849";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        connect(cfg, {}, "opc"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    ASSERT_NE(client, nullptr);

    auto ser = ASSERT_NIL_P(opcua::testutil::simple_read(client, "NS=1;S=test"));
    ASSERT_EQ(ser.data_type(), x::telem::FLOAT32_T);
    ASSERT_EQ(ser.at<float>(0), 5.0f);

    server.stop();

    UA_Variant_clear(&float_val);
}

/// @brief it should log open62541 messages with their UA_String arguments formatted.
TEST(ConnectionTest, testLogFormatsUAString) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4850;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4850";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    CaptureSink logs;
    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));
    EXPECT_TRUE(logs.contains(
        "for SecurityPolicy http://opcfoundation.org/UA/SecurityPolicy#None and a "
        "revised lifetime of"
    ));

    server.stop();
}

/// @brief it should return unreachable error when connection is refused.
TEST(ConnectionTest, connectionRefused) {
    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:9999";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::UNREACHABLE);
}

/// @brief it should return invalid endpoint error for malformed endpoint.
TEST(ConnectionTest, invalidEndpointFormat) {
    Config cfg;
    cfg.endpoint = "not-a-valid-endpoint";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::INVALID_ENDPOINT);
}

/// @brief it should return invalid endpoint error for empty endpoint.
TEST(ConnectionTest, emptyEndpoint) {
    Config cfg;
    cfg.endpoint = "";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::INVALID_ENDPOINT);
}

/// @brief it should return unreachable error for invalid hostname.
TEST(ConnectionTest, invalidHostname) {
    Config cfg;
    cfg.endpoint = "opc.tcp://nonexistent.invalid.hostname:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::UNREACHABLE);
}

/// @brief it should reconnect successfully after disconnect.
TEST(ConnectionTest, disconnectAndReconnect) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4841;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4841";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));

    UA_SessionState session_state = UA_SESSIONSTATE_CLOSED;
    UA_SecureChannelState channel_state = UA_SECURECHANNELSTATE_CLOSED;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    UA_Client_disconnect(client.get());

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    ASSERT_NIL(reconnect(client, cfg.endpoint));

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

/// @brief it should handle server stop during active connection.
TEST(ConnectionTest, serverStopDuringConnection) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4842;
    auto server = std::make_unique<testutil::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4842";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    auto node_id = ASSERT_NIL_P(types::NodeId::parse("NS=1;S=TestFloat"));

    UA_ReadValueId ids[1];
    UA_ReadValueId_init(&ids[0]);
    ids[0].nodeId = node_id.resolve({}).value();
    ids[0].attributeId = UA_ATTRIBUTEID_VALUE;

    UA_ReadRequest req;
    UA_ReadRequest_init(&req);
    req.nodesToRead = ids;
    req.nodesToReadSize = 1;

    UA_ReadResponse res = UA_Client_Service_read(client.get(), req);
    EXPECT_NE(res.responseHeader.serviceResult, UA_STATUSCODE_GOOD);
    UA_ReadResponse_clear(&res);
}

/// @brief it should connect successfully after server restart.
TEST(ConnectionTest, connectionAfterServerRestart) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4844;

    auto server = std::make_unique<testutil::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4844";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client1 = ASSERT_NIL_P(connect(cfg, {}, "test"));

    server->stop();
    server.reset();

    server = std::make_unique<testutil::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    auto client2 = ASSERT_NIL_P(connect(cfg, {}, "test"));

    server->stop();
}

/// @brief it should read successfully before disconnect changes session state.
TEST(ConnectionTest, readAfterDisconnect) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4845;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4845";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));

    auto ser1 = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=TestFloat"));

    UA_Client_disconnect(client.get());

    UA_SessionState session_state = UA_SESSIONSTATE_CLOSED;
    UA_SecureChannelState channel_state = UA_SECURECHANNELSTATE_CLOSED;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

/// @brief it should handle multiple consecutive disconnects gracefully.
TEST(ConnectionTest, multipleDisconnects) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4846;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4846";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));

    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());

    server.stop();
}

/// @brief it should reject username/password authentication without encryption.
TEST(ConnectionTest, usernamePasswordWithoutEncryption) {
    // Mock server rejects username/password without encryption to prevent
    // credential leaks over the network
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4847;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4847";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "any_user";
    cfg.password = "any_password";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::SESSION_REJECTED);

    server.stop();
}

/// @brief it should reject empty username with password without encryption.
TEST(ConnectionTest, emptyUsernameWithPassword) {
    // When password is provided without encryption, the server rejects the
    // identity token to prevent credential leaks
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4851;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4851";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "";
    cfg.password = "password";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::SESSION_REJECTED);

    server.stop();
}

/// @brief it should reject username with empty password without encryption.
TEST(ConnectionTest, usernameWithEmptyPassword) {
    // When username is provided without encryption, the server rejects the
    // identity token to prevent credential leaks
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4852;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4852";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "username";
    cfg.password = "";

    ASSERT_OCCURRED_AS_P(connect(cfg, {}, "test"), opcua::errors::SESSION_REJECTED);

    server.stop();
}

/// @brief it should apply default timeout of 5 seconds when client_timeout_ms is 0.
TEST(ConnectionTest, DefaultClientTimeout) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4854;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4854";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.client_timeout_ms = 0;

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));
    auto *config = UA_Client_getConfig(client.get());
    EXPECT_EQ(config->timeout, 5000);

    server.stop();
}

/// @brief it should apply custom client timeout when specified.
TEST(ConnectionTest, CustomClientTimeout) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4855;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4855";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.client_timeout_ms = 15000;

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));
    auto *config = UA_Client_getConfig(client.get());
    EXPECT_EQ(config->timeout, 15000);

    server.stop();
}

/// @brief it should apply default secure channel (10 min) and session (20 min)
/// timeouts when not specified.
TEST(ConnectionTest, DefaultSessionAndChannelTimeouts) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4856;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4856";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.secure_channel_lifetime_ms = 0;
    cfg.session_timeout_ms = 0;

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));
    auto *config = UA_Client_getConfig(client.get());
    EXPECT_EQ(config->secureChannelLifeTime, 600000); // 10 minutes
    EXPECT_EQ(config->requestedSessionTimeout, 1200000); // 20 minutes

    server.stop();
}

/// @brief it should apply custom secure channel and session timeouts.
TEST(ConnectionTest, CustomSessionAndChannelTimeouts) {
    testutil::ServerConfig server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = 4857;
    testutil::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4857";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.secure_channel_lifetime_ms = 30000;
    cfg.session_timeout_ms = 60000;

    auto client = ASSERT_NIL_P(connect(cfg, {}, "test"));
    auto *config = UA_Client_getConfig(client.get());
    EXPECT_EQ(config->secureChannelLifeTime, 30000);
    EXPECT_EQ(config->requestedSessionTimeout, 60000);

    server.stop();
}

/// @brief it should read the configured certificate files instead of the Driver
/// certificate, and report a file that is missing.
TEST(ConnectionTest, resolveSecurityMissingFiles) {
    const auto store = testutil::create_store();
    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4848";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";
    ASSERT_OCCURRED_AS_P(
        resolve_security(cfg, *store),
        opcua::errors::CLIENT_CERTIFICATE_INVALID
    );
}

/// @brief it should leave the identity empty for an unsecured connection without
/// making a Driver certificate.
TEST(ConnectionTest, resolveSecurityUnsecured) {
    const auto dir = std::filesystem::temp_directory_path() /
                     ("opcua-unused-" + x::uuid::create().to_string());
    certificate::Store store(certificate::StoreConfig{.dir = dir});
    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4848";
    const auto sec = ASSERT_NIL_P(resolve_security(cfg, store));
    ASSERT_FALSE(sec.identity.has_value());
    ASSERT_FALSE(std::filesystem::exists(dir));
}

/// @brief it should refuse a secured connection without a client identity.
TEST(ConnectionTest, securedWithoutIdentity) {
    Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4848";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256Sha256";
    ASSERT_OCCURRED_AS_P(
        connect(cfg, {}, "test"),
        opcua::errors::ENCRYPTION_CONFIG_FAILED
    );
}
}
