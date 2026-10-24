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
    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    const ScanTaskConfig cfg;
    const Scanner scanner(ctx, task, cfg, nullptr);
    EXPECT_EQ(scanner.config().make, INTEGRATION_NAME);
}

TEST_F(EtherCATScanTest, ScanConfigParsesCorrectly) {
    x::json::json cfg_json = {{"rate", 2.0}, {"disabled", false}};

    x::json::Parser parser(cfg_json);
    ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_EQ(cfg.rate.hz(), 2.0);
    EXPECT_FALSE(cfg.disabled);
}

TEST_F(EtherCATScanTest, ScanConfigDefaultValues) {
    x::json::json cfg_json = {};

    x::json::Parser parser(cfg_json);
    ScanTaskConfig cfg(parser);

    ASSERT_NIL(parser.error());
    EXPECT_FALSE(cfg.disabled);
}

TEST_F(EtherCATScanTest, ScannerConfigReturnsCorrectValues) {
    synnax::task::Task task{
        .rack = rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true,
    };
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    auto config = scanner.config();
    EXPECT_EQ(config.make, INTEGRATION_NAME);
    EXPECT_EQ(config.log_prefix, SCAN_LOG_PREFIX);
}

TEST_F(EtherCATScanTest, ScannerStartStopSucceed) {
    synnax::task::Task task{
        .rack = rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true,
    };
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    ASSERT_NIL(scanner.start());
    ASSERT_NIL(scanner.stop());
}

TEST_F(EtherCATScanTest, TestInterfaceCommandWithInvalidArgs) {
    synnax::task::Task task{
        .rack = rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true,
    };
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    synnax::task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = x::json::json::object_t{};
    cmd.key = '1';

    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_TRUE(handled);
    EXPECT_FALSE(ctx->statuses.empty());
}

TEST_F(EtherCATScanTest, UnknownCommandNotHandled) {
    synnax::task::Task task{
        .rack = rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true,
    };
    ScanTaskConfig cfg;
    Scanner scanner(ctx, task, cfg, nullptr);

    synnax::task::Command cmd;
    cmd.type = "unknown_command";
    cmd.args = x::json::json::object_t{};

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

    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    synnax::task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "eth0"}};
    cmd.key = '1';

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, synnax::status::VARIANT_SUCCESS);
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

    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    synnax::task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "enp3s0"}};
    cmd.key = '2';

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, synnax::status::VARIANT_SUCCESS);
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

    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    ScanTaskConfig cfg;
    Scanner scanner(this->ctx, task, cfg, pool);

    synnax::task::Command cmd;
    cmd.type = TEST_INTERFACE_CMD_TYPE;
    cmd.args = {{"interface", "eth0"}};
    cmd.key = '3';

    bool handled = scanner.exec(cmd, task, this->ctx);
    EXPECT_TRUE(handled);
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().variant, synnax::status::VARIANT_ERROR);
}

/// @brief a scanner over a mock bus with one slave, for device set specs.
class EtherCATScanDeviceSetTest : public EtherCATScanTest {
protected:
    std::shared_ptr<mock::Master> master;
    std::unique_ptr<Scanner> scanner;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        this->master = std::make_shared<mock::Master>("eth0");
        this->master->add_slave(slave::Properties{.position = 0, .name = "EL3102"});
        auto manager = std::make_unique<mock::Manager>();
        manager->configure("eth0", this->master);
        this->scanner = std::make_unique<Scanner>(
            this->ctx,
            synnax::task::Task{
                .rack = this->rack.key,
                .name = "EtherCAT Scanner",
                .type = SCAN_TASK_TYPE,
                .internal = true,
            },
            ScanTaskConfig{},
            std::make_shared<engine::Pool>(std::move(manager))
        );
    }

    /// @brief creates a slave device with the given startup parameters.
    synnax::device::Device create_slave(const x::json::json &startup_parameters) {
        synnax::device::Device dev{
            .key = "ecat_startup_" + std::to_string(this->rack.key),
            .rack = this->rack.key,
            .location = "eth0.Slot 0",
            .make = INTEGRATION_NAME,
            .model = SLAVE_DEVICE_MODEL,
            .name = "EL3102",
            .properties = {
                {"serial", 0},
                {"vendor_id", 0x2},
                {"product_code", 0xC1E3052},
                {"revision", 1},
                {"name", "EL3102"},
                {"network", "eth0"},
                {"position", 0},
                {"enabled", true},
                {"pdos",
                 {{"inputs", x::json::json::array()},
                  {"outputs", x::json::json::array()}}},
                {"startup_parameters", startup_parameters},
            },
        };
        const auto err = this->client->devices.create(dev);
        EXPECT_FALSE(err) << err;
        return dev;
    }

    [[nodiscard]] synnax::device::Status
    status(const synnax::device::Device &dev) const {
        return ASSERT_NIL_P(
            this->client->statuses.retrieve<synnax::device::StatusDetails>(
                synnax::device::status_key(dev)
            )
        );
    }
};

/// @brief it should pass the startup parameters of a changed device to the master.
TEST_F(EtherCATScanDeviceSetTest, PassesStartupParametersToTheMaster) {
    const auto dev = this->create_slave(
        {{{"index", 0x8000},
          {"sub_index", 0x15},
          {"data_type", "uint16"},
          {"value", 3},
          {"name", "Filter"}}}
    );
    this->scanner->on_device_set(dev);
    EXPECT_EQ(
        this->master->startup_parameters(0),
        (std::vector<slave::StartupParameter>{{
            .index = 0x8000,
            .sub_index = 0x15,
            .data_type = x::telem::UINT16_T,
            .value = 3,
            .name = "Filter",
        }})
    );
    EXPECT_EQ(this->status(dev).message, "Device enabled");
}

/// @brief it should pass the startup parameters of existing devices to the master on
/// the first scan.
TEST_F(EtherCATScanDeviceSetTest, FirstScanPassesStartupParametersToTheMaster) {
    const auto dev = this->create_slave(
        {{{"index", 0x8000},
          {"sub_index", 0x15},
          {"data_type", "uint16"},
          {"value", 3},
          {"name", "Filter"}}}
    );
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    ASSERT_NIL_P(
        this->scanner->scan(common::ScannerContext{.count = 0, .devices = &devices})
    );
    EXPECT_EQ(
        this->master->startup_parameters(0),
        (std::vector<slave::StartupParameter>{{
            .index = 0x8000,
            .sub_index = 0x15,
            .data_type = x::telem::UINT16_T,
            .value = 3,
            .name = "Filter",
        }})
    );
}

/// @brief it should pass no startup parameters and report an error status that names
/// the invalid field.
TEST_F(EtherCATScanDeviceSetTest, ReportsAnInvalidStartupParameter) {
    const auto dev = this->create_slave(
        {{{"index", 0x8000},
          {"sub_index", 0x15},
          {"data_type", "uint8"},
          {"value", 300}}}
    );
    this->scanner->on_device_set(dev);
    EXPECT_TRUE(this->master->startup_parameters(0).empty());
    const auto status = this->status(dev);
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Invalid device properties");
    EXPECT_EQ(
        status.description,
        "startup_parameters.0.value: 300 does not fit in uint8"
    );
}
}
