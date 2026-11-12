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

#include "driver/ethercat/slaves.h"

namespace driver::ethercat {
namespace {
/// @brief returns a slave at position with a product.
slave::DiscoveryResult
on_bus(const uint16_t position, const uint32_t vendor_id, const uint32_t product_code) {
    slave::DiscoveryResult result;
    result.properties.position = position;
    result.properties.vendor_id = vendor_id;
    result.properties.product_code = product_code;
    result.properties.name = "Slave " + std::to_string(position);
    return result;
}

/// @brief returns the slaves with the saved properties of each slave on bus, keyed as
/// dev<position>.
Slaves saved(const std::vector<slave::DiscoveryResult> &bus) {
    Slaves slaves;
    for (const auto &s: bus)
        slaves.by_key.emplace(
            "dev" + std::to_string(s.properties.position),
            s.properties
        );
    return slaves;
}
}

/// @brief it should accept a bus that holds each saved slave at its position, and
/// slaves the task does not name.
TEST(SlavesValidateTest, AcceptsTheSavedSlaves) {
    const auto slaves = saved({on_bus(1, 2, 0x12345678), on_bus(2, 2, 0xABCDEF00)});
    ASSERT_NIL(slaves.validate(
        {on_bus(1, 2, 0x12345678), on_bus(2, 2, 0xABCDEF00), on_bus(3, 2, 1)}
    ));
}

/// @brief it should reject a bus with no slave at a saved position.
TEST(SlavesValidateTest, RejectsAMissingSlave) {
    const auto err = saved({on_bus(2, 2, 0xABCDEF00)}).validate({on_bus(1, 2, 1)});
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Slave 2 (Slave 2) is not on the bus.");
}

/// @brief it should reject a slave of another vendor at a saved position.
TEST(SlavesValidateTest, RejectsAnotherVendor) {
    const auto err = saved({on_bus(1, 0x99, 0x12345678)})
                         .validate({on_bus(1, 2, 0x12345678)});
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 1 (Slave 1): expected vendor_id 0x00000099, found 0x00000002."
    );
}

/// @brief it should reject another product at a saved position.
TEST(SlavesValidateTest, RejectsAnotherProduct) {
    const auto err = saved({on_bus(1, 2, 0x87654321)})
                         .validate({on_bus(1, 2, 0x12345678)});
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 1 (Slave 1): expected product_code 0x87654321, found 0x12345678."
    );
}

/// @brief it should report the mismatch at the lowest position when several slaves
/// mismatch.
TEST(SlavesValidateTest, ReportsTheLowestMismatchedPosition) {
    std::vector<slave::DiscoveryResult> bus;
    for (uint16_t pos = 1; pos <= 8; ++pos)
        bus.push_back(on_bus(pos, 2, pos));
    auto slaves = saved(bus);
    for (auto &[_, s]: slaves.by_key)
        if (s.position > 2) s.vendor_id = 0x99;
    const auto err = slaves.validate(bus);
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 3 (Slave 3): expected vendor_id 0x00000099, found 0x00000002."
    );
}
}
