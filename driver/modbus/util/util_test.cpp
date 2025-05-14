// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/modbus/util/util.h"

class ModbusUtilTest : public ::testing::Test {
protected:
    template<typename T>
    void expect_format_parse_eq(
        const T &value,
        const telem::DataType &dt,
        bool swap_bytes = false,
        bool swap_words = false
    ) {
        uint16_t registers[4] = {0}; // Max 4 registers needed (64-bit types)
        ASSERT_NIL(util::format_register(value, registers, dt, swap_bytes, swap_words));
        const auto parsed_value = ASSERT_NIL_P(
            util::parse_register(registers, 0, dt, swap_bytes, swap_words)
        );
        ASSERT_EQ(telem::cast<T>(parsed_value), value);
    }
};

TEST_F(ModbusUtilTest, test16BitTypes) {
    // Test uint16
    expect_format_parse_eq<uint16_t>(12345, telem::UINT16_T);
    expect_format_parse_eq<uint16_t>(0xFFFF, telem::UINT16_T);
    expect_format_parse_eq<uint16_t>(0, telem::UINT16_T);

    // Test int16
    expect_format_parse_eq<int16_t>(-12345, telem::INT16_T);
    expect_format_parse_eq<int16_t>(12345, telem::INT16_T);
    expect_format_parse_eq<int16_t>(0, telem::INT16_T);
}

TEST_F(ModbusUtilTest, test32BitTypes) {
    expect_format_parse_eq<uint32_t>(0xFFFFFFFF, telem::UINT32_T);
    expect_format_parse_eq<uint32_t>(12345678, telem::UINT32_T);

    expect_format_parse_eq<int32_t>(-12345678, telem::INT32_T);
    expect_format_parse_eq<int32_t>(12345678, telem::INT32_T);

    expect_format_parse_eq<float>(3.14159f, telem::FLOAT32_T);
    expect_format_parse_eq<float>(-3.14159f, telem::FLOAT32_T);
    expect_format_parse_eq<float>(0.0f, telem::FLOAT32_T);
    expect_format_parse_eq<float>(100.0, telem::FLOAT32_T);
}

TEST_F(ModbusUtilTest, test64BitTypes) {
    expect_format_parse_eq<uint64_t>(0xFFFFFFFFFFFFFFFF, telem::UINT64_T);
    expect_format_parse_eq<uint64_t>(12345678901234, telem::UINT64_T);

    expect_format_parse_eq<int64_t>(-12345678901234, telem::INT64_T);
    expect_format_parse_eq<int64_t>(12345678901234, telem::INT64_T);

    expect_format_parse_eq<double>(3.14159265359, telem::FLOAT64_T);
    expect_format_parse_eq<double>(-3.14159265359, telem::FLOAT64_T);
    expect_format_parse_eq<double>(0.0, telem::FLOAT64_T);
}

TEST_F(ModbusUtilTest, test8BitTypes) {
    expect_format_parse_eq<uint8_t>(255, telem::UINT8_T);
    expect_format_parse_eq<uint8_t>(0, telem::UINT8_T);
    expect_format_parse_eq<uint8_t>(127, telem::UINT8_T);

    expect_format_parse_eq<int8_t>(-128, telem::INT8_T);
    expect_format_parse_eq<int8_t>(0, telem::INT8_T);
    expect_format_parse_eq<int8_t>(127, telem::INT8_T);
}

TEST_F(ModbusUtilTest, testByteSwapping) {
    // Test with byte swapping enabled
    expect_format_parse_eq<uint16_t>(0x1234, telem::UINT16_T, true);
    expect_format_parse_eq<uint32_t>(0x12345678, telem::UINT32_T, true);
    expect_format_parse_eq<float>(3.14159f, telem::FLOAT32_T, true);
}

TEST_F(ModbusUtilTest, testWordSwapping) {
    // Test with word swapping enabled
    expect_format_parse_eq<uint32_t>(0x12345678, telem::UINT32_T, false, true);
    expect_format_parse_eq<float>(3.14159f, telem::FLOAT32_T, false, true);
    expect_format_parse_eq<double>(3.14159265359, telem::FLOAT64_T, false, true);
}

TEST_F(ModbusUtilTest, testByteAndWordSwapping) {
    // Test with both byte and word swapping enabled
    expect_format_parse_eq<uint32_t>(0x12345678, telem::UINT32_T, true, true);
    expect_format_parse_eq<float>(3.14159f, telem::FLOAT32_T, true, true);
    expect_format_parse_eq<double>(3.14159265359, telem::FLOAT64_T, true, true);
}

TEST_F(ModbusUtilTest, testInvalidDataType) {
    uint16_t registers[4] = {0};

    // Test parsing with invalid data type
    auto [_, parse_err] = util::parse_register(
        registers,
        0,
        telem::UNKNOWN_T,
        false,
        false
    );
    ASSERT_OCCURRED_AS(parse_err, xerrors::VALIDATION);

    // Test formatting with invalid data type
    auto format_err = util::format_register(
        0,
        registers,
        telem::UNKNOWN_T,
        false,
        false
    );
    ASSERT_OCCURRED_AS(format_err, xerrors::VALIDATION);
}
