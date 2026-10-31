// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <map>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/testutil/slave.h"

namespace driver::modbus::device {
/// @brief it should share one connection among the devices on an endpoint and route
/// each request to the device's unit.
TEST(ManagerTest, SharesConnectionPerEndpoint) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1520;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());

    Manager manager;
    ConnectionConfig first{"127.0.0.1", 1520};
    first.unit_id = 1;
    ConnectionConfig second{"127.0.0.1", 1520};
    second.unit_id = 2;
    auto dev1 = ASSERT_NIL_P(manager.acquire(first));
    auto dev2 = ASSERT_NIL_P(manager.acquire(second));
    uint16_t reg = 0;
    ASSERT_NIL(dev1->read_registers(HoldingRegister, 0, 1, &reg));
    ASSERT_NIL(dev2->read_registers(HoldingRegister, 0, 1, &reg));
    ASSERT_NIL(dev2->read_registers(HoldingRegister, 0, 1, &reg));

    EXPECT_EQ(slave.connections(), 1);
    EXPECT_EQ(slave.unit_requests(1), 1);
    EXPECT_EQ(slave.unit_requests(2), 2);
    slave.stop();
}

/// @brief it should close the connection when its last device is released, and open a
/// new one for the next device.
TEST(ManagerTest, ClosesConnectionWithLastDevice) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1521;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());

    Manager manager;
    uint16_t reg = 0;
    {
        auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1521}));
        ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    }
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1521}));
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(slave.connections(), 2);
    slave.stop();
}

/// @brief it should keep each request whole when threads share a connection.
TEST(ManagerTest, ConcurrentRequestsShareConnection) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1522;
    slave_config.holding_registers[0] = static_cast<uint16_t>(42);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());

    Manager manager;
    std::atomic<int> failures{0};
    std::vector<std::thread> threads;
    threads.reserve(8);
    for (int i = 0; i < 8; i++)
        threads.emplace_back([&manager, &failures, i] {
            ConnectionConfig cfg{"127.0.0.1", 1522};
            cfg.unit_id = static_cast<std::uint8_t>(i % 2 + 1);
            auto [dev, err] = manager.acquire(cfg);
            if (err) {
                failures++;
                return;
            }
            for (int j = 0; j < 25; j++) {
                uint16_t reg = 0;
                if (dev->read_registers(HoldingRegister, 0, 1, &reg) || reg != 42)
                    failures++;
            }
        });
    for (auto &t: threads)
        t.join();

    EXPECT_EQ(failures.load(), 0);
    EXPECT_EQ(slave.unit_requests(1) + slave.unit_requests(2), 200);
    slave.stop();
}

/// @brief it should open one connection for each endpoint when threads acquire and read
/// at once.
TEST(ManagerTest, ConcurrentAcquireIsThreadSafe) {
    std::vector<std::unique_ptr<testutil::Slave>> slaves;
    for (int i = 0; i < 4; i++) {
        testutil::SlaveConfig config;
        config.host = "127.0.0.1";
        config.port = (1510 + i);
        auto slave = std::make_unique<testutil::Slave>(config);
        ASSERT_NIL(slave->start());
        slaves.push_back(std::move(slave));
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    std::atomic<int> error_count{0};
    std::vector<std::shared_ptr<Device>> devs(20);

    std::vector<std::thread> threads;
    threads.reserve(20);
    for (int i = 0; i < 20; i++) {
        threads.emplace_back([&manager, &error_count, &devs, i]() {
            ConnectionConfig config{"127.0.0.1", static_cast<uint16_t>(1510 + (i % 4))};
            auto [dev, err] = manager.acquire(config);
            uint16_t reg = 0;
            if (err || dev->read_registers(HoldingRegister, 0, 1, &reg)) error_count++;
            devs[i] = std::move(dev);
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(error_count.load(), 0);
    for (auto &slave: slaves) {
        EXPECT_EQ(slave->connections(), 1);
        slave->stop();
    }
}

TEST(DeviceTest, ReadCoilsWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1543;
    slave_config.coils[0] = 1;
    slave_config.coils[1] = 0;
    slave_config.coils[2] = 1;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1543};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint8_t bits[3];
    ASSERT_NIL(dev->read_bits(Coil, 0, 3, bits));

    EXPECT_EQ(bits[0], 1);
    EXPECT_EQ(bits[1], 0);
    EXPECT_EQ(bits[2], 1);

    slave.stop();
}

TEST(DeviceTest, ReadDiscreteInputsWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1549;
    slave_config.discrete_inputs[0] = 1;
    slave_config.discrete_inputs[1] = 1;
    slave_config.discrete_inputs[2] = 0;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1549};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint8_t bits[3];
    ASSERT_NIL(dev->read_bits(DiscreteInput, 0, 3, bits));

    EXPECT_EQ(bits[0], 1);
    EXPECT_EQ(bits[1], 1);
    EXPECT_EQ(bits[2], 0);

    slave.stop();
}

TEST(DeviceTest, ReadHoldingRegistersWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1544;
    slave_config.holding_registers[0] = static_cast<uint16_t>(0x1234);
    slave_config.holding_registers[1] = static_cast<uint16_t>(0x5678);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1544};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint16_t regs[2];
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 2, regs));

    EXPECT_EQ(regs[0], 0x1234);
    EXPECT_EQ(regs[1], 0x5678);

    slave.stop();
}

TEST(DeviceTest, ReadInputRegistersWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1550;
    slave_config.input_registers[0] = static_cast<uint16_t>(0xAAAA);
    slave_config.input_registers[1] = static_cast<uint16_t>(0xBBBB);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1550};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint16_t regs[2];
    ASSERT_NIL(dev->read_registers(InputRegister, 0, 2, regs));

    EXPECT_EQ(regs[0], 0xAAAA);
    EXPECT_EQ(regs[1], 0xBBBB);

    slave.stop();
}

TEST(DeviceTest, WriteBitsWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1545;
    slave_config.coils[0] = 0;
    slave_config.coils[1] = 0;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1545};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint8_t bits_to_write[2] = {1, 1};
    ASSERT_NIL(dev->write_bits(0, 2, bits_to_write));

    uint8_t bits_read[2];
    ASSERT_NIL(dev->read_bits(Coil, 0, 2, bits_read));

    EXPECT_EQ(bits_read[0], 1);
    EXPECT_EQ(bits_read[1], 1);

    slave.stop();
}

TEST(DeviceTest, WriteRegistersWorks) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1546;
    slave_config.holding_registers[0] = static_cast<uint16_t>(0);
    slave_config.holding_registers[1] = static_cast<uint16_t>(0);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1546};

    auto dev = ASSERT_NIL_P(manager.acquire(config));

    uint16_t regs_to_write[2] = {0xABCD, 0xEF01};
    ASSERT_NIL(dev->write_registers(0, 2, regs_to_write));

    uint16_t regs_read[2];
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 2, regs_read));

    EXPECT_EQ(regs_read[0], 0xABCD);
    EXPECT_EQ(regs_read[1], 0xEF01);

    slave.stop();
}

TEST(DeviceTest, ServerStopsWhileConnected) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1547;
    slave_config.coils[0] = 1;
    auto slave = std::make_unique<testutil::Slave>(slave_config);
    ASSERT_NIL(slave->start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1547};

    auto dev = ASSERT_NIL_P(manager.acquire(config));
    uint8_t bits[1];
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, bits));

    slave->stop();
    slave.reset();

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    ASSERT_OCCURRED_AS(dev->read_bits(Coil, 0, 1, bits), UNREACHABLE);
}

TEST(DeviceTest, ReconnectAfterServerRestart) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1548;
    slave_config.coils[0] = 1;

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1548};

    {
        testutil::Slave slave(slave_config);
        ASSERT_NIL(slave.start());
        std::this_thread::sleep_for(std::chrono::milliseconds(100));

        auto dev = ASSERT_NIL_P(manager.acquire(config));

        uint8_t bits[1];
        ASSERT_NIL(dev->read_bits(Coil, 0, 1, bits));
        EXPECT_EQ(bits[0], 1);

        slave.stop();
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    {
        slave_config.coils[0] = 0;
        testutil::Slave slave(slave_config);
        ASSERT_NIL(slave.start());
        std::this_thread::sleep_for(std::chrono::milliseconds(100));

        auto dev = ASSERT_NIL_P(manager.acquire(config));

        uint8_t bits[1];
        ASSERT_NIL(dev->read_bits(Coil, 0, 1, bits));
        EXPECT_EQ(bits[0], 0);

        slave.stop();
    }
}

struct ClassifyCase {
    int errnum;
    x::errors::Error expected;
};

class ClassifyTest : public ::testing::TestWithParam<ClassifyCase> {};

/// @brief it should classify each libmodbus and system error and keep its message.
TEST_P(ClassifyTest, Classifies) {
    const auto [errnum, expected] = GetParam();
    const auto err = classify(errnum);
    ASSERT_OCCURRED_AS(err, expected);
    EXPECT_EQ(err.data, modbus_strerror(errnum));
    EXPECT_EQ(err.matches(UNREACHABLE), expected.matches(UNREACHABLE));
}

INSTANTIATE_TEST_SUITE_P(
    Errors,
    ClassifyTest,
    ::testing::Values(
        ClassifyCase{EMBXILFUN, CRITICAL_ERROR},
        ClassifyCase{EMBXILADD, CRITICAL_ERROR},
        ClassifyCase{EMBXILVAL, CRITICAL_ERROR},
        ClassifyCase{EMBMDATA, CRITICAL_ERROR},
        ClassifyCase{EINVAL, CRITICAL_ERROR},
        ClassifyCase{ENOMEM, CRITICAL_ERROR},
        ClassifyCase{EMBXSFAIL, TEMPORARY_ERROR},
        ClassifyCase{EMBXACK, TEMPORARY_ERROR},
        ClassifyCase{EMBXSBUSY, TEMPORARY_ERROR},
        ClassifyCase{EMBXNACK, TEMPORARY_ERROR},
        ClassifyCase{EMBXMEMPAR, TEMPORARY_ERROR},
        ClassifyCase{EMBXGPATH, TEMPORARY_ERROR},
        ClassifyCase{EMBXGTAR, TEMPORARY_ERROR},
        ClassifyCase{EMBBADCRC, TEMPORARY_ERROR},
        ClassifyCase{ETIMEDOUT, UNREACHABLE},
        ClassifyCase{ECONNRESET, UNREACHABLE},
        ClassifyCase{ECONNREFUSED, UNREACHABLE},
        ClassifyCase{EPIPE, UNREACHABLE},
        ClassifyCase{EBADF, UNREACHABLE},
        ClassifyCase{EMBBADDATA, UNREACHABLE}
    )
);

/// @brief it should retry a request on a new connection after the server drops the old
/// one.
TEST(DeviceTest, ReconnectsAfterDroppedConnection) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1620;
    slave_config.coils[0] = 1;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1620}));
    uint8_t bit = 0;
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));

    slave.drop_connections();
    bit = 0;
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    EXPECT_EQ(bit, 1);
    EXPECT_EQ(slave.connections(), 2);
    slave.stop();
}

/// @brief it should fail the request after a dropped connection when the device has no
/// retries, and reconnect on the next one.
TEST(DeviceTest, ReconnectsOnNextRequestWithoutRetries) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1626;
    slave_config.coils[0] = 1;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    ConnectionConfig cfg{"127.0.0.1", 1626};
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    uint8_t bit = 0;
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));

    slave.drop_connections();
    ASSERT_OCCURRED_AS(dev->read_bits(Coil, 0, 1, &bit), UNREACHABLE);
    bit = 0;
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    EXPECT_EQ(bit, 1);
    slave.stop();
}

/// @brief it should report an unreachable server when the server is gone, and connect
/// once it is back.
TEST(DeviceTest, ReusesDeviceAcrossServerRestart) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1621;
    slave_config.coils[0] = 1;
    auto slave = std::make_unique<testutil::Slave>(slave_config);
    ASSERT_NIL(slave->start());
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1621}));
    slave->stop();
    slave.reset();

    uint8_t bit = 0;
    ASSERT_OCCURRED_AS(dev->read_bits(Coil, 0, 1, &bit), UNREACHABLE);
    ASSERT_OCCURRED_AS(dev->read_bits(Coil, 0, 1, &bit), UNREACHABLE);

    slave = std::make_unique<testutil::Slave>(slave_config);
    ASSERT_NIL(slave->start());
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    EXPECT_EQ(bit, 1);
    slave->stop();
}

/// @brief it should not count a timed-out connect against the unit, since the request
/// never reached it.
TEST(DeviceTest, ConnectTimeoutDoesNotDemoteUnit) {
#ifdef __APPLE__
    GTEST_SKIP() << "macOS resets a connect to a full accept queue";
#endif
    // A full accept queue drops new connects, so they time out instead of failing.
    auto *listener = modbus_new_tcp("127.0.0.1", 1690);
    const int fd = modbus_tcp_listen(listener, 1);
    ASSERT_NE(fd, -1);
    modbus_set_socket(listener, fd);
    std::vector<modbus_t *> queued;
    for (int i = 0; i < 8; i++) {
        auto *ctx = modbus_new_tcp("127.0.0.1", 1690);
        modbus_set_response_timeout(ctx, 0, 100000);
        modbus_connect(ctx);
        queued.push_back(ctx);
    }
    Manager manager;
    ConnectionConfig cfg{"127.0.0.1", 1690};
    cfg.timeout = x::telem::MILLISECOND * 100;
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));

    uint8_t bit = 0;
    for (std::size_t i = 0; i <= DEMOTE_AFTER; i++) {
        const auto err = dev->read_bits(Coil, 0, 1, &bit);
        ASSERT_OCCURRED_AS(err, UNREACHABLE);
        EXPECT_EQ(err.data, modbus_strerror(ETIMEDOUT));
    }
    EXPECT_EQ(dev->stats().requests, DEMOTE_AFTER + 1);
    EXPECT_EQ(dev->stats().timeouts, 0U);

    for (auto *ctx: queued) {
        modbus_close(ctx);
        modbus_free(ctx);
    }
    modbus_close(listener);
    modbus_free(listener);
}

/// @brief it should return a critical error for an address the server does not have,
/// and keep the connection.
TEST(DeviceTest, IllegalAddressIsCritical) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1622;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1622}));
    uint16_t reg = 0;
    const auto err = dev->read_registers(HoldingRegister, 500, 1, &reg);
    ASSERT_OCCURRED_AS(err, CRITICAL_ERROR);
    EXPECT_EQ(err.data, "Illegal data address");
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    slave.stop();
}

/// @brief it should return a critical error for a function the server rejects.
TEST(DeviceTest, RejectedFunctionIsCritical) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1623;
    slave_config.reject_function_codes = {MODBUS_FC_READ_COILS};
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1623}));
    uint8_t bit = 0;
    const auto err = dev->read_bits(Coil, 0, 1, &bit);
    ASSERT_OCCURRED_AS(err, CRITICAL_ERROR);
    EXPECT_EQ(err.data, "Illegal function");
    slave.stop();
}

/// @brief it should connect to a server by host name.
TEST(ManagerTest, AcquiresByHostName) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1624;
    slave_config.coils[0] = 1;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"localhost", 1624}));
    uint8_t bit = 0;
    ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    EXPECT_EQ(bit, 1);
    slave.stop();
}

/// @brief it should acquire a device without connecting, and fail its requests as
/// unreachable when no server listens.
TEST(ManagerTest, RequestWithoutServerIsUnreachable) {
    Manager manager;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1625}));
    uint8_t bit = 0;
    ASSERT_OCCURRED_AS(dev->read_bits(Coil, 0, 1, &bit), UNREACHABLE);
}

TEST(ConnectionConfigTest, ToJsonWorks) {
    const ConnectionConfig config{"192.168.1.100", 502, true, false};
    x::json::json j = config.to_json();
    EXPECT_EQ(j["host"], "192.168.1.100");
    EXPECT_EQ(j["port"], 502);
    EXPECT_EQ(j["swap_bytes"], true);
    EXPECT_EQ(j["swap_words"], false);
}

TEST(ConnectionConfigTest, FromJsonWorks) {
    x::json::json j = {
        {"host", "10.0.0.50"},
        {"port", 1502},
        {"swap_bytes", false},
        {"swap_words", true}
    };

    x::json::Parser parser(j);
    ConnectionConfig config(parser);

    EXPECT_EQ(config.host, "10.0.0.50");
    EXPECT_EQ(config.port, 1502);
    EXPECT_EQ(config.swap_bytes, false);
    EXPECT_EQ(config.swap_words, true);
    EXPECT_EQ(config.unit_id, 255);
    EXPECT_EQ(config.timeout, x::telem::SECOND);
    EXPECT_EQ(config.retries, 1);
}

/// @brief it should parse the unit, timeout, and retries, and round trip them.
TEST(ConnectionConfigTest, ParsesUnitTimeoutAndRetries) {
    x::json::Parser parser(
        x::json::json{
            {"transport", "tcp"},
            {"host", "gateway"},
            {"port", 502},
            {"swap_bytes", false},
            {"swap_words", false},
            {"unit_id", 7},
            {"timeout_ms", 250},
            {"retries", 3},
        }
    );
    const ConnectionConfig config(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(config.unit_id, 7);
    EXPECT_EQ(config.timeout, x::telem::MILLISECOND * 250);
    EXPECT_EQ(config.retries, 3);
    const auto j = config.to_json();
    EXPECT_EQ(j["unit_id"], 7);
    EXPECT_EQ(j["timeout_ms"], 250);
    EXPECT_EQ(j["retries"], 3);
}

struct InvalidConnectionCase {
    std::string field;
    x::json::json value;
    std::string message;
};

class InvalidConnectionTest : public ::testing::TestWithParam<InvalidConnectionCase> {};

/// @brief it should reject a connection field out of range, naming the field.
TEST_P(InvalidConnectionTest, Rejects) {
    const auto &[field, value, message] = GetParam();
    x::json::json j = {
        {"host", "gateway"},
        {"port", 502},
        {"swap_bytes", false},
        {"swap_words", false},
    };
    j[field] = value;
    x::json::Parser parser(j);
    const ConnectionConfig config(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errs = parser.error_json()["errors"];
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], field);
    EXPECT_EQ(errs[0]["message"], message);
}

INSTANTIATE_TEST_SUITE_P(
    Fields,
    InvalidConnectionTest,
    ::testing::Values(
        InvalidConnectionCase{"transport", "ascii", "must be tcp or rtu"},
        InvalidConnectionCase{"unit_id", 256, "must be between 0 and 255"},
        InvalidConnectionCase{"timeout_ms", 0, "must be between 1 and 60000"},
        InvalidConnectionCase{"retries", 11, "must be between 0 and 10"}
    )
);

/// @brief it should parse every serial field and round trip them.
TEST(ConnectionConfigTest, ParsesSerial) {
    x::json::Parser parser(
        x::json::json{
            {"transport", "rtu"},
            {"serial_port", "/dev/ttyUSB0"},
            {"baud_rate", 9600},
            {"parity", "none"},
            {"stop_bits", 2},
            {"swap_bytes", false},
            {"swap_words", false},
            {"unit_id", 17},
        }
    );
    const ConnectionConfig config(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(config.transport, Transport::RTU);
    EXPECT_EQ(config.serial.port, "/dev/ttyUSB0");
    EXPECT_EQ(config.serial.baud_rate, 9600);
    EXPECT_EQ(config.serial.parity, 'N');
    EXPECT_EQ(config.serial.stop_bits, 2);
    EXPECT_EQ(config.unit_id, 17);
    EXPECT_EQ(config.endpoint(), canonical_port("/dev/ttyUSB0"));
    const auto j = config.to_json();
    EXPECT_EQ(j["transport"], "rtu");
    EXPECT_EQ(j["serial_port"], "/dev/ttyUSB0");
    EXPECT_EQ(j["baud_rate"], 9600);
    EXPECT_EQ(j["parity"], "none");
    EXPECT_EQ(j["stop_bits"], 2);
    EXPECT_EQ(j["unit_id"], 17);
    EXPECT_FALSE(j.contains("data_bits"));
    EXPECT_FALSE(j.contains("host"));
    EXPECT_FALSE(j.contains("port"));
}

/// @brief it should default the line settings to 19200 8E1 and the unit to 1.
TEST(ConnectionConfigTest, DefaultsSerial) {
    x::json::Parser parser(
        x::json::json{
            {"transport", "rtu"},
            {"serial_port", "COM3"},
            {"swap_bytes", false},
            {"swap_words", false},
        }
    );
    const ConnectionConfig config(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(config.serial.line(), "19200 8E1");
    EXPECT_EQ(config.unit_id, 1);
    EXPECT_EQ(config.to_json()["parity"], "even");
}

/// @brief it should accept the lowest and the highest baud rate.
TEST(ConnectionConfigTest, AcceptsBaudRateBounds) {
    for (const int rate: {1200, 230400}) {
        x::json::Parser parser(
            x::json::json{
                {"transport", "rtu"},
                {"serial_port", "/dev/ttyUSB0"},
                {"baud_rate", rate},
                {"swap_bytes", false},
                {"swap_words", false},
            }
        );
        const ConnectionConfig config(parser);
        ASSERT_NIL(parser.error());
        EXPECT_EQ(config.serial.baud_rate, rate);
    }
}

const std::string BAUD_RATE_ERROR = "must be one of 1200, 2400, 4800, 9600, 19200, "
                                    "38400, 57600, 115200, 230400";

class InvalidSerialTest : public ::testing::TestWithParam<InvalidConnectionCase> {};

/// @brief it should reject a serial field out of range, naming the field.
TEST_P(InvalidSerialTest, Rejects) {
    const auto &[field, value, message] = GetParam();
    x::json::json j = {
        {"transport", "rtu"},
        {"serial_port", "/dev/ttyUSB0"},
        {"swap_bytes", false},
        {"swap_words", false},
    };
    j[field] = value;
    x::json::Parser parser(j);
    const ConnectionConfig config(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errs = parser.error_json()["errors"];
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], field);
    EXPECT_EQ(errs[0]["message"], message);
}

INSTANTIATE_TEST_SUITE_P(
    Fields,
    InvalidSerialTest,
    ::testing::Values(
        InvalidConnectionCase{"serial_port", "", "must not be empty"},
        InvalidConnectionCase{"baud_rate", 14400, BAUD_RATE_ERROR},
        InvalidConnectionCase{"baud_rate", 600, BAUD_RATE_ERROR},
        InvalidConnectionCase{"baud_rate", 460800, BAUD_RATE_ERROR},
        InvalidConnectionCase{"parity", "mark", "must be none, even, or odd"},
        InvalidConnectionCase{"stop_bits", 3, "must be 1 or 2"},
        InvalidConnectionCase{"unit_id", 0, "must be between 1 and 247"},
        InvalidConnectionCase{"unit_id", 248, "must be between 1 and 247"}
    )
);

/// @brief it should reject an empty serial port when another field is also invalid.
TEST(ConnectionConfigTest, RejectsEmptySerialPortWithOtherErrors) {
    x::json::Parser parser(
        x::json::json{
            {"transport", "rtu"},
            {"serial_port", ""},
            {"swap_words", false},
        }
    );
    const ConnectionConfig config(parser);
    const auto errs = parser.error_json()["errors"];
    ASSERT_EQ(errs.size(), 2);
    EXPECT_EQ(errs[0]["path"], "swap_bytes");
    EXPECT_EQ(errs[1]["path"], "serial_port");
    EXPECT_EQ(errs[1]["message"], "must not be empty");
}

/// @brief it should report a missing serial port once.
TEST(ConnectionConfigTest, RejectsMissingSerialPortOnce) {
    x::json::Parser parser(
        x::json::json{
            {"transport", "rtu"},
            {"swap_bytes", false},
            {"swap_words", false}
        }
    );
    const ConnectionConfig config(parser);
    const auto errs = parser.error_json()["errors"];
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], "serial_port");
    EXPECT_EQ(errs[0]["message"], "This field is required");
}

/// @brief it should prefix a Windows COM port with the device namespace and leave every
/// other path as it is.
TEST(SerialPathTest, PrefixesComPorts) {
    EXPECT_EQ(serial_path("COM3"), R"(\\.\COM3)");
    EXPECT_EQ(serial_path("com12"), R"(\\.\com12)");
    EXPECT_EQ(serial_path("COM256"), R"(\\.\COM256)");
    EXPECT_EQ(serial_path(R"(\\.\COM3)"), R"(\\.\COM3)");
    EXPECT_EQ(serial_path("COM"), "COM");
    EXPECT_EQ(serial_path("COMX"), "COMX");
    EXPECT_EQ(serial_path("/dev/ttyUSB0"), "/dev/ttyUSB0");
}

/// @brief it should wait 3.5 characters between frames, and 1.75 ms above 19200 baud.
TEST(SerialTest, FrameGap) {
    EXPECT_EQ(
        (Serial{.baud_rate = 9600, .parity = 'E'}).frame_gap(),
        x::telem::TimeSpan(4010416)
    );
    EXPECT_EQ(
        (Serial{.baud_rate = 19200, .parity = 'N'}).frame_gap(),
        x::telem::TimeSpan(1822916)
    );
    EXPECT_EQ((Serial{.baud_rate = 115200}).frame_gap(), x::telem::MICROSECOND * 1750);
}

/// @brief it should refuse a device whose line settings differ from the open serial
/// port, naming both.
TEST(ManagerTest, RejectsMismatchedLineSettings) {
    Manager manager;
    const ConnectionConfig first(Serial{.port = "/dev/ttyMISMATCH"});
    const auto dev = ASSERT_NIL_P(manager.acquire(first));
    ConnectionConfig second = first;
    second.unit_id = 2;
    const auto same_line = ASSERT_NIL_P(manager.acquire(second));
    second.serial.baud_rate = 9600;
    second.serial.parity = 'N';
    const auto err = manager.acquire(second).second;
    ASSERT_OCCURRED_AS(err, CRITICAL_ERROR);
    EXPECT_EQ(
        err.data,
        "serial port /dev/ttyMISMATCH is open at 19200 8E1, but this device uses "
        "9600 8N1"
    );
}

/// @brief it should default every request field when the object is missing.
TEST(RequestConfigTest, DefaultsWhenMissing) {
    const x::json::Parser parser(
        x::json::json{
            {"connection",
             {{"host", "a"},
              {"port", 502},
              {"swap_bytes", false},
              {"swap_words", false}}},
        }
    );
    const Properties props(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(props.requests.max_registers, 125);
    EXPECT_EQ(props.requests.max_gap, 125);
    EXPECT_EQ(props.requests.write_codes, WriteCodes::Multiple);
    EXPECT_FALSE(props.requests.mask_write_supported);
    EXPECT_EQ(props.requests.delay, x::telem::TimeSpan(0));
}

/// @brief it should parse every request field.
TEST(RequestConfigTest, ParsesFields) {
    x::json::Parser parser(
        x::json::json{
            {"max_registers", 60},
            {"max_gap", 0},
            {"write_codes", "single"},
            {"mask_write_supported", true},
            {"delay_ms", 20},
        }
    );
    const RequestConfig requests(parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(requests.max_registers, 60);
    EXPECT_EQ(requests.max_gap, 0);
    EXPECT_EQ(requests.write_codes, WriteCodes::Single);
    EXPECT_TRUE(requests.mask_write_supported);
    EXPECT_EQ(requests.delay, x::telem::MILLISECOND * 20);
}

/// @brief it should reject a request field out of range, naming the field.
TEST(RequestConfigTest, RejectsInvalidFields) {
    x::json::Parser parser(
        x::json::json{
            {"max_registers", 126},
            {"max_gap", -1},
            {"write_codes", "both"},
            {"delay_ms", 5001},
        }
    );
    const RequestConfig requests(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errs = parser.error_json()["errors"];
    ASSERT_EQ(errs.size(), 4);
    EXPECT_EQ(errs[0]["path"], "max_registers");
    EXPECT_EQ(errs[0]["message"], "must be between 1 and 125");
    EXPECT_EQ(errs[1]["path"], "max_gap");
    EXPECT_EQ(errs[1]["message"], "must be between 0 and 65535");
    EXPECT_EQ(errs[2]["path"], "write_codes");
    EXPECT_EQ(errs[2]["message"], "must be multiple or single");
    EXPECT_EQ(errs[3]["path"], "delay_ms");
    EXPECT_EQ(errs[3]["message"], "must be between 0 and 5000");
}

namespace {
/// @returns a connection to a unit on the local test slave with a short timeout.
ConnectionConfig unit(const std::uint16_t port, const std::uint8_t id) {
    ConnectionConfig cfg{"127.0.0.1", port};
    cfg.unit_id = id;
    cfg.timeout = x::telem::MILLISECOND * 50;
    return cfg;
}
}

/// @brief it should retry a timed-out request, demote the unit after three timed-out
/// attempts in a row, and keep serving the other units on the connection.
TEST(DeviceTest, DemotesSilentUnit) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1627;
    slave_config.silent_units = {9};
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto silent = ASSERT_NIL_P(manager.acquire(unit(1627, 9)));
    auto healthy = ASSERT_NIL_P(manager.acquire(unit(1627, 1)));
    uint16_t reg = 0;

    ASSERT_OCCURRED_AS(
        silent->read_registers(HoldingRegister, 0, 1, &reg),
        UNREACHABLE
    );
    EXPECT_EQ(slave.unit_requests(9), 2);
    const auto err = silent->read_registers(HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, UNREACHABLE);
    EXPECT_EQ(
        err.data,
        "unit 9 timed out 3 times in a row, so its requests fail for 10s"
    );
    EXPECT_EQ(slave.unit_requests(9), 3);

    ASSERT_OCCURRED_AS(
        silent->read_registers(HoldingRegister, 0, 1, &reg),
        UNREACHABLE
    );
    EXPECT_EQ(slave.unit_requests(9), 3);
    ASSERT_NIL(healthy->read_registers(HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(slave.connections(), 1);
    slave.stop();
}

/// @brief it should send the requests of a demoted unit again once the demotion time
/// passes.
TEST(DeviceTest, ServesUnitAfterDemotion) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1654;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager(x::telem::MILLISECOND * 100);
    auto cfg = unit(1654, 6);
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    uint16_t reg = 0;

    slave.set_silent(6, true);
    for (int i = 0; i < 2; i++)
        ASSERT_OCCURRED_AS(
            dev->read_registers(HoldingRegister, 0, 1, &reg),
            UNREACHABLE
        );
    const auto err = dev->read_registers(HoldingRegister, 0, 1, &reg);
    ASSERT_OCCURRED_AS(err, UNREACHABLE);
    EXPECT_EQ(
        err.data,
        "unit 6 timed out 3 times in a row, so its requests fail for 100ms"
    );
    slave.set_silent(6, false);
    ASSERT_OCCURRED_AS(dev->read_registers(HoldingRegister, 0, 1, &reg), UNREACHABLE);
    EXPECT_EQ(slave.unit_requests(6), 3);

    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    EXPECT_EQ(slave.unit_requests(6), 4);
    slave.stop();
}

/// @brief it should reset the timed-out count of a unit when it answers.
TEST(DeviceTest, ReplyResetsTimeouts) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1628;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto cfg = unit(1628, 4);
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    uint16_t reg = 0;

    for (int i = 0; i < 2; i++) {
        slave.set_silent(4, true);
        ASSERT_OCCURRED_AS(
            dev->read_registers(HoldingRegister, 0, 1, &reg),
            UNREACHABLE
        );
        slave.set_silent(4, false);
        ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    }
    slave.set_silent(4, true);
    ASSERT_OCCURRED_AS(dev->read_registers(HoldingRegister, 0, 1, &reg), UNREACHABLE);
    slave.set_silent(4, false);
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    slave.stop();
}

/// @brief it should discard a reply that arrives after its request timed out, so the
/// next request reads its own reply.
TEST(DeviceTest, DiscardsLateReply) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1629;
    slave_config.holding_registers[0] = static_cast<uint16_t>(11);
    slave_config.holding_registers[1] = static_cast<uint16_t>(22);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto cfg = unit(1629, 1);
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));

    slave.stall_next(1, std::chrono::milliseconds(150));
    uint16_t reg = 0;
    ASSERT_OCCURRED_AS(dev->read_registers(HoldingRegister, 0, 1, &reg), UNREACHABLE);
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    ASSERT_NIL(dev->read_registers(HoldingRegister, 1, 1, &reg));
    EXPECT_EQ(reg, 22);
    EXPECT_EQ(slave.connections(), 1);
    slave.stop();
}

/// @brief it should send a waiting write before reads that waited longer.
TEST(DeviceTest, WriteGoesBeforeWaitingReads) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1650;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    ConnectionConfig cfg{"127.0.0.1", 1650};
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));

    slave.stall_next(MODBUS_TCP_SLAVE, std::chrono::milliseconds(300));
    std::thread slow([&] {
        uint16_t r = 0;
        ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &r));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    std::thread read([&] {
        uint8_t bit = 0;
        ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    std::thread write([&] {
        const uint16_t value = 5;
        ASSERT_NIL(dev->write_registers(0, 1, &value));
    });
    slow.join();
    read.join();
    write.join();

    const std::vector<int> expected{
        MODBUS_FC_READ_HOLDING_REGISTERS,
        MODBUS_FC_READ_HOLDING_REGISTERS,
        MODBUS_FC_WRITE_MULTIPLE_REGISTERS,
        MODBUS_FC_READ_COILS,
    };
    EXPECT_EQ(slave.log(), expected);
    slave.stop();
}

/// @brief it should count the requests, retries, timeouts, and exceptions of each unit
/// on a shared connection, and time each reply.
TEST(DeviceTest, CountsRequestsOfEachUnit) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1677;
    slave_config.gateway_failed_units = {2};
    slave_config.silent_units = {3};
    slave_config.reject_function_codes = {MODBUS_FC_WRITE_MULTIPLE_COILS};
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    const auto acquire = [&](const std::uint8_t unit) {
        ConnectionConfig cfg{"127.0.0.1", 1677};
        cfg.unit_id = unit;
        cfg.timeout = x::telem::MILLISECOND * 50;
        return manager.acquire(cfg);
    };
    auto healthy = ASSERT_NIL_P(acquire(1));
    auto gateway = ASSERT_NIL_P(acquire(2));
    auto silent = ASSERT_NIL_P(acquire(3));
    uint16_t reg = 0;
    slave.stall_next(1, std::chrono::milliseconds(30));
    ASSERT_NIL(healthy->read_registers(HoldingRegister, 0, 1, &reg));
    ASSERT_NIL(healthy->read_registers(HoldingRegister, 0, 1, &reg));
    const uint8_t bit = 1;
    ASSERT_OCCURRED_AS(healthy->write_bits(0, 1, &bit), CRITICAL_ERROR);
    ASSERT_OCCURRED_AS(
        gateway->read_registers(HoldingRegister, 0, 1, &reg),
        TEMPORARY_ERROR
    );
    ASSERT_OCCURRED_AS(
        silent->read_registers(HoldingRegister, 0, 1, &reg),
        UNREACHABLE
    );

    const auto h = healthy->stats();
    EXPECT_EQ(h.requests, 3U);
    EXPECT_EQ(h.retries, 0U);
    EXPECT_EQ(h.timeouts, 0U);
    EXPECT_EQ(h.exceptions, 1U);
    EXPECT_GE(h.max_response, x::telem::MILLISECOND * 30);
    EXPECT_LT(h.last_response, h.max_response);
    const auto g = gateway->stats();
    EXPECT_EQ(g.requests, 2U);
    EXPECT_EQ(g.retries, 1U);
    EXPECT_EQ(g.timeouts, 0U);
    EXPECT_EQ(g.exceptions, 2U);
    EXPECT_GT(g.max_response, x::telem::TimeSpan(0));
    EXPECT_EQ(gateway->last_reply(), Connection::Clock::time_point{});
    const auto s = silent->stats();
    EXPECT_EQ(s.requests, 2U);
    EXPECT_EQ(s.retries, 1U);
    EXPECT_EQ(s.timeouts, 2U);
    EXPECT_EQ(s.exceptions, 0U);
    EXPECT_EQ(s.max_response, x::telem::TimeSpan(0));
    EXPECT_EQ(
        s.to_json(),
        x::json::json(
            {{"requests", 2},
             {"retries", 1},
             {"timeouts", 2},
             {"exceptions", 0},
             {"last_response", 0},
             {"max_response", 0}}
        )
    );
}

/// @brief it should keep the bits that another device sets in the same register while
/// a read-then-write runs.
TEST(MaskWrite, ReadThenWriteKeepsConcurrentBits) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1679;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    const ConnectionConfig conn{"127.0.0.1", 1679};
    auto low = ASSERT_NIL_P(manager.acquire(conn));
    auto high = ASSERT_NIL_P(manager.acquire(conn));
    const auto set_bits = [](const std::shared_ptr<Device> &dev, const int first) {
        for (int bit = first; bit < first + 8; bit++) {
            const auto mask = static_cast<std::uint16_t>(1U << bit);
            ASSERT_NIL(
                dev->mask_write_register(0, static_cast<std::uint16_t>(~mask), mask)
            );
        }
    };

    for (int round = 0; round < 10; round++) {
        ASSERT_NIL(low->mask_write_register(0, 0, 0));
        std::thread writing([&] { set_bits(high, 8); });
        set_bits(low, 0);
        writing.join();
        ASSERT_EQ(slave.get_holding_register(0), 0xFFFF) << "round " << round;
    }
    slave.stop();
}

struct MaskWriteCase {
    std::string name;
    bool mask_write_supported;
    /// @brief the requests the write sends, by function code.
    std::map<int, std::size_t> requests;
};

class MaskWriteTest : public ::testing::TestWithParam<MaskWriteCase> {};

/// @brief it should set the cleared bits of a register and keep the rest, with one
/// function code 22 request when the device has mask writes, and a read and a write
/// otherwise.
TEST_P(MaskWriteTest, SetsBitsAndKeepsTheRest) {
    const auto &[_, mask_write_supported, expected] = GetParam();
    testutil::SlaveConfig slave_config;
    slave_config.port = 1676;
    slave_config.holding_registers[4] = static_cast<uint16_t>(0xFF0F);
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    RequestConfig requests;
    requests.mask_write_supported = mask_write_supported;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1676}, requests));
    ASSERT_NIL(dev->mask_write_register(4, 0xFFF0, 0x0005));
    EXPECT_EQ(slave.get_holding_register(4), 0xFF05);
    for (const auto &[fc, count]: expected)
        EXPECT_EQ(slave.requests(fc), count) << "function code " << fc;
    slave.stop();
}

INSTANTIATE_TEST_SUITE_P(
    Writes,
    MaskWriteTest,
    ::testing::Values(
        MaskWriteCase{
            "MaskWrite",
            true,
            {{MODBUS_FC_MASK_WRITE_REGISTER, 1},
             {MODBUS_FC_READ_HOLDING_REGISTERS, 0},
             {MODBUS_FC_WRITE_MULTIPLE_REGISTERS, 0}},
        },
        MaskWriteCase{
            "ReadThenWrite",
            false,
            {{MODBUS_FC_MASK_WRITE_REGISTER, 0},
             {MODBUS_FC_READ_HOLDING_REGISTERS, 1},
             {MODBUS_FC_WRITE_MULTIPLE_REGISTERS, 1}},
        }
    ),
    [](const ::testing::TestParamInfo<MaskWriteCase> &info) { return info.param.name; }
);

/// @brief it should give a waiting read the turn after one write, so steady writes
/// cannot hold reads back.
TEST(DeviceTest, ReadWaitsBehindOneWrite) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1657;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    ConnectionConfig cfg{"127.0.0.1", 1657};
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));

    slave.stall_next(MODBUS_TCP_SLAVE, std::chrono::milliseconds(300));
    std::thread slow([&] {
        uint16_t r = 0;
        ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &r));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    std::thread read([&] {
        uint8_t bit = 0;
        ASSERT_NIL(dev->read_bits(Coil, 0, 1, &bit));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    std::vector<std::thread> writes;
    writes.reserve(2);
    for (int i = 0; i < 2; i++)
        writes.emplace_back([&] {
            const uint16_t value = 5;
            ASSERT_NIL(dev->write_registers(0, 1, &value));
        });
    slow.join();
    read.join();
    for (auto &w: writes)
        w.join();

    const std::vector<int> expected{
        MODBUS_FC_READ_HOLDING_REGISTERS,
        MODBUS_FC_READ_HOLDING_REGISTERS,
        MODBUS_FC_WRITE_MULTIPLE_REGISTERS,
        MODBUS_FC_READ_COILS,
        MODBUS_FC_WRITE_MULTIPLE_REGISTERS,
    };
    EXPECT_EQ(slave.log(), expected);
    slave.stop();
}

/// @brief it should stop a probe that waits too long for its turn behind a slow
/// request.
TEST(DeviceTest, ProbeStopsWaitingForBusyConnection) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1658;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto busy = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1658}));
    ConnectionConfig probed_cfg{"127.0.0.1", 1658};
    probed_cfg.unit_id = 2;
    auto probed = ASSERT_NIL_P(manager.acquire(probed_cfg));
    uint16_t reg = 0;
    ASSERT_NIL(busy->read_registers(HoldingRegister, 0, 1, &reg));

    slave.stall_next(MODBUS_TCP_SLAVE, std::chrono::milliseconds(500));
    std::thread slow([&] {
        uint16_t r = 0;
        ASSERT_NIL(busy->read_registers(HoldingRegister, 0, 1, &r));
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    const auto start = std::chrono::steady_clock::now();
    const auto err = probed->probe(x::telem::MILLISECOND * 100);
    const auto waited = std::chrono::steady_clock::now() - start;
    slow.join();
    ASSERT_OCCURRED_AS(err, UNREACHABLE);
    EXPECT_EQ(err.data, "other requests kept the connection busy for 100ms");
    EXPECT_LT(waited, std::chrono::milliseconds(400));
    ASSERT_NIL(probed->probe());
    slave.stop();
}

/// @brief it should write each coil and register with its own request when the device
/// writes with single function codes.
TEST(DeviceTest, WritesWithSingleCodes) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1651;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    RequestConfig requests;
    requests.write_codes = WriteCodes::Single;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1651}, requests));

    const std::vector<uint16_t> registers{0x1234, 0x5678};
    ASSERT_NIL(dev->write_registers(3, registers.size(), registers.data()));
    const std::vector<uint8_t> bits{1, 0, 1};
    ASSERT_NIL(dev->write_bits(4, bits.size(), bits.data()));

    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_SINGLE_REGISTER), 2);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_MULTIPLE_REGISTERS), 0);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_SINGLE_COIL), 3);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_MULTIPLE_COILS), 0);
    EXPECT_EQ(slave.get_holding_register(3), 0x1234);
    EXPECT_EQ(slave.get_holding_register(4), 0x5678);
    EXPECT_EQ(slave.get_coil(4), 1);
    EXPECT_EQ(slave.get_coil(5), 0);
    EXPECT_EQ(slave.get_coil(6), 1);
    slave.stop();
}

/// @brief it should wait the device's delay between requests on the connection.
TEST(DeviceTest, WaitsDelayBetweenRequests) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1652;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    RequestConfig requests;
    requests.delay = x::telem::MILLISECOND * 100;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1652}, requests));
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    EXPECT_GE(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(190));
    slave.stop();
}

/// @brief it should wait the device's delay before each request of a single-code write.
TEST(DeviceTest, WaitsDelayBetweenSingleCodeWrites) {
    testutil::SlaveConfig slave_config;
    slave_config.port = 1656;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    RequestConfig requests;
    requests.write_codes = WriteCodes::Single;
    requests.delay = x::telem::MILLISECOND * 100;
    auto dev = ASSERT_NIL_P(manager.acquire({"127.0.0.1", 1656}, requests));
    uint16_t reg = 0;
    ASSERT_NIL(dev->read_registers(HoldingRegister, 0, 1, &reg));
    const auto start = std::chrono::steady_clock::now();
    const std::vector<uint8_t> bits{1, 0};
    ASSERT_NIL(dev->write_bits(0, bits.size(), bits.data()));
    const std::vector<uint16_t> registers{1, 2};
    ASSERT_NIL(dev->write_registers(0, registers.size(), registers.data()));
    EXPECT_GE(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(390));
    slave.stop();
}

struct ProbeCase {
    std::string name;
    testutil::SlaveConfig slave;
    /// @brief the error of the probe, or NIL when the unit answers.
    x::errors::Error expected;
    /// @brief the error number whose message the expected error carries.
    int errnum = 0;
};

class ProbeTest : public ::testing::TestWithParam<ProbeCase> {};

/// @brief it should treat any reply from the unit as alive, an exception included, and
/// a timeout or a gateway exception as unreachable.
TEST_P(ProbeTest, Probes) {
    const auto &param = GetParam();
    auto slave_config = param.slave;
    slave_config.port = 1653;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    Manager manager;
    auto cfg = unit(1653, 3);
    cfg.retries = 0;
    auto dev = ASSERT_NIL_P(manager.acquire(cfg));
    const auto err = dev->probe();
    if (!param.expected)
        ASSERT_NIL(err);
    else {
        ASSERT_OCCURRED_AS(err, param.expected);
        EXPECT_EQ(err.data, modbus_strerror(param.errnum));
    }
    slave.stop();
}

INSTANTIATE_TEST_SUITE_P(
    Outcomes,
    ProbeTest,
    ::testing::Values(
        ProbeCase{"Answers", {}, x::errors::NIL},
        ProbeCase{
            "Exception",
            {.reject_function_codes = {MODBUS_FC_READ_HOLDING_REGISTERS}},
            x::errors::NIL
        },
        ProbeCase{"Silent", {.silent_units = {3}}, UNREACHABLE, ETIMEDOUT},
        ProbeCase{"Gateway", {.gateway_failed_units = {3}}, TEMPORARY_ERROR, EMBXGTAR}
    ),
    [](const auto &info) { return info.param.name; }
);
}
