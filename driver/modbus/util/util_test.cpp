// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/modbus/util/util.h"

namespace driver::modbus::util {
class ModbusUtilTest : public ::testing::Test {
protected:
    template<typename T>
    void expect_format_parse_eq(
        const T &value,
        const x::telem::DataType &dt,
        bool swap_bytes = false,
        bool swap_words = false
    ) {
        uint16_t registers[4] = {0}; // Max 4 registers needed (64-bit types)
        ASSERT_NIL(util::format_register(value, registers, dt, swap_bytes, swap_words));
        const auto parsed_value = ASSERT_NIL_P(
            util::parse_register_value(registers, dt, swap_bytes, swap_words)
        );
        ASSERT_EQ(x::telem::cast<T>(parsed_value), value);
    }
};

/// @brief it should format and parse 16-bit register types correctly.
TEST_F(ModbusUtilTest, test16BitTypes) {
    expect_format_parse_eq<uint16_t>(12345, x::telem::UINT16_T);
    expect_format_parse_eq<uint16_t>(0xFFFF, x::telem::UINT16_T);
    expect_format_parse_eq<uint16_t>(0, x::telem::UINT16_T);

    expect_format_parse_eq<int16_t>(-12345, x::telem::INT16_T);
    expect_format_parse_eq<int16_t>(12345, x::telem::INT16_T);
    expect_format_parse_eq<int16_t>(0, x::telem::INT16_T);
}

/// @brief it should format and parse 32-bit register types correctly.
TEST_F(ModbusUtilTest, test32BitTypes) {
    expect_format_parse_eq<uint32_t>(0xFFFFFFFF, x::telem::UINT32_T);
    expect_format_parse_eq<uint32_t>(12345678, x::telem::UINT32_T);

    expect_format_parse_eq<int32_t>(-12345678, x::telem::INT32_T);
    expect_format_parse_eq<int32_t>(12345678, x::telem::INT32_T);

    expect_format_parse_eq<float>(3.14159f, x::telem::FLOAT32_T);
    expect_format_parse_eq<float>(-3.14159f, x::telem::FLOAT32_T);
    expect_format_parse_eq<float>(0.0f, x::telem::FLOAT32_T);
    expect_format_parse_eq<float>(100.0, x::telem::FLOAT32_T);
}

/// @brief it should format and parse 64-bit register types correctly.
TEST_F(ModbusUtilTest, test64BitTypes) {
    expect_format_parse_eq<uint64_t>(0xFFFFFFFFFFFFFFFF, x::telem::UINT64_T);
    expect_format_parse_eq<uint64_t>(12345678901234, x::telem::UINT64_T);

    expect_format_parse_eq<int64_t>(-12345678901234, x::telem::INT64_T);
    expect_format_parse_eq<int64_t>(12345678901234, x::telem::INT64_T);

    expect_format_parse_eq<double>(3.14159265359, x::telem::FLOAT64_T);
    expect_format_parse_eq<double>(-3.14159265359, x::telem::FLOAT64_T);
    expect_format_parse_eq<double>(0.0, x::telem::FLOAT64_T);
}

/// @brief it should format and parse 8-bit register types correctly.
TEST_F(ModbusUtilTest, test8BitTypes) {
    expect_format_parse_eq<uint8_t>(255, x::telem::UINT8_T);
    expect_format_parse_eq<uint8_t>(0, x::telem::UINT8_T);
    expect_format_parse_eq<uint8_t>(127, x::telem::UINT8_T);

    expect_format_parse_eq<int8_t>(-128, x::telem::INT8_T);
    expect_format_parse_eq<int8_t>(0, x::telem::INT8_T);
    expect_format_parse_eq<int8_t>(127, x::telem::INT8_T);
}

/// @brief it should handle byte swapping in register operations.
TEST_F(ModbusUtilTest, testByteSwapping) {
    expect_format_parse_eq<uint16_t>(0x1234, x::telem::UINT16_T, true);
    expect_format_parse_eq<uint32_t>(0x12345678, x::telem::UINT32_T, true);
    expect_format_parse_eq<float>(3.14159f, x::telem::FLOAT32_T, true);
}

/// @brief it should handle word swapping in register operations.
TEST_F(ModbusUtilTest, testWordSwapping) {
    expect_format_parse_eq<uint32_t>(0x12345678, x::telem::UINT32_T, false, true);
    expect_format_parse_eq<float>(3.14159f, x::telem::FLOAT32_T, false, true);
    expect_format_parse_eq<double>(3.14159265359, x::telem::FLOAT64_T, false, true);
}

/// @brief it should handle combined byte and word swapping.
TEST_F(ModbusUtilTest, testByteAndWordSwapping) {
    expect_format_parse_eq<uint32_t>(0x12345678, x::telem::UINT32_T, true, true);
    expect_format_parse_eq<float>(3.14159f, x::telem::FLOAT32_T, true, true);
    expect_format_parse_eq<double>(3.14159265359, x::telem::FLOAT64_T, true, true);
}

/// @brief it should reject invalid data type in register operations.
TEST_F(ModbusUtilTest, testInvalidDataType) {
    uint16_t registers[4] = {0};
    ASSERT_OCCURRED_AS_P(
        util::parse_register_value(registers, x::telem::UNKNOWN_T, false, false),
        x::errors::VALIDATION
    );

    ASSERT_OCCURRED_AS(
        util::format_register(0, registers, x::telem::UNKNOWN_T, false, false),
        x::errors::VALIDATION
    );
}
}
