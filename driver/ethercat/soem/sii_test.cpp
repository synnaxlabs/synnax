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

#include "driver/ethercat/soem/sii.h"

namespace driver::ethercat::soem::sii {
namespace {
/// @brief returns the strings of a slave whose SII holds strings.
Strings lookup(const std::map<uint8_t, std::string> &strings) {
    return [strings](const uint8_t index) {
        const auto it = strings.find(index);
        return it == strings.end() ? std::string{} : it->second;
    };
}

/// @brief keeps the PDOs on sync managers 0 through 7.
bool below_eight(const uint8_t sync) {
    return sync < 8;
}
}

/// @brief it should read each PDO with its name, the objects it maps with their names
/// and CoE data types, and its padding.
TEST(ParsePDOsTest, ReadsPDOsWithNamesTypesAndPadding) {
    // A category of one TxPDO, 0x1A00 on SM3 named by string 1, with 0x6000:01 (1 bit,
    // BOOLEAN, string 2), 7 bits of padding, and 0x6000:11 (INTEGER16, string 3).
    const std::vector<uint8_t> category = {
        16, 0,    0x00, 0x1A, 3, 3, 0, 1, 0, 0,    0x00, 0x60, 1, 2,    0x01, 1, 0,
        0,  0x00, 0x00, 0,    0, 0, 7, 0, 0, 0x00, 0x60, 17,   3, 0x03, 16,   0, 0,
    };
    const auto data = std::span(category).subspan(2);
    const auto pdos = parse_pdos(
        data,
        below_eight,
        lookup({{1, "Channel 1"}, {2, "Underrange"}, {3, "Value"}})
    );
    ASSERT_EQ(pdos.size(), 1);
    EXPECT_EQ(pdos[0].index, 0x1A00);
    EXPECT_EQ(pdos[0].name, "Channel 1");
    ASSERT_EQ(pdos[0].entries.size(), 3);
    EXPECT_EQ(pdos[0].entries[0].index, 0x6000);
    EXPECT_EQ(pdos[0].entries[0].sub_index, 1);
    EXPECT_EQ(pdos[0].entries[0].bit_length, 1);
    EXPECT_EQ(pdos[0].entries[0].name, "Underrange");
    EXPECT_EQ(pdos[0].entries[0].type, telem::DataType::EC_BOOLEAN);
    EXPECT_EQ(pdos[0].entries[1].index, 0);
    EXPECT_EQ(pdos[0].entries[1].bit_length, 7);
    EXPECT_EQ(pdos[0].entries[2].sub_index, 17);
    EXPECT_EQ(pdos[0].entries[2].bit_length, 16);
    EXPECT_EQ(pdos[0].entries[2].name, "Value");
    EXPECT_EQ(pdos[0].entries[2].type, telem::DataType::EC_INTEGER16);
}

/// @brief it should read PDOs one after another, and give a PDO no name when its SII
/// string index is 0.
TEST(ParsePDOsTest, ReadsConsecutivePDOs) {
    const std::vector<uint8_t> data = {
        0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0,
        0x01, 0x1A, 1, 3, 0, 0, 0, 0, 0x10, 0x60, 1, 0, 0x08, 32, 0, 0,
    };
    const auto pdos = parse_pdos(data, below_eight, lookup({}));
    ASSERT_EQ(pdos.size(), 2);
    EXPECT_EQ(pdos[0].name, std::nullopt);
    EXPECT_EQ(pdos[1].index, 0x1A01);
    EXPECT_EQ(pdos[1].entries[0].index, 0x6010);
    EXPECT_EQ(pdos[1].entries[0].type, telem::DataType::EC_REAL32);
}

/// @brief it should leave out a PDO on a sync manager that keep rejects.
TEST(ParsePDOsTest, LeavesOutAPDOOnARejectedSyncManager) {
    const std::vector<uint8_t> data = {
        0x00,
        0x1A,
        1,
        8,
        0,
        0,
        0,
        0,
        0x00,
        0x60,
        1,
        0,
        0x03,
        16,
        0,
        0,
    };
    EXPECT_TRUE(parse_pdos(data, below_eight, lookup({})).empty());
}

/// @brief it should leave out a PDO that counts more objects than the category holds.
TEST(ParsePDOsTest, LeavesOutAPDOThatRunsPastTheCategory) {
    const std::vector<uint8_t> data = {
        0x00,
        0x1A,
        2,
        3,
        0,
        0,
        0,
        0,
        0x00,
        0x60,
        1,
        0,
        0x03,
        16,
        0,
        0,
    };
    EXPECT_TRUE(parse_pdos(data, below_eight, lookup({})).empty());
}

/// @brief it should read each synchronization mode in SII order.
TEST(ParseOpModesTest, ReadsModesInOrder) {
    // The DC category of a slave on the test bench: SM, DC with a 1 s SYNC0, and DC
    // with a 1 us SYNC0 and a SYNC1 of 1000 bus cycles.
    const std::vector<uint8_t> category = {
        36, 0, 0,    0,    0, 0,    0,    0, 0, 0, 0, 0,    0,    0,    0,
        0,  0, 0,    1,    0, 3,    0,    0, 0, 0, 0, 0,    0xCA, 0x9A, 0x3B,
        0,  0, 0,    0,    0, 0,    0,    0, 0, 0, 0, 0x07, 0,    0,    4,
        0,  0, 0,    0,    0, 0xE8, 0x03, 0, 0, 0, 0, 0,    0,    0,    0,
        0,  0, 0x18, 0xFC, 0, 0x07, 0,    0, 5, 0, 0, 0,    0,    0,
    };
    const auto data = std::span(category).subspan(2);
    const std::vector<dc::OpMode> expected = {
        {.name = "SM-Synchron", .sync0_factor = 1},
        {.name = "DC", .assign_activate = 0x0700, .sync0_cycle = 1000000000},
        {.name = "DC-bus1",
         .assign_activate = 0x0700,
         .sync0_cycle = 1000,
         .sync1_factor = -1000},
    };
    EXPECT_EQ(
        parse_op_modes(data, lookup({{3, "SM-Synchron"}, {4, "DC"}, {5, "DC-bus1"}})),
        expected
    );
}

/// @brief it should read negative shift times and leave the name of a mode empty when
/// the SII gives none.
TEST(ParseOpModesTest, ReadsSignedShiftsAndUnnamedModes) {
    // One DC mode: SYNC0 = bus cycle, SYNC0 shift -100 us, SYNC1 shift 25 us, no name.
    const std::vector<uint8_t> category = {
        12, 0,    0,    0,    0,    0,    0x60, 0x79, 0xFE, 0xFF, 0xA8, 0x61, 0,
        0,  0x01, 0x00, 0x00, 0x03, 0x01, 0x00, 0,    0,    0,    0,    0,    0,
    };
    const auto data = std::span(category).subspan(2);
    const std::vector<dc::OpMode> expected = {{
        .assign_activate = 0x0300,
        .sync0_factor = 1,
        .sync0_shift = -100000,
        .sync1_factor = 1,
        .sync1_shift = 25000,
    }};
    EXPECT_EQ(parse_op_modes(data, lookup({})), expected);
}

/// @brief it should leave out a mode that the category holds only part of.
TEST(ParseOpModesTest, LeavesOutAPartialMode) {
    EXPECT_TRUE(parse_op_modes(std::vector<uint8_t>(23, 0), lookup({})).empty());
}
}
