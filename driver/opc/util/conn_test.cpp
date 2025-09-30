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
        .ns = 1,
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
