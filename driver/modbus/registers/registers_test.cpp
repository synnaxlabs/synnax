// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/plan.h"
#include "driver/modbus/registers/registers.h"

namespace driver::modbus::registers {
namespace {
codec::Plan compile(synnax::library::BinaryField f) {
    f.key = x::uuid::create();
    synnax::library::MessageEntry m;
    m.payload = synnax::library::BinaryPayload{.fields = {f}};
    return ASSERT_NIL_P(codec::Plan::compile(m));
}

/// @brief encodes a value with the write layout into zeroed registers.
std::vector<std::uint16_t> encode(
    const x::telem::SampleValue &value,
    const x::telem::DataType &dt,
    const bool bytes_swapped,
    const bool words_swapped
) {
    const auto plan = compile(
        ASSERT_NIL_P(whole_field(dt, bytes_swapped, words_swapped, 0))
    );
    auto values = plan.values();
    EXPECT_FALSE(values.set(0, value));
    std::vector<std::uint8_t> bytes((plan.length() + 1) / 2 * 2);
    EXPECT_FALSE(plan.encode(values, bytes));
    std::vector<std::uint16_t> regs(bytes.size() / 2);
    to_registers(bytes, regs);
    return regs;
}

/// @brief decodes a value with the read layout from registers.
double decode(
    const std::vector<std::uint16_t> &regs,
    const x::telem::DataType &dt,
    const bool bytes_swapped,
    const bool words_swapped
) {
    const auto plan = compile(ASSERT_NIL_P(field(dt, bytes_swapped, words_swapped, 0)));
    std::vector<std::uint8_t> bytes(regs.size() * 2);
    to_bytes(regs, bytes);
    auto values = plan.values();
    EXPECT_FALSE(plan.decode(bytes, values));
    return values.get(0);
}
}

TEST(Registers, SerializesEachRegisterBigEndian) {
    const std::vector<std::uint16_t> regs = {0x1234, 0xABCD};
    std::vector<std::uint8_t> bytes(4);
    to_bytes(regs, bytes);
    EXPECT_EQ(bytes, (std::vector<std::uint8_t>{0x12, 0x34, 0xAB, 0xCD}));
    std::vector<std::uint16_t> back(2);
    to_registers(bytes, back);
    EXPECT_EQ(back, regs);
}

TEST(Registers, LaysOut16BitValues) {
    EXPECT_EQ(
        encode(uint16_t{0x1234}, x::telem::UINT16_T, false, false),
        (std::vector<std::uint16_t>{0x1234})
    );
    EXPECT_EQ(
        encode(uint16_t{0x1234}, x::telem::UINT16_T, true, false),
        (std::vector<std::uint16_t>{0x3412})
    );
    EXPECT_EQ(
        encode(int16_t{-2}, x::telem::INT16_T, false, false),
        (std::vector<std::uint16_t>{0xFFFE})
    );
    EXPECT_EQ(decode({0xFFFE}, x::telem::INT16_T, false, false), -2);
}

TEST(Registers, PutsTheLowWordFirstUnlessWordsAreSwapped) {
    const auto v = uint32_t{0x12345678};
    const auto dt = x::telem::UINT32_T;
    EXPECT_EQ(
        encode(v, dt, false, false),
        (std::vector<std::uint16_t>{0x5678, 0x1234})
    );
    EXPECT_EQ(encode(v, dt, false, true), (std::vector<std::uint16_t>{0x1234, 0x5678}));
    EXPECT_EQ(encode(v, dt, true, false), (std::vector<std::uint16_t>{0x7856, 0x3412}));
    EXPECT_EQ(encode(v, dt, true, true), (std::vector<std::uint16_t>{0x3412, 0x7856}));
    EXPECT_EQ(decode({0x5678, 0x1234}, dt, false, false), 0x12345678);
    EXPECT_EQ(decode({0x1234, 0x5678}, dt, false, true), 0x12345678);
    EXPECT_EQ(decode({0x7856, 0x3412}, dt, true, false), 0x12345678);
    EXPECT_EQ(decode({0x3412, 0x7856}, dt, true, true), 0x12345678);
}

TEST(Registers, LaysOut64BitValues) {
    const auto v = uint64_t{0x0102030405060708};
    const auto dt = x::telem::UINT64_T;
    EXPECT_EQ(
        encode(v, dt, false, false),
        (std::vector<std::uint16_t>{0x0708, 0x0506, 0x0304, 0x0102})
    );
    EXPECT_EQ(
        encode(v, dt, false, true),
        (std::vector<std::uint16_t>{0x0102, 0x0304, 0x0506, 0x0708})
    );
    EXPECT_EQ(
        encode(v, dt, true, true),
        (std::vector<std::uint16_t>{0x0201, 0x0403, 0x0605, 0x0807})
    );
}

TEST(Registers, RoundTripsEveryTypeAndSwap) {
    for (const bool bytes_swapped: {false, true})
        for (const bool words_swapped: {false, true}) {
            const auto round_trip = [&](const x::telem::SampleValue &v) {
                const auto dt = x::telem::DataType::infer(v);
                return decode(
                    encode(v, dt, bytes_swapped, words_swapped),
                    dt,
                    bytes_swapped,
                    words_swapped
                );
            };
            EXPECT_EQ(round_trip(uint8_t{255}), 255);
            EXPECT_EQ(round_trip(int8_t{-128}), -128);
            EXPECT_EQ(round_trip(uint16_t{0xFFFF}), 0xFFFF);
            EXPECT_EQ(round_trip(int16_t{-12345}), -12345);
            EXPECT_EQ(round_trip(uint32_t{0xFFFFFFFF}), 0xFFFFFFFF);
            EXPECT_EQ(round_trip(int32_t{-12345678}), -12345678);
            EXPECT_EQ(round_trip(int64_t{-12345678901234}), -12345678901234);
            EXPECT_EQ(round_trip(uint64_t{12345678901234}), 12345678901234);
            EXPECT_EQ(round_trip(3.14159f), static_cast<double>(3.14159f));
            EXPECT_EQ(round_trip(-3.14159265359), -3.14159265359);
        }
}

TEST(Registers, Reads8BitValuesFromOneByteOfTheRegister) {
    EXPECT_EQ(decode({0xAB12}, x::telem::UINT8_T, false, false), 0x12);
    EXPECT_EQ(decode({0xAB12}, x::telem::UINT8_T, true, false), 0xAB);
    EXPECT_EQ(decode({0x00FB}, x::telem::INT8_T, false, false), -5);
}

TEST(Registers, Writes8BitValuesAcrossTheWholeRegister) {
    EXPECT_EQ(
        encode(int8_t{-5}, x::telem::INT8_T, false, false),
        (std::vector<std::uint16_t>{0xFFFB})
    );
    EXPECT_EQ(
        encode(int8_t{-5}, x::telem::INT8_T, true, false),
        (std::vector<std::uint16_t>{0xFBFF})
    );
    EXPECT_EQ(
        encode(uint8_t{200}, x::telem::UINT8_T, false, false),
        (std::vector<std::uint16_t>{0x00C8})
    );
}

TEST(Registers, RejectsANonNumericDataType) {
    ASSERT_OCCURRED_AS_P(
        field(x::telem::UNKNOWN_T, false, false, 0),
        x::errors::VALIDATION
    );
    ASSERT_OCCURRED_AS_P(
        field(x::telem::STRING_T, false, false, 0),
        x::errors::VALIDATION
    );
    ASSERT_OCCURRED_AS_P(
        whole_field(x::telem::TIMESTAMP_T, false, false, 0),
        x::errors::VALIDATION
    );
}

TEST(Registers, OffsetsTheFieldByTwoBytesPerRegister) {
    const auto f = ASSERT_NIL_P(field(x::telem::UINT16_T, false, false, 3));
    const auto plan = compile(f);
    std::vector<std::uint8_t> bytes = {0, 0, 0, 0, 0, 0, 0x12, 0x34};
    auto values = plan.values();
    ASSERT_NIL(plan.decode(bytes, values));
    EXPECT_EQ(values.get(0), 0x1234);
}
}
