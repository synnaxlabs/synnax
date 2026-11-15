// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/slaves.h"

namespace driver::ethercat {
namespace {
/// @brief returns a slave at position with a serial number.
slave::DiscoveryResult on_bus(const uint16_t position, const uint32_t serial) {
    slave::DiscoveryResult result;
    result.properties.position = position;
    result.properties.vendor_id = 2;
    result.properties.product_code = 0x12345678;
    result.properties.serial = serial;
    result.properties.name = "Slave " + std::to_string(position);
    return result;
}
}

/// @brief it should save the position where it finds each slave on the bus.
TEST(SlavesBindTest, SavesThePositionOfASlaveThatMoved) {
    Slaves slaves;
    slaves.by_key.emplace("moved", on_bus(2, 777).properties);
    slaves.by_key.emplace("kept", on_bus(1, 888).properties);
    const auto positions = ASSERT_NIL_P(slaves.bind({on_bus(1, 888), on_bus(4, 777)}));
    EXPECT_EQ(positions, (std::map<uint16_t, uint16_t>{{1, 1}, {2, 4}}));
    EXPECT_EQ(slaves.by_key.at("moved").position, 4);
    EXPECT_EQ(slaves.by_key.at("kept").position, 1);
}

/// @brief it should keep the saved positions when it does not find a slave.
TEST(SlavesBindTest, KeepsThePositionsWhenASlaveIsMissing) {
    Slaves slaves;
    slaves.by_key.emplace("missing", on_bus(2, 777).properties);
    const auto err = slaves.bind({on_bus(1, 888)}).second;
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Slave 2 (serial 777) is not on the bus.");
    EXPECT_EQ(slaves.by_key.at("missing").position, 2);
}
}
