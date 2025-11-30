// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/mock/slave.h"
#include "x/cpp/xtest/xtest.h"

/// @brief Tests that the Manager correctly reuses cached connections.
/// This test verifies that acquiring the same connection twice returns
/// the same shared_ptr (connection reuse), not two different connections.
TEST(ManagerTest, ConnectionCacheReusesConnections) {
    // Start a mock slave to connect to
    modbus::mock::SlaveConfig slave_config;
    slave_config.host = "127.0.0.1";
    slave_config.port = 1520;  // Use unique port to avoid conflicts
    modbus::mock::Slave slave(slave_config);
    ASSERT_NIL(slave.start());

    // Give the slave time to start
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    modbus::device::Manager manager;
    modbus::device::ConnectionConfig config{"127.0.0.1", 1520};

    // First acquire
    auto [dev1, err1] = manager.acquire(config);
    ASSERT_NIL(err1);
    ASSERT_NE(dev1, nullptr);

    // Second acquire to the same host:port should return the SAME connection
    auto [dev2, err2] = manager.acquire(config);
    ASSERT_NIL(err2);
    ASSERT_NE(dev2, nullptr);

    // This assertion will FAIL with the bug because the Manager stores
    // connections under `config.host` but looks them up under `host:port`
    EXPECT_EQ(dev1.get(), dev2.get())
        << "Expected connection reuse - both acquires should return the same device";

    slave.stop();
}

/// @brief Tests that different ports on the same host get different connections.
TEST(ManagerTest, DifferentPortsGetDifferentConnections) {
    // Start two mock slaves on different ports
    modbus::mock::SlaveConfig slave_config1;
    slave_config1.host = "127.0.0.1";
    slave_config1.port = 1530;  // Use unique port to avoid conflicts
    modbus::mock::Slave slave1(slave_config1);
    ASSERT_NIL(slave1.start());

    modbus::mock::SlaveConfig slave_config2;
    slave_config2.host = "127.0.0.1";
    slave_config2.port = 1531;  // Use unique port to avoid conflicts
    modbus::mock::Slave slave2(slave_config2);
    ASSERT_NIL(slave2.start());

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    modbus::device::Manager manager;

    // Acquire connection to port 1530
    modbus::device::ConnectionConfig config1{"127.0.0.1", 1530};
    auto [dev1, err1] = manager.acquire(config1);
    ASSERT_NIL(err1);
    ASSERT_NE(dev1, nullptr);

    // Acquire connection to port 1531 - should be different connection
    modbus::device::ConnectionConfig config2{"127.0.0.1", 1531};
    auto [dev2, err2] = manager.acquire(config2);
    ASSERT_NIL(err2);
    ASSERT_NE(dev2, nullptr);

    // These should be different connections since they're different ports
    EXPECT_NE(dev1.get(), dev2.get())
        << "Expected different connections for different ports";

    slave1.stop();
    slave2.stop();
}

/// @brief Tests that the Manager is thread-safe when multiple threads
/// acquire connections concurrently.
/// Without mutex protection, this test may crash or produce inconsistent results.
TEST(ManagerTest, ConcurrentAcquireIsThreadSafe) {
    // Start mock slaves on multiple ports
    std::vector<std::unique_ptr<modbus::mock::Slave>> slaves;
    for (int i = 0; i < 4; i++) {
        modbus::mock::SlaveConfig config;
        config.host = "127.0.0.1";
        config.port = static_cast<int>(1510 + i);
        auto slave = std::make_unique<modbus::mock::Slave>(config);
        ASSERT_NIL(slave->start());
        slaves.push_back(std::move(slave));
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    modbus::device::Manager manager;
    std::atomic<int> success_count{0};
    std::atomic<int> error_count{0};

    // Launch multiple threads that all try to acquire connections concurrently
    std::vector<std::thread> threads;
    for (int i = 0; i < 20; i++) {
        threads.emplace_back([&manager, &success_count, &error_count, i]() {
            // Each thread acquires a connection to one of the 4 ports
            modbus::device::ConnectionConfig config{
                "127.0.0.1",
                static_cast<uint16_t>(1510 + (i % 4))
            };
            auto [dev, err] = manager.acquire(config);
            if (err) {
                error_count++;
            } else if (dev != nullptr) {
                success_count++;
            }
        });
    }

    // Wait for all threads to complete
    for (auto &t : threads) {
        t.join();
    }

    // All acquisitions should succeed without crashes or data corruption
    EXPECT_EQ(error_count.load(), 0) << "Some acquisitions failed";
    EXPECT_EQ(success_count.load(), 20) << "Not all acquisitions succeeded";

    for (auto &slave : slaves) {
        slave->stop();
    }
}