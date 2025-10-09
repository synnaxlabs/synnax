// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/modbus/scan_task.h"

#include "driver/modbus/mock/slave.h"
#include "x/cpp/xtest/xtest.h"

TEST(ScanTask, testConnection) {
    auto slave = modbus::mock::Slave(modbus::mock::SlaveConfig{});
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";

    auto dev_manager = std::make_shared<modbus::device::Manager>();

    auto conn_cfg = modbus::device::ConnectionConfig{"127.0.0.1", 1502};
    auto cmd_args = json{{"connection", conn_cfg.to_json()}};
    auto cmd = task::Command(t.key, modbus::TEST_CONNECTION_CMD_TYPE, cmd_args);
    cmd.key = "electric_boogaloo";

    auto scan_task = modbus::ScanTask(ctx, t, dev_manager);

    scan_task.exec(cmd);
    ASSERT_EQ(ctx->states.size(), 1);
    auto first = ctx->states[0];
    EXPECT_EQ(first.variant, "success");
    EXPECT_EQ(first.key, cmd.key);
    EXPECT_EQ(first.details.task, t.key);
    EXPECT_EQ(first.message, "Connection successful");
}
