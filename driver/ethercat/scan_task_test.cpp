// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/mock/master.h"
#include "driver/ethercat/scan_task.h"

class EtherCATScanTest : public ::testing::Test {
protected:
    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        rack = ASSERT_NIL_P(client->racks.create("test_rack"));
        ctx = std::make_shared<task::MockContext>(client);
    }

    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    synnax::Rack rack;
};

TEST_F(EtherCATScanTest, GenerateNetworkKey) {
    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    ethercat::InterfaceInfo iface{"eth0", "Ethernet Adapter"};
    std::vector<ethercat::SlaveInfo> slaves;
    slaves.emplace_back(1, 0x1, 0x2, 0, 12345, "Test Slave", ethercat::SlaveState::OP);

    auto dev = scanner.scan({}).first;
}

TEST_F(EtherCATScanTest, GenerateSlaveKeyWithSerial) {
    ethercat::SlaveInfo
        slave(1, 1000, 2000, 1, 12345, "EL3004", ethercat::SlaveState::OP);
    std::string expected = "ethercat_1000_2000_12345";

    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    auto [devices, err] = scanner.scan({});
    ASSERT_NIL(err);
}

TEST_F(EtherCATScanTest, GenerateSlaveKeyWithoutSerial) {
    ethercat::SlaveInfo slave(3, 1000, 2000, 1, 0, "EL3004", ethercat::SlaveState::OP);
    std::string expected = "ethercat_eth0_1000_2000_3";

    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    auto [devices, err] = scanner.scan({});
    ASSERT_NIL(err);
}

TEST_F(EtherCATScanTest, ScanConfigParsesCorrectly) {
    json cfg_json = {{"scan_rate", 2.0}, {"enabled", true}, {"backend", "soem"}};

    xjson::Parser parser(cfg_json);
    ethercat::ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_EQ(cfg.scan_rate.hz(), 2.0);
    EXPECT_TRUE(cfg.enabled);
    EXPECT_EQ(cfg.backend, "soem");
}

TEST_F(EtherCATScanTest, ScanConfigDefaultValues) {
    json cfg_json = {};

    xjson::Parser parser(cfg_json);
    ethercat::ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_EQ(cfg.backend, "auto");
    EXPECT_TRUE(cfg.enabled);
}

TEST_F(EtherCATScanTest, ScannerConfigReturnsCorrectValues) {
    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    auto config = scanner.config();
    EXPECT_EQ(config.make, ethercat::INTEGRATION_NAME);
    EXPECT_EQ(config.log_prefix, ethercat::SCAN_LOG_PREFIX);
}

TEST_F(EtherCATScanTest, ScannerStartStopSucceed) {
    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    ASSERT_NIL(scanner.start());
    ASSERT_NIL(scanner.stop());
}

TEST_F(EtherCATScanTest, TestInterfaceCommandWithInvalidArgs) {
    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    task::Command cmd;
    cmd.type = ethercat::TEST_INTERFACE_CMD_TYPE;
    cmd.args = "{}";
    cmd.key = 1;

    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_TRUE(handled);
    EXPECT_FALSE(ctx->statuses.empty());
}

TEST_F(EtherCATScanTest, UnknownCommandNotHandled) {
    synnax::Task task(rack.key, "EtherCAT Scanner", ethercat::SCAN_TASK_TYPE, "", true);
    ethercat::ScanTaskConfig cfg;
    ethercat::Scanner scanner(ctx, task, cfg, nullptr);

    task::Command cmd;
    cmd.type = "unknown_command";
    cmd.args = "{}";

    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_FALSE(handled);
}
