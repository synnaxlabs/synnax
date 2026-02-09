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

#include "x/cpp/test/test.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/mock/slave.h"

namespace driver::modbus::device {
TEST(ManagerTest, AcquireAlwaysCreatesNewConnection) {
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1520;
    mock::Slave slave(slave_config);
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
    std::vector<std::unique_ptr<mock::Slave>> slaves;
    for (int i = 0; i < 4; i++) {
        mock::SlaveConfig config;
        config.host = "127.0.0.1";
        config.port = static_cast<int>(1510 + i);
        auto slave = std::make_unique<mock::Slave>(config);
        ASSERT_NIL(slave->start());
        slaves.push_back(std::move(slave));
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Manager manager;
    std::atomic<int> success_count{0};
    std::atomic<int> error_count{0};

    std::vector<std::thread> threads;
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1543;
    slave_config.coils[0] = 1;
    slave_config.coils[1] = 0;
    slave_config.coils[2] = 1;
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1549;
    slave_config.discrete_inputs[0] = 1;
    slave_config.discrete_inputs[1] = 1;
    slave_config.discrete_inputs[2] = 0;
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1544;
    slave_config.holding_registers[0] = static_cast<uint16_t>(0x1234);
    slave_config.holding_registers[1] = static_cast<uint16_t>(0x5678);
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1550;
    slave_config.input_registers[0] = static_cast<uint16_t>(0xAAAA);
    slave_config.input_registers[1] = static_cast<uint16_t>(0xBBBB);
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1545;
    slave_config.coils[0] = 0;
    slave_config.coils[1] = 0;
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1546;
    slave_config.holding_registers[0] = static_cast<uint16_t>(0);
    slave_config.holding_registers[1] = static_cast<uint16_t>(0);
    mock::Slave slave(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1547;
    slave_config.coils[0] = 1;
    auto slave = std::make_unique<mock::Slave>(slave_config);
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
    mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1548;
    slave_config.coils[0] = 1;

    Manager manager;
    ConnectionConfig config{"127.0.0.1", 1548};

    {
        mock::Slave slave(slave_config);
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
        mock::Slave slave(slave_config);
        ASSERT_NIL(slave.start());
        std::this_thread::sleep_for(std::chrono::milliseconds(100));

        auto dev = ASSERT_NIL_P(manager.acquire(config));

        uint8_t bits[1];
        ASSERT_NIL(dev->read_bits(Coil, 0, 1, bits));
        EXPECT_EQ(bits[0], 0);

        slave.stop();
    }
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
