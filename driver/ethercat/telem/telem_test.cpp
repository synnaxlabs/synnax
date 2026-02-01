// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/telem/telem.h"

TEST(InferTypeFromBitLength, ZeroBits) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(0), telem::UINT8_T);
}

TEST(InferTypeFromBitLength, OneBit) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(1), telem::UINT8_T);
}

TEST(InferTypeFromBitLength, EightBits) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(8), telem::UINT8_T);
}

TEST(InferTypeFromBitLength, SixteenBits) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(16), telem::UINT16_T);
}

TEST(InferTypeFromBitLength, ThirtyTwoBits) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(32), telem::UINT32_T);
}

TEST(InferTypeFromBitLength, SixtyFourBits) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(64), telem::UINT64_T);
}

TEST(InferTypeFromBitLength, NonStandardSizes) {
    EXPECT_EQ(ethercat::infer_type_from_bit_length(4), telem::UINT8_T);
    EXPECT_EQ(ethercat::infer_type_from_bit_length(12), telem::UINT16_T);
    EXPECT_EQ(ethercat::infer_type_from_bit_length(24), telem::UINT32_T);
    EXPECT_EQ(ethercat::infer_type_from_bit_length(48), telem::UINT64_T);
}

TEST(MapEthercatToSynnax, Boolean) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_BOOLEAN, 1),
        telem::UINT8_T
    );
}

TEST(MapEthercatToSynnax, BitTypes) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_BIT1, 1),
        telem::UINT8_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_BIT4, 4),
        telem::UINT8_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_BIT8, 8),
        telem::UINT8_T
    );
}

TEST(MapEthercatToSynnax, SignedIntegers) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER8, 8),
        telem::INT8_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER16, 16),
        telem::INT16_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER32, 32),
        telem::INT32_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER64, 64),
        telem::INT64_T
    );
}

TEST(MapEthercatToSynnax, UnsignedIntegers) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED8, 8),
        telem::UINT8_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED16, 16),
        telem::UINT16_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED32, 32),
        telem::UINT32_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED64, 64),
        telem::UINT64_T
    );
}

TEST(MapEthercatToSynnax, NonStandardIntegerSizes) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER24, 24),
        telem::INT32_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED24, 24),
        telem::UINT32_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_INTEGER48, 48),
        telem::INT64_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNSIGNED48, 48),
        telem::UINT64_T
    );
}

TEST(MapEthercatToSynnax, FloatingPoint) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_REAL32, 32),
        telem::FLOAT32_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_REAL64, 64),
        telem::FLOAT64_T
    );
}

TEST(MapEthercatToSynnax, StringTypes) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_VISIBLE_STRING, 0),
        telem::STRING_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_OCTET_STRING, 0),
        telem::STRING_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNICODE_STRING, 0),
        telem::STRING_T
    );
}

TEST(MapEthercatToSynnax, TimeTypes) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_TIME_OF_DAY, 48),
        telem::INT64_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_TIME_DIFFERENCE, 48),
        telem::INT64_T
    );
}

TEST(MapEthercatToSynnax, UnknownFallsBackToBitLength) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNKNOWN, 8),
        telem::UINT8_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNKNOWN, 16),
        telem::UINT16_T
    );
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_UNKNOWN, 32),
        telem::UINT32_T
    );
}

TEST(MapEthercatToSynnax, DomainFallsBackToBitLength) {
    EXPECT_EQ(
        ethercat::map_ethercat_to_synnax(ethercat::ECDataType::EC_DOMAIN, 64),
        telem::UINT64_T
    );
}

TEST(GeneratePdoEntryName, UsesCoENameWhenAvailable) {
    const std::string result = ethercat::generate_pdo_entry_name(
        "Status Word",
        0x6041,
        0x00,
        true,
        telem::UINT16_T
    );
    EXPECT_EQ(result, "Status Word");
}

TEST(GeneratePdoEntryName, GeneratesInputNameWhenCoEEmpty) {
    const std::string result = ethercat::generate_pdo_entry_name(
        "",
        0x6000,
        0x01,
        true,
        telem::UINT16_T
    );
    EXPECT_EQ(result, "Input (uint16) 0x6000:01");
}

TEST(GeneratePdoEntryName, GeneratesOutputNameWhenCoEEmpty) {
    const std::string result = ethercat::generate_pdo_entry_name(
        "",
        0x7000,
        0x02,
        false,
        telem::INT32_T
    );
    EXPECT_EQ(result, "Output (int32) 0x7000:02");
}

TEST(GeneratePdoEntryName, FormatsHighSubindex) {
    const std::string result = ethercat::generate_pdo_entry_name(
        "",
        0x1A00,
        0xFF,
        true,
        telem::UINT8_T
    );
    EXPECT_EQ(result, "Input (uint8) 0x1A00:FF");
}

TEST(FormatIndexSubindex, FormatsCorrectly) {
    EXPECT_EQ(ethercat::format_index_subindex(0x6000, 0x01), "0x6000:01");
    EXPECT_EQ(ethercat::format_index_subindex(0x1A00, 0xFF), "0x1A00:FF");
    EXPECT_EQ(ethercat::format_index_subindex(0x0000, 0x00), "0x0000:00");
    EXPECT_EQ(ethercat::format_index_subindex(0xFFFF, 0xAB), "0xFFFF:AB");
}
