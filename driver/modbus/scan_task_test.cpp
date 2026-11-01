// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <chrono>
#include <thread>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/scan_task.h"
#include "driver/modbus/scan_task.h"
#include "driver/modbus/testutil/slave.h"

namespace driver::modbus {
/// @brief it should successfully test connection to Modbus device.
TEST(ScanTask, testConnection) {
    auto slave = testutil::Slave(testutil::SlaveConfig{.port = 1664});
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::task::Task t;
    t.key = x::uuid::create();
    t.type = "modbus_scan";

    auto dev_manager = std::make_shared<device::Manager>();

    const auto cfg = ScanTaskConfig{};
    auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, t, dev_manager),
        ctx,
        t,
        x::breaker::default_config(t.name),
        cfg.rate
    );

    auto conn_cfg = device::ConnectionConfig{"127.0.0.1", 1664};
    auto cmd_args = x::json::json{{"connection", conn_cfg.to_json()}};
    auto cmd = synnax::task::Command{
        .task = t.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "electric_boogaloo",
        .args = cmd_args
    };

    scan_task->exec(cmd);
    ASSERT_EQ(ctx->statuses.size(), 1);
    auto first = ctx->statuses[0];
    EXPECT_EQ(first.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first.key, synnax::task::status_key(t));
    EXPECT_EQ(first.details.cmd, cmd.key);
    EXPECT_EQ(first.details.task, t.key);
    EXPECT_EQ(first.message, "Connected to device");
}

TEST(ScanTask, testConfigReturnsCorrectValues) {
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<device::Manager>();

    const Scanner scanner(ctx, t, dev_manager);
    auto cfg = scanner.config();
    EXPECT_EQ(cfg.make, "Modbus");
    EXPECT_EQ(cfg.log_prefix, "[modbus.scan_task]");
}

TEST(ScanTask, testExecReturnsFalseForUnknownCommand) {
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<device::Manager>();

    Scanner scanner(ctx, t, dev_manager);
    synnax::task::Command cmd{.task = t.key, .type = "unknown_command"};
    bool handled = scanner.exec(cmd, t, ctx);
    EXPECT_FALSE(handled);
}

TEST(ScanTask, testScanChecksDeviceHealth) {
    auto slave = testutil::Slave(testutil::SlaveConfig{.port = 1665});
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<device::Manager>();

    Scanner scanner(ctx, t, dev_manager);

    // Create device with valid Modbus connection properties
    synnax::device::Device dev;
    dev.key = "health-test-device";
    dev.name = "Health Test Device";
    dev.make = MAKE;
    dev.rack = t.rack;
    dev.properties = x::json::json{
        {"connection",
         {{"host", "127.0.0.1"},
          {"port", 1665},
          {"swap_bytes", false},
          {"swap_words", false}}}
    };

    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto devices = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(devices[0].status.value().message, "Device connected");
}

TEST(ScanTask, testScanReportsDisconnectedDevice) {
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.type = "modbus_scan";
    auto dev_manager = std::make_shared<device::Manager>();

    Scanner scanner(ctx, t, dev_manager);

    // Create device with invalid connection (no server running on this port)
    synnax::device::Device dev;
    dev.key = "disconnected-device";
    dev.name = "Disconnected Device";
    dev.make = MAKE;
    dev.rack = t.rack;
    dev.properties = x::json::json{
        {"connection",
         {{"host", "127.0.0.1"},
          {"port", 9999},
          {"swap_bytes", false},
          {"swap_words", false}}}
    };

    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto devices = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(devices[0].status.value().message, "Failed to reach device");
}

namespace {
/// @returns a Modbus device on the local test slave.
synnax::device::Device local_device(
    const std::string &key,
    const int port,
    const int unit_id = MODBUS_TCP_SLAVE
) {
    synnax::device::Device dev;
    dev.key = key;
    dev.name = key;
    dev.make = MAKE;
    dev.properties = x::json::json{
        {"connection",
         {{"host", "127.0.0.1"},
          {"port", port},
          {"swap_bytes", false},
          {"swap_words", false},
          {"unit_id", unit_id},
          {"timeout_ms", 50},
          {"retries", 0}}}
    };
    return dev;
}

/// @returns the devices of one scan of a single device, with their statuses.
std::vector<synnax::device::Device>
scan(Scanner &scanner, const synnax::device::Device &dev) {
    std::unordered_map<std::string, synnax::device::Device> devices{{dev.key, dev}};
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;
    auto [out, err] = scanner.scan(scan_ctx);
    EXPECT_FALSE(err) << err;
    return out;
}
}

/// @brief it should keep the connection to a device open between scans.
TEST(ScanTask, testScanKeepsConnectionOpen) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1660;
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    const auto dev = local_device("kept", 1660);

    for (int i = 0; i < 3; i++)
        EXPECT_EQ(
            scan(scanner, dev)[0].status.value().variant,
            synnax::status::VARIANT_SUCCESS
        );
    EXPECT_EQ(slave.connections(), 1);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 3);
}

/// @brief it should skip the probe of a device that answered a task since the last
/// scan.
TEST(ScanTask, testScanSkipsProbeAfterTaskReply) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1661;
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    auto devs = std::make_shared<device::Manager>();
    Scanner scanner(ctx, t, devs);
    const auto dev = local_device("busy", 1661, 2);
    device::ConnectionConfig task_conn{"127.0.0.1", 1661};
    task_conn.unit_id = 2;
    auto task_dev = ASSERT_NIL_P(devs->acquire(task_conn));

    scan(scanner, dev);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 1);
    uint8_t bit = 0;
    ASSERT_NIL(task_dev->read_bits(device::Coil, 0, 1, &bit));
    const auto statuses = scan(scanner, dev);
    EXPECT_EQ(statuses[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 1);
    scan(scanner, dev);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 2);
}

/// @brief it should report a device whose unit does not answer, even though its server
/// accepts the connection.
TEST(ScanTask, testScanReportsSilentUnit) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1662;
    slave_cfg.silent_units = {5};
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());

    const auto status = scan(scanner, local_device("silent", 1662, 5))[0]
                            .status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Failed to reach device");
    EXPECT_EQ(status.description, modbus_strerror(ETIMEDOUT));
    EXPECT_EQ(status.details.data.value().at("stats")["timeouts"], 1);
}

/// @brief it should report the request counters of a device in its status.
TEST(ScanTask, testScanReportsRequestStats) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1678;
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    const auto dev = local_device("counted", 1678);

    scan(scanner, dev);
    const auto status = scan(scanner, dev)[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    const auto stats = status.details.data.value().at("stats");
    EXPECT_EQ(stats["requests"], 2);
    EXPECT_EQ(stats["retries"], 0);
    EXPECT_EQ(stats["timeouts"], 0);
    EXPECT_EQ(stats["exceptions"], 0);
    EXPECT_GT(stats["max_response"].get<std::int64_t>(), 0);
}

/// @brief it should test a connection while a scan waits on a silent server.
TEST(ScanTask, testConnectionDuringSlowScan) {
    testutil::SlaveConfig silent_cfg;
    silent_cfg.port = 1666;
    silent_cfg.silent_units = {5};
    testutil::Slave silent(silent_cfg);
    ASSERT_NIL(silent.start());
    x::defer::defer stop_silent([&silent] { silent.stop(); });
    testutil::Slave live(testutil::SlaveConfig{.port = 1667});
    ASSERT_NIL(live.start());
    x::defer::defer stop_live([&live] { live.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    auto slow = local_device("slow", 1666, 5);
    slow.properties["connection"]["timeout_ms"] = 2000;
    std::thread scanning([&] { scan(scanner, slow); });
    x::defer::defer join([&scanning] { scanning.join(); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    synnax::task::Command cmd{
        .task = t.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "test",
        .args = {
            {"connection", local_device("live", 1667).properties.at("connection")}
        },
    };

    const auto start = std::chrono::steady_clock::now();
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(1));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
}

/// @brief it should wait the request delay of the tested device after a task's request
/// on the shared connection.
TEST(ScanTask, testConnectionWaitsRequestDelay) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1680;
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    auto devs = std::make_shared<device::Manager>();
    Scanner scanner(ctx, t, devs);
    const auto dev = local_device("paced", 1680);
    auto task_dev = ASSERT_NIL_P(
        devs->acquire(device::ConnectionConfig{"127.0.0.1", 1680})
    );
    synnax::task::Command cmd{
        .task = t.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "test",
        .args = {
            {"connection", dev.properties.at("connection")},
            {"requests", {{"delay_ms", 300}}},
        },
    };

    uint8_t bit = 0;
    ASSERT_NIL(task_dev->read_bits(device::Coil, 0, 1, &bit));
    const auto start = std::chrono::steady_clock::now();
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    EXPECT_GE(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(250));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
}

/// @brief it should fail a connection test when the unit does not answer.
TEST(ScanTask, testConnectionFailsForSilentUnit) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1663;
    slave_cfg.silent_units = {5};
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    const auto dev = local_device("silent", 1663, 5);
    synnax::task::Command cmd{
        .task = t.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "test",
        .args = {{"connection", dev.properties.at("connection")}},
    };

    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to connect to device");
    EXPECT_EQ(ctx->statuses[0].description, modbus_strerror(ETIMEDOUT));
}
}
