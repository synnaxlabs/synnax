// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/dc/dc.h"

namespace driver::ethercat::dc {

/// @brief it should give back the same mode after a JSON round trip, negative factors
/// and shifts included.
TEST(OpMode, RoundTripsThroughJSON) {
    const OpMode mode{
        .name = "DC-Synchron",
        .assign_activate = 0x0700,
        .sync0_cycle = 1000,
        .sync0_factor = -2,
        .sync0_shift = -100000,
        .sync1_factor = -1000,
        .sync1_shift = 25000,
    };
    auto parser = x::json::Parser(mode.to_json());
    EXPECT_EQ(OpMode::parse(parser), mode);
    EXPECT_TRUE(parser.ok());
}

/// @brief it should report a field that the JSON leaves out.
TEST(OpMode, ParseReportsAMissingField) {
    auto j = OpMode{.name = "DC"}.to_json();
    j.erase("sync1_shift");
    auto parser = x::json::Parser(j);
    OpMode::parse(parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "sync1_shift");
}

/// @brief it should leave a slave that lists an SM mode, or no mode, without DC.
TEST(Select, ReturnsNoModeForASlaveThatListsSM) {
    EXPECT_EQ(select({}), std::nullopt);
    EXPECT_EQ(
        select({OpMode{.assign_activate = 0x0700}, OpMode{.assign_activate = 0}}),
        std::nullopt
    );
}

/// @brief it should choose the first DC mode of a slave that lists no SM mode.
TEST(Select, ReturnsTheFirstDCMode) {
    const OpMode first{.name = "DC", .assign_activate = 0x0300};
    EXPECT_EQ(select({first, OpMode{.assign_activate = 0x0700}}), first);
}

/// @brief it should use a fixed SYNC0 cycle at any bus period.
TEST(Resolve, UsesAFixedSYNC0Cycle) {
    const OpMode mode{.assign_activate = 0x0300, .sync0_cycle = 1'000'000'000};
    const auto regs = ASSERT_NIL_P(resolve(mode, x::telem::MILLISECOND * 2));
    EXPECT_EQ(regs.sync0_cycle, 1'000'000'000U);
    EXPECT_EQ(regs.sync1_cycle, 0U);
}

/// @brief it should set the SYNC0 cycle to a multiple or a fraction of the bus cycle
/// from the SYNC0 factor.
TEST(Resolve, ScalesTheBusCycleBySYNC0Factor) {
    const auto period = x::telem::MILLISECOND;
    const OpMode multiple{.assign_activate = 0x0300, .sync0_factor = 2};
    EXPECT_EQ(ASSERT_NIL_P(resolve(multiple, period)).sync0_cycle, 2'000'000U);
    const OpMode fraction{.assign_activate = 0x0300, .sync0_factor = -4};
    EXPECT_EQ(ASSERT_NIL_P(resolve(fraction, period)).sync0_cycle, 250'000U);
}

/// @brief it should set the SYNC1 register to the SYNC1 cycle less the SYNC0 cycle,
/// plus the SYNC1 shift.
TEST(Resolve, SetsSYNC1FromItsFactor) {
    const auto period = x::telem::MILLISECOND;
    const OpMode of_sync0{
        .assign_activate = 0x0700,
        .sync0_cycle = 500'000,
        .sync1_factor = 20,
        .sync1_shift = 1'000,
    };
    EXPECT_EQ(ASSERT_NIL_P(resolve(of_sync0, period)).sync1_cycle, 9'501'000U);
    const OpMode of_bus{
        .assign_activate = 0x0700,
        .sync0_cycle = 1'000,
        .sync1_factor = -1000,
    };
    EXPECT_EQ(ASSERT_NIL_P(resolve(of_bus, period)).sync1_cycle, 999'999'000U);
    const OpMode each_sync0{.assign_activate = 0x0700, .sync0_factor = 1};
    EXPECT_EQ(ASSERT_NIL_P(resolve(each_sync0, period)).sync1_cycle, 0U);
}

/// @brief it should write only the activation and shift of a mode without SYNC0, and
/// not check its cycles.
TEST(Resolve, SkipsTheCyclesOfAModeWithoutSYNC0) {
    const OpMode mode{.assign_activate = 0x0001, .sync0_shift = -5, .sync1_factor = -1};
    EXPECT_EQ(
        ASSERT_NIL_P(resolve(mode, x::telem::MILLISECOND)),
        (Registers{.assign_activate = 0x0001, .sync0_shift = -5})
    );
}

/// @brief it should not check the SYNC1 cycle of a mode without SYNC1.
TEST(Resolve, SkipsTheSYNC1CycleOfAModeWithoutSYNC1) {
    const OpMode mode{
        .assign_activate = 0x0300,
        .sync0_cycle = 1'000,
        .sync1_factor = -1000,
    };
    EXPECT_EQ(ASSERT_NIL_P(resolve(mode, x::telem::MILLISECOND * 10)).sync1_cycle, 0U);
}

/// @brief it should refuse a mode that gives no SYNC0 cycle.
TEST(Resolve, RefusesAModeWithoutASYNC0Cycle) {
    const OpMode mode{.name = "DC-Synchron", .assign_activate = 0x0300};
    const auto err = resolve(mode, x::telem::MILLISECOND).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "mode DC-Synchron gives no SYNC0 cycle");
}

/// @brief it should refuse a SYNC0 fraction that is not a whole number of
/// nanoseconds.
TEST(Resolve, RefusesASYNC0CycleOfPartNanoseconds) {
    const OpMode mode{.assign_activate = 0x0300, .sync0_factor = -3};
    const auto err = resolve(mode, x::telem::MILLISECOND).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "SYNC0 cycle of 1ms / 3 is not a whole number of nanoseconds");
}

/// @brief it should refuse a SYNC0 cycle that its register cannot hold.
TEST(Resolve, RefusesASYNC0CycleOverTheRegisterLimit) {
    const OpMode mode{.assign_activate = 0x0300, .sync0_factor = 5};
    const auto err = resolve(mode, x::telem::SECOND).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "SYNC0 cycle of 5s is over the limit of 4s 294ms 967us 295ns");
}

/// @brief it should refuse a SYNC1 cycle that is not a multiple of the SYNC0 cycle.
TEST(Resolve, RefusesASYNC1CycleThatIsNotAMultipleOfSYNC0) {
    const OpMode mode{
        .assign_activate = 0x0700,
        .sync0_cycle = 1'000'000'000,
        .sync1_factor = -1,
    };
    const auto err = resolve(mode, x::telem::MILLISECOND * 2).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "SYNC1 cycle of 2ms is not a multiple of the SYNC0 cycle of 1s"
    );
}

/// @brief it should refuse a SYNC1 cycle that its register cannot hold, as the
/// formula of some modes gives at a slow bus.
TEST(Resolve, RefusesASYNC1CycleOverTheRegisterLimit) {
    const OpMode mode{
        .assign_activate = 0x0700,
        .sync0_cycle = 1'000,
        .sync1_factor = -1000,
    };
    const auto err = resolve(mode, x::telem::MILLISECOND * 10).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "SYNC1 cycle of 10s is over the limit of 4s 294ms 968us 295ns");
}

/// @brief it should refuse a SYNC1 shift that puts SYNC1 before its SYNC0 event.
TEST(Resolve, RefusesASYNC1ShiftBeforeSYNC0) {
    const OpMode mode{
        .assign_activate = 0x0700,
        .sync0_factor = 1,
        .sync1_shift = -100'000,
    };
    const auto err = resolve(mode, x::telem::MILLISECOND).second;
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "SYNC1 shift of -100us is under the limit of 0ns");
}

/// @brief it should mark process data with a SYNC0 cycle of 100 us or more, and not
/// with a faster SYNC0 or with SYNC0 off.
TEST(Registers, MarkDataWithASYNC0CycleOf100usOrMore) {
    EXPECT_TRUE((Registers{.sync0_cycle = 100'000}).marks_data());
    EXPECT_FALSE((Registers{.sync0_cycle = 99'999}).marks_data());
    EXPECT_FALSE(Registers{}.marks_data());
}

/// @brief it should divide a SYNC0 cycle into the period of 300 Hz, which misses a
/// third of 10 ms by a third of a nanosecond.
TEST(Grid, DividesTheRoundedPeriodWithinANanosecond) {
    EXPECT_DOUBLE_EQ(
        grid(x::telem::Rate(300).period(), {x::telem::MILLISECOND * 10}),
        10'000'000.0 / 3
    );
}

/// @brief it should give a grid of 1 ns at 300 Hz for SYNC0 cycles of 10 ms and
/// 3,333,333 ns, whose events drift apart by 100 ns each second.
TEST(Grid, IsANanosecondForSYNC0CyclesThatDriftApart) {
    EXPECT_DOUBLE_EQ(
        grid(
            x::telem::Rate(300).period(),
            {x::telem::TimeSpan(3'333'333), x::telem::MILLISECOND * 10}
        ),
        1
    );
}
}
