// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "driver/ethercat/esi/known_devices.h"

namespace ethercat::esi {

class KnownDevicesTest : public ::testing::Test {
protected:
    SlaveInfo slave;
};

TEST_F(KnownDevicesTest, RegistryStatsArePopulated) {
    EXPECT_GT(RegistryStats::DEVICE_COUNT, 0);
    EXPECT_GT(RegistryStats::VENDOR_COUNT, 0);
    EXPECT_GT(RegistryStats::PDO_ENTRY_COUNT, 0);
}

TEST_F(KnownDevicesTest, LookupUnknownDeviceReturnsFalse) {
    bool found = lookup_device_pdos(0xDEADBEEF, 0x12345678, 0x00000001, slave);
    EXPECT_FALSE(found);
    EXPECT_FALSE(slave.pdos_discovered);
}

TEST_F(KnownDevicesTest, IsDeviceKnownReturnsFalseForUnknown) {
    bool known = is_device_known(0xDEADBEEF, 0x12345678);
    EXPECT_FALSE(known);
}

TEST_F(KnownDevicesTest, VendorNameReturnsNulloptForUnknown) {
    auto name = vendor_name(0xDEADBEEF);
    EXPECT_FALSE(name.has_value());
}

// Beckhoff vendor tests (vendor ID: 0x00000002)
class BeckhoffDevicesTest : public ::testing::Test {
protected:
    static constexpr uint32_t BECKHOFF_VENDOR_ID = 0x00000002;
    SlaveInfo slave;
};

TEST_F(BeckhoffDevicesTest, VendorNameReturnsBeckhoff) {
    auto name = vendor_name(BECKHOFF_VENDOR_ID);
    ASSERT_TRUE(name.has_value());
    EXPECT_NE(name->find("Beckhoff"), std::string_view::npos);
}

TEST_F(BeckhoffDevicesTest, BeckhoffDevicesExist) {
    // Beckhoff should have many devices registered
    bool any_known = false;
    for (uint32_t pc = 1; pc <= 1000; ++pc) {
        if (is_device_known(BECKHOFF_VENDOR_ID, pc)) {
            any_known = true;
            break;
        }
    }
    EXPECT_TRUE(any_known) << "No Beckhoff devices found in registry";
}

TEST_F(BeckhoffDevicesTest, LookupPopulatesPDOs) {
    // Find a Beckhoff device
    uint32_t found_product_code = 0;
    for (uint32_t pc = 1; pc <= 10000; ++pc) {
        if (is_device_known(BECKHOFF_VENDOR_ID, pc)) {
            found_product_code = pc;
            break;
        }
    }
    ASSERT_NE(found_product_code, 0) << "No Beckhoff devices found";

    bool found = lookup_device_pdos(
        BECKHOFF_VENDOR_ID,
        found_product_code,
        0x00000001,
        slave
    );
    EXPECT_TRUE(found);
    EXPECT_TRUE(slave.pdos_discovered);

    size_t total_pdos = slave.input_pdos.size() + slave.output_pdos.size();
    EXPECT_GT(total_pdos, 0);
}

TEST_F(BeckhoffDevicesTest, RevisionFallbackWorks) {
    // Find a Beckhoff device
    uint32_t found_product_code = 0;
    for (uint32_t pc = 1; pc <= 10000; ++pc) {
        if (is_device_known(BECKHOFF_VENDOR_ID, pc)) {
            found_product_code = pc;
            break;
        }
    }
    ASSERT_NE(found_product_code, 0);

    // Lookup with non-existent revision should still work (fallback)
    bool found = lookup_device_pdos(
        BECKHOFF_VENDOR_ID,
        found_product_code,
        0xFFFFFFFF,
        slave
    );
    EXPECT_TRUE(found) << "Revision fallback should work";
    EXPECT_TRUE(slave.pdos_discovered);
}

// DEWESoft vendor tests (vendor ID: 0xDEBE50F7)
class DEWESoftDevicesTest : public ::testing::Test {
protected:
    static constexpr uint32_t DEWESOFT_VENDOR_ID = 0xDEBE50F7;
    SlaveInfo slave;
};

TEST_F(DEWESoftDevicesTest, VendorNameReturnsDEWESoft) {
    auto name = vendor_name(DEWESOFT_VENDOR_ID);
    if (!name.has_value()) { GTEST_SKIP() << "DEWESoft not in registry"; }
    EXPECT_NE(name->find("ewesoft"), std::string_view::npos);
}

TEST_F(DEWESoftDevicesTest, DEWESoftDevicesExist) {
    if (!vendor_name(DEWESOFT_VENDOR_ID).has_value()) {
        GTEST_SKIP() << "DEWESoft not in registry";
    }

    bool any_known = false;
    for (uint32_t pc = 1; pc <= 500; ++pc) {
        if (is_device_known(DEWESOFT_VENDOR_ID, pc)) {
            any_known = true;
            break;
        }
    }
    EXPECT_TRUE(any_known) << "No DEWESoft devices found in registry";
}

} // namespace ethercat::esi
