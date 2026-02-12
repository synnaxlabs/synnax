// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/json/json.h"

#include "driver/ethercat/pdo/pdo.h"

namespace driver::ethercat::pdo {
/// @brief it should return true when all Key fields match and false otherwise.
TEST(PDOKeyTest, EqualityOperator) {
    const Key key1{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    const Key key2{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    EXPECT_TRUE(key1 == key2);

    EXPECT_FALSE(
        (key1 ==
         Key{.slave_position = 2, .index = 0x6000, .sub_index = 1, .is_input = true})
    );
    EXPECT_FALSE(
        (key1 ==
         Key{.slave_position = 1, .index = 0x7000, .sub_index = 1, .is_input = true})
    );
    EXPECT_FALSE(
        (key1 ==
         Key{.slave_position = 1, .index = 0x6000, .sub_index = 2, .is_input = true})
    );
    EXPECT_FALSE(
        (key1 ==
         Key{.slave_position = 1, .index = 0x6000, .sub_index = 1, .is_input = false})
    );
}

/// @brief it should produce consistent hashes for equal keys.
TEST(PDOKeyTest, HashConsistency) {
    const Key key1{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    const Key key2{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };

    const KeyHash hasher;
    EXPECT_EQ(hasher(key1), hasher(key2));

    const Key key3{
        .slave_position = 2,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    EXPECT_NE(hasher(key1), hasher(key3));
}

/// @brief it should work correctly as a key in unordered_map.
TEST(PDOKeyTest, WorksInUnorderedMap) {
    Offsets offsets;

    const Key key1{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    offsets[key1] = {.byte = 0, .bit = 0};

    const Key key2{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 2,
        .is_input = true
    };
    offsets[key2] = {.byte = 2, .bit = 0};

    EXPECT_EQ(offsets.size(), 2);
    EXPECT_EQ(offsets[key1].byte, 0);
    EXPECT_EQ(offsets[key2].byte, 2);

    const Key lookup{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .is_input = true
    };
    EXPECT_EQ(offsets[lookup].byte, 0);
}

/// @brief it should correctly round up bit lengths to bytes.
TEST(PDOEntryTest, ByteLengthCalculation) {
    Entry entry;

    entry.bit_length = 8;
    EXPECT_EQ(entry.byte_length(), 1);

    entry.bit_length = 1;
    EXPECT_EQ(entry.byte_length(), 1);

    entry.bit_length = 16;
    EXPECT_EQ(entry.byte_length(), 2);

    entry.bit_length = 9;
    EXPECT_EQ(entry.byte_length(), 2);

    entry.bit_length = 32;
    EXPECT_EQ(entry.byte_length(), 4);
}

/// @brief it should correctly round up bit lengths to bytes for Properties.
TEST(PDOPropertiesTest, ByteLengthCalculation) {
    Properties props{
        .pdo_index = 0x1A00,
        .index = 0x6000,
        .sub_index = 1,
        .bit_length = 8,
        .is_input = true,
        .name = "Test",
        .data_type = x::telem::UINT8_T
    };

    EXPECT_EQ(props.byte_length(), 1);

    props.bit_length = 1;
    EXPECT_EQ(props.byte_length(), 1);

    props.bit_length = 16;
    EXPECT_EQ(props.byte_length(), 2);

    props.bit_length = 9;
    EXPECT_EQ(props.byte_length(), 2);
}

/// @brief it should correctly parse Properties from JSON.
TEST(PDOPropertiesTest, ParseFromJSON) {
    x::json::Parser parser(std::string(R"({
        "pdo_index": 6656,
        "index": 24576,
        "sub_index": 1,
        "bit_length": 16,
        "name": "Position",
        "data_type": "uint16"
    })"));

    auto props = Properties::parse(parser, true);

    ASSERT_TRUE(parser.ok());
    EXPECT_EQ(props.pdo_index, 0x1A00);
    EXPECT_EQ(props.index, 0x6000);
    EXPECT_EQ(props.sub_index, 1);
    EXPECT_EQ(props.bit_length, 16);
    EXPECT_TRUE(props.is_input);
    EXPECT_EQ(props.name, "Position");
    EXPECT_EQ(props.data_type, x::telem::UINT16_T);
}

/// @brief it should correctly serialize Properties to JSON.
TEST(PDOPropertiesTest, ToJSON) {
    Properties props{
        .pdo_index = 0x1A00,
        .index = 0x6000,
        .sub_index = 1,
        .bit_length = 16,
        .is_input = true,
        .name = "Position",
        .data_type = x::telem::UINT16_T
    };

    auto json = props.to_json();

    EXPECT_EQ(json["name"], "Position");
    EXPECT_EQ(json["pdo_index"], 0x1A00);
    EXPECT_EQ(json["index"], 0x6000);
    EXPECT_EQ(json["sub_index"], 1);
    EXPECT_EQ(json["bit_length"], 16);
    EXPECT_EQ(json["data_type"], "uint16");
}

TEST(FindOffset, ReturnsOffsetForExistingEntry) {
    Offsets offsets;
    offsets[Key{1, 0x6000, 1, true}] = {10, 3};
    Entry entry{.slave_position = 1, .index = 0x6000, .sub_index = 1, .is_input = true};
    auto result = find_offset(offsets, entry);
    EXPECT_EQ(result.byte, 10);
    EXPECT_EQ(result.bit, 3);
}

TEST(FindOffset, ReturnsZeroOffsetForMissingEntry) {
    Offsets offsets;
    Entry entry{.slave_position = 1, .index = 0x6000, .sub_index = 1, .is_input = true};
    auto result = find_offset(offsets, entry);
    EXPECT_EQ(result.byte, 0);
    EXPECT_EQ(result.bit, 0);
}

TEST(ComputeOffsetsProperties, ByteAlignedOffsets) {
    Offsets offsets;
    std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 16},
        {.index = 0x6000, .sub_index = 2, .bit_length = 32},
    };
    compute_offsets(offsets, 1, pdos, true, 0);

    const Key k1{1, 0x6000, 1, true};
    const Key k2{1, 0x6000, 2, true};
    EXPECT_EQ(offsets[k1].byte, 0);
    EXPECT_EQ(offsets[k1].bit, 0);
    EXPECT_EQ(offsets[k2].byte, 2);
    EXPECT_EQ(offsets[k2].bit, 0);
}

TEST(ComputeOffsetsProperties, SubByteBitOffsets) {
    Offsets offsets;
    std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 1},
        {.index = 0x6000, .sub_index = 2, .bit_length = 1},
        {.index = 0x6000, .sub_index = 3, .bit_length = 1},
    };
    compute_offsets(offsets, 1, pdos, true, 0);

    const Key k1{1, 0x6000, 1, true};
    const Key k2{1, 0x6000, 2, true};
    const Key k3{1, 0x6000, 3, true};
    EXPECT_EQ(offsets[k1].byte, 0);
    EXPECT_EQ(offsets[k1].bit, 0);
    EXPECT_EQ(offsets[k2].byte, 0);
    EXPECT_EQ(offsets[k2].bit, 1);
    EXPECT_EQ(offsets[k3].byte, 0);
    EXPECT_EQ(offsets[k3].bit, 2);
}

TEST(ComputeOffsetsProperties, BaseOffsetPropagation) {
    Offsets offsets;
    std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 8},
    };
    compute_offsets(offsets, 1, pdos, true, 100);

    const Key k1{1, 0x6000, 1, true};
    EXPECT_EQ(offsets[k1].byte, 100);
    EXPECT_EQ(offsets[k1].bit, 0);
}

TEST(ComputeOffsetsProperties, EmptyPdoList) {
    Offsets offsets;
    std::vector<Properties> pdos;
    compute_offsets(offsets, 1, pdos, true, 0);
    EXPECT_TRUE(offsets.empty());
}

TEST(ComputeOffsetsProperties, MultiSlave) {
    Offsets offsets;
    std::vector<Properties> slave1_pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 16},
    };
    std::vector<Properties> slave2_pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 8},
    };
    compute_offsets(offsets, 1, slave1_pdos, true, 0);
    compute_offsets(offsets, 2, slave2_pdos, true, 10);

    const Key k1{1, 0x6000, 1, true};
    const Key k2{2, 0x6000, 1, true};
    EXPECT_EQ(offsets[k1].byte, 0);
    EXPECT_EQ(offsets[k2].byte, 10);
}

TEST(ComputeOffsetsEntries, InputOutputDistinction) {
    Offsets offsets;
    std::vector<Entry> entries = {
        {.slave_position = 1,
         .index = 0x6000,
         .sub_index = 1,
         .bit_length = 16,
         .is_input = true},
        {.slave_position = 1,
         .index = 0x7000,
         .sub_index = 1,
         .bit_length = 8,
         .is_input = false},
        {.slave_position = 1,
         .index = 0x6000,
         .sub_index = 2,
         .bit_length = 32,
         .is_input = true},
    };
    compute_offsets(offsets, entries, 0, 50);

    const Key ki1{1, 0x6000, 1, true};
    const Key ko1{1, 0x7000, 1, false};
    const Key ki2{1, 0x6000, 2, true};
    EXPECT_EQ(offsets[ki1].byte, 0);
    EXPECT_EQ(offsets[ko1].byte, 50);
    EXPECT_EQ(offsets[ki2].byte, 2);
}

TEST(ComputeOffsetsEntries, BaseOffsets) {
    Offsets offsets;
    std::vector<Entry> entries = {
        {.slave_position = 1,
         .index = 0x6000,
         .sub_index = 1,
         .bit_length = 8,
         .is_input = true},
        {.slave_position = 1,
         .index = 0x7000,
         .sub_index = 1,
         .bit_length = 16,
         .is_input = false},
    };
    compute_offsets(offsets, entries, 10, 20);

    const Key ki{1, 0x6000, 1, true};
    const Key ko{1, 0x7000, 1, false};
    EXPECT_EQ(offsets[ki].byte, 10);
    EXPECT_EQ(offsets[ko].byte, 20);
}

/// @brief it should preserve values through JSON parse and serialize round-trip.
TEST(PDOPropertiesTest, JSONRoundTrip) {
    x::json::Parser parser(std::string(R"({
        "pdo_index": 6400,
        "index": 28672,
        "sub_index": 2,
        "bit_length": 32,
        "name": "Velocity",
        "data_type": "int32"
    })"));

    auto props = Properties::parse(parser, false);
    ASSERT_TRUE(parser.ok());

    auto json = props.to_json();

    EXPECT_EQ(json["pdo_index"], 0x1900);
    EXPECT_EQ(json["index"], 0x7000);
    EXPECT_EQ(json["sub_index"], 2);
    EXPECT_EQ(json["bit_length"], 32);
    EXPECT_EQ(json["name"], "Velocity");
    EXPECT_EQ(json["data_type"], "int32");
}
}
