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
#include "x/cpp/test/test.h"

#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/mock/master.h"
#include "driver/ethercat/scan_task.h"

namespace driver::ethercat {
class EtherCATScanTest : public ::testing::Test {
protected:
    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->rack = ASSERT_NIL_P(this->client->racks.create("test_rack"));
        this->ctx = std::make_shared<task::MockContext>(this->client);
    }

    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    synnax::rack::Rack rack;
};

TEST_F(EtherCATScanTest, ScannerCreation) {
    const synnax::task::Task
        task(rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    const ScanTaskConfig cfg;
    const Scanner scanner(ctx, task, cfg, nullptr);
    EXPECT_EQ(scanner.config().make, INTEGRATION_NAME);
}

TEST_F(EtherCATScanTest, ScanConfigParsesCorrectly) {
    x::json::json cfg_json = {{"scan_rate", 2.0}, {"enabled", true}};

    x::json::Parser parser(cfg_json);
    ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_EQ(cfg.scan_rate.hz(), 2.0);
    EXPECT_TRUE(cfg.enabled);
}

TEST_F(EtherCATScanTest, ScanConfigDefaultValues) {
    x::json::json cfg_json = {};

    x::json::Parser parser(cfg_json);
    ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_TRUE(cfg.enabled);
}

TEST_F(EtherCATScanTest, ScannerConfigReturnsCorrectValues) {
    synnax::task::Task task(rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    auto config = scanner.config();
    EXPECT_EQ(config.make, INTEGRATION_NAME);
    EXPECT_EQ(config.log_prefix, SCAN_LOG_PREFIX);
}

TEST_F(EtherCATScanTest, ScannerStartStopSucceed) {
    synnax::task::Task task(rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    ASSERT_NIL(scanner.start());
    ASSERT_NIL(scanner.stop());
}

TEST_F(EtherCATScanTest, TestInterfaceCommandWithInvalidArgs) {
    synnax::task::Task task(rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = "{}";
    cmd.key = 1;

    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_TRUE(handled);
    EXPECT_FALSE(ctx->statuses.empty());
}

TEST_F(EtherCATScanTest, UnknownCommandNotHandled) {
    synnax::task::Task task(rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    task::Command cmd;
    cmd.type = "unknown_command";
    cmd.args = "{}";

    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_FALSE(handled);
}

TEST_F(EtherCATScanTest, TestInterfaceCommandSuccess) {
    auto mock_master = std::make_shared<mock::Master>("eth0");
    mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 12345,
            .name = "Test Slave",
        }
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("eth0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager));

    synnax::task::Task
        task(this->rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "eth0"}};
    cmd.key = 1;

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, x::status::VARIANT_SUCCESS);
    EXPECT_NE(this->ctx->statuses.back().message.find("1 slaves"), std::string::npos);
}

TEST_F(EtherCATScanTest, TestInterfaceCommandWithMultipleSlaves) {
    auto mock_master = std::make_shared<mock::Master>("enp3s0");
    mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 11111,
            .name = "Slave 1",
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x1,
            .product_code = 0x3,
            .serial = 22222,
            .name = "Slave 2",
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 2,
            .vendor_id = 0x1,
            .product_code = 0x4,
            .serial = 33333,
            .name = "Slave 3",
        }
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("enp3s0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager));

    synnax::task::Task
        task(this->rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "enp3s0"}};
    cmd.key = 2;

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, x::status::VARIANT_SUCCESS);
    EXPECT_NE(this->ctx->statuses.back().message.find("3 slaves"), std::string::npos);
}

TEST_F(EtherCATScanTest, TestInterfaceCommandInitError) {
    auto mock_master = std::make_shared<mock::Master>("eth0");
    mock_master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("eth0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager));

    synnax::task::Task
        task(this->rack.key, "EtherCAT Scanner", SCAN_TASK_TYPE, "", true);
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "eth0"}};
    cmd.key = 3;

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, x::status::VARIANT_ERROR);
}
}
