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

namespace driver::ethercat::esi {

class KnownDevicesTest : public ::testing::Test {
protected:
    slave::Properties slave;
};

TEST_F(KnownDevicesTest, LookupUnknownDeviceReturnsFalse) {
    bool found = lookup_device_pdos(0xDEADBEEF, 0x12345678, 0x00000001, slave);
    EXPECT_FALSE(found);
    EXPECT_TRUE(slave.input_pdos.empty());
    EXPECT_TRUE(slave.output_pdos.empty());
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
    slave::Properties slave;
};

TEST_F(BeckhoffDevicesTest, VendorNameReturnsBeckhoff) {
    auto name = vendor_name(BECKHOFF_VENDOR_ID);
    ASSERT_TRUE(name.has_value());
    EXPECT_NE(name->find("Beckhoff"), std::string_view::npos);
}

// DEWESoft vendor tests (vendor ID: 0xDEBE50F7)
class DEWESoftDevicesTest : public ::testing::Test {
protected:
    static constexpr uint32_t DEWESOFT_VENDOR_ID = 0xDEBE50F7;
    slave::Properties slave;
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

TEST_F(DEWESoftDevicesTest, LookupPopulatesPDOs) {
    uint32_t found_product_code = 0;
    for (uint32_t pc = 1; pc <= 500; ++pc) {
        if (is_device_known(DEWESOFT_VENDOR_ID, pc)) {
            found_product_code = pc;
            break;
        }
    }
    if (found_product_code == 0) GTEST_SKIP() << "No DEWESoft devices in registry";

    bool found = lookup_device_pdos(
        DEWESOFT_VENDOR_ID,
        found_product_code,
        0x00000001,
        this->slave
    );
    EXPECT_TRUE(found);

    size_t total_pdos = this->slave.input_pdos.size() + this->slave.output_pdos.size();
    EXPECT_GT(total_pdos, 0);
}

TEST_F(DEWESoftDevicesTest, RevisionFallbackWorks) {
    uint32_t found_product_code = 0;
    for (uint32_t pc = 1; pc <= 500; ++pc) {
        if (is_device_known(DEWESOFT_VENDOR_ID, pc)) {
            found_product_code = pc;
            break;
        }
    }
    if (found_product_code == 0) GTEST_SKIP() << "No DEWESoft devices in registry";

    bool found = lookup_device_pdos(
        DEWESOFT_VENDOR_ID,
        found_product_code,
        0xFFFFFFFF,
        this->slave
    );
    EXPECT_TRUE(found) << "Revision fallback should work";
}

}
