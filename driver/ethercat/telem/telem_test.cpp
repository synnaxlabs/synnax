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

namespace driver::ethercat::telem {
TEST(InferTypeFromBitLength, ZeroBits) {
    EXPECT_EQ(infer_type_from_bit_length(0), x::telem::UINT8_T);
}

TEST(InferTypeFromBitLength, OneBit) {
    EXPECT_EQ(infer_type_from_bit_length(1), x::telem::UINT8_T);
}

TEST(InferTypeFromBitLength, EightBits) {
    EXPECT_EQ(infer_type_from_bit_length(8), x::telem::UINT8_T);
}

TEST(InferTypeFromBitLength, SixteenBits) {
    EXPECT_EQ(infer_type_from_bit_length(16), x::telem::UINT16_T);
}

TEST(InferTypeFromBitLength, ThirtyTwoBits) {
    EXPECT_EQ(infer_type_from_bit_length(32), x::telem::UINT32_T);
}

TEST(InferTypeFromBitLength, SixtyFourBits) {
    EXPECT_EQ(infer_type_from_bit_length(64), x::telem::UINT64_T);
}

TEST(InferTypeFromBitLength, NonStandardSizes) {
    EXPECT_EQ(infer_type_from_bit_length(4), x::telem::UINT8_T);
    EXPECT_EQ(infer_type_from_bit_length(12), x::telem::UINT16_T);
    EXPECT_EQ(infer_type_from_bit_length(24), x::telem::UINT32_T);
    EXPECT_EQ(infer_type_from_bit_length(48), x::telem::UINT64_T);
}

TEST(MapEthercatToSynnax, Boolean) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_BOOLEAN, 1), x::telem::UINT8_T);
}

TEST(MapEthercatToSynnax, BitTypes) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_BIT1, 1), x::telem::UINT8_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_BIT4, 4), x::telem::UINT8_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_BIT8, 8), x::telem::UINT8_T);
}

TEST(MapEthercatToSynnax, SignedIntegers) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER8, 8), x::telem::INT8_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER16, 16), x::telem::INT16_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER32, 32), x::telem::INT32_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER64, 64), x::telem::INT64_T);
}

TEST(MapEthercatToSynnax, UnsignedIntegers) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED8, 8), x::telem::UINT8_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED16, 16), x::telem::UINT16_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED32, 32), x::telem::UINT32_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED64, 64), x::telem::UINT64_T);
}

TEST(MapEthercatToSynnax, NonStandardIntegerSizes) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER24, 24), x::telem::INT32_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED24, 24), x::telem::UINT32_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_INTEGER48, 48), x::telem::INT64_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNSIGNED48, 48), x::telem::UINT64_T);
}

TEST(MapEthercatToSynnax, FloatingPoint) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_REAL32, 32), x::telem::FLOAT32_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_REAL64, 64), x::telem::FLOAT64_T);
}

TEST(MapEthercatToSynnax, StringTypes) {
    EXPECT_EQ(
        map_ethercat_to_synnax(DataType::EC_VISIBLE_STRING, 0),
        x::telem::STRING_T
    );
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_OCTET_STRING, 0), x::telem::STRING_T);
    EXPECT_EQ(
        map_ethercat_to_synnax(DataType::EC_UNICODE_STRING, 0),
        x::telem::STRING_T
    );
}

TEST(MapEthercatToSynnax, TimeTypes) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_TIME_OF_DAY, 48), x::telem::INT64_T);
    EXPECT_EQ(
        map_ethercat_to_synnax(DataType::EC_TIME_DIFFERENCE, 48),
        x::telem::INT64_T
    );
}

TEST(MapEthercatToSynnax, UnknownFallsBackToBitLength) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNKNOWN, 8), x::telem::UINT8_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNKNOWN, 16), x::telem::UINT16_T);
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_UNKNOWN, 32), x::telem::UINT32_T);
}

TEST(MapEthercatToSynnax, DomainFallsBackToBitLength) {
    EXPECT_EQ(map_ethercat_to_synnax(DataType::EC_DOMAIN, 64), x::telem::UINT64_T);
}

TEST(GeneratePdoEntryName, UsesCoENameWhenAvailable) {
    const std::string result = generate_pdo_entry_name(
        "Status Word",
        0x6041,
        0x00,
        true,
        x::telem::UINT16_T
    );
    EXPECT_EQ(result, "Status Word");
}

TEST(GeneratePdoEntryName, GeneratesInputNameWhenCoEEmpty) {
    const std::string
        result = generate_pdo_entry_name("", 0x6000, 0x01, true, x::telem::UINT16_T);
    EXPECT_EQ(result, "Input (uint16) 0x6000:01");
}

TEST(GeneratePdoEntryName, GeneratesOutputNameWhenCoEEmpty) {
    const std::string
        result = generate_pdo_entry_name("", 0x7000, 0x02, false, x::telem::INT32_T);
    EXPECT_EQ(result, "Output (int32) 0x7000:02");
}

TEST(GeneratePdoEntryName, FormatsHighSubindex) {
    const std::string
        result = generate_pdo_entry_name("", 0x1A00, 0xFF, true, x::telem::UINT8_T);
    EXPECT_EQ(result, "Input (uint8) 0x1A00:FF");
}

TEST(FormatIndexSubindex, FormatsCorrectly) {
    EXPECT_EQ(format_index_sub_index(0x6000, 0x01), "0x6000:01");
    EXPECT_EQ(format_index_sub_index(0x1A00, 0xFF), "0x1A00:FF");
    EXPECT_EQ(format_index_sub_index(0x0000, 0x00), "0x0000:00");
    EXPECT_EQ(format_index_sub_index(0xFFFF, 0xAB), "0xFFFF:AB");
}

TEST(ReadPdoToSeries, SingleBitAtOffset0) {
    uint8_t buffer[] = {0b00000001};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 0, 1, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 1);
}

TEST(ReadPdoToSeries, SingleBitAtOffset7) {
    uint8_t buffer[] = {0b10000000};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 7, 1, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 1);
}

TEST(ReadPdoToSeries, FourBitsAtOffset0) {
    uint8_t buffer[] = {0b00001111};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 0, 4, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 0x0F);
}

TEST(ReadPdoToSeries, FourBitsAtOffset4) {
    uint8_t buffer[] = {0b11110000};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 4, 4, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 0x0F);
}

TEST(ReadPdoToSeries, SubByteSpanningByteBoundary) {
    uint8_t buffer[] = {0b11100000, 0b00000011};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 5, 6, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 0b00011111);
}

TEST(ReadPdoToSeries, ByteAligned8Bit) {
    uint8_t buffer[] = {0xAB};
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 0, 8, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 0xAB);
}

TEST(ReadPdoToSeries, ByteAligned16Bit) {
    uint8_t buffer[] = {0x34, 0x12};
    x::telem::Series series(x::telem::UINT16_T, 1);
    read_pdo_to_series(buffer, 0, 16, x::telem::UINT16_T, series);
    EXPECT_EQ(series.at<uint16_t>(0), 0x1234);
}

TEST(ReadPdoToSeries, ByteAligned32Bit) {
    uint8_t buffer[] = {0x78, 0x56, 0x34, 0x12};
    x::telem::Series series(x::telem::UINT32_T, 1);
    read_pdo_to_series(buffer, 0, 32, x::telem::UINT32_T, series);
    EXPECT_EQ(series.at<uint32_t>(0), 0x12345678);
}

TEST(ReadPdoToSeries, Unsigned24BitAligned) {
    uint8_t buffer[] = {0x56, 0x34, 0x12, 0x00};
    x::telem::Series series(x::telem::UINT32_T, 1);
    read_pdo_to_series(buffer, 0, 24, x::telem::UINT32_T, series);
    EXPECT_EQ(series.at<uint32_t>(0), 0x00123456);
}

TEST(ReadPdoToSeries, Signed24BitPositive) {
    uint8_t buffer[] = {0x56, 0x34, 0x12, 0x00};
    x::telem::Series series(x::telem::INT32_T, 1);
    read_pdo_to_series(buffer, 0, 24, x::telem::INT32_T, series);
    EXPECT_EQ(series.at<int32_t>(0), 0x00123456);
}

TEST(ReadPdoToSeries, Signed24BitNegative) {
    uint8_t buffer[] = {0xFF, 0xFF, 0xFF, 0x00};
    x::telem::Series series(x::telem::INT32_T, 1);
    read_pdo_to_series(buffer, 0, 24, x::telem::INT32_T, series);
    EXPECT_EQ(series.at<int32_t>(0), -1);
}

TEST(ReadPdoToSeries, Signed24BitNegativeValue) {
    uint8_t buffer[] = {0x00, 0x00, 0x80, 0x00};
    x::telem::Series series(x::telem::INT32_T, 1);
    read_pdo_to_series(buffer, 0, 24, x::telem::INT32_T, series);
    EXPECT_EQ(series.at<int32_t>(0), static_cast<int32_t>(0xFF800000));
}

TEST(ReadPdoToSeries, TwentyFourBitWithBitOffset) {
    uint8_t buffer[] = {0x58, 0xD1, 0x48, 0x00};
    x::telem::Series series(x::telem::UINT32_T, 1);
    read_pdo_to_series(buffer, 2, 24, x::telem::UINT32_T, series);
    EXPECT_EQ(series.at<uint32_t>(0), 0x00123456);
}

TEST(WritePdoFromValue, SingleBitAtOffset0Set) {
    uint8_t buffer[] = {0x00};
    write_pdo_from_value(buffer, 0, 1, x::telem::UINT8_T, uint8_t{1});
    EXPECT_EQ(buffer[0], 0b00000001);
}

TEST(WritePdoFromValue, SingleBitAtOffset0Clear) {
    uint8_t buffer[] = {0xFF};
    write_pdo_from_value(buffer, 0, 1, x::telem::UINT8_T, uint8_t{0});
    EXPECT_EQ(buffer[0], 0b11111110);
}

TEST(WritePdoFromValue, SingleBitAtOffset7Set) {
    uint8_t buffer[] = {0x00};
    write_pdo_from_value(buffer, 7, 1, x::telem::UINT8_T, uint8_t{1});
    EXPECT_EQ(buffer[0], 0b10000000);
}

TEST(WritePdoFromValue, FourBitsAtOffset0) {
    uint8_t buffer[] = {0xF0};
    write_pdo_from_value(buffer, 0, 4, x::telem::UINT8_T, uint8_t{0x0A});
    EXPECT_EQ(buffer[0], 0xFA);
}

TEST(WritePdoFromValue, FourBitsAtOffset4) {
    uint8_t buffer[] = {0x0F};
    write_pdo_from_value(buffer, 4, 4, x::telem::UINT8_T, uint8_t{0x0A});
    EXPECT_EQ(buffer[0], 0xAF);
}

TEST(WritePdoFromValue, SubByteSpanningByteBoundary) {
    uint8_t buffer[] = {0x00, 0x00};
    write_pdo_from_value(buffer, 5, 6, x::telem::UINT8_T, uint8_t{0b00011111});
    EXPECT_EQ(buffer[0], 0b11100000);
    EXPECT_EQ(buffer[1], 0b00000011);
}

TEST(WritePdoFromValue, SubByteSpanningPreservesOtherBits) {
    uint8_t buffer[] = {0b00011111, 0b11111100};
    write_pdo_from_value(buffer, 5, 6, x::telem::UINT8_T, uint8_t{0b00101010});
    EXPECT_EQ(buffer[0], 0b01011111);
    EXPECT_EQ(buffer[1], 0b11111101);
}

TEST(WritePdoFromValue, ByteAligned8Bit) {
    uint8_t buffer[] = {0x00};
    write_pdo_from_value(buffer, 0, 8, x::telem::UINT8_T, uint8_t{0xAB});
    EXPECT_EQ(buffer[0], 0xAB);
}

TEST(WritePdoFromValue, ByteAligned16Bit) {
    uint8_t buffer[] = {0x00, 0x00};
    write_pdo_from_value(buffer, 0, 16, x::telem::UINT16_T, uint16_t{0x1234});
    EXPECT_EQ(buffer[0], 0x34);
    EXPECT_EQ(buffer[1], 0x12);
}

TEST(WritePdoFromValue, ByteAligned32Bit) {
    uint8_t buffer[] = {0x00, 0x00, 0x00, 0x00};
    write_pdo_from_value(buffer, 0, 32, x::telem::UINT32_T, uint32_t{0x12345678});
    EXPECT_EQ(buffer[0], 0x78);
    EXPECT_EQ(buffer[1], 0x56);
    EXPECT_EQ(buffer[2], 0x34);
    EXPECT_EQ(buffer[3], 0x12);
}

TEST(WritePdoFromValue, TwentyFourBitAligned) {
    uint8_t buffer[] = {0x00, 0x00, 0x00, 0x00};
    write_pdo_from_value(buffer, 0, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(buffer[0], 0x56);
    EXPECT_EQ(buffer[1], 0x34);
    EXPECT_EQ(buffer[2], 0x12);
}

TEST(WritePdoFromValue, TwentyFourBitWithBitOffset) {
    uint8_t buffer[] = {0x00, 0x00, 0x00, 0x00};
    write_pdo_from_value(buffer, 2, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(buffer[0], 0x58);
    EXPECT_EQ(buffer[1], 0xD1);
    EXPECT_EQ(buffer[2], 0x48);
    EXPECT_EQ(buffer[3], 0x00);
}

TEST(WritePdoFromValue, TwentyFourBitWithBitOffsetPreservesOtherBits) {
    uint8_t buffer[] = {0x03, 0x00, 0x00, 0xFC};
    write_pdo_from_value(buffer, 2, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(buffer[0], 0x5B);
    EXPECT_EQ(buffer[1], 0xD1);
    EXPECT_EQ(buffer[2], 0x48);
    EXPECT_EQ(buffer[3], 0xFC);
}

TEST(PdoRequiredBytes, StandardByteLengths) {
    EXPECT_EQ(pdo_required_bytes(0, 8), 1u);
    EXPECT_EQ(pdo_required_bytes(0, 16), 2u);
    EXPECT_EQ(pdo_required_bytes(0, 32), 4u);
    EXPECT_EQ(pdo_required_bytes(0, 64), 8u);
}

TEST(PdoRequiredBytes, SubByteWithinSingleByte) {
    EXPECT_EQ(pdo_required_bytes(0, 1), 1u);
    EXPECT_EQ(pdo_required_bytes(0, 4), 1u);
    EXPECT_EQ(pdo_required_bytes(4, 4), 1u);
    EXPECT_EQ(pdo_required_bytes(7, 1), 1u);
}

TEST(PdoRequiredBytes, SubByteSpanningByteBoundary) {
    EXPECT_EQ(pdo_required_bytes(5, 4), 2u);
    EXPECT_EQ(pdo_required_bytes(7, 2), 2u);
}

TEST(PdoRequiredBytes, TwentyFourBitAligned) {
    EXPECT_EQ(pdo_required_bytes(0, 24), 3u);
}

TEST(PdoRequiredBytes, TwentyFourBitWithBitOffset) {
    EXPECT_EQ(pdo_required_bytes(1, 24), 4u);
    EXPECT_EQ(pdo_required_bytes(4, 24), 4u);
    EXPECT_EQ(pdo_required_bytes(7, 24), 4u);
}

TEST(ReadWriteRoundTrip, SingleBit) {
    uint8_t buffer[] = {0x00};
    write_pdo_from_value(buffer, 3, 1, x::telem::UINT8_T, uint8_t{1});
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 3, 1, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 1);
}

TEST(ReadWriteRoundTrip, FourBitsSpanning) {
    uint8_t buffer[] = {0xFF, 0xFF};
    write_pdo_from_value(buffer, 6, 4, x::telem::UINT8_T, uint8_t{0x09});
    x::telem::Series series(x::telem::UINT8_T, 1);
    read_pdo_to_series(buffer, 6, 4, x::telem::UINT8_T, series);
    EXPECT_EQ(series.at<uint8_t>(0), 0x09);
}

TEST(ReadWriteRoundTrip, TwentyFourBitWithOffset) {
    uint8_t buffer[] = {0x00, 0x00, 0x00, 0x00};
    write_pdo_from_value(buffer, 4, 24, x::telem::UINT32_T, uint32_t{0xABCDEF});
    x::telem::Series series(x::telem::UINT32_T, 1);
    read_pdo_to_series(buffer, 4, 24, x::telem::UINT32_T, series);
    EXPECT_EQ(series.at<uint32_t>(0), 0xABCDEF);
}

TEST(ReadWriteRoundTrip, StandardTypes) {
    uint8_t buffer[8] = {0};

    write_pdo_from_value(buffer, 0, 32, x::telem::FLOAT32_T, 3.14159f);
    x::telem::Series series(x::telem::FLOAT32_T, 1);
    read_pdo_to_series(buffer, 0, 32, x::telem::FLOAT32_T, series);
    EXPECT_FLOAT_EQ(series.at<float>(0), 3.14159f);
}
}
