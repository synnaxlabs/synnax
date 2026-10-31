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
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/scan_task.h"
#include "driver/ethercat/testutil/master.h"

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

    [[nodiscard]] synnax::task::Task scan_task() const {
        return synnax::task::Task{
            .rack = this->rack.key,
            .name = "EtherCAT Scanner",
            .type = SCAN_TASK_TYPE,
            .internal = true
        };
    }
};

class EtherCATScanStatusTest : public EtherCATScanTest {
protected:
    const pdo::Entry entry{0, 0x6000, 1, 16, true, x::telem::INT16_T};
    std::shared_ptr<testutil::Master> master;
    std::shared_ptr<engine::Pool> pool;
    std::unique_ptr<Scanner> scanner;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        this->master = std::make_shared<testutil::Master>("eth0");
        this->master->add_slave(
            slave::Properties{
                .position = 0,
                .vendor_id = 0x1,
                .product_code = 0x2,
                .serial = 12345,
                .name = "Test Slave",
                .input_pdos = {
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 1,
                     .bit_length = 16,
                     .is_input = true,
                     .data_type = x::telem::INT16_T}
                },
            }
        );
        auto manager = std::make_unique<testutil::Manager>();
        manager->configure("eth0", this->master);
        this->pool = std::make_shared<engine::Pool>(std::move(manager));
        this->scanner = std::make_unique<Scanner>(
            this->ctx,
            this->scan_task(),
            ScanTaskConfig{},
            this->pool
        );
    }

    /// @brief scans once and returns the status of the only slave device.
    [[nodiscard]] synnax::device::Status scan_status() const {
        const auto devices = ASSERT_NIL_P(
            this->scanner->scan(common::ScannerContext{.count = 1})
        );
        EXPECT_EQ(devices.size(), 1);
        return devices.at(0).status.value();
    }
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
    ASSERT_FALSE(ctx->statuses.empty());
    EXPECT_EQ(ctx->statuses.back().message, "Failed to parse test command");
    EXPECT_EQ(ctx->statuses.back().description, "interface: This field is required");
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
    auto mock_master = std::make_shared<testutil::Master>("eth0");
    mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 12345,
            .name = "Test Slave",
        }
    );

    auto manager = std::make_unique<testutil::Manager>();
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
    auto mock_master = std::make_shared<testutil::Master>("enp3s0");
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

    auto manager = std::make_unique<testutil::Manager>();
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

/// @brief it should return a permission error from a scan, so the error reaches the
/// status of the scan task.
TEST_F(EtherCATScanTest, ScanReturnsPermissionDenied) {
    auto mock_master = std::make_shared<testutil::Master>("eth0");
    mock_master->inject_init_error(
        x::errors::Error(errors::PERMISSION_DENIED, "permission denied opening eth0")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager));
    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    Scanner scanner(this->ctx, task, ScanTaskConfig{}, pool);
    ASSERT_OCCURRED_AS_P(scanner.scan({}), errors::PERMISSION_DENIED);
}

TEST_F(EtherCATScanTest, TestInterfaceCommandInitError) {
    auto mock_master = std::make_shared<testutil::Master>("eth0");
    mock_master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );

    auto manager = std::make_unique<testutil::Manager>();
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
    EXPECT_EQ(this->ctx->statuses.back().message, "Failed to probe interface");
    EXPECT_EQ(this->ctx->statuses.back().description, "no interface");
}

/// @brief a scanner over a mock bus on eth0, for scan specs.
class EtherCATScanProbeTest : public EtherCATScanTest {
protected:
    std::shared_ptr<testutil::Master> master;
    std::unique_ptr<Scanner> scanner;
    std::size_t scans = 0;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        this->master = std::make_shared<testutil::Master>("eth0");
        auto manager = std::make_unique<testutil::Manager>();
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

    void scan() { ASSERT_NIL_P(this->scanner->scan({.count = this->scans++})); }
};

/// @brief it should probe an interface with no slaves only on every sixth scan.
TEST_F(EtherCATScanProbeTest, ProbesAnInterfaceWithNoSlavesEverySixthScan) {
    this->master->inject_init_error(
        x::errors::Error(errors::NO_SLAVES_FOUND, "no EtherCAT slaves found on eth0")
    );
    for (int i = 0; i < 7; ++i)
        this->scan();
    EXPECT_EQ(this->master->init_call_count(), 2);
}

/// @brief it should return a partial scan error for an interface it cannot open, then
/// scan the interface again when it opens.
TEST_F(EtherCATScanProbeTest, ReturnsAnInterfaceItCannotScanAsAPartialScan) {
    this->master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "cannot open eth0")
    );
    const auto [devices, err] = this->scanner->scan({});
    ASSERT_OCCURRED_AS(err, common::PARTIAL_SCAN_ERROR);
    EXPECT_EQ(err.data, "cannot open eth0");
    EXPECT_TRUE(devices.empty());
    this->master->inject_init_error(x::errors::NIL);
    ASSERT_NIL_P(this->scanner->scan({.count = 1}));
}

/// @brief it should fail to configure the scan task and report an error status when the
/// packet capture library is missing.
TEST_F(EtherCATScanTest, ScanTaskFailsToConfigureWithoutAPacketCaptureLibrary) {
    Factory factory(std::unique_ptr<master::Manager>{});
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true,
    };
    const auto [configured, handled] = factory.configure_task(
        this->ctx,
        task,
        driver::task::NO_COMMAND
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(configured, nullptr);
    ASSERT_EQ(this->ctx->statuses.size(), 1);
    EXPECT_EQ(this->ctx->statuses.back().variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(this->ctx->statuses.back().message, "Failed to configure task");
    EXPECT_EQ(
        this->ctx->statuses.back().description,
        "Npcap (packet capture for EtherCAT) library is not installed. Download here: "
        "https://npcap.com/#download. Restart Driver after installation."
    );
}

/// @brief it should show the discovery result in the status of a slave while the bus is
/// stopped.
TEST_F(EtherCATScanStatusTest, SlaveOnStoppedBusShowsDiscovery) {
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Discovered (1 inputs, 0 outputs)");
}

/// @brief it should show a slave in OP on a running bus as operational.
TEST_F(EtherCATScanStatusTest, SlaveInOpShowsOperational) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(engine->health().size(), 1);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Operational");
}

/// @brief it should show a faulted slave as an error and put its AL status in the
/// description.
TEST_F(EtherCATScanStatusTest, FaultedSlaveShowsItsALStatus) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(0, slave::State::OP);
    this->master->fault_slave(0, slave::State::SAFE_OP, "Sync manager watchdog");
    ASSERT_EVENTUALLY_TRUE(!engine->health().empty() && engine->health()[0].faulted);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Error in SAFE-OP");
    EXPECT_EQ(status.description, "Sync manager watchdog");
    this->master->clear_transition_failures();
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && engine->health()[0].operational()
    );
    EXPECT_EQ(this->scan_status().message, "Operational");
}

/// @brief it should tell the user where to find the error code of a faulted slave when
/// the master reads no AL status code.
TEST_F(EtherCATScanStatusTest, FaultedSlaveWithoutALStatusPointsToTheKernelLog) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(0, slave::State::OP);
    this->master->fault_slave(0, slave::State::SAFE_OP, "");
    ASSERT_EVENTUALLY_TRUE(!engine->health().empty() && engine->health()[0].faulted);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Error in SAFE-OP");
    EXPECT_EQ(
        status.description,
        "The device reports an error without an error code. With IgH, the kernel "
        "log (dmesg) shows it."
    );
}

/// @brief it should show a lost slave as not responding.
TEST_F(EtherCATScanStatusTest, LostSlaveShowsNotResponding) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(0, true);
    ASSERT_EVENTUALLY_TRUE(!engine->health().empty() && engine->health()[0].lost);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Not responding");
    EXPECT_EQ(status.description, "Check the power and cabling of the device.");
}

/// @brief it should show a slave that left OP as a warning while the Driver returns it
/// to OP.
TEST_F(EtherCATScanStatusTest, SlaveLeavingOpShowsAWarning) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(0, slave::State::OP);
    this->master->set_slave_state(0, slave::State::PRE_OP);
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && engine->health()[0].state == slave::State::PRE_OP
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "In PRE-OP");
    EXPECT_EQ(status.description, "The Driver is returning the device to OP.");
}
}
