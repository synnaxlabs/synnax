// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Hardware integration tests for EtherCAT driver with IOLITE R8 hardware.
///
/// These tests require:
/// - IOLITE R8 hardware connected via EtherCAT
/// - Root privileges for raw socket access
/// - ETHERCAT_INTERFACE environment variable set (e.g., "en7")
///
/// Run with: sudo bazel test //driver/ethercat:ethercat_hardware_test \
///           --test_env=ETHERCAT_INTERFACE=en7 --test_output=all

#include <chrono>
#include <cstdlib>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/master.h"
#include "engine/engine.h"

namespace ethercat {

/// Expected number of slaves on IOLITE R8.
constexpr int EXPECTED_SLAVE_COUNT = 7;

/// Expected slaves in OP state (excludes 6xSTG modules).
constexpr int EXPECTED_OP_SLAVES = 5;

/// Gets the interface name from environment variable.
std::string get_interface_name() {
    const char *iface = std::getenv("ETHERCAT_INTERFACE");
    return iface ? std::string(iface) : "en7";
}

class HardwareTest : public ::testing::Test {
protected:
    std::string interface_name;
    std::unique_ptr<soem::Master> master;

    void SetUp() override {
        interface_name = get_interface_name();
        master = std::make_unique<soem::Master>(interface_name);
    }

    void TearDown() override {
        if (master) master->deactivate();
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
    }
};

TEST_F(HardwareTest, ScanDiscoversSlavesIOLITE) {
    ASSERT_NIL(master->initialize());

    auto slaves = master->slaves();
    ASSERT_EQ(slaves.size(), EXPECTED_SLAVE_COUNT)
        << "Expected " << EXPECTED_SLAVE_COUNT << " slaves, found " << slaves.size();

    for (const auto &slave: slaves) {
        EXPECT_NE(slave.properties.vendor_id, 0u)
            << "Slave " << slave.properties.position << " has no vendor ID";
        EXPECT_FALSE(slave.properties.name.empty())
            << "Slave " << slave.properties.position << " has no name";
    }
}

TEST_F(HardwareTest, ActivatePartialIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    auto slaves = master->slaves();
    int op_count = 0;

    for (const auto &slave: slaves) {
        auto state = master->slave_state(slave.properties.position);
        if (state == slave::State::OP) op_count++;
    }

    EXPECT_GE(op_count, EXPECTED_OP_SLAVES)
        << "Expected at least " << EXPECTED_OP_SLAVES << " slaves in OP, got "
        << op_count;
}

TEST_F(HardwareTest, CyclicExchangeIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    constexpr int NUM_CYCLES = 100;
    constexpr auto CYCLE_PERIOD = std::chrono::milliseconds(10);
    int successful_cycles = 0;
    int error_count = 0;

    auto *domain = master->active_domain();
    ASSERT_NE(domain, nullptr);

    for (int i = 0; i < NUM_CYCLES; i++) {
        if (master->send()) {
            error_count++;
            continue;
        }

        std::this_thread::sleep_for(CYCLE_PERIOD);

        if (master->receive()) {
            error_count++;
            continue;
        }

        if (!master->process(*domain)) successful_cycles++;
        if (master->queue(*domain)) error_count++;
    }

    EXPECT_GE(successful_cycles, static_cast<int>(NUM_CYCLES * 0.95))
        << "Expected at least 95% successful cycles, got "
        << (successful_cycles * 100 / NUM_CYCLES) << "%";
    EXPECT_LE(error_count, static_cast<int>(NUM_CYCLES * 0.05))
        << "Too many errors: " << error_count << "/" << NUM_CYCLES;
}

TEST_F(HardwareTest, ReadInputDataIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    auto *domain = master->active_domain();
    ASSERT_NE(domain, nullptr);
    ASSERT_GT(domain->input_size(), 0u) << "No input data available";

    ASSERT_NIL(master->send());
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    ASSERT_NIL(master->receive());
    ASSERT_NIL(master->process(*domain));

    uint8_t *data = domain->data();
    ASSERT_NE(data, nullptr);
}

TEST_F(HardwareTest, WorkingCounterValidationIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    auto *domain = master->active_domain();
    ASSERT_NE(domain, nullptr);

    constexpr int NUM_CYCLES = 10;
    int wkc_mismatch_count = 0;

    for (int i = 0; i < NUM_CYCLES; i++) {
        ASSERT_NIL(master->send());
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
        ASSERT_NIL(master->receive());

        if (master->process(*domain).matches(WORKING_COUNTER_ERROR))
            wkc_mismatch_count++;

        (void) master->queue(*domain);
    }

    EXPECT_EQ(wkc_mismatch_count, 0)
        << "WKC mismatch occurred " << wkc_mismatch_count << " times";
}

TEST_F(HardwareTest, GracefulShutdownIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    ASSERT_NIL(master->send());
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    ASSERT_NIL(master->receive());

    master->deactivate();

    auto slaves = master->slaves();
    for (const auto &slave: slaves) {
        auto state = master->slave_state(slave.properties.position);
        EXPECT_TRUE(state == slave::State::INIT || state == slave::State::PRE_OP)
            << "Slave " << slave.properties.position
            << " in unexpected state after deactivate";
    }
}

TEST_F(HardwareTest, SlaveDataOffsetsIOLITE) {
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    auto slaves = master->slaves();
    for (const auto &slave: slaves) {
        auto offsets = master->slave_data_offsets(slave.properties.position);
        EXPECT_GE(offsets.input_size + offsets.output_size, 0u)
            << "Slave " << slave.properties.position << " has no data";
    }
}

class CyclicEngineHardwareTest : public ::testing::Test {
protected:
    std::string interface_name;
    std::shared_ptr<soem::Master> master;
    std::unique_ptr<Loop> engine;
    std::function<void()> cleanup;

    void SetUp() override {
        interface_name = get_interface_name();
        master = std::make_shared<soem::Master>(interface_name);
        std::shared_ptr<Master> master_base = master;
        engine = std::make_unique<Loop>(
            master_base,
            LoopConfig(telem::MILLISECOND * 10)
        );
        cleanup = nullptr;
    }

    void TearDown() override {
        if (cleanup) cleanup();
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
    }
};

TEST_F(CyclicEngineHardwareTest, StartAndStopCyclicExchange) {
    TaskRegistration reg;
    auto registered = ASSERT_NIL_P(engine->register_task(reg));
    cleanup = registered.unregister;

    EXPECT_TRUE(engine->is_running());

    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    EXPECT_GT(engine->cycle_count(), 0u);

    registered.unregister();
    cleanup = nullptr;
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineHardwareTest, MultipleTasksRefCounting) {
    TaskRegistration reg;
    auto reg1 = ASSERT_NIL_P(engine->register_task(reg));
    auto reg2 = ASSERT_NIL_P(engine->register_task(reg));
    EXPECT_EQ(engine->task_count(), 2);
    EXPECT_TRUE(engine->is_running());

    reg1.unregister();
    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(engine->task_count(), 1);

    reg2.unregister();
    EXPECT_FALSE(engine->is_running());
    EXPECT_EQ(engine->task_count(), 0);
}

TEST_F(CyclicEngineHardwareTest, WaitForInputsWithHardware) {
    TaskRegistration reg;
    reg.inputs.push_back(pdo::Entry(1, 0x6000, 1, 8, true));
    auto registered = ASSERT_NIL_P(engine->register_task(reg));
    cleanup = registered.unregister;

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;

    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));
    EXPECT_FALSE(buffer.empty());

    registered.unregister();
    cleanup = nullptr;
}

TEST_F(CyclicEngineHardwareTest, SustainedCyclicExchange) {
    TaskRegistration reg;
    auto registered = ASSERT_NIL_P(engine->register_task(reg));
    cleanup = registered.unregister;

    constexpr int TEST_DURATION_MS = 1000;
    std::this_thread::sleep_for(std::chrono::milliseconds(TEST_DURATION_MS));

    uint64_t cycles = engine->cycle_count();
    constexpr uint64_t expected_cycles = static_cast<uint64_t>(
        TEST_DURATION_MS * 0.9 / 10
    );
    EXPECT_GE(cycles, expected_cycles)
        << "Expected ~" << TEST_DURATION_MS / 10 << " cycles, got " << cycles;

    registered.unregister();
    cleanup = nullptr;
}

TEST_F(CyclicEngineHardwareTest, DynamicPDORegistrationWhileRunning) {
    TaskRegistration reg1;
    reg1.inputs.push_back(pdo::Entry(1, 0x6000, 1, 8, true));
    auto registered1 = ASSERT_NIL_P(engine->register_task(reg1));
    EXPECT_TRUE(engine->is_running());

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;

    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));

    TaskRegistration reg2;
    reg2.inputs.push_back(pdo::Entry(1, 0x6000, 2, 8, true));
    auto registered2 = ASSERT_NIL_P(engine->register_task(reg2));

    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(engine->task_count(), 2);

    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));

    registered1.unregister();
    registered2.unregister();
}

TEST_F(CyclicEngineHardwareTest, MultipleRestartsWithHardware) {
    TaskRegistration reg1;
    reg1.inputs.push_back(pdo::Entry(1, 0x6000, 1, 8, true));
    auto registered1 = ASSERT_NIL_P(engine->register_task(reg1));

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));

    TaskRegistration reg2;
    reg2.inputs.push_back(pdo::Entry(1, 0x6000, 2, 8, true));
    auto registered2 = ASSERT_NIL_P(engine->register_task(reg2));
    EXPECT_TRUE(engine->is_running());

    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));

    TaskRegistration reg3;
    reg3.inputs.push_back(pdo::Entry(1, 0x6000, 3, 8, true));
    auto registered3 = ASSERT_NIL_P(engine->register_task(reg3));
    EXPECT_TRUE(engine->is_running());

    ASSERT_NIL(engine->wait_for_inputs(buffer, stopped));

    registered1.unregister();
    registered2.unregister();
    registered3.unregister();
}

}
