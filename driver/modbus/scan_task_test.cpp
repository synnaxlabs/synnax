// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/test/xtest.h"

#include "driver/modbus/mock/slave.h"
#include "driver/modbus/scan_task.h"
#include "driver/task/common/scan_task.h"

/// @brief it should successfully test connection to Modbus device.
TEST(ScanTask, testConnection) {
    auto slave = driver::modbus::mock::Slave(driver::modbus::mock::SlaveConfig{});
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);

    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";

    auto dev_manager = std::make_shared<driver::modbus::device::Manager>();

    const auto cfg = driver::modbus::ScanTaskConfig{};
    auto scan_task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::modbus::Scanner>(ctx, t, dev_manager),
        ctx,
        t,
        x::breaker::default_config(t.name),
        cfg.scan_rate
    );

    auto conn_cfg = driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502};
    auto cmd_args = json{{"connection", conn_cfg.to_json()}};
    auto cmd = driver::task::Command(t.key, driver::modbus::TEST_CONNECTION_CMD_TYPE, cmd_args);
    cmd.key = "electric_boogaloo";

    scan_task->exec(cmd);
    ASSERT_EQ(ctx->statuses.size(), 1);
    auto first = ctx->statuses[0];
    EXPECT_EQ(first.variant, status::variant::SUCCESS);
    EXPECT_EQ(first.key, t.status_key());
    EXPECT_EQ(first.details.cmd, cmd.key);
    EXPECT_EQ(first.details.task, t.key);
    EXPECT_EQ(first.message, "Connection successful");
}

TEST(ScanTask, testConfigReturnsCorrectValues) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<driver::modbus::device::Manager>();

    const driver::modbus::Scanner scanner(ctx, t, dev_manager);
    auto cfg = scanner.config();
    EXPECT_EQ(cfg.make, "modbus");
    EXPECT_EQ(cfg.log_prefix, "[modbus.scan_task]");
}

TEST(ScanTask, testExecReturnsFalseForUnknownCommand) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<driver::modbus::device::Manager>();

    driver::modbus::Scanner scanner(ctx, t, dev_manager);
    driver::task::Command cmd(t.key, "unknown_command", json{});
    bool handled = scanner.exec(cmd, t, ctx);
    EXPECT_FALSE(handled);
}

TEST(ScanTask, testScanChecksDeviceHealth) {
    auto slave = driver::modbus::mock::Slave(driver::modbus::mock::SlaveConfig{});
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<driver::modbus::device::Manager>();

    driver::modbus::Scanner scanner(ctx, t, dev_manager);

    // Create device with valid Modbus connection properties
    synnax::Device dev;
    dev.key = "health-test-device";
    dev.name = "Health Test Device";
    dev.make = "modbus";
    dev.rack = synnax::rack_key_from_task_key(t.key);
    dev.properties = json{
        {"connection",
         {{"host", "127.0.0.1"},
          {"port", 1502},
          {"swap_bytes", false},
          {"swap_words", false}}}
    }.dump();

    std::unordered_map<std::string, synnax::Device> devices_map;
    devices_map[dev.key] = dev;
    driver::task::common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto [devices, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status.variant, status::variant::SUCCESS);
    EXPECT_EQ(devices[0].status.message, "Device connected");
}

TEST(ScanTask, testScanReportsDisconnectedDevice) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::Task t;
    t.key = 12345;
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<driver::modbus::device::Manager>();

    driver::modbus::Scanner scanner(ctx, t, dev_manager);

    // Create device with invalid connection (no server running on this port)
    synnax::Device dev;
    dev.key = "disconnected-device";
    dev.name = "Disconnected Device";
    dev.make = "modbus";
    dev.rack = synnax::rack_key_from_task_key(t.key);
    dev.properties = json{
        {"connection",
         {{"host", "127.0.0.1"},
          {"port", 9999},
          {"swap_bytes", false},
          {"swap_words", false}}}
    }.dump();

    std::unordered_map<std::string, synnax::Device> devices_map;
    devices_map[dev.key] = dev;
    driver::task::common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto [devices, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status.variant, status::variant::WARNING);
    EXPECT_EQ(devices[0].status.message, "Failed to reach device");
}
