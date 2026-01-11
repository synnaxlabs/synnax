// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "x/cpp/test/test.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"
#include "driver/opc/mock/server.h"
#include "driver/opc/testutil/testutil.h"

/// @brief it should establish basic connection and read node value.
TEST(ConnectionTest, testBasicConn) {
    UA_Variant float_val;
    UA_Variant_init(&float_val);
    UA_Float float_data = 5.0f;
    UA_Variant_setScalarCopy(&float_val, &float_data, &UA_TYPES[UA_TYPES_FLOAT]);

    mock::TestNode
        node(1, "test", &UA_TYPES[UA_TYPES_FLOAT], float_val, "Test Float Node");

    mock::ServerConfig server_cfg;
    server_cfg.test_nodes = {node};
    server_cfg.port = 4840;

    mock::Server server(server_cfg);
    server.start();

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        driver::opc::connection::connect(cfg, "opc"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    ASSERT_NE(client, nullptr);

    auto ser = ASSERT_NIL_P(driver::opc::testutil::simple_read(client, "NS=1;S=test"));
    ASSERT_EQ(ser.data_type(), x::telem::FLOAT32_T);
    ASSERT_EQ(ser.at<float>(0), 5.0f);

    server.stop();

    UA_Variant_clear(&float_val);
}

/// @brief it should return unreachable error when connection is refused.
TEST(ConnectionTest, connectionRefused) {
    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:9999";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::UNREACHABLE
    );
}

/// @brief it should return invalid endpoint error for malformed endpoint.
TEST(ConnectionTest, invalidEndpointFormat) {
    driver::opc::connection::Config cfg;
    cfg.endpoint = "not-a-valid-endpoint";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::INVALID_ENDPOINT
    );
}

/// @brief it should return invalid endpoint error for empty endpoint.
TEST(ConnectionTest, emptyEndpoint) {
    driver::opc::connection::Config cfg;
    cfg.endpoint = "";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::INVALID_ENDPOINT
    );
}

/// @brief it should return unreachable error for invalid hostname.
TEST(ConnectionTest, invalidHostname) {
    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://nonexistent.invalid.hostname:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::UNREACHABLE
    );
}

/// @brief it should reconnect successfully after disconnect.
TEST(ConnectionTest, disconnectAndReconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4841;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4841";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    UA_Client_disconnect(client.get());

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    ASSERT_NIL(driver::opc::connection::reconnect(client, cfg.endpoint));

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

/// @brief it should handle server stop during active connection.
TEST(ConnectionTest, serverStopDuringConnection) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4842;
    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4842";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    auto node_id = ASSERT_NIL_P(driver::opc::NodeId::parse("NS=1;S=TestFloat"));

    UA_ReadValueId ids[1];
    UA_ReadValueId_init(&ids[0]);
    ids[0].nodeId = node_id; // Implicit conversion to const UA_NodeId&
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
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4844;

    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4844";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client1 = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    server->stop();
    server.reset();

    server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());

    auto client2 = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    server->stop();
}

/// @brief it should read successfully before disconnect changes session state.
TEST(ConnectionTest, readAfterDisconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4845;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4845";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    auto ser1 = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestFloat")
    );

    UA_Client_disconnect(client.get());

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

/// @brief it should handle multiple consecutive disconnects gracefully.
TEST(ConnectionTest, multipleDisconnects) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4846;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4846";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_NIL_P(driver::opc::connection::connect(cfg, "test"));

    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());

    server.stop();
}

/// @brief it should reject username/password authentication without encryption.
TEST(ConnectionTest, usernamePasswordWithoutEncryption) {
    // Mock server rejects username/password without encryption to prevent
    // credential leaks over the network
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4847;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4847";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "any_user";
    cfg.password = "any_password";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject sign mode connection with missing certificates.
TEST(ConnectionTest, signModeWithMissingCertificates) {
    // When security mode requires encryption but certificates are missing,
    // the connection fails with identity token rejected because the server
    // won't accept the connection without proper security
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4848;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4848";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject sign and encrypt mode with missing certificates.
TEST(ConnectionTest, signAndEncryptModeWithMissingCertificates) {
    // When security mode requires encryption but certificates are missing,
    // the connection fails with identity token rejected
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4849;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4849";
    cfg.security_mode = "SignAndEncrypt";
    cfg.security_policy = "Basic256Sha256";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject connection when client certificate is missing.
TEST(ConnectionTest, missingClientCertificate) {
    // Missing certificate files cause connection to fail with identity token
    // rejected because the server won't accept unencrypted identity tokens
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4850;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4850";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256";
    cfg.client_cert = "/path/to/missing/cert.pem";
    cfg.client_private_key = "/path/to/missing/key.pem";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject empty username with password without encryption.
TEST(ConnectionTest, emptyUsernameWithPassword) {
    // When password is provided without encryption, the server rejects the
    // identity token to prevent credential leaks
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4851;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4851";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "";
    cfg.password = "password";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject username with empty password without encryption.
TEST(ConnectionTest, usernameWithEmptyPassword) {
    // When username is provided without encryption, the server rejects the
    // identity token to prevent credential leaks
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4852;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4852";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "username";
    cfg.password = "";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}

/// @brief it should reject invalid security policy with missing certificates.
TEST(ConnectionTest, invalidSecurityPolicy) {
    // Invalid security policy with missing certificates causes the server
    // to reject the identity token
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4853;
    mock::Server server(server_cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());

    driver::opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4853";
    cfg.security_mode = "Sign";
    cfg.security_policy = "InvalidPolicy999";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    ASSERT_OCCURRED_AS_P(
        driver::opc::connection::connect(cfg, "test"),
        driver::opc::errors::IDENTITY_TOKEN_REJECTED
    );

    server.stop();
}
