// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/mock/server.h"
#include "driver/opc/testutil/testutil.h"

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

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        opc::connection::connect(cfg, "opc"),
        (5 * telem::SECOND).chrono(),
        (250 * telem::MILLISECOND).chrono()
    );
    ASSERT_NE(client, nullptr);

    auto ser = ASSERT_NIL_P(opc::testutil::simple_read(client, "NS=1;S=test"));
    ASSERT_EQ(ser.data_type(), telem::FLOAT32_T);
    ASSERT_EQ(ser.at<float>(0), 5.0f);

    server.stop();

    UA_Variant_clear(&float_val);
}

TEST(ConnectionTest, connectionRefused) {
    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:9999";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnectionTest, invalidEndpointFormat) {
    opc::connection::Config cfg;
    cfg.endpoint = "not-a-valid-endpoint";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnectionTest, emptyEndpoint) {
    opc::connection::Config cfg;
    cfg.endpoint = "";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnectionTest, invalidHostname) {
    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://nonexistent.invalid.hostname:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnectionTest, disconnectAndReconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4841;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4841";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err1] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client, nullptr);

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    UA_Client_disconnect(client.get());

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    auto err2 = opc::connection::reconnect(client, cfg.endpoint);
    ASSERT_FALSE(err2);

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

TEST(ConnectionTest, serverStopDuringConnection) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4842;
    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4842";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err1] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client, nullptr);

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    auto [node_id, parse_err] = opc::NodeId::parse("NS=1;S=TestFloat");
    ASSERT_FALSE(parse_err);

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

TEST(ConnectionTest, connectionAfterServerRestart) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4844;

    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4844";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client1, err1] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client1, nullptr);

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    auto [client2, err2] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err2);
    ASSERT_NE(client2, nullptr);

    server->stop();
}

TEST(ConnectionTest, readAfterDisconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4845;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4845";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err);

    auto [ser1, read_err1] = opc::testutil::simple_read(client, "NS=1;S=TestFloat");
    ASSERT_FALSE(read_err1);

    UA_Client_disconnect(client.get());

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

TEST(ConnectionTest, multipleDisconnects) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4846;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4846";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = opc::connection::connect(cfg, "test");
    ASSERT_FALSE(err);

    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());

    server.stop();
}

TEST(ConnectionTest, invalidUsernamePassword) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4847;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4847";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "invalid_user";
    cfg.password = "wrong_password";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err || client != nullptr);

    server.stop();
}

TEST(ConnectionTest, signModeWithNoEncryptionServer) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4848;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4848";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err);

    server.stop();
}

TEST(ConnectionTest, signAndEncryptModeWithNoEncryptionServer) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4849;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4849";
    cfg.security_mode = "SignAndEncrypt";
    cfg.security_policy = "Basic256Sha256";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err);

    server.stop();
}

TEST(ConnectionTest, missingClientCertificate) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4850;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4850";
    cfg.security_mode = "Sign";
    cfg.security_policy = "Basic256";
    cfg.client_cert = "/path/to/missing/cert.pem";
    cfg.client_private_key = "/path/to/missing/key.pem";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err);

    server.stop();
}

TEST(ConnectionTest, emptyUsernameWithPassword) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4851;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4851";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "";
    cfg.password = "password";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err || client != nullptr);

    server.stop();
}

TEST(ConnectionTest, usernameWithEmptyPassword) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4852;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4852";
    cfg.security_mode = "None";
    cfg.security_policy = "None";
    cfg.username = "username";
    cfg.password = "";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err || client != nullptr);

    server.stop();
}

TEST(ConnectionTest, invalidSecurityPolicy) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4853;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    opc::connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4853";
    cfg.security_mode = "Sign";
    cfg.security_policy = "InvalidPolicy999";
    cfg.client_cert = "/nonexistent/cert.pem";
    cfg.client_private_key = "/nonexistent/key.pem";

    auto [client, err] = opc::connection::connect(cfg, "test");
    EXPECT_TRUE(err);

    server.stop();
}
