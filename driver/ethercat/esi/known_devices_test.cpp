// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

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

// Beckhoff vendor tests (vendor ID: 0x00000002)
class BeckhoffDevicesTest : public ::testing::Test {
protected:
    static constexpr uint32_t BECKHOFF_VENDOR_ID = 0x00000002;
    slave::Properties slave;
};

constexpr uint32_t DEWESOFT_VENDOR_ID = 0xDEBE50F7;
constexpr uint32_t IOLITE_6XSTG = 0xFC;

/// @brief it should fill the six strain inputs of an IOLITE 6xSTG from the registry.
TEST(DewesoftDevices, LookupFillsTheEntriesOfAnIOLITE6xSTG) {
    slave::Properties slave;
    ASSERT_TRUE(lookup_device_pdos(DEWESOFT_VENDOR_ID, IOLITE_6XSTG, 0x10001, slave));
    EXPECT_EQ(slave.input_pdos.size(), 6);
    EXPECT_TRUE(slave.output_pdos.empty());
}

/// @brief it should use a listed revision of the device when the slave's revision is
/// not listed.
TEST(DewesoftDevices, LookupFallsBackToAListedRevision) {
    slave::Properties slave;
    ASSERT_TRUE(
        lookup_device_pdos(DEWESOFT_VENDOR_ID, IOLITE_6XSTG, 0xFFFFFFFF, slave)
    );
    EXPECT_EQ(slave.input_pdos.size(), 6);
}

/// @brief it should fill the name and data type of a live entry from the registry entry
/// at the same index, sub-index, and bit length.
TEST_F(BeckhoffDevicesTest, FillKnownEntriesUsesRegistryEntryAtSameAddress) {
    constexpr uint32_t el3102 = 0x0C1E3052;
    ASSERT_TRUE(lookup_device_pdos(BECKHOFF_VENDOR_ID, el3102, 1, this->slave));
    const auto known = this->slave.input_pdos.back();
    slave::Properties live{
        .vendor_id = BECKHOFF_VENDOR_ID,
        .product_code = el3102,
        .revision = 1,
        .input_pdos = {
            {.index = known.index,
             .sub_index = known.sub_index,
             .bit_length = known.bit_length,
             .name = "Input (unknown) 0x6000:01",
             .data_type = x::telem::UNKNOWN_T},
            {.index = 0x7FFF,
             .sub_index = 1,
             .bit_length = 8,
             .name = "Input (uint8) 0x7FFF:01",
             .data_type = x::telem::UINT8_T},
        },
    };
    fill_known_entries(live);
    EXPECT_EQ(live.input_pdos[0].name, known.name);
    EXPECT_EQ(live.input_pdos[0].data_type, known.data_type);
    EXPECT_EQ(live.input_pdos[1].name, "Input (uint8) 0x7FFF:01");
    EXPECT_EQ(live.input_pdos[1].data_type, x::telem::UINT8_T);
}

/// @brief it should leave a live entry unchanged when the registry entry at its address
/// has another bit length.
TEST_F(BeckhoffDevicesTest, FillKnownEntriesSkipsEntryOfOtherBitLength) {
    constexpr uint32_t el3102 = 0x0C1E3052;
    ASSERT_TRUE(lookup_device_pdos(BECKHOFF_VENDOR_ID, el3102, 1, this->slave));
    const auto known = this->slave.input_pdos.back();
    slave::Properties live{
        .vendor_id = BECKHOFF_VENDOR_ID,
        .product_code = el3102,
        .revision = 1,
        .input_pdos = {
            {.index = known.index,
             .sub_index = known.sub_index,
             .bit_length = static_cast<uint8_t>(known.bit_length + 8),
             .name = "Input (unknown) 0x6000:01",
             .data_type = x::telem::UNKNOWN_T},
        },
    };
    fill_known_entries(live);
    EXPECT_EQ(live.input_pdos[0].name, "Input (unknown) 0x6000:01");
    EXPECT_EQ(live.input_pdos[0].data_type, x::telem::UNKNOWN_T);
}

}
