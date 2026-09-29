// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <span>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/bench/bench.h"
#include "x/cpp/test/test.h"

#include "driver/codec/errors.h"
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

namespace {
pdo::Entry entry(const uint8_t bit_length, const x::telem::DataType &dt) {
    return pdo::Entry(0, 0x6000, 1, bit_length, true, dt);
}

/// @brief decodes an entry from the buffer into a one-sample series of its type.
x::telem::Series decode(
    const std::vector<uint8_t> &buffer,
    const uint8_t bit,
    const uint8_t bit_length,
    const x::telem::DataType &dt
) {
    const auto p = ASSERT_NIL_P(plan(entry(bit_length, dt), bit));
    auto values = p.values();
    EXPECT_FALSE(p.decode(buffer, values));
    x::telem::Series series(dt, 1);
    values.write(0, series);
    return series;
}

/// @brief encodes a value into the buffer in place, as an RxPDO write does.
void encode(
    std::vector<uint8_t> &buffer,
    const uint8_t bit,
    const uint8_t bit_length,
    const x::telem::DataType &dt,
    const x::telem::SampleValue &value
) {
    const auto p = ASSERT_NIL_P(plan(entry(bit_length, dt), bit));
    auto values = p.values();
    ASSERT_NIL(values.set(0, dt.cast(value)));
    ASSERT_NIL(p.encode(values, std::span(buffer)));
}
}

TEST(Plan, ReadsSubByteValuesAtAnyBitOffset) {
    EXPECT_EQ(decode({0b00000001}, 0, 1, x::telem::UINT8_T).at<uint8_t>(0), 1);
    EXPECT_EQ(decode({0b10000000}, 7, 1, x::telem::UINT8_T).at<uint8_t>(0), 1);
    EXPECT_EQ(decode({0b00001111}, 0, 4, x::telem::UINT8_T).at<uint8_t>(0), 0x0F);
    EXPECT_EQ(decode({0b11110000}, 4, 4, x::telem::UINT8_T).at<uint8_t>(0), 0x0F);
    EXPECT_EQ(
        decode({0b11100000, 0b00000011}, 5, 6, x::telem::UINT8_T).at<uint8_t>(0),
        0b00011111
    );
}

TEST(Plan, SignExtendsSignedSubByteValues) {
    EXPECT_EQ(decode({0b11110000}, 4, 4, x::telem::INT8_T).at<int8_t>(0), -1);
    EXPECT_EQ(decode({0b01110000}, 4, 4, x::telem::INT8_T).at<int8_t>(0), 7);
}

TEST(Plan, ReadsByteAlignedValuesLittleEndian) {
    EXPECT_EQ(decode({0xAB}, 0, 8, x::telem::UINT8_T).at<uint8_t>(0), 0xAB);
    EXPECT_EQ(decode({0x34, 0x12}, 0, 16, x::telem::UINT16_T).at<uint16_t>(0), 0x1234);
    EXPECT_EQ(
        decode({0x78, 0x56, 0x34, 0x12}, 0, 32, x::telem::UINT32_T).at<uint32_t>(0),
        0x12345678
    );
}

TEST(Plan, Reads24BitValues) {
    EXPECT_EQ(
        decode({0x56, 0x34, 0x12, 0x00}, 0, 24, x::telem::UINT32_T).at<uint32_t>(0),
        0x00123456
    );
    EXPECT_EQ(
        decode({0x56, 0x34, 0x12, 0x00}, 0, 24, x::telem::INT32_T).at<int32_t>(0),
        0x00123456
    );
    EXPECT_EQ(
        decode({0xFF, 0xFF, 0xFF, 0x00}, 0, 24, x::telem::INT32_T).at<int32_t>(0),
        -1
    );
    EXPECT_EQ(
        decode({0x00, 0x00, 0x80, 0x00}, 0, 24, x::telem::INT32_T).at<int32_t>(0),
        static_cast<int32_t>(0xFF800000)
    );
    EXPECT_EQ(
        decode({0x58, 0xD1, 0x48, 0x00}, 2, 24, x::telem::UINT32_T).at<uint32_t>(0),
        0x00123456
    );
}

TEST(Plan, ReadsAByteLengthValueAtABitOffset) {
    EXPECT_EQ(
        decode({0x40, 0x23, 0x01}, 4, 16, x::telem::UINT16_T).at<uint16_t>(0),
        0x1234
    );
}

TEST(Plan, WritesSubByteValuesPreservingOtherBits) {
    std::vector<uint8_t> b = {0x00};
    encode(b, 0, 1, x::telem::UINT8_T, uint8_t{1});
    EXPECT_EQ(b[0], 0b00000001);
    b = {0xFF};
    encode(b, 0, 1, x::telem::UINT8_T, uint8_t{0});
    EXPECT_EQ(b[0], 0b11111110);
    b = {0x00};
    encode(b, 7, 1, x::telem::UINT8_T, uint8_t{1});
    EXPECT_EQ(b[0], 0b10000000);
    b = {0xF0};
    encode(b, 0, 4, x::telem::UINT8_T, uint8_t{0x0A});
    EXPECT_EQ(b[0], 0xFA);
    b = {0x0F};
    encode(b, 4, 4, x::telem::UINT8_T, uint8_t{0x0A});
    EXPECT_EQ(b[0], 0xAF);
    b = {0x00, 0x00};
    encode(b, 5, 6, x::telem::UINT8_T, uint8_t{0b00011111});
    EXPECT_EQ(b, (std::vector<uint8_t>{0b11100000, 0b00000011}));
    b = {0b00011111, 0b11111100};
    encode(b, 5, 6, x::telem::UINT8_T, uint8_t{0b00101010});
    EXPECT_EQ(b, (std::vector<uint8_t>{0b01011111, 0b11111101}));
}

TEST(Plan, WritesByteAlignedValuesLittleEndian) {
    std::vector<uint8_t> b(1);
    encode(b, 0, 8, x::telem::UINT8_T, uint8_t{0xAB});
    EXPECT_EQ(b[0], 0xAB);
    b.assign(2, 0);
    encode(b, 0, 16, x::telem::UINT16_T, uint16_t{0x1234});
    EXPECT_EQ(b, (std::vector<uint8_t>{0x34, 0x12}));
    b.assign(4, 0);
    encode(b, 0, 32, x::telem::UINT32_T, uint32_t{0x12345678});
    EXPECT_EQ(b, (std::vector<uint8_t>{0x78, 0x56, 0x34, 0x12}));
}

TEST(Plan, Writes24BitValuesPreservingOtherBits) {
    std::vector<uint8_t> b(4);
    encode(b, 0, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(b, (std::vector<uint8_t>{0x56, 0x34, 0x12, 0x00}));
    b.assign(4, 0);
    encode(b, 2, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(b, (std::vector<uint8_t>{0x58, 0xD1, 0x48, 0x00}));
    b = {0x03, 0x00, 0x00, 0xFC};
    encode(b, 2, 24, x::telem::UINT32_T, uint32_t{0x123456});
    EXPECT_EQ(b, (std::vector<uint8_t>{0x5B, 0xD1, 0x48, 0xFC}));
}

TEST(Plan, RoundTripsValues) {
    std::vector<uint8_t> b = {0x00};
    encode(b, 3, 1, x::telem::UINT8_T, uint8_t{1});
    EXPECT_EQ(decode(b, 3, 1, x::telem::UINT8_T).at<uint8_t>(0), 1);
    b = {0xFF, 0xFF};
    encode(b, 6, 4, x::telem::UINT8_T, uint8_t{0x09});
    EXPECT_EQ(decode(b, 6, 4, x::telem::UINT8_T).at<uint8_t>(0), 0x09);
    b.assign(4, 0);
    encode(b, 4, 24, x::telem::UINT32_T, uint32_t{0xABCDEF});
    EXPECT_EQ(decode(b, 4, 24, x::telem::UINT32_T).at<uint32_t>(0), 0xABCDEF);
    b.assign(4, 0);
    encode(b, 0, 32, x::telem::FLOAT32_T, 3.14159f);
    EXPECT_FLOAT_EQ(decode(b, 0, 32, x::telem::FLOAT32_T).at<float>(0), 3.14159f);
}

TEST(Plan, SpansTheBytesTheEntryTouches) {
    const auto length = [](const uint8_t bit, const uint8_t bit_length) {
        return ASSERT_NIL_P(plan(entry(bit_length, x::telem::UINT64_T), bit)).length();
    };
    EXPECT_EQ(length(0, 1), 1);
    EXPECT_EQ(length(7, 1), 1);
    EXPECT_EQ(length(4, 4), 1);
    EXPECT_EQ(length(5, 4), 2);
    EXPECT_EQ(length(0, 16), 2);
    EXPECT_EQ(length(0, 24), 3);
    EXPECT_EQ(length(1, 24), 4);
    EXPECT_EQ(length(7, 24), 4);
    EXPECT_EQ(length(0, 64), 8);
}

TEST(Plan, RejectsEntriesTheCodecCannotLayOut) {
    ASSERT_OCCURRED_AS_P(plan(entry(0, x::telem::UINT8_T), 0), codec::LAYOUT_ERROR);
    ASSERT_OCCURRED_AS_P(plan(entry(72, x::telem::STRING_T), 0), codec::LAYOUT_ERROR);
    ASSERT_OCCURRED_AS_P(plan(entry(16, x::telem::FLOAT32_T), 0), codec::LAYOUT_ERROR);
}

TEST(Plan, DecodesAndEncodesACycleWithoutAllocating) {
    const std::vector<std::pair<pdo::Entry, pdo::Offset>> pdos = {
        {entry(1, x::telem::UINT8_T), {0, 3}},
        {entry(4, x::telem::INT8_T), {0, 4}},
        {entry(16, x::telem::UINT16_T), {1, 0}},
        {entry(24, x::telem::INT32_T), {3, 2}},
        {entry(32, x::telem::FLOAT32_T), {7, 0}},
        {entry(64, x::telem::INT64_T), {11, 0}},
    };
    std::vector<codec::Plan> plans;
    std::vector<x::telem::Series> series;
    for (const auto &[e, offset]: pdos) {
        plans.push_back(ASSERT_NIL_P(plan(e, offset.bit)));
        series.emplace_back(e.data_type, 1);
    }
    std::vector<uint8_t> image(19, 0xA5);
    auto value = plans.front().values();
    size_t failures = 0;
    const auto before = x::bench::alloc_count.load();
    for (int cycle = 0; cycle < 100; cycle++)
        for (size_t i = 0; i < plans.size(); i++) {
            const auto window = std::span(image).subspan(pdos[i].second.byte);
            series[i].clear();
            if (plans[i].decode(window, value)) failures++;
            value.write(0, series[i]);
            if (plans[i].encode(value, window)) failures++;
        }
    EXPECT_EQ(x::bench::alloc_count.load() - before, 0);
    EXPECT_EQ(failures, 0);
}
}
