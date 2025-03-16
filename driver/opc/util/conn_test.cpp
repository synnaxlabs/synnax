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
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/util/util.h"
#include "driver/opc/mock/server.h"

TEST(ConnTest, testBasicConn) {
    synnax::Channel ch;
    ch.data_type = telem::FLOAT32_T;

    mock::ServerChannel server_ch{
        .ns = 1,
        .node = "test",
        .ch = ch
    };

    mock::ServerConfig server_cfg{
        .channels = {server_ch}
    };

    mock::Server server{mock::ServerConfig(server_cfg)};
    server.start();
    util::ConnectionConfig cfg;
    cfg.endpoint = "opc.tcp://0.0.0.0:4840";
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
    ASSERT_EQ(ser.at<float>(0), 5);
}
