// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/testutil/slave.h"

namespace driver::modbus::device {
TEST(ManagerTest, AcquireAlwaysCreatesNewConnection) {
    testutil::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1520;
    testutil::Slave slave(slave_config);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1520};

    auto dev1 = ASSERT_NIL_P(manager.acquire(config));
    auto dev2 = ASSERT_NIL_P(manager.acquire(config));

    EXPECT_NE(dev1.get(), dev2.get());

    slave.stop();
}

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
    std::atomic<int> success_count{0};
    std::atomic<int> error_count{0};

    std::vector<std::thread> threads;
    threads.reserve(20);
    for (int i = 0; i < 20; i++) {
        threads.emplace_back([&manager, &success_count, &error_count, i]() {
            ConnectionConfig config{"127.0.0.1", static_cast<uint16_t>(1510 + (i % 4))};
            auto [dev, err] = manager.acquire(config);
            if (err) {
                error_count++;
            } else if (dev != nullptr) {
                success_count++;
            }
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(error_count.load(), 0);
    EXPECT_EQ(success_count.load(), 20);

    for (auto &slave: slaves) {
        slave->stop();
    }
}

TEST(ManagerTest, AcquireFailsWhenServerNotRunning) {
    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1599};

    auto [dev, err] = manager.acquire(config);

    EXPECT_TRUE(err);
    EXPECT_EQ(dev, nullptr);
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

    slave->stop();
    slave.reset();

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    uint8_t bits[1];
    auto read_err = dev->read_bits(Coil, 0, 1, bits);
    EXPECT_TRUE(read_err);
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

/// @brief it should reconnect on the next request after the server drops the
/// connection, without a new acquire.
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

/// @brief it should return an unreachable error when no server listens.
TEST(ManagerTest, AcquireUnreachable) {
    Manager manager;
    ASSERT_OCCURRED_AS_P(manager.acquire({"127.0.0.1", 1625}), UNREACHABLE);
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
}
}
