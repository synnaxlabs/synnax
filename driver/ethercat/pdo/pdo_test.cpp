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
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
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
    offsets[Key{1, 0x6000, 1, true}] = {10, 3, 16};
    Entry entry{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .bit_length = 16,
        .is_input = true
    };
    const auto result = ASSERT_NIL_P(find_offset(offsets, entry));
    EXPECT_EQ(result.byte, 10);
    EXPECT_EQ(result.bit, 3);
}

/// @brief it should return a mapping error when the entry in the offsets has another
/// bit length.
TEST(FindOffset, ReturnsMappingErrorForAnotherBitLength) {
    Offsets offsets;
    offsets[Key{1, 0x6000, 1, true}] = {10, 3, 16};
    Entry entry{
        .slave_position = 1,
        .index = 0x6000,
        .sub_index = 1,
        .bit_length = 32,
        .is_input = true
    };
    const auto err = find_offset(offsets, entry).second;
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        err.data,
        "PDO 0x6000:01 on module 1 is 16 bits, not 32. Scan the network again."
    );
}

/// @brief it should return a mapping error that names the entry and slave when the
/// entry is not in the offsets.
TEST(FindOffset, ReturnsMappingErrorForMissingEntry) {
    Offsets offsets;
    Entry entry{.slave_position = 1, .index = 0x6000, .sub_index = 1, .is_input = true};
    const auto err = find_offset(offsets, entry).second;
    ASSERT_OCCURRED_AS(err, errors::PDO_NOT_MAPPED);
    EXPECT_EQ(err.data, "PDO 0x6000:01 is not in the process image of module 1.");
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
    EXPECT_EQ(offsets[k1].bit_length, 16);
    EXPECT_EQ(offsets[k2].byte, 2);
    EXPECT_EQ(offsets[k2].bit, 0);
    EXPECT_EQ(offsets[k2].bit_length, 32);
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

/// @brief it should start at the given bit and carry later entries into the next byte.
TEST(ComputeOffsetsProperties, StartBitCarriesIntoTheNextByte) {
    Offsets offsets;
    const std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 2},
        {.index = 0x6000, .sub_index = 2, .bit_length = 2},
    };
    compute_offsets(offsets, 1, pdos, true, 3, 6);

    const Key k1{1, 0x6000, 1, true};
    const Key k2{1, 0x6000, 2, true};
    EXPECT_EQ(offsets[k1].byte, 3);
    EXPECT_EQ(offsets[k1].bit, 6);
    EXPECT_EQ(offsets[k2].byte, 4);
    EXPECT_EQ(offsets[k2].bit, 0);
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

/// @brief it should parse the PDO name and write it back to JSON.
TEST(PDOPropertiesTest, PDONameRoundTrip) {
    x::json::Parser parser(std::string(R"({
        "pdo_index": 6656,
        "index": 24576,
        "sub_index": 1,
        "bit_length": 1,
        "name": "Input",
        "pdo_name": "Channel 1",
        "data_type": "uint8"
    })"));
    const auto props = Properties::parse(parser, true);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(props.pdo_name, "Channel 1");
    EXPECT_EQ(props.to_json()["pdo_name"], "Channel 1");
}

/// @brief it should leave the PDO name empty when the JSON has none, and not write it
/// back.
TEST(PDOPropertiesTest, PDONameAbsent) {
    x::json::Parser parser(std::string(R"({
        "pdo_index": 6656,
        "index": 24576,
        "sub_index": 1,
        "bit_length": 1,
        "name": "Input",
        "data_type": "uint8"
    })"));
    const auto props = Properties::parse(parser, true);
    ASSERT_NIL(parser.error());
    EXPECT_FALSE(props.pdo_name.has_value());
    EXPECT_FALSE(props.to_json().contains("pdo_name"));
}

/// @brief it should treat an entry with index 0 as padding.
TEST(PDOPropertiesTest, IsPadding) {
    EXPECT_TRUE((Properties{.index = 0, .bit_length = 7}).is_padding());
    EXPECT_FALSE((Properties{.index = 0x6000, .sub_index = 1}).is_padding());
}

/// @brief it should give padding no offset, and shift the offsets of later entries by
/// its bit length.
TEST(ComputeOffsetsProperties, PaddingShiftsLaterEntries) {
    Offsets offsets;
    const std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 1},
        {.index = 0, .bit_length = 7},
        {.index = 0x6000, .sub_index = 17, .bit_length = 16},
    };
    compute_offsets(offsets, 1, pdos, true, 0);

    EXPECT_FALSE(offsets.contains(Key{1, 0, 0, true}));
    const auto value = offsets.at(Key{1, 0x6000, 17, true});
    EXPECT_EQ(value.byte, 1);
    EXPECT_EQ(value.bit, 0);
}

/// @brief it should count padding bits in the total bit length.
TEST(TotalBits, CountsPadding) {
    const std::vector<Properties> pdos = {
        {.index = 0x6000, .sub_index = 1, .bit_length = 1},
        {.index = 0, .bit_length = 7},
        {.index = 0x6000, .sub_index = 17, .bit_length = 16},
    };
    EXPECT_EQ(total_bits(pdos), 24);
}

/// @brief it should append each object with its name, its data type, and the name of
/// its PDO, and keep padding.
TEST(AppendTest, AppendsObjectsAndPadding) {
    std::vector<pdo::Properties> pdos;
    append(
        pdos,
        true,
        {
            .index = 0x1A00,
            .entries =
                {{.index = 0x6000,
                  .sub_index = 1,
                  .bit_length = 16,
                  .name = "Value",
                  .type = telem::DataType::EC_INTEGER16},
                 {.index = 0, .sub_index = 0, .bit_length = 8},
                 {.index = 0, .sub_index = 0, .bit_length = 0},
                 {.index = 0x6000, .sub_index = 2, .bit_length = 32}},
            .name = "Channel 1",
        }
    );
    ASSERT_EQ(pdos.size(), 3);
    EXPECT_EQ(pdos[0].name, "Value");
    EXPECT_EQ(pdos[0].data_type, x::telem::INT16_T);
    EXPECT_EQ(pdos[0].pdo_index, 0x1A00);
    EXPECT_EQ(pdos[0].pdo_name, "Channel 1");
    EXPECT_TRUE(pdos[1].is_padding());
    EXPECT_EQ(pdos[1].bit_length, 8);
    EXPECT_EQ(pdos[2].name, "Input (uint32) 0x6000:02");
    EXPECT_EQ(pdos[2].data_type, x::telem::UINT32_T);
}
}
