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

#include "driver/modbus/block.h"
#include "driver/modbus/device/device.h"

namespace driver::modbus::block {
namespace {
struct Range {
    std::uint16_t address;
    std::size_t w;

    [[nodiscard]] std::size_t width() const { return this->w; }
};

void expect_block(
    const Block &b,
    const std::uint16_t start,
    const std::size_t count,
    const std::size_t first,
    const std::size_t last
) {
    EXPECT_EQ(b.start, start);
    EXPECT_EQ(b.count, count);
    EXPECT_EQ(b.first, first);
    EXPECT_EQ(b.last, last);
}
}

/// @brief it should return no blocks for no channels.
TEST(Plan, Empty) {
    EXPECT_TRUE(plan(std::vector<Range>{}, 125, true).empty());
}

/// @brief it should span gaps between channels when gaps are allowed.
TEST(Plan, SpansGaps) {
    const std::vector<Range> chs{{0, 1}, {10, 2}, {50, 4}};
    const auto blocks = plan(chs, 125, true);
    ASSERT_EQ(blocks.size(), 1);
    expect_block(blocks[0], 0, 54, 0, 3);
}

/// @brief it should split a block that would pass the limit.
TEST(Plan, SplitsAtLimit) {
    const std::vector<Range> chs{{0, 1}, {124, 1}, {125, 1}, {300, 2}};
    const auto blocks = plan(chs, 125, true);
    ASSERT_EQ(blocks.size(), 3);
    expect_block(blocks[0], 0, 125, 0, 2);
    expect_block(blocks[1], 125, 1, 2, 3);
    expect_block(blocks[2], 300, 2, 3, 4);
}

/// @brief it should split a block when a multi-register channel crosses the limit.
TEST(Plan, SplitsWideChannelAtLimit) {
    const std::vector<Range> chs{{0, 1}, {123, 4}};
    const auto blocks = plan(chs, 125, true);
    ASSERT_EQ(blocks.size(), 2);
    expect_block(blocks[0], 0, 1, 0, 1);
    expect_block(blocks[1], 123, 4, 1, 2);
}

/// @brief it should size a block by the channel that ends last, not the one that
/// starts last.
TEST(Plan, SizesByFurthestEnd) {
    const std::vector<Range> chs{{10, 4}, {11, 1}};
    const auto blocks = plan(chs, 125, true);
    ASSERT_EQ(blocks.size(), 1);
    expect_block(blocks[0], 10, 4, 0, 2);
}

/// @brief it should merge only channels that touch end to end when gaps are not
/// allowed.
TEST(Plan, NoGaps) {
    const std::vector<Range> chs{{0, 2}, {2, 1}, {3, 4}, {8, 1}, {9, 1}};
    const auto blocks = plan(chs, 123, false);
    ASSERT_EQ(blocks.size(), 2);
    expect_block(blocks[0], 0, 7, 0, 3);
    expect_block(blocks[1], 8, 2, 3, 5);
}

/// @brief it should split adjacent channels at the limit when gaps are not allowed.
TEST(Plan, NoGapsSplitsAtLimit) {
    std::vector<Range> chs;
    chs.reserve(1970);
    for (std::uint16_t i = 0; i < 1970; i++)
        chs.push_back({i, 1});
    const auto blocks = plan(chs, 1968, false);
    ASSERT_EQ(blocks.size(), 2);
    expect_block(blocks[0], 0, 1968, 0, 1968);
    expect_block(blocks[1], 1968, 2, 1968, 1970);
}

/// @brief it should print one address or a range.
TEST(Addresses, Format) {
    EXPECT_EQ(addresses(Block{.start = 5, .count = 1}), "5");
    EXPECT_EQ(addresses(Block{.start = 5, .count = 4}), "5-8");
}

/// @brief it should name the channels and addresses and keep the error type.
TEST(Wrap, NamesChannelsAndAddresses) {
    const x::errors::Error err(device::CRITICAL_ERROR, "Illegal data address");
    const auto wrapped = wrap(
        err,
        "holding registers",
        Block{.start = 100, .count = 25},
        {"pump_speed", "flow"}
    );
    ASSERT_OCCURRED_AS(wrapped, device::CRITICAL_ERROR);
    EXPECT_EQ(
        wrapped.data,
        "pump_speed, flow (holding registers 100-124): Illegal data address"
    );
}
}
