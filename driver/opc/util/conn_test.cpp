// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "gtest/gtest.h"

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/mock/server.h"
#include "driver/opc/util/util.h"

TEST(ConnTest, testBasicConn) {
    UA_Variant float_val;
    UA_Variant_init(&float_val);
    UA_Float float_data = 5.0f;
    UA_Variant_setScalarCopy(&float_val, &float_data, &UA_TYPES[UA_TYPES_FLOAT]);

    mock::TestNode node{
        re.ns = 1,
        .node_id = "test",
        .data_type = &UA_TYPES[UA_TYPES_FLOAT],
        .initial_value = float_val,
        .description = "Test Float Node"
    };

    mock::ServerConfig server_cfg;
    server_cfg.test_nodes = {node};
    server_cfg.port = 4840;

    mock::Server server(server_cfg);
    server.start();

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        util::connect(cfg, "opc"),
        (5 * telem::SECOND).chrono(),
        (250 * telem::MILLISECOND).chrono()
    );
    ASSERT_NE(client, nullptr);

    auto ser = ASSERT_NIL_P(util::simple_read(client, "NS=1;S=test"));
    ASSERT_EQ(ser.data_type(), telem::FLOAT32_T);
    ASSERT_EQ(ser.at<float>(0), 5.0f);

    server.stop();
}

TEST(ConnTest, connectionRefused) {
    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:9999";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnTest, invalidEndpointFormat) {
    util::ConnectionConfig cfg;
    cfg.endpoint = "not-a-valid-endpoint";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnTest, emptyEndpoint) {
    util::ConnectionConfig cfg;
    cfg.endpoint = "";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnTest, invalidHostname) {
    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://nonexistent.invalid.hostname:4840";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_TRUE(err);
}

TEST(ConnTest, disconnectAndReconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4841;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4841";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err1] = util::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client, nullptr);

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    UA_Client_disconnect(client.get());

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    auto err2 = util::reconnect(client, cfg.endpoint);
    ASSERT_FALSE(err2);

    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

TEST(ConnTest, serverStopDuringConnection) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4842;
    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4842";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err1] = util::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client, nullptr);

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    auto [node_id, parse_err] = util::parse_node_id("NS=1;S=TestFloat");
    ASSERT_FALSE(parse_err);

    UA_ReadValueId ids[1];
    UA_ReadValueId_init(&ids[0]);
    ids[0].nodeId = node_id;
    ids[0].attributeId = UA_ATTRIBUTEID_VALUE;

    UA_ReadRequest req;
    UA_ReadRequest_init(&req);
    req.nodesToRead = ids;
    req.nodesToReadSize = 1;

    UA_ReadResponse res = UA_Client_Service_read(client.get(), req);
    EXPECT_NE(res.responseHeader.serviceResult, UA_STATUSCODE_GOOD);
    UA_ReadResponse_clear(&res);
}

TEST(ConnTest, concurrentConnections) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4843;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4843";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};
    std::atomic<int> failure_count{0};

    for (int i = 0; i < 5; ++i) {
        threads.emplace_back([&cfg, &success_count, &failure_count]() {
            auto [client, err] = util::connect(cfg, "test");
            if (!err && client) {
                success_count++;
            } else {
                failure_count++;
            }
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(success_count, 5);
    EXPECT_EQ(failure_count, 0);

    server.stop();
}

TEST(ConnTest, connectionAfterServerRestart) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4844;

    auto server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4844";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client1, err1] = util::connect(cfg, "test");
    ASSERT_FALSE(err1);
    ASSERT_NE(client1, nullptr);

    server->stop();
    server.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    server = std::make_unique<mock::Server>(server_cfg);
    server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    auto [client2, err2] = util::connect(cfg, "test");
    ASSERT_FALSE(err2);
    ASSERT_NE(client2, nullptr);

    server->stop();
}

TEST(ConnTest, readAfterDisconnect) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4845;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4845";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_FALSE(err);

    auto [ser1, read_err1] = util::simple_read(client, "NS=1;S=TestFloat");
    ASSERT_FALSE(read_err1);

    UA_Client_disconnect(client.get());

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
    EXPECT_NE(session_state, UA_SESSIONSTATE_ACTIVATED);

    server.stop();
}

TEST(ConnTest, multipleDisconnects) {
    mock::ServerConfig server_cfg = mock::ServerConfig::create_default();
    server_cfg.port = 4846;
    mock::Server server(server_cfg);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://localhost:4846";
    cfg.security_mode = "None";
    cfg.security_policy = "None";

    auto [client, err] = util::connect(cfg, "test");
    ASSERT_FALSE(err);

    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());
    UA_Client_disconnect(client.get());

    server.stop();
}
