// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cerrno>
#include <chrono>
#include <filesystem>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <termios.h>
#include <unistd.h>

#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/scan_task.h"
#include "driver/modbus/device/device.h"
#include "driver/modbus/scan_task.h"
#include "driver/modbus/testutil/slave.h"

namespace driver::modbus {
namespace {
testutil::SlaveConfig serial_slave() {
    testutil::SlaveConfig cfg;
    cfg.serial = true;
    return cfg;
}

/// @returns the connection of a unit on the serial slave.
device::ConnectionConfig rtu(const testutil::Slave &slave, const std::uint8_t unit_id) {
    device::ConnectionConfig cfg(device::Serial{.port = slave.serial_port()});
    cfg.unit_id = unit_id;
    cfg.timeout = x::telem::MILLISECOND * 50;
    cfg.retries = 0;
    return cfg;
}

/// @returns a Modbus device for a unit on the serial slave.
synnax::device::Device rtu_device(
    const testutil::Slave &slave,
    const std::string &key,
    const std::uint8_t unit_id
) {
    synnax::device::Device dev;
    dev.key = key;
    dev.name = key;
    dev.make = MAKE;
    dev.properties = {{"connection", rtu(slave, unit_id).to_json()}};
    return dev;
}

/// @returns a connection test command for cfg.
synnax::task::Command
test_command(const synnax::task::Task &t, const device::ConnectionConfig &cfg) {
    return synnax::task::Command{
        .task = t.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "rtu",
        .args = {{"connection", cfg.to_json()}},
    };
}
}

/// @brief it should read every table and write coils and registers over RTU, with
/// multiple and single write codes.
TEST(RTU, ReadsAndWrites) {
    auto cfg = serial_slave();
    cfg.coils[1] = 1;
    cfg.discrete_inputs[2] = 1;
    cfg.holding_registers[3] = static_cast<uint16_t>(1234);
    cfg.input_registers[4] = static_cast<uint16_t>(5678);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    device::Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire(rtu(slave, 1)));

    uint8_t bits[2] = {};
    ASSERT_NIL(dev->read_bits(device::Coil, 0, 2, bits));
    EXPECT_EQ(bits[1], 1);
    ASSERT_NIL(dev->read_bits(device::DiscreteInput, 2, 1, bits));
    EXPECT_EQ(bits[0], 1);
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(device::HoldingRegister, 3, 1, &reg));
    EXPECT_EQ(reg, 1234);
    ASSERT_NIL(dev->read_registers(device::InputRegister, 4, 1, &reg));
    EXPECT_EQ(reg, 5678);

    const uint8_t coils[] = {1, 0, 1};
    ASSERT_NIL(dev->write_bits(5, 3, coils));
    EXPECT_EQ(slave.get_coil(5), 1);
    EXPECT_EQ(slave.get_coil(6), 0);
    EXPECT_EQ(slave.get_coil(7), 1);
    const uint16_t registers[] = {11, 22};
    ASSERT_NIL(dev->write_registers(0, 2, registers));
    EXPECT_EQ(slave.get_holding_register(0), 11);
    EXPECT_EQ(slave.get_holding_register(1), 22);

    device::RequestConfig single;
    single.write_codes = device::WriteCodes::Single;
    auto single_dev = ASSERT_NIL_P(manager.acquire(rtu(slave, 1), single));
    ASSERT_NIL(single_dev->write_bits(8, 1, coils));
    EXPECT_EQ(slave.get_coil(8), 1);
    const uint16_t value = 33;
    ASSERT_NIL(single_dev->write_registers(2, 1, &value));
    EXPECT_EQ(slave.get_holding_register(2), 33);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_SINGLE_COIL), 1);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_SINGLE_REGISTER), 1);
}

/// @brief it should route the concurrent requests of two units through one shared
/// serial port without garbling a frame.
TEST(RTU, SharesPortAmongUnits) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    device::Manager manager;
    const std::vector<std::shared_ptr<device::Device>> devs = {
        ASSERT_NIL_P(manager.acquire(rtu(slave, 1))),
        ASSERT_NIL_P(manager.acquire(rtu(slave, 2))),
    };

    constexpr int requests = 20;
    std::vector<std::thread> threads;
    threads.reserve(devs.size());
    std::atomic<int> failures = 0;
    for (const auto &dev: devs)
        threads.emplace_back([&dev, &failures] {
            uint16_t reg = 0;
            for (int i = 0; i < requests; i++)
                if (dev->read_registers(device::HoldingRegister, 0, 1, &reg))
                    failures++;
        });
    for (auto &t: threads)
        t.join();
    EXPECT_EQ(failures, 0);
    EXPECT_EQ(slave.unit_requests(1), requests);
    EXPECT_EQ(slave.unit_requests(2), requests);
}

/// @brief it should let one connection at a time use a serial port, as when another
/// program holds it, even through a link to the port.
TEST(RTU, LocksPortToOneConnection) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    device::Manager manager;
    device::Manager other;
    auto link_cfg = rtu(slave, 1);
    link_cfg.serial.port = link.string();
    auto by_link = ASSERT_NIL_P(other.acquire(link_cfg));
    std::filesystem::create_symlink(slave.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    auto by_path = ASSERT_NIL_P(manager.acquire(rtu(slave, 1)));

    uint16_t reg = 0;
    ASSERT_NIL(by_path->read_registers(device::HoldingRegister, 0, 1, &reg));
    const auto err = by_link->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::UNREACHABLE);
    EXPECT_EQ(err.data, "serial port is in use by another connection");
    by_path.reset();
    ASSERT_NIL(by_link->read_registers(device::HoldingRegister, 0, 1, &reg));
}

/// @brief it should refuse a second connection to a port before it changes the port's
/// line settings.
TEST(RTU, RefusesPortBeforeChangingLineSettings) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    device::Manager manager;
    device::Manager other;
    auto link_cfg = rtu(slave, 1);
    link_cfg.serial.port = link.string();
    link_cfg.serial.baud_rate = 9600;
    auto by_link = ASSERT_NIL_P(other.acquire(link_cfg));
    std::filesystem::create_symlink(slave.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    auto by_path = ASSERT_NIL_P(manager.acquire(rtu(slave, 1)));
    uint16_t reg = 0;
    ASSERT_NIL(by_path->read_registers(device::HoldingRegister, 0, 1, &reg));

    const int fd = open(slave.serial_port().c_str(), O_RDONLY | O_NOCTTY | O_NONBLOCK);
    ASSERT_NE(fd, -1);
    x::defer::defer close_fd([fd] { close(fd); });
    std::atomic done = false;
    std::atomic changed = false;
    std::thread watch([&] {
        termios tios{};
        while (!done)
            if (tcgetattr(fd, &tios) == 0 && cfgetospeed(&tios) != B19200)
                changed = true;
    });
    std::size_t refused = 0;
    for (int i = 0; i < 200; i++)
        if (by_link->read_registers(device::HoldingRegister, 0, 1, &reg)
                .matches(device::UNREACHABLE))
            refused++;
    done = true;
    watch.join();
    EXPECT_EQ(refused, 200);
    EXPECT_FALSE(changed);
}

/// @brief it should demote a silent unit while another unit on the port answers.
TEST(RTU, DemotesSilentUnitWhileOthersAnswer) {
    auto cfg = serial_slave();
    cfg.silent_units = {3};
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    device::Manager manager;
    auto silent = ASSERT_NIL_P(manager.acquire(rtu(slave, 3)));
    auto live = ASSERT_NIL_P(manager.acquire(rtu(slave, 1)));

    uint16_t reg = 0;
    for (std::size_t i = 0; i < device::DEMOTE_AFTER; i++) {
        ASSERT_OCCURRED_AS(
            silent->read_registers(device::HoldingRegister, 0, 1, &reg),
            device::UNREACHABLE
        );
        ASSERT_NIL(live->read_registers(device::HoldingRegister, 0, 1, &reg));
    }
    const auto err = silent->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::UNREACHABLE);
    EXPECT_EQ(
        err.data,
        "unit 3 timed out 3 times in a row, so its requests fail for 10s"
    );
    EXPECT_EQ(slave.unit_requests(3), device::DEMOTE_AFTER);
    ASSERT_NIL(live->read_registers(device::HoldingRegister, 0, 1, &reg));
}

/// @brief it should leave a silence of 3.5 characters between two frames.
TEST(RTU, WaitsFrameGapBetweenRequests) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto cfg = rtu(slave, 1);
    cfg.serial.baud_rate = 1200;
    device::Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));

    constexpr int requests = 5;
    uint16_t reg = 0;
    const auto start = std::chrono::steady_clock::now();
    for (int i = 0; i < requests; i++)
        ASSERT_NIL(dev->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_GE(
        std::chrono::steady_clock::now() - start,
        cfg.serial.frame_gap().chrono() * (requests - 1)
    );
}

/// @brief it should report a reply that line noise damaged as a temporary error, and
/// serve the next request on the same port.
TEST(RTU, RecoversFromDamagedReply) {
    auto cfg = serial_slave();
    cfg.holding_registers[0] = static_cast<uint16_t>(1234);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    device::Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire(rtu(slave, 1)));

    uint16_t reg = 0;
    slave.damage_next(1);
    const auto err = dev->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::TEMPORARY_ERROR);
    EXPECT_EQ(err.data, modbus_strerror(EMBBADCRC));
    ASSERT_NIL(dev->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(reg, 1234);
}

/// @brief it should report an unplugged port, whose link is gone, as unreachable, and
/// serve the unit again once a port is back at the same path.
TEST(RTU, RecoversAfterUnplug) {
    auto cfg = serial_slave();
    cfg.holding_registers[0] = static_cast<uint16_t>(1234);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    std::filesystem::create_symlink(slave.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    device::Manager manager;
    auto link_cfg = rtu(slave, 1);
    link_cfg.serial.port = link.string();
    auto dev = ASSERT_NIL_P(manager.acquire(link_cfg));
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(device::HoldingRegister, 0, 1, &reg));

    slave.stop();
    std::filesystem::remove(link);
    const auto err = dev->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::UNREACHABLE);
    EXPECT_EQ(err.data, modbus_strerror(ENOENT));

    ASSERT_NIL(slave.start());
    std::filesystem::create_symlink(slave.serial_port(), link);
    reg = 0;
    ASSERT_NIL(dev->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(reg, 1234);
}

/// @brief it should share the connection of a link that moved to a new port, as after
/// a replug, with a device acquired after the move.
TEST(RTU, SharesConnectionAfterPortMoves) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    testutil::Slave moved(serial_slave());
    ASSERT_NIL(moved.start());
    x::defer::defer stop_moved([&moved] { moved.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    std::filesystem::create_symlink(slave.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    device::Manager manager;
    auto link_cfg = rtu(slave, 1);
    link_cfg.serial.port = link.string();
    auto first = ASSERT_NIL_P(manager.acquire(link_cfg));
    uint16_t reg = 0;
    ASSERT_NIL(first->read_registers(device::HoldingRegister, 0, 1, &reg));

    slave.stop();
    ASSERT_OCCURRED_AS(
        first->read_registers(device::HoldingRegister, 0, 1, &reg),
        device::UNREACHABLE
    );
    std::filesystem::remove(link);
    std::filesystem::create_symlink(moved.serial_port(), link);
    ASSERT_NIL(first->read_registers(device::HoldingRegister, 0, 1, &reg));
    auto second = ASSERT_NIL_P(manager.acquire(link_cfg));
    ASSERT_NIL(second->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(moved.unit_requests(1), 2);
}

/// @brief it should give a link and the path it names separate connections, so that a
/// device on the path stays on its port when the link moves to another port.
TEST(RTU, KeepsPathOnPortWhenLinkMoves) {
    auto before_cfg = serial_slave();
    before_cfg.holding_registers[0] = static_cast<uint16_t>(1);
    testutil::Slave before(before_cfg);
    ASSERT_NIL(before.start());
    x::defer::defer stop_before([&before] { before.stop(); });
    auto after_cfg = serial_slave();
    after_cfg.holding_registers[0] = static_cast<uint16_t>(2);
    testutil::Slave after(after_cfg);
    ASSERT_NIL(after.start());
    x::defer::defer stop_after([&after] { after.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    std::filesystem::create_symlink(before.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    device::Manager manager;
    auto link_cfg = rtu(before, 1);
    link_cfg.serial.port = link.string();
    auto linked = ASSERT_NIL_P(manager.acquire(link_cfg));
    auto direct = ASSERT_NIL_P(manager.acquire(rtu(before, 1)));
    uint16_t reg = 0;
    ASSERT_NIL(linked->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(reg, 1);
    const auto err = direct->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::UNREACHABLE);
    EXPECT_EQ(err.data, "serial port is in use by another connection");

    std::filesystem::remove(link);
    std::filesystem::create_symlink(after.serial_port(), link);
    ASSERT_NIL(linked->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(reg, 2);
    ASSERT_NIL(direct->read_registers(device::HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(reg, 1);
}

/// @brief it should return an unreachable error when the serial port does not exist.
TEST(RTU, MissingPortIsUnreachable) {
    device::Manager manager;
    auto dev = ASSERT_NIL_P(
        manager.acquire(device::ConnectionConfig(device::Serial{.port = "/dev/nope"}))
    );
    uint16_t reg = 0;
    const auto err = dev->read_registers(device::HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, device::UNREACHABLE);
    EXPECT_EQ(err.data, modbus_strerror(ENOENT));
}

/// @brief it should pass a connection test of an RTU unit.
TEST(RTU, ScannerTestsConnection) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    auto cmd = test_command(t, rtu(slave, 1));

    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(ctx->statuses[0].message, "Connected to device");
}

/// @brief it should accept new line settings for a serial port that only the scanner
/// holds open, in a connection test and in the next scans.
TEST(RTU, ScannerAcceptsNewLineSettings) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    std::unordered_map<std::string, synnax::device::Device> devices{
        {"dev", rtu_device(slave, "dev", 1)},
    };
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;
    ASSERT_NIL_P(scanner.scan(scan_ctx));
    auto cfg = rtu(slave, 1);
    cfg.serial.baud_rate = 9600;

    auto cmd = test_command(t, cfg);
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_NIL_P(scanner.scan(scan_ctx));
    devices["dev"].properties = {{"connection", cfg.to_json()}};
    const auto scanned = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(scanned.size(), 1);
    const auto status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Device connected");
}

/// @brief it should accept a new path to a serial port that only the scanner holds
/// open, in a connection test and in the next scans.
TEST(RTU, ScannerAcceptsNewPathToPort) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    const auto link = std::filesystem::temp_directory_path() /
                      ("modbus-rtu-" + x::uuid::create().to_string());
    std::filesystem::create_symlink(slave.serial_port(), link);
    x::defer::defer remove_link([&link] { std::filesystem::remove(link); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    std::unordered_map<std::string, synnax::device::Device> devices{
        {"dev", rtu_device(slave, "dev", 1)},
    };
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;
    ASSERT_NIL_P(scanner.scan(scan_ctx));
    auto cfg = rtu(slave, 1);
    cfg.serial.port = link.string();

    auto cmd = test_command(t, cfg);
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_NIL_P(scanner.scan(scan_ctx));
    devices["dev"].properties = {{"connection", cfg.to_json()}};
    const auto scanned = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(scanned.size(), 1);
    const auto status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Device connected");
}

/// @brief it should wait for a scan's probe of a serial port to end before it tests the
/// port at new line settings.
TEST(RTU, ScannerTestsNewLineSettingsDuringProbe) {
    auto slave_cfg = serial_slave();
    slave_cfg.silent_units = {4};
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    auto silent = rtu(slave, 4);
    silent.timeout = x::telem::MILLISECOND * 500;
    std::unordered_map<std::string, synnax::device::Device> devices{
        {"silent", rtu_device(slave, "silent", 4)},
    };
    devices["silent"].properties = {{"connection", silent.to_json()}};
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;
    std::thread scanning([&] { ASSERT_NIL_P(scanner.scan(scan_ctx)); });
    x::defer::defer join([&scanning] { scanning.join(); });
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    auto cfg = rtu(slave, 1);
    cfg.serial.baud_rate = 9600;

    auto cmd = test_command(t, cfg);
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
}

/// @brief it should fail a connection test at new line settings for a serial port that
/// a task holds open.
TEST(RTU, ScannerRejectsLineSettingsOfOpenPort) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    t.key = x::uuid::create();
    auto manager = std::make_shared<device::Manager>();
    const auto held = ASSERT_NIL_P(manager->acquire(rtu(slave, 1)));
    Scanner scanner(ctx, t, manager);
    auto cfg = rtu(slave, 1);
    cfg.serial.baud_rate = 9600;

    auto cmd = test_command(t, cfg);
    ASSERT_TRUE(scanner.exec(cmd, t, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to connect to device");
    EXPECT_EQ(
        ctx->statuses[0].description,
        "serial port " + slave.serial_port() +
            " is open at 19200 8E1, but this device uses 9600 8E1"
    );
}

/// @brief it should keep a serial port at the line settings of the first saved server
/// on it, and report the conflict on a saved server with other settings in every scan.
TEST(RTU, ScannerKeepsPortForFirstServer) {
    testutil::Slave slave(serial_slave());
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    auto slow = rtu(slave, 2);
    slow.serial.baud_rate = 9600;
    std::unordered_map<std::string, synnax::device::Device> devices{
        {"fast", rtu_device(slave, "fast", 1)},
        {"slow", rtu_device(slave, "slow", 2)},
    };
    devices["slow"].properties = {{"connection", slow.to_json()}};
    const std::unordered_map<std::string, std::string> lines{
        {"fast", "19200 8E1"},
        {"slow", "9600 8E1"},
    };
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;

    std::string first;
    for (int scan = 0; scan < 2; scan++) {
        const auto scanned = ASSERT_NIL_P(scanner.scan(scan_ctx));
        ASSERT_EQ(scanned.size(), 2);
        for (const auto &dev: scanned)
            if (dev.status.value().variant == synnax::status::VARIANT_SUCCESS) {
                if (first.empty()) first = dev.key;
                EXPECT_EQ(dev.key, first);
            }
        ASSERT_FALSE(first.empty());
        for (const auto &dev: scanned) {
            if (dev.key == first) continue;
            const auto status = dev.status.value();
            EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
            EXPECT_EQ(status.message, "Failed to reach device");
            EXPECT_EQ(
                status.description,
                "serial port " + slave.serial_port() + " is open at " +
                    lines.at(first) + ", but this device uses " + lines.at(dev.key)
            );
        }
    }
}

/// @brief it should report each unit on a serial port by whether it answers.
TEST(RTU, ScannerChecksEachUnit) {
    auto cfg = serial_slave();
    cfg.silent_units = {4};
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });
    auto ctx = std::make_shared<task::MockContext>(nullptr);
    synnax::task::Task t;
    Scanner scanner(ctx, t, std::make_shared<device::Manager>());
    std::unordered_map<std::string, synnax::device::Device> devices{
        {"live", rtu_device(slave, "live", 1)},
        {"silent", rtu_device(slave, "silent", 4)},
    };
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;

    const auto scanned = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(scanned.size(), 2);
    for (const auto &dev: scanned) {
        const auto status = dev.status.value();
        if (dev.key == "live") {
            EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
            EXPECT_EQ(status.message, "Device connected");
        } else {
            EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
            EXPECT_EQ(status.message, "Failed to reach device");
            EXPECT_EQ(status.description, modbus_strerror(ETIMEDOUT));
        }
    }
}
}
