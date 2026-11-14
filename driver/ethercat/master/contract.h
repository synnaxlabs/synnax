// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <cstdint>
#include <map>
#include <set>
#include <thread>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::master::contract {
/// @brief returns the entry that each contract slave maps: 0x6000:01, a 16-bit input.
inline pdo::Entry input(const uint16_t position) {
    return {
        .slave_position = position,
        .index = 0x6000,
        .sub_index = 0x01,
        .bit_length = 16,
        .is_input = true,
    };
}

/// @brief the behavior that every master::Master shares. Bus wraps one master and
/// provides:
/// - `master::Master &master()`.
/// - `void add_slave()`: adds a slave at the next position that maps input().
/// - `void refuse(uint16_t position)`: makes the slave at position refuse SAFE-OP.
template<typename Bus>
class Contract : public ::testing::Test {
protected:
    Bus bus;
    x::breaker::Breaker brk;
    /// @brief the process image of the latest activate().
    master::Image image;
    /// @brief how many times activate() started the cycle.
    int cycles = 0;
    std::atomic<bool> cycling = false;
    std::thread cycle;

    void SetUp() override { this->brk.start(); }

    void TearDown() override {
        this->stop_cycle();
        this->brk.stop();
    }

    /// @brief activates the master with slaves, cycling the bus on its own thread from
    /// the start of the cycle until stop_cycle().
    Activation activate(const std::map<uint16_t, slave::Config> &slaves = {}) {
        return this->bus.master().activate(
            {.period = x::telem::MILLISECOND, .slaves = slaves},
            [this](const Image &image) {
                this->image = image;
                this->cycles++;
                this->cycling = true;
                this->cycle = std::thread([this] {
                    while (this->cycling) {
                        static_cast<void>(this->bus.master().send());
                        std::this_thread::sleep_for(std::chrono::microseconds(100));
                        static_cast<void>(this->bus.master().receive());
                    }
                });
            }
        );
    }

    void stop_cycle() {
        this->cycling = false;
        if (this->cycle.joinable()) this->cycle.join();
    }
};

TYPED_TEST_SUITE_P(Contract);

/// @brief it should fail to count slaves before a scan.
TYPED_TEST_P(Contract, CountSlavesFailsBeforeScan) {
    this->bus.add_slave();
    ASSERT_OCCURRED_AS_P(this->bus.master().count_slaves(), errors::MASTER_INIT_ERROR);
}

/// @brief it should count the slaves on the bus after a scan.
TYPED_TEST_P(Contract, CountSlavesAfterScan) {
    this->bus.add_slave();
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    EXPECT_EQ(ASSERT_NIL_P(this->bus.master().count_slaves()), 2);
}

/// @brief it should return the slaves of the first scan until deactivate().
TYPED_TEST_P(Contract, ScanReturnsTheSlavesItFoundUntilDeactivate) {
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    this->bus.add_slave();
    EXPECT_EQ(ASSERT_NIL_P(this->bus.master().scan()).size(), 1);
}

/// @brief it should fail activation before a scan.
TYPED_TEST_P(Contract, ActivateFailsBeforeScan) {
    this->bus.add_slave();
    ASSERT_OCCURRED_AS(this->activate().err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(this->cycles, 0);
}

/// @brief it should fail receive() and send() between a scan and activation.
TYPED_TEST_P(Contract, ReceiveAndSendFailBeforeActivation) {
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_OCCURRED_AS(this->bus.master().receive(), errors::CYCLIC_ERROR);
    ASSERT_OCCURRED_AS(this->bus.master().send(), errors::CYCLIC_ERROR);
}

/// @brief it should start the cycle once, map each enabled slave, and report each
/// enabled slave in OP.
TYPED_TEST_P(Contract, ActivateRunsEachEnabledSlave) {
    this->bus.add_slave();
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate({{2, {.enabled = false}}}).err);
    EXPECT_EQ(this->cycles, 1);
    ASSERT_NIL_P(this->image.offset(input(1)));
    ASSERT_OCCURRED_AS_P(this->image.offset(input(2)), errors::PDO_MAPPING_ERROR);
    const auto health = this->bus.master().check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].position, 1);
    EXPECT_EQ(health[0].state, slave::State::OP);
}

/// @brief it should give a working counter share to each enabled slave that maps
/// input(), and none to a disabled slave.
TYPED_TEST_P(Contract, ActivateGivesEachEnabledSlaveAWorkingCounterShare) {
    this->bus.add_slave();
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate({{2, {.enabled = false}}}).err);
    EXPECT_EQ(
        this->image.working_counter_shares,
        (std::map<uint16_t, master::WorkingCounterShare>{{1, {.inputs = 1}}})
    );
}

/// @brief it should map every entry inside the process image.
TYPED_TEST_P(Contract, ActivateMapsEachEntryInsideTheImage) {
    this->bus.add_slave();
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate().err);
    ASSERT_FALSE(this->image.offsets.empty());
    for (const auto &[key, offset]: this->image.offsets) {
        const auto size = key.is_input ? this->image.inputs.size()
                                       : this->image.outputs.size();
        EXPECT_LE(
            offset.byte + telem::pdo_required_bytes(offset.bit, offset.bit_length),
            size
        );
    }
}

/// @brief it should fail a second activation.
TYPED_TEST_P(Contract, ActivateFailsTwice) {
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate().err);
    const auto err = this->bus.master()
                         .activate(
                             {.period = x::telem::MILLISECOND},
                             [](const Image &) {}
                         )
                         .err;
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(this->cycles, 1);
}

/// @brief it should return the refusal of a slave, keyed by its position, after it
/// starts the cycle.
TYPED_TEST_P(Contract, ActivateReturnsTheRefusalOfASlave) {
    this->bus.add_slave();
    this->bus.add_slave();
    this->bus.refuse(2);
    ASSERT_NIL(this->bus.master().scan().second);
    const auto activation = this->activate();
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(this->cycles, 1);
    std::set<uint16_t> refused;
    for (const auto &[position, refusal]: activation.refusals)
        refused.insert(position);
    EXPECT_EQ(refused, std::set<uint16_t>{2});
}

/// @brief it should fail receive(), send(), and activate() after deactivate() until the
/// next scan.
TYPED_TEST_P(Contract, CycleAndActivateFailAfterDeactivate) {
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate().err);
    this->stop_cycle();
    this->bus.master().deactivate();
    ASSERT_OCCURRED_AS(this->bus.master().receive(), errors::CYCLIC_ERROR);
    ASSERT_OCCURRED_AS(this->bus.master().send(), errors::CYCLIC_ERROR);
    ASSERT_OCCURRED_AS(this->activate().err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(this->cycles, 1);
}

/// @brief it should discover and start the bus again after deactivate().
TYPED_TEST_P(Contract, ScanAndActivateAgainAfterDeactivate) {
    this->bus.add_slave();
    ASSERT_NIL(this->bus.master().scan().second);
    ASSERT_NIL(this->activate().err);
    this->stop_cycle();
    this->bus.master().deactivate();
    EXPECT_EQ(ASSERT_NIL_P(this->bus.master().scan()).size(), 1);
    ASSERT_NIL(this->activate().err);
    EXPECT_EQ(this->cycles, 2);
}

REGISTER_TYPED_TEST_SUITE_P(
    Contract,
    CountSlavesFailsBeforeScan,
    CountSlavesAfterScan,
    ScanReturnsTheSlavesItFoundUntilDeactivate,
    ActivateFailsBeforeScan,
    ReceiveAndSendFailBeforeActivation,
    ActivateRunsEachEnabledSlave,
    ActivateGivesEachEnabledSlaveAWorkingCounterShare,
    ActivateMapsEachEntryInsideTheImage,
    ActivateFailsTwice,
    ActivateReturnsTheRefusalOfASlave,
    CycleAndActivateFailAfterDeactivate,
    ScanAndActivateAgainAfterDeactivate
);
}
