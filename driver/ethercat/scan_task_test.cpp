// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <map>
#include <random>
#include <string>
#include <thread>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/scan_task.h"
#include "driver/ethercat/testutil/master.h"

namespace driver::ethercat {
namespace {
/// @brief a loader for a network with no saved slave settings.
const engine::SlaveConfigs NO_SLAVE_CONFIGS = [](const std::string &,
                                                 const std::string &) {
    return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
};
}

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
    const pdo::Entry entry{1, 0x6000, 1, 16, true, x::telem::INT16_T};
    std::shared_ptr<testutil::Master> master;
    std::shared_ptr<engine::Pool> pool;
    std::unique_ptr<Scanner> scanner;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        this->master = std::make_shared<testutil::Master>();
        this->master->add_slave(
            slave::Properties{
                .network = "eth0",
                .position = 1,
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
        this->pool = std::make_shared<engine::Pool>(
            std::move(manager),
            NO_SLAVE_CONFIGS
        );
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

/// @brief it should report a test command that has no interface.
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

/// @brief it should leave a command it does not know to another handler.
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

/// @brief it should report how many slaves a test command finds, and call the network
/// by the name of its master.
TEST_F(EtherCATScanTest, TestInterfaceCommandWithMultipleSlaves) {
    auto mock_master = std::make_shared<testutil::Master>();
    mock_master->set_name("Ethernet 2");
    mock_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 11111,
            .name = "Slave 1",
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 2,
            .vendor_id = 0x1,
            .product_code = 0x3,
            .serial = 22222,
            .name = "Slave 2",
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 3,
            .vendor_id = 0x1,
            .product_code = 0x4,
            .serial = 33333,
            .name = "Slave 3",
        }
    );

    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("enp3s0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS);

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
    EXPECT_EQ(this->ctx->statuses.back().message, "Found 3 slaves on Ethernet 2");
}

/// @brief it should return a permission error from a scan, so the error reaches the
/// status of the scan task.
TEST_F(EtherCATScanTest, ScanReturnsPermissionDenied) {
    auto mock_master = std::make_shared<testutil::Master>();
    mock_master->inject_scan_error(
        x::errors::Error(errors::PERMISSION_DENIED, "permission denied opening eth0")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS);
    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "EtherCAT Scanner",
        .type = SCAN_TASK_TYPE,
        .internal = true
    };
    Scanner scanner(this->ctx, task, ScanTaskConfig{}, pool);
    ASSERT_OCCURRED_AS_P(scanner.scan({}), errors::PERMISSION_DENIED);
}

/// @brief it should report an interface that fails to open.
TEST_F(EtherCATScanTest, TestInterfaceCommandInitError) {
    auto mock_master = std::make_shared<testutil::Master>();
    mock_master->inject_scan_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );

    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", mock_master);
    auto pool = std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS);

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
    EXPECT_EQ(this->ctx->statuses.back().description, "No interface.");
}

/// @brief a scanner over a mock bus on eth0, for scan specs.
class EtherCATScanProbeTest : public EtherCATScanTest {
protected:
    std::shared_ptr<testutil::Master> master;
    std::unique_ptr<Scanner> scanner;
    std::size_t scans = 0;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        this->master = std::make_shared<testutil::Master>();
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
            std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS)
        );
    }

    void scan() { ASSERT_NIL_P(this->scanner->scan({.count = this->scans++})); }
};

/// @brief it should probe an interface with no slaves only on every sixth scan.
TEST_F(EtherCATScanProbeTest, ProbesAnInterfaceWithNoSlavesEverySixthScan) {
    this->master->inject_scan_error(
        x::errors::Error(errors::NO_SLAVES_FOUND, "no EtherCAT slaves found on eth0")
    );
    for (int i = 0; i < 7; ++i)
        this->scan();
    EXPECT_EQ(this->master->call_count("scan"), 2);
}

/// @brief it should report an interface whose replies a filter blocks on each scan, and
/// still probe it only on every sixth scan.
TEST_F(EtherCATScanProbeTest, ReportsAnInterfaceWhoseRepliesAreBlockedOnEachScan) {
    const auto blocked = x::errors::Error(
        errors::REPLIES_BLOCKED,
        "No EtherCAT slaves found on Ethernet 2. The DewesoftX \"EthercatDAQ Filter\" "
        "is on this adapter."
    );
    this->master->inject_scan_error(blocked);
    for (std::size_t i = 0; i < 7; ++i) {
        const auto [devices, err] = this->scanner->scan({.count = i});
        ASSERT_OCCURRED_AS(err, common::PARTIAL_SCAN_ERROR);
        EXPECT_EQ(err.data, blocked.data);
        EXPECT_TRUE(devices.empty());
    }
    EXPECT_EQ(this->master->call_count("scan"), 2);
    this->master->inject_scan_error(x::errors::NIL);
    for (std::size_t i = 7; i < 12; ++i)
        ASSERT_OCCURRED_AS_P(
            this->scanner->scan({.count = i}),
            common::PARTIAL_SCAN_ERROR
        );
    ASSERT_NIL_P(this->scanner->scan({.count = 12}));
}

/// @brief scans an adapter with a filter and no devices, and an adapter with one
/// device.
class EtherCATScanFilterTest : public EtherCATScanTest {
protected:
    const x::errors::Error blocked = x::errors::Error(
        errors::REPLIES_BLOCKED,
        "No EtherCAT slaves found on Ethernet. The DewesoftX \"EthercatDAQ Filter\" is "
        "on this adapter."
    );
    std::unique_ptr<Scanner> scanner;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        auto filtered = std::make_shared<testutil::Master>();
        filtered->inject_scan_error(this->blocked);
        auto bus = std::make_shared<testutil::Master>();
        bus->add_slave(
            slave::Properties{
                .position = 1,
                .vendor_id = 0x2,
                .serial = 1,
                .name = "EL3102"
            }
        );
        auto manager = std::make_unique<testutil::Manager>();
        manager->configure("eth0", filtered);
        manager->configure("eth1", bus);
        this->scanner = std::make_unique<Scanner>(
            this->ctx,
            synnax::task::Task{
                .rack = this->rack.key,
                .name = "EtherCAT Scanner",
                .type = SCAN_TASK_TYPE,
                .internal = true,
            },
            ScanTaskConfig{},
            std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS)
        );
    }
};

/// @brief it should not report the filter of an adapter with no devices while another
/// adapter shows devices.
TEST_F(EtherCATScanFilterTest, IgnoresTheFilterOfAnAdapterWithNoDevices) {
    EXPECT_EQ(ASSERT_NIL_P(this->scanner->scan({})).size(), 1);
}

/// @brief it should report the filter of an adapter that the Core holds devices from.
TEST_F(EtherCATScanFilterTest, ReportsTheFilterOfAnAdapterThatHadDevices) {
    const std::unordered_map<std::string, synnax::device::Device> saved{
        {"saved",
         synnax::device::Device{.key = "saved", .properties = {{"network", "eth0"}}}},
    };
    const auto [devices, err] = this->scanner->scan({.devices = &saved});
    ASSERT_OCCURRED_AS(err, common::PARTIAL_SCAN_ERROR);
    EXPECT_EQ(err.data, this->blocked.data);
    EXPECT_EQ(devices.size(), 1);
}

/// @brief it should return a partial scan error for an interface it cannot open, then
/// scan the interface again when it opens.
TEST_F(EtherCATScanProbeTest, ReturnsAnInterfaceItCannotScanAsAPartialScan) {
    this->master->inject_scan_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "cannot open eth0")
    );
    const auto [devices, err] = this->scanner->scan({});
    ASSERT_OCCURRED_AS(err, common::PARTIAL_SCAN_ERROR);
    EXPECT_EQ(err.data, "Cannot open eth0.");
    EXPECT_TRUE(devices.empty());
    this->master->inject_scan_error(x::errors::NIL);
    ASSERT_NIL_P(this->scanner->scan({.count = 1}));
}

/// @brief it should fail to configure the scan task and report an error status when the
/// packet capture library is missing.
TEST_F(EtherCATScanTest, ScanTaskFailsToConfigureWithoutAPacketCaptureLibrary) {
    Factory factory(std::unique_ptr<master::Manager>{}, this->client, this->rack.key);
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
    EXPECT_TRUE(status.details.connected);
}

/// @brief it should key a slave with a serial number by its vendor, product, and
/// serial, not by its position.
TEST_F(EtherCATScanStatusTest, KeysASlaveBySerialNumber) {
    const auto devices = ASSERT_NIL_P(
        this->scanner->scan(common::ScannerContext{.count = 1})
    );
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices.at(0).key, "ethercat_1_2_12345");
}

namespace {
using Devices = std::unordered_map<std::string, synnax::device::Device>;

/// @brief adds a stored slave device with key for the slave that props describes.
void store(Devices &devices, const std::string &key, const slave::Properties &props) {
    devices[key] = synnax::device::Device{
        .key = key,
        .model = SLAVE_DEVICE_MODEL,
        .name = props.name,
        .properties = props.to_json().get<x::json::json::object_t>(),
    };
}

/// @brief the properties of a slave on network with the given identity.
slave::Properties identity(
    const uint16_t position,
    const uint32_t product_code,
    const uint32_t serial,
    const std::string &network = "eth0"
) {
    return {
        .network = network,
        .position = position,
        .vendor_id = 0x1,
        .product_code = product_code,
        .serial = serial,
        .name = "Slave",
    };
}
}

/// @brief it should keep the device of a slave with its serial number after the slave
/// moves to another position.
TEST_F(EtherCATScanStatusTest, KeepsTheDeviceOfASlaveThatMoved) {
    Devices devices;
    store(devices, "saved", identity(4, 0x2, 12345));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "saved");
    EXPECT_EQ(scanned.at(0).properties.at("position"), 1);
}

/// @brief it should keep the device of a slave with no serial number at its position.
TEST_F(EtherCATScanStatusTest, KeepsTheDeviceOfASlaveWithNoSerialAtItsPosition) {
    this->master->replace_slave(1, identity(1, 0x2, 0));
    Devices devices;
    store(devices, "saved", identity(1, 0x2, 0));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "saved");
}

/// @brief it should give the serial number to a device saved without one at the
/// position of a slave of the same model.
TEST_F(EtherCATScanStatusTest, GivesTheSerialToADeviceSavedWithoutOne) {
    Devices devices;
    store(devices, "ethercat_eth0_1_2_1", identity(1, 0x2, 0));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "ethercat_eth0_1_2_1");
    EXPECT_EQ(scanned.at(0).properties.at("serial"), 12345);
}

/// @brief it should give a new device to a slave at the position of a saved device of
/// another model.
TEST_F(EtherCATScanStatusTest, KeepsADeviceOfAnotherModelAtThePosition) {
    Devices devices;
    store(devices, "ethercat_eth0_1_3_1", identity(1, 0x3, 0));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "ethercat_1_2_12345");
}

/// @brief it should give a new device to a slave at the position of a saved device of
/// another unit of its model.
TEST_F(EtherCATScanStatusTest, KeepsTheDeviceOfAnotherUnitAtThePosition) {
    Devices devices;
    store(devices, "replaced", identity(1, 0x2, 777));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "ethercat_1_2_12345");
}

/// @brief it should give a slave with no serial number a new key when a device that
/// moved with another slave holds the key of its position.
TEST_F(EtherCATScanStatusTest, GivesANewKeyWhenAMovedDeviceHoldsTheKey) {
    this->master->replace_slave(1, identity(1, 0x2, 0));
    this->master->add_slave(identity(2, 0x2, 777));
    Devices devices;
    store(devices, "ethercat_eth0_1_2_1", identity(1, 0x2, 777));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 2);
    EXPECT_EQ(scanned.at(0).key, "ethercat_eth0_1_2_1_2");
    EXPECT_EQ(scanned.at(1).key, "ethercat_eth0_1_2_1");
    EXPECT_EQ(scanned.at(1).properties.at("position"), 2);
}

/// @brief it should keep a moved device with invalid properties on its slave.
TEST_F(EtherCATScanStatusTest, KeepsAMovedDeviceWithInvalidProperties) {
    this->master->replace_slave(1, identity(1, 0x2, 0));
    this->master->add_slave(identity(2, 0x2, 777));
    Devices devices;
    store(devices, "ethercat_eth0_1_2_1", identity(1, 0x2, 777));
    devices["ethercat_eth0_1_2_1"].properties["op_mode"] = {
        {"index", 1},
        {"assign_activate", 0x0700},
        {"sync0_cycle", -1},
    };
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 2);
    EXPECT_EQ(scanned.at(0).key, "ethercat_eth0_1_2_1_2");
    EXPECT_EQ(scanned.at(1).key, "ethercat_eth0_1_2_1");
}

/// @brief it should give the device of a serial number to one slave when slaves on two
/// networks have the serial number.
TEST_F(EtherCATScanTest, GivesADeviceToOneSlaveAcrossNetworks) {
    auto manager = std::make_unique<testutil::Manager>();
    for (const auto *network: {"eth0", "eth1"}) {
        auto master = std::make_shared<testutil::Master>();
        master->add_slave(identity(1, 0x2, 777, network));
        manager->configure(network, master);
    }
    const auto pool = std::make_shared<engine::Pool>(
        std::move(manager),
        NO_SLAVE_CONFIGS
    );
    Scanner scanner(this->ctx, this->scan_task(), ScanTaskConfig{}, pool);
    Devices devices;
    store(devices, "saved", identity(1, 0x2, 777));
    const auto scanned = ASSERT_NIL_P(scanner.scan({.count = 1, .devices = &devices}));
    ASSERT_EQ(scanned.size(), 2);
    EXPECT_EQ(scanned.at(0).key, "saved");
    EXPECT_EQ(scanned.at(1).key, "ethercat_1_2_777");
}

/// @brief it should save no new device for a slave whose serial number read fails, and
/// discover the slaves again every sixth scan.
TEST_F(EtherCATScanStatusTest, SkipsASlaveWhoseSerialReadFails) {
    this->master->fail_serial_read(1, "failed to read 0x1018:04: no response");
    Devices devices;
    store(devices, "elsewhere", identity(2, 0x2, 777));
    for (size_t i = 0; i < 13; ++i)
        EXPECT_TRUE(
            ASSERT_NIL_P(this->scanner->scan({.count = i, .devices = &devices})).empty()
        );
    EXPECT_EQ(this->master->call_count("deactivate"), 2);
}

/// @brief it should keep the stored device and serial number at the position of a
/// slave whose serial number read fails.
TEST_F(EtherCATScanStatusTest, KeepsTheDeviceOfASlaveWhoseSerialReadFails) {
    this->master->fail_serial_read(1, "failed to read 0x1018:04: no response");
    Devices devices;
    store(devices, "saved", identity(1, 0x2, 777));
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned.at(0).key, "saved");
    EXPECT_EQ(scanned.at(0).properties.at("serial"), 777);
}

/// @brief it should show a failed PDO discovery as a warning that says the Driver tries
/// again.
TEST_F(EtherCATScanStatusTest, FailedPDODiscoveryShowsAWarning) {
    this->master->fail_pdo_discovery(1, "no PDO objects found in object dictionary");
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Failed to discover PDOs");
    EXPECT_TRUE(status.details.connected);
    EXPECT_EQ(
        status.description,
        "No PDO objects found in object dictionary. Channels can use only the entries "
        "a scan discovers. The Driver tries again while no task uses the port."
    );
}

/// @brief it should show the PDOs of a fallback discovery with its error as a warning.
TEST_F(EtherCATScanStatusTest, FallbackPDODiscoveryShowsItsPDOsAsAWarning) {
    this->master->fall_back_pdo_discovery(
        1,
        "failed to read the PDO layout: no answer, so the Driver uses the PDOs in its "
        "SII"
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Discovered (1 inputs, 0 outputs)");
    EXPECT_EQ(
        status.description,
        "Failed to read the PDO layout: no answer, so the Driver uses the PDOs in its "
        "SII. PDO order may be unreliable. Verify data correctness after configuration."
    );
}

/// @brief it should discover the slaves again every sixth scan while a slave has a
/// failed PDO discovery.
TEST_F(EtherCATScanStatusTest, FailedPDODiscoveryRediscoversEverySixthScan) {
    this->master->fail_pdo_discovery(1, "no PDO objects found in object dictionary");
    for (size_t i = 0; i < 13; ++i)
        ASSERT_NIL_P(this->scanner->scan({.count = i}));
    EXPECT_EQ(this->master->call_count("deactivate"), 2);
}

/// @brief it should save no device for a slave with no vendor and product, and discover
/// the slaves again every sixth scan.
TEST_F(EtherCATScanStatusTest, SkipsASlaveWithNoIdentity) {
    this->master->add_slave(slave::Properties{.network = "eth0", .position = 2});
    for (size_t i = 0; i < 13; ++i) {
        const auto devices = ASSERT_NIL_P(this->scanner->scan({.count = i}));
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices.at(0).key, "ethercat_1_2_12345");
    }
    EXPECT_EQ(this->master->call_count("deactivate"), 2);
}

/// @brief it should show a failed settings read as a warning that says the Driver tries
/// again.
TEST_F(EtherCATScanStatusTest, FailedSettingsReadShowsAWarning) {
    this->master->fail_settings_read(
        1,
        "failed to read 0x8050:03: Object does not exist in the object dictionary"
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Discovered (1 inputs, 0 outputs)");
    EXPECT_EQ(
        status.description,
        "Failed to read the settings: failed to read 0x8050:03: Object does not exist in the object dictionary. "
        "The Driver tries again while no task uses the port."
    );
}

/// @brief it should discover the slaves again every sixth scan while a slave has a
/// failed settings read.
TEST_F(EtherCATScanStatusTest, FailedSettingsReadRediscoversEverySixthScan) {
    this->master->fail_settings_read(
        1,
        "failed to read 0x8050:03: Object does not exist in the object dictionary"
    );
    for (size_t i = 0; i < 13; ++i)
        ASSERT_NIL_P(this->scanner->scan({.count = i}));
    EXPECT_EQ(this->master->call_count("deactivate"), 2);
}

/// @brief it should not discover the slaves again for a disabled slave whose PDO
/// discovery failed.
TEST_F(EtherCATScanStatusTest, DisabledSlaveIsNotRediscovered) {
    this->master->fail_pdo_discovery(1, "no PDO objects found in object dictionary");
    auto dev = ASSERT_NIL_P(this->scanner->scan({.count = 0})).at(0);
    dev.properties["enabled"] = false;
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    for (size_t i = 1; i < 13; ++i)
        ASSERT_NIL_P(this->scanner->scan({.count = i, .devices = &devices}));
    EXPECT_EQ(this->master->call_count("deactivate"), 0);
}

/// @brief it should keep the chosen mode of a slave when a scan finds the slave again.
TEST_F(EtherCATScanStatusTest, KeepsTheChosenModeOnARescan) {
    auto dev = ASSERT_NIL_P(this->scanner->scan({.count = 0})).at(0);
    const dc::Choice choice{.index = 1, .assign_activate = 0x0700};
    dev.properties["op_mode"] = choice.to_json();
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    const auto rescanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    EXPECT_EQ(rescanned.at(0).properties.at("op_mode"), choice.to_json());
}

/// @brief it should store the settings that a scan reads in the device properties, in
/// place of the stored ones.
TEST_F(EtherCATScanStatusTest, StoresTheSettingsThatAScanReads) {
    const x::json::json settings{
        {"device", x::json::json::array()},
        {"channels", x::json::json::array()},
    };
    this->master->add_slave(
        slave::Properties{
            .network = "eth0",
            .position = 2,
            .vendor_id = 0x1,
            .product_code = 0x3,
            .serial = 6789,
            .name = "6xSTG",
            .settings = settings,
        }
    );
    auto dev = ASSERT_NIL_P(this->scanner->scan({.count = 0})).at(1);
    EXPECT_EQ(dev.properties.at("settings"), settings);
    dev.properties["settings"] = {{"device", x::json::json::array()}};
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    const auto rescanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    EXPECT_FALSE(rescanned.at(0).properties.contains("settings"));
    EXPECT_EQ(rescanned.at(1).properties.at("settings"), settings);
}

/// @brief it should keep the stored settings of a slave when a rescan fails to read
/// them.
TEST_F(EtherCATScanStatusTest, KeepsTheStoredSettingsWhenTheirReadFails) {
    this->master->fail_settings_read(
        1,
        "failed to read 0x8050:03: Object does not exist in the object dictionary"
    );
    auto dev = ASSERT_NIL_P(this->scanner->scan({.count = 0})).at(0);
    EXPECT_FALSE(dev.properties.contains("settings"));
    const x::json::json settings{
        {"device", x::json::json::array()},
        {"channels", x::json::json::array()},
    };
    dev.properties["settings"] = settings;
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    const auto rescanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    EXPECT_EQ(rescanned.at(0).properties.at("settings"), settings);
}

/// @brief it should keep the invalid properties error of a saved device on a rescan.
TEST_F(EtherCATScanStatusTest, KeepsTheInvalidPropertiesErrorOnARescan) {
    auto dev = ASSERT_NIL_P(this->scanner->scan({.count = 0})).at(0);
    dev.properties["op_mode"] = {
        {"index", 1},
        {"assign_activate", 0x0700},
        {"sync0_cycle", -1},
    };
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    const auto rescanned = ASSERT_NIL_P(
        this->scanner->scan({.count = 1, .devices = &devices})
    );
    const auto status = rescanned.at(0).status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Invalid device properties");
    EXPECT_EQ(status.description, "op_mode.sync0_cycle: must be 0 or more");
    EXPECT_FALSE(status.details.connected);
}

/// @brief it should not discover the slaves again while every slave has its PDOs.
TEST_F(EtherCATScanStatusTest, DiscoveredSlavesAreNotRediscovered) {
    for (size_t i = 0; i < 13; ++i)
        ASSERT_NIL_P(this->scanner->scan({.count = i}));
    EXPECT_EQ(this->master->call_count("deactivate"), 0);
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
    EXPECT_TRUE(status.details.connected);
}

/// @brief it should show a faulted slave as an error and put its AL status in the
/// description.
TEST_F(EtherCATScanStatusTest, FaultedSlaveShowsItsALStatus) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->fault_slave(1, slave::State::SAFE_OP, "Sync manager watchdog");
    ASSERT_EVENTUALLY_TRUE(!engine->health().empty() && engine->health()[0].faulted);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Error in SAFE-OP");
    EXPECT_EQ(status.description, "Sync manager watchdog");
    EXPECT_TRUE(status.details.connected);
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
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->fault_slave(1, slave::State::SAFE_OP, "");
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

/// @brief it should show the startup parameter that a slave rejected as a sentence.
TEST_F(EtherCATScanStatusTest, SlaveThatRejectsAStartupParameterShowsItAsASentence) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_state(1, slave::State::PRE_OP);
    this->master->fail_startup(
        1,
        "rejected startup parameter 0x8000:15 (Filter): Value of parameter written "
        "too high"
    );
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && !engine->health()[0].startup_error.empty()
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Error in PRE-OP");
    EXPECT_EQ(
        status.description,
        "Rejected startup parameter 0x8000:15 (Filter): Value of parameter written too "
        "high."
    );
}

/// @brief it should show a slave whose distributed clock the master could not restore
/// as an error that tells why.
TEST_F(EtherCATScanStatusTest, SlaveWithAFailedClockRestoreShowsWhy) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_state(1, slave::State::PRE_OP);
    this->master->fail_dc_restore(1, "no slave latched its receive time");
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && !engine->health()[0].dc_restore_error.empty()
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to restore the distributed clock");
    EXPECT_EQ(
        status.description,
        "The device lost its clock settings, and the Driver could not set them again: "
        "no slave latched its receive time. The Driver keeps the device out of OP and "
        "tries again. If the error stays, check the power and cabling of the device."
    );
    EXPECT_TRUE(status.details.connected);
}

/// @brief it should show a lost slave as not responding.
TEST_F(EtherCATScanStatusTest, LostSlaveShowsNotResponding) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(!engine->health().empty() && engine->health()[0].lost);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Not responding");
    EXPECT_FALSE(status.details.connected);
    EXPECT_EQ(status.description, "Check the power and cabling of the device.");
}

/// @brief it should show a slave that the bus started without as an error that tells
/// why.
TEST_F(EtherCATScanStatusTest, LeftOutSlaveShowsFailedToStart) {
    this->master->add_slave(
        slave::Properties{
            .position = 2,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 54321,
            .name = "Left Out Slave",
        }
    );
    this->master->refuse(2, {.reason = "refused SAFE-OP: Inconsistent settings"});
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(engine->health().size(), 2);
    const auto devices = ASSERT_NIL_P(
        this->scanner->scan(common::ScannerContext{.count = 1})
    );
    const auto dev = std::ranges::find(
        devices,
        "Left Out Slave",
        &synnax::device::Device::name
    );
    ASSERT_NE(dev, devices.end());
    const auto status = dev->status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to start");
    EXPECT_EQ(
        status.description,
        "It refused SAFE-OP: Inconsistent settings. The other devices on the network "
        "started without it. The Driver tries it again the next time the network "
        "starts."
    );
}

/// @brief it should end the status of a left-out slave that reports "Invalid DC SYNC
/// configuration" with the DC sync advice.
TEST_F(EtherCATScanStatusTest, LeftOutSlaveShowsTheDCSyncAdvice) {
    this->master->add_slave(
        slave::Properties{
            .position = 2,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = 54321,
            .name = "Left Out Slave",
        }
    );
    this->master->refuse(
        2,
        {.reason = "refused SAFE-OP: Invalid DC SYNC configuration",
         .advice = slave::DC_SYNC_ADVICE}
    );
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(engine->health().size(), 2);
    const auto devices = ASSERT_NIL_P(
        this->scanner->scan(common::ScannerContext{.count = 1})
    );
    const auto dev = std::ranges::find(
        devices,
        "Left Out Slave",
        &synnax::device::Device::name
    );
    ASSERT_NE(dev, devices.end());
    const auto status = dev->status.value();
    EXPECT_EQ(
        status.description,
        "It refused SAFE-OP: Invalid DC SYNC configuration. The other devices on the "
        "network started without it. The Driver tries it again the next time the "
        "network starts. " +
            std::string(slave::DC_SYNC_ADVICE)
    );
}

/// @brief it should show a slave that left OP as a warning while the Driver returns it
/// to OP.
TEST_F(EtherCATScanStatusTest, SlaveLeavingOpShowsAWarning) {
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({this->entry}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->set_slave_state(1, slave::State::PRE_OP);
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && engine->health()[0].state == slave::State::PRE_OP
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "In PRE-OP");
    EXPECT_TRUE(status.details.connected);
    EXPECT_EQ(status.description, "The Driver is returning the device to OP.");
}
/// @brief it should show a disabled slave as not connected.
TEST_F(EtherCATScanStatusTest, DisabledSlaveIsNotConnected) {
    auto dev = ASSERT_NIL_P(this->scanner->scan(common::ScannerContext{.count = 1}))
                   .at(0);
    dev.properties["enabled"] = false;
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    const auto scanned = ASSERT_NIL_P(
        this->scanner->scan(common::ScannerContext{.count = 2, .devices = &devices})
    );
    ASSERT_EQ(scanned.size(), 1);
    const auto &status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_DISABLED);
    EXPECT_EQ(status.message, "Device disabled");
    EXPECT_FALSE(status.details.connected);
}

/// @brief it should keep an enabled slave connected when its device is edited.
TEST_F(EtherCATScanStatusTest, EditedSlaveStaysConnected) {
    auto dev = ASSERT_NIL_P(this->scanner->scan(common::ScannerContext{.count = 1}))
                   .at(0);
    // A scan derives the key from the serial, so every run would get the same key.
    dev.key = make_unique_channel_name(dev.key);
    ASSERT_NIL(this->client->devices.create(dev));
    this->scanner->on_device_set(dev);
    const auto status = ASSERT_NIL_P(
        this->client->statuses.retrieve_by_owner<synnax::device::StatusDetails>(
            synnax::device::resource_id(dev.key)
        )
    );
    EXPECT_EQ(status.message, "Device enabled");
    EXPECT_TRUE(status.details.connected);
}

/// @brief a scanner over a mock bus with one slave, for device set specs.
class EtherCATScanDeviceSetTest : public EtherCATScanTest {
protected:
    std::unique_ptr<Scanner> scanner;

    void SetUp() override {
        EtherCATScanTest::SetUp();
        auto master = std::make_shared<testutil::Master>();
        master->add_slave(
            slave::Properties{
                .position = 1,
                .vendor_id = 0x2,
                .product_code = 0xC1E3052,
                .name = "EL3102",
            }
        );
        auto manager = std::make_unique<testutil::Manager>();
        manager->configure("eth0", master);
        this->scanner = std::make_unique<Scanner>(
            this->ctx,
            this->scan_task(),
            ScanTaskConfig{},
            std::make_shared<engine::Pool>(std::move(manager), NO_SLAVE_CONFIGS)
        );
    }

    /// @brief creates a device of the model with the given enabled value.
    synnax::device::Device create_slave(
        const x::json::json &enabled,
        const std::string &model = SLAVE_DEVICE_MODEL
    ) {
        synnax::device::Device dev{
            .key = "ecat_device_set_" + std::to_string(this->rack.key),
            .rack = this->rack.key,
            .location = "eth0.Slot 1",
            .make = INTEGRATION_NAME,
            .model = model,
            .name = "EL3102",
            .properties = {
                {"serial", 0},
                {"vendor_id", 0x2},
                {"product_code", 0xC1E3052},
                {"revision", 1},
                {"name", "EL3102"},
                {"network", "eth0"},
                {"position", 1},
                {"enabled", enabled},
                {"pdos",
                 {{"inputs", x::json::json::array()},
                  {"outputs", x::json::json::array()}}},
            },
        };
        const auto err = this->client->devices.create(dev);
        EXPECT_NIL(err);
        return dev;
    }

    [[nodiscard]] synnax::device::Status
    status(const synnax::device::Device &dev) const {
        return ASSERT_NIL_P(
            this->client->statuses.retrieve_by_owner<synnax::device::StatusDetails>(
                synnax::device::resource_id(dev.key)
            )
        );
    }
};

/// @brief it should report a changed device that is disabled as not connected.
TEST_F(EtherCATScanDeviceSetTest, ReportsADisabledDevice) {
    const auto dev = this->create_slave(false);
    this->scanner->on_device_set(dev);
    const auto status = this->status(dev);
    EXPECT_EQ(status.variant, synnax::status::VARIANT_DISABLED);
    EXPECT_EQ(status.message, "Device disabled");
    EXPECT_FALSE(status.details.connected);
}

/// @brief it should write no status for a device that is not a slave.
TEST_F(EtherCATScanDeviceSetTest, IgnoresADeviceThatIsNotASlave) {
    const auto dev = this->create_slave(nullptr, "network");
    const auto before = this->status(dev);
    this->scanner->on_device_set(dev);
    const auto after = this->status(dev);
    EXPECT_EQ(after.message, before.message);
    EXPECT_EQ(after.variant, before.variant);
    EXPECT_EQ(after.time, before.time);
}

/// @brief it should report an error status that names the invalid field.
TEST_F(EtherCATScanDeviceSetTest, ReportsInvalidProperties) {
    const auto dev = this->create_slave(nullptr);
    this->scanner->on_device_set(dev);
    const auto status = this->status(dev);
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Invalid device properties");
    EXPECT_EQ(status.description, "enabled: type must be boolean, but is null");
}

/// @brief it should show an invalid startup parameter on the device status at each
/// scan.
TEST_F(EtherCATScanDeviceSetTest, ScanReportsAnInvalidStartupParameter) {
    const synnax::device::Device dev{
        .key = "ethercat_eth0_2_203305042_1",
        .model = SLAVE_DEVICE_MODEL,
        .name = "EL3102",
        .properties = {
            {"startup_parameters",
             {{{"index", 0x8000},
               {"sub_index", 0x15},
               {"data_type", "uint8"},
               {"value", 300}}}},
        },
    };
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {dev.key, dev}
    };
    for (std::size_t count = 0; count < 2; ++count) {
        const auto scanned = ASSERT_NIL_P(this->scanner->scan(
            common::ScannerContext{.count = count, .devices = &devices}
        ));
        ASSERT_EQ(scanned.size(), 1);
        const auto status = scanned[0].status.value();
        EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
        EXPECT_EQ(status.message, "Invalid device properties");
        EXPECT_EQ(
            status.description,
            "startup_parameters.0.value: must be a whole number from 0 to 255"
        );
    }
}

/// @brief a scanner of eth0 with a slave whose input 0x6000:1 might be one that only
/// DewesoftX reads.
class EtherCATScanModeTest : public EtherCATScanTest {
protected:
    const std::vector<slave::StartupParameter> standalone{
        {.index = 0x8020, .sub_index = 9, .data_type = x::telem::UINT16_T, .value = 0},
        {.index = 0x8020, .sub_index = 2, .data_type = x::telem::UINT16_T, .value = 2},
    };
    slave::Properties props;
    std::shared_ptr<testutil::Master> master;
    std::shared_ptr<engine::Pool> pool;
    std::unique_ptr<Scanner> scanner;
    std::mt19937 rng = random_generator("EtherCATScanModeTest");

    /// @brief puts the slave on eth0, on bus 1 of a slave with two buses. Only
    /// DewesoftX reads the entry at only_read_index, and shared says whether DewesoftX
    /// also sets up the slave.
    void start(const bool shared, const uint16_t only_read_index = 0x6000) {
        auto change = x::json::json::array();
        if (shared)
            for (const auto &param: this->standalone)
                change.push_back({
                    {"index", param.index},
                    {"sub_index", param.sub_index},
                    {"data_type", "uint16"},
                    {"value", param.value},
                });
        this->props = slave::Properties{
            .network = "eth0",
            .position = 1,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = std::uniform_int_distribution<uint32_t>(1)(this->rng),
            .name = "Test Slave",
            .input_pdos =
                {{.pdo_index = 0x1A00,
                  .index = 0x6000,
                  .sub_index = 1,
                  .bit_length = 32,
                  .is_input = true,
                  .data_type = x::telem::UINT32_T}},
            .settings = x::json::json{
                {"device", x::json::json::array()},
                {"channels",
                 {{{"entries",
                    {{{"index", only_read_index},
                      {"sub_index", 1},
                      {"only_read_by", "DewesoftX"}}}},
                   {"settings", x::json::json::array()},
                   {"requires", x::json::json::array()}}}},
                {"mode",
                 {{"value", shared ? 1 : 0},
                  {"shared", shared},
                  {"bus", 1},
                  {"switch", change}}},
            },
        };
        if (shared) this->props.settings.value()["mode"]["shared_with"] = "DewesoftX";
        this->serve();
    }

    /// @brief merges fields into the mode of the slave and serves it again.
    void set_mode(const x::json::json &fields) {
        this->props.settings.value()["mode"].merge_patch(fields);
        this->serve();
    }

    /// @brief serves props on eth0 to a new scanner.
    void serve() {
        this->master = std::make_shared<testutil::Master>();
        this->master->add_slave(this->props);
        auto manager = std::make_unique<testutil::Manager>();
        manager->configure("eth0", this->master);
        this->pool = std::make_shared<engine::Pool>(
            std::move(manager),
            NO_SLAVE_CONFIGS
        );
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

    /// @brief creates the device of the slave and sends the set_standalone command for
    /// it. Returns the status that the command sets.
    synnax::task::Status set_standalone() {
        synnax::device::Device dev{
            .key = "ethercat_1_2_" + std::to_string(this->props.serial),
            .rack = this->rack.key,
            .location = "eth0.Slot 1",
            .make = INTEGRATION_NAME,
            .model = SLAVE_DEVICE_MODEL,
            .name = "Test Slave",
            .properties = this->props.to_json(),
        };
        EXPECT_NIL(this->client->devices.create(dev));
        synnax::task::Command cmd;
        cmd.type = SET_STANDALONE_CMD_TYPE;
        cmd.args = {{"device", dev.key}};
        cmd.key = "1";
        auto task = this->scan_task();
        EXPECT_TRUE(this->scanner->exec(cmd, task, this->ctx));
        EXPECT_FALSE(this->ctx->statuses.empty());
        return this->ctx->statuses.back();
    }
};

/// @brief it should warn about a slave that sends data that only DewesoftX reads, and
/// advise the switch.
TEST_F(EtherCATScanModeTest, WarnsAboutASlaveSetUpForAnotherProgram) {
    this->start(true);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Set up for DewesoftX");
    EXPECT_EQ(
        status.description,
        "Synnax cannot read its data. In Configure, switch the device to Synnax."
    );
}

/// @brief it should offer only the switch for a slave with one bus that sends data that
/// only DewesoftX reads.
TEST_F(EtherCATScanModeTest, OffersOnlyTheSwitchForASingleBusSlave) {
    this->start(true, 0x6041);
    this->set_mode(
        {{"bus", nullptr}, {"shared_with", nullptr}, {"only_read_by", "DewesoftX"}}
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Set up for DewesoftX");
    EXPECT_EQ(
        status.description,
        "Synnax cannot read its data. In Configure, switch the device to Synnax."
    );
}

/// @brief it should not warn about a slave in dual mode on bus 1 that sends no data
/// that only DewesoftX reads.
TEST_F(EtherCATScanModeTest, ShowsDiscoveryOfADualSlaveOnBus1) {
    this->start(true, 0x6041);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Discovered (1 inputs, 0 outputs)");
}

/// @brief it should ask for bus 1 when a slave that runs Synnax only is on bus 2.
TEST_F(EtherCATScanModeTest, AsksForBus1ForAStandaloneSlaveOnBus2) {
    this->start(false, 0x6041);
    this->set_mode({{"bus", 2}});
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Connect Synnax to EtherCAT bus 1 IN");
    EXPECT_EQ(status.description, "");
}

/// @brief it should ask for a power restart of a slave that holds Synnax only but still
/// sends data that only DewesoftX reads.
TEST_F(EtherCATScanModeTest, AsksForAPowerRestartOfASwitchedSlave) {
    this->start(false);
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Restart the power of the device");
    EXPECT_EQ(status.description, "");
}

/// @brief it should ask for bus 1 before the power restart of a switched slave on bus
/// 2.
TEST_F(EtherCATScanModeTest, AsksForBus1BeforeThePowerRestartOnBus2) {
    this->start(false);
    this->set_mode({{"bus", 2}});
    const auto status = this->scan_status();
    EXPECT_EQ(status.message, "Restart the power of the device");
    EXPECT_EQ(status.description, "First, connect Synnax to EtherCAT bus 1 IN.");
}

/// @brief it should not advise a switch for a slave whose mode the scan did not read.
TEST_F(EtherCATScanModeTest, AdvisesNoSwitchWithoutAMode) {
    this->start(true);
    this->props.settings.value().erase("mode");
    this->serve();
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Set up for DewesoftX");
    EXPECT_EQ(status.description, "Synnax cannot read its data.");
}

/// @brief it should not warn about a slave in dual mode on bus 2, the bus that
/// DewesoftX does not use.
TEST_F(EtherCATScanModeTest, ShowsDiscoveryOfADualSlaveOnBus2) {
    this->start(true, 0x6041);
    this->set_mode({{"bus", 2}});
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Discovered (1 inputs, 0 outputs)");
}

/// @brief it should show the live health of a running slave over its setup warning.
TEST_F(EtherCATScanModeTest, ShowsTheLiveHealthOverTheSetupWarning) {
    this->start(true);
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const pdo::Entry entry{1, 0x6000, 1, 32, true, x::telem::UINT32_T};
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({entry}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_TRUE(
        !engine->health().empty() && engine->health()[0].operational()
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Operational");
}

/// @brief it should add a failed settings read to the setup warning.
TEST_F(EtherCATScanModeTest, AddsAFailedSettingsReadToTheSetupWarning) {
    this->start(true);
    this->props.settings.value().erase("mode");
    this->serve();
    this->master->fail_settings_read(
        1,
        "failed to read 0x8020:09: Object does not exist in the object dictionary"
    );
    const auto status = this->scan_status();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Set up for DewesoftX");
    EXPECT_EQ(
        status.description,
        "Synnax cannot read its data. Failed to read the settings: "
        "failed to read 0x8020:09: Object does not exist in the object dictionary. The "
        "Driver tries again while no task uses the port."
    );
}

/// @brief it should write the switch that the scan read to the slave on a stopped bus.
TEST_F(EtherCATScanModeTest, SetStandaloneWritesTheSwitchToTheSlave) {
    this->start(true);
    const auto status = this->set_standalone();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Switched Test Slave to Synnax only");
    EXPECT_EQ(status.description, "Restart the power of the device to finish.");
    EXPECT_EQ(
        this->master->written(),
        (std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>>{
            {1, this->standalone},
        })
    );
}

/// @brief it should write nothing to a slave that already runs Synnax only.
TEST_F(EtherCATScanModeTest, SetStandaloneWritesNothingToAStandaloneSlave) {
    this->start(false, 0x6041);
    const auto status = this->set_standalone();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.description, "The device already runs Synnax only.");
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should refuse to switch a slave while a task runs its bus.
TEST_F(EtherCATScanModeTest, SetStandaloneRefusesWhileATaskRunsTheBus) {
    this->start(true);
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const pdo::Entry entry{1, 0x6000, 1, 32, true, x::telem::UINT32_T};
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({entry}, x::telem::Rate(100), 1)
    );
    const auto status = this->set_standalone();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to switch the device");
    EXPECT_EQ(
        status.description,
        "A task uses eth0. Stop the tasks that use it first."
    );
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should refuse at once to switch a slave while a task starts its bus.
TEST_F(EtherCATScanModeTest, SetStandaloneRefusesWhileATaskStartsTheBus) {
    this->start(true);
    this->master->hold_activate();
    const auto engine = ASSERT_NIL_P(this->pool->acquire("eth0"));
    const pdo::Entry entry{1, 0x6000, 1, 32, true, x::telem::UINT32_T};
    std::thread opener([&engine, &entry] {
        auto [reader, err] = engine->open_reader({entry}, x::telem::Rate(100), 1);
    });
    this->master->wait_until_activating();
    const auto status = this->set_standalone();
    this->master->release_activate();
    opener.join();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to switch the device");
    EXPECT_EQ(
        status.description,
        "A task is starting on eth0. Try again when the start ends, or stop the task."
    );
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should report a set_standalone command that names no device.
TEST_F(EtherCATScanModeTest, SetStandaloneReportsAMissingDevice) {
    this->start(true);
    synnax::task::Command cmd;
    cmd.type = SET_STANDALONE_CMD_TYPE;
    cmd.args = x::json::json::object_t{};
    auto task = this->scan_task();
    EXPECT_TRUE(this->scanner->exec(cmd, task, this->ctx));
    ASSERT_FALSE(this->ctx->statuses.empty());
    EXPECT_EQ(this->ctx->statuses.back().message, "Failed to switch the device");
    EXPECT_EQ(
        this->ctx->statuses.back().description,
        "Device: This field is required."
    );
}

/// @brief it should refuse to switch a device that is not on its network, and call the
/// network by the name of its master.
TEST_F(EtherCATScanModeTest, SetStandaloneRefusesADeviceOffTheNetwork) {
    this->start(true);
    this->master->set_name("Ethernet 2");
    this->props.serial++;
    const auto status = this->set_standalone();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to switch the device");
    EXPECT_EQ(
        status.description,
        "Test Slave is not on Ethernet 2. Check its power and cabling."
    );
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should refuse to switch a slave whose settings hold no mode.
TEST_F(EtherCATScanModeTest, SetStandaloneRefusesASlaveWithNoMode) {
    this->start(true);
    this->props.settings.value().erase("mode");
    this->serve();
    const auto status = this->set_standalone();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to switch the device");
    EXPECT_EQ(status.description, "Synnax does not know how to switch this device.");
    EXPECT_TRUE(this->master->written().empty());
}
}
