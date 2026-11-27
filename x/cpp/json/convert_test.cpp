// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstdint>
#include <limits>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

using nlohmann::json;

TEST(ParseTimeFormat, ISO8601) {
    const auto fmt = ASSERT_NIL_P(x::json::parse_time_format("iso8601"));
    ASSERT_EQ(fmt, x::json::TimeFormat::ISO8601);
}

TEST(ParseTimeFormat, UnixSecond) {
    const auto fmt = ASSERT_NIL_P(x::json::parse_time_format("unix_sec"));
    ASSERT_EQ(fmt, x::json::TimeFormat::UnixSecond);
}

TEST(ParseTimeFormat, UnixMillisecond) {
    const auto fmt = ASSERT_NIL_P(x::json::parse_time_format("unix_ms"));
    ASSERT_EQ(fmt, x::json::TimeFormat::UnixMillisecond);
}

TEST(ParseTimeFormat, UnixMicrosecond) {
    const auto fmt = ASSERT_NIL_P(x::json::parse_time_format("unix_us"));
    ASSERT_EQ(fmt, x::json::TimeFormat::UnixMicrosecond);
}

TEST(ParseTimeFormat, UnixNanosecond) {
    const auto fmt = ASSERT_NIL_P(x::json::parse_time_format("unix_ns"));
    ASSERT_EQ(fmt, x::json::TimeFormat::UnixNanosecond);
}

TEST(ParseTimeFormat, InvalidFormat) {
    ASSERT_OCCURRED_AS_P(x::json::parse_time_format("invalid"), x::json::BASE_ERROR);
}

TEST(ParseTimeFormat, EmptyString) {
    ASSERT_OCCURRED_AS_P(x::json::parse_time_format(""), x::json::BASE_ERROR);
}

// --- ToSampleValue (Number) ---

TEST(ToSampleValue, NumberToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::FLOAT64_T)
    );
    ASSERT_EQ(std::get<double>(sv), 42.5);
}

TEST(ToSampleValue, NumberToFloat32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::FLOAT32_T)
    );
    ASSERT_EQ(std::get<float>(sv), 42.5);
}

TEST(ToSampleValue, NumberToInt64) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::INT64_T));
    ASSERT_EQ(std::get<int64_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt32) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::INT32_T));
    ASSERT_EQ(std::get<int32_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt16) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::INT16_T));
    ASSERT_EQ(std::get<int16_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt8) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::INT8_T));
    ASSERT_EQ(std::get<int8_t>(sv), 7);
}

TEST(ToSampleValue, NumberTo64) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::UINT64_T));
    ASSERT_EQ(std::get<uint64_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint32) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::UINT32_T));
    ASSERT_EQ(std::get<uint32_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint16) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::UINT16_T));
    ASSERT_EQ(std::get<uint16_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint8) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::UINT8_T));
    ASSERT_EQ(std::get<uint8_t>(sv), 7);
}

TEST(ToSampleValue, NumberNegativeZeroToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(-0.0), x::telem::FLOAT64_T)
    );
    const auto value = std::get<double>(sv);
    ASSERT_EQ(value, -0.0);
    ASSERT_EQ(std::signbit(value), true);
}

TEST(ToSampleValue, NumberToInt64TruncationError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(3.7), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToInt64Min) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(INT64_MIN), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), INT64_MIN);
}

TEST(ToSampleValue, NumberToInt64Max) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(INT64_MAX), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), INT64_MAX);
}

TEST(ToSampleValue, NumberToUint64Max) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(UINT64_MAX), x::telem::UINT64_T)
    );
    ASSERT_EQ(std::get<uint64_t>(sv), UINT64_MAX);
}

TEST(ToSampleValue, NumberWholeFloatToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(35.0), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 35);
}

TEST(ToSampleValue, NumberFloatAtInt64LimitError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(9223372036854775808.0), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberFloatAtUint64LimitError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(18446744073709551616.0), x::telem::UINT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToFloat32OverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(1e39), x::telem::FLOAT32_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToFloat32OverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("-1e39"), x::telem::FLOAT32_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToUint8Overflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(300), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToUint8Underflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(-1), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToStringDecimal) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "42.5");
}

TEST(ToSampleValue, NumberToStringInteger) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(7), x::telem::STRING_T));
    ASSERT_EQ(std::get<std::string>(sv), "7");
}

TEST(ToSampleValue, NumberToStringKeepTrailingZeroes) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.0), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "42.0");
}

TEST(ToSampleValue, NumberToTimestampNanosecond) {
    const int64_t value = 1000000000000000000;
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(value),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(value));
}

TEST(ToSampleValue, NumberToTimestampNanosecondKeepsEveryNanosecond) {
    const int64_t value = 1700000000123456789;
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(value),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(value));
}

TEST(ToSampleValue, NumberToTimestampMicrosecondKeepsEveryMicrosecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(int64_t{1700000000123457}),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMicrosecond
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(int64_t{1700000000123457000})
    );
}

TEST(ToSampleValue, StringToTimestampNanosecondKeepsEveryNanosecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1700000000123456789"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(int64_t{1700000000123456789})
    );
}

TEST(ToSampleValue, NumberToTimestampOverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(int64_t{10000000000000}),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberDecimalToTimestampOverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(1e13),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToTimestampMicrosecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(int64_t(1000000)),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMicrosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1000000000));
}

TEST(ToSampleValue, NumberToTimestampMillisecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(int64_t(1500)),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMillisecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, NumberToTimestampSecondInteger) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(int64_t(1000000000)),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, NumberToTimestampSecondDecimal) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(1.5),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, NumberToTimestampMillisecondDecimal) {
    // 1500.5 ms = 1500500000 ns
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(1500.5),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMillisecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500500000));
}

TEST(ToSampleValue, NumberToTimestampMicrosecondDecimal) {
    // 1000000.5 us = 1000000500 ns
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(1000000.5),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMicrosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1000000500));
}

TEST(ToSampleValue, NumberToTimestampISO8601Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(42),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(42), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NumberToByteArray) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(42), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, NumberToJSON) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(42), x::telem::JSON_T));
    ASSERT_EQ(std::get<std::string>(sv), "42");
}

// --- ToSampleValue (String) ---

TEST(ToSampleValue, StringToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("hello"), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "hello");
}

/// @brief it should convert a string to its quoted JSON text on a JSON channel.
TEST(ToSampleValue, StringToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("hello"), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "\"hello\"");
}

/// @brief it should convert UUID text to the 16 bytes of the UUID.
TEST(ToSampleValue, StringToUUID) {
    const std::string text = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(text), x::telem::UUID_T)
    );
    const auto uuid = ASSERT_NIL_P(x::uuid::UUID::parse(text));
    ASSERT_EQ(
        std::get<std::string>(sv),
        std::string(reinterpret_cast<const char *>(uuid.data()), uuid.size())
    );
}

/// @brief it should reject a string that is not a UUID.
TEST(ToSampleValue, InvalidStringToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("not-a-uuid"), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should decode base64 text to the bytes that it holds.
TEST(ToSampleValue, StringToBytes) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("AP8BAg=="), x::telem::BYTES_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), std::string("\x00\xff\x01\x02", 4));
}

/// @brief it should decode empty base64 text to no bytes.
TEST(ToSampleValue, EmptyStringToBytes) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(json(""), x::telem::BYTES_T));
    ASSERT_EQ(std::get<std::string>(sv), "");
}

/// @brief it should reject text that is not canonical, padded base64.
TEST(ToSampleValue, InvalidStringToBytes) {
    for (const auto *text: {"AP8BAg", "AP8B*g==", "AP8BAh==", "AP8 BAg=="}) {
        SCOPED_TRACE(text);
        ASSERT_OCCURRED_AS_P(
            x::json::to_sample_value(json(text), x::telem::BYTES_T),
            x::json::CONVERSION_ERROR
        );
    }
}

// --- String → Numeric ---

TEST(ToSampleValue, StringToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("34.5"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 34.5);
}

TEST(ToSampleValue, StringToFloat32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("34.5"), x::telem::FLOAT32_T)
    );
    ASSERT_FLOAT_EQ(std::get<float>(sv), 34.5f);
}

TEST(ToSampleValue, StringToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("7"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 7);
}

TEST(ToSampleValue, StringToUint8) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("200"), x::telem::UINT8_T)
    );
    ASSERT_EQ(std::get<uint8_t>(sv), 200);
}

TEST(ToSampleValue, StringToInt64TruncationError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("3.7"), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToUint8Overflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("300"), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringNonNumericToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("hello"), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringPartialNumericToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("34.5abc"), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringEmptyToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(""), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringEmptyToInt64Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(""), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

// --- String exponent notation ---

TEST(ToSampleValue, StringExponentToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("1e5"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 100000.0);
}

TEST(ToSampleValue, StringExponentUpperToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("1E5"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 100000.0);
}

TEST(ToSampleValue, StringDecimalExponentToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("1.5e2"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 150.0);
}

TEST(ToSampleValue, StringNegativeExponentToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("1.5e-2"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 0.015);
}

TEST(ToSampleValue, StringExponentToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("1e5"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 100000);
}

TEST(ToSampleValue, StringExponentToUint32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("2.5e3"), x::telem::UINT32_T)
    );
    ASSERT_EQ(std::get<uint32_t>(sv), 2500);
}

TEST(ToSampleValue, StringExponentToInt64TruncationError) {
    // 1.5e0 = 1.5 (fractional, causes truncation error)
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("1.5e0"), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

// --- String large integer precision ---

TEST(ToSampleValue, StringLargeInt64PreservesPrecision) {
    // 2^53 + 1: would lose precision through double
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("9007199254740993"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 9007199254740993LL);
}

TEST(ToSampleValue, StringLargeNegativeInt64PreservesPrecision) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("-9007199254740993"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), -9007199254740993LL);
}

TEST(ToSampleValue, StringInt64Max) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("9223372036854775807"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), INT64_MAX);
}

TEST(ToSampleValue, StringInt64Min) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("-9223372036854775808"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), INT64_MIN);
}

TEST(ToSampleValue, StringInt64OverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("9223372036854775808"), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringLargeUint64PreservesPrecision) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("18446744073709551615"), x::telem::UINT64_T)
    );
    ASSERT_EQ(std::get<uint64_t>(sv), UINT64_MAX);
}

TEST(ToSampleValue, StringUint64OverflowError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("18446744073709551616"), x::telem::UINT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringNegativeToUint8Overflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("-1"), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringDecimalToInt64TruncationError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("3.7"), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringWholeDecimalToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("35.0"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 35);
}

TEST(ToSampleValue, StringLeadingPlusToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("+42"), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 42);
}

TEST(ToSampleValue, StringZeroToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("0"), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 0.0);
}

TEST(ToSampleValue, StringNegativeZeroToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("-0"), x::telem::FLOAT64_T)
    );
    const auto value = std::get<double>(sv);
    ASSERT_EQ(std::signbit(value), true);
    ASSERT_DOUBLE_EQ(value, -0.0);
}

// --- ToSampleValue (Boolean) ---

TEST(ToSampleValue, BooleanTrueToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(true), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 1);
}

TEST(ToSampleValue, BooleanFalseToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(false), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 0);
}

TEST(ToSampleValue, BooleanTrueToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(true), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 1.0);
}

TEST(ToSampleValue, BooleanFalseToUint8) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(false), x::telem::UINT8_T)
    );
    ASSERT_EQ(std::get<uint8_t>(sv), 0);
}

TEST(ToSampleValue, BooleanTrueToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(true), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "true");
}

TEST(ToSampleValue, BooleanFalseToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(false), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "false");
}

TEST(ToSampleValue, BooleanTrueToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(true), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, BooleanFalseToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(false), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, BooleanTrueToBytes) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(true), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, BooleanFalseToBytes) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(false), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, BooleanTrueToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(true), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "true");
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, BooleanFalseToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(false), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "false");
}

TEST(ToSampleValue, BooleanTrueToTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(true), x::telem::TIMESTAMP_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, BooleanFalseToTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(false), x::telem::TIMESTAMP_T),
        x::json::CONVERSION_ERROR
    );
}

// --- ToSampleValue (Null) ---

TEST(ToSampleValue, NullToFloat64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NullToInt64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NullToUint8) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NullToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(nullptr), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "null");
}

TEST(ToSampleValue, NullToBytes) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, NullToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(nullptr), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "null");
}

TEST(ToSampleValue, NullToTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::TIMESTAMP_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, NullToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(nullptr), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

/// --- ToSampleValue (Object) ---

TEST(ToSampleValue, ObjectToFloat64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ObjectToInt64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ObjectToUint8) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ObjectToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(json::object()), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "{}");
}

TEST(ToSampleValue, ObjectToStringWithKeys) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json(json::object({{"key", "value"}})),
            x::telem::STRING_T
        )
    );
    ASSERT_EQ(std::get<std::string>(sv), "{\"key\":\"value\"}");
}

TEST(ToSampleValue, ObjectToBytes) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, ObjectToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(json::object()), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "{}");
}

TEST(ToSampleValue, ObjectToTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::TIMESTAMP_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ObjectToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::object()), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

/// --- ToSampleValue (Array) ---

TEST(ToSampleValue, ArrayToFloat64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::FLOAT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ArrayToInt64) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::INT64_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ArrayToUint8) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::UINT8_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ArrayToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(json::array()), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "[]");
}

TEST(ToSampleValue, ArrayToStringWithElements) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(json::array({1, 2, 3})), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "[1,2,3]");
}

TEST(ToSampleValue, ArrayToBytes) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::BYTES_T),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should convert the value to its JSON text on a JSON channel.
TEST(ToSampleValue, ArrayToJSON) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(json::array()), x::telem::JSON_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "[]");
}

TEST(ToSampleValue, ArrayToTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::TIMESTAMP_T),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, ArrayToUUID) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(json::array()), x::telem::UUID_T),
        x::json::CONVERSION_ERROR
    );
}

// --- Strings → Timestamps ---

TEST(ToSampleValue, StringToTimestampUnixNanosecond) {
    const int64_t value = 1000000000000000000;
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1000000000000000000"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(value));
}

TEST(ToSampleValue, StringToTimestampUnixSecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1000000000"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampUnixMillisecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1500"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMillisecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, StringToTimestampUnixMicrosecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1000000"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixMicrosecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1000000000));
}

TEST(ToSampleValue, StringToTimestampUnixSecondDecimal) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1.5"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixSecond
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, StringToTimestampNonNumericError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("not-a-number"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringISO8601WithUnixFormatError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::UnixNanosecond
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40.5Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000500000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601WithOffset) {
    // 02:46:40.5+01:00 = 01:46:40.5 UTC
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T02:46:40.5+01:00"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000500000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601WithoutSubSecond) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

// --- ISO8601 parsing edge cases ---

TEST(ToSampleValue, StringToTimestampISO8601Epoch) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1970-01-01T00:00:00Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(0));
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondThreeDigits) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40.123Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondSixDigits) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40.123456Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondNineDigits) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40.123456789Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456789)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601ExcessDigitsTruncated) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40.1234567891111Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456789)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NegativeOffset) {
    // 00:46:40-01:00 = 01:46:40 UTC
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T00:46:40-01:00"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NegativeOffsetCrossesMidnight) {
    // 23:46:40-05:00 on Sep 8 = 04:46:40 UTC on Sep 9
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-08T23:46:40-05:00"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    const int64_t expected = 1000000000000000000 + int64_t(3) * 3600 * 1000000000;
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(expected));
}

TEST(ToSampleValue, StringToTimestampISO8601ExplicitPlusZero) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40+00:00"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MinusZero) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40-00:00"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LowercaseZ) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LowercaseT) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09t01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SpaceSeparator) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09 01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LeapYearFeb29) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2000-02-29T00:00:00Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(int64_t(11016) * 86400 * 1000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601PreEpoch) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1969-12-31T23:59:59Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(-1000000000));
}

TEST(ToSampleValue, StringToTimestampISO8601PreEpochWithFraction) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1969-12-31T23:59:59.5Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(-500000000));
}

TEST(ToSampleValue, StringToTimestampISO8601HalfHourOffset) {
    // India: +05:30
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2001-09-09T07:16:40+05:30"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LeapSecondRejected) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T23:59:60Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

// --- Invalid ISO8601 inputs ---

TEST(ToSampleValue, StringToTimestampISO8601EmptyString) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(""),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601TooShort) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:4"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MissingTimezone) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601BadSeparator) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09X01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NonDigitYear) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("20X1-09-09T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MonthZero) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-00-09T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MonthThirteen) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-13-09T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should reject an ISO 8601 day past the end of its month.
TEST(ToSampleValue, StringToTimestampISO8601DayPastEndOfMonth) {
    for (const auto *input:
         {"2023-02-29T00:00:00Z",
          "2024-02-30T00:00:00Z",
          "2100-02-29T00:00:00Z",
          "2001-04-31T00:00:00Z",
          "2001-11-31T00:00:00Z",
          "2001-01-32T00:00:00Z",
          "2001-12-99T00:00:00Z"})
        ASSERT_OCCURRED_AS_P(
            x::json::to_sample_value(
                json(input),
                x::telem::TIMESTAMP_T,
                x::json::TimeFormat::ISO8601
            ),
            x::json::CONVERSION_ERROR
        );
}

/// @brief it should accept an ISO 8601 date on the last day of its month.
TEST(ToSampleValue, StringToTimestampISO8601LastDayOfMonth) {
    for (const auto *input:
         {"2024-02-29T00:00:00Z",
          "2000-02-29T00:00:00Z",
          "2001-04-30T00:00:00Z",
          "2001-12-31T00:00:00Z"})
        ASSERT_NIL_P(
            x::json::to_sample_value(
                json(input),
                x::telem::TIMESTAMP_T,
                x::json::TimeFormat::ISO8601
            )
        );
}

TEST(ToSampleValue, StringToTimestampISO8601DayZero) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-00T01:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601HourTwentyFour) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T24:46:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MinuteSixty) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:60:40Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SecondSixty) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:60Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601InvalidTimezoneChar) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40X"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601OffsetMissingColon) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40+0100"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601OffsetTruncated) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40+01"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601JustDate) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should reject an ISO 8601 timestamp after the maximum timestamp.
TEST(ToSampleValue, StringToTimestampISO8601AfterMaxTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2300-01-01T00:00:00Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should reject an ISO 8601 timestamp before the minimum timestamp.
TEST(ToSampleValue, StringToTimestampISO8601BeforeMinTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("1600-01-01T00:00:00Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should parse the maximum ISO 8601 timestamp.
TEST(ToSampleValue, StringToTimestampISO8601MaxTimestamp) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("2262-04-11T23:47:16.854775807Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(std::numeric_limits<int64_t>::max())
    );
}

/// @brief it should parse the minimum ISO 8601 timestamp.
TEST(ToSampleValue, StringToTimestampISO8601MinTimestamp) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(
            json("1677-09-21T00:12:43.145224192Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        )
    );
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(std::numeric_limits<int64_t>::min())
    );
}

/// @brief it should reject an ISO 8601 timestamp one nanosecond after the maximum.
TEST(ToSampleValue, StringToTimestampISO8601JustAfterMaxTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2262-04-11T23:47:16.854775808Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should reject an ISO 8601 timestamp one nanosecond before the minimum.
TEST(ToSampleValue, StringToTimestampISO8601JustBeforeMinTimestamp) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("1677-09-21T00:12:43.145224191Z"),
            x::telem::TIMESTAMP_T,
            x::json::TimeFormat::ISO8601
        ),
        x::json::CONVERSION_ERROR
    );
}

// ==================== EnumMap ====================

TEST(ToSampleValueEnum, StringToFloat64ViaEnum) {
    x::json::EnumMap enums = {{"ON", 1.0}, {"OFF", 0.0}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("ON"), x::telem::FLOAT64_T, {}, &enums)
    );
    ASSERT_EQ(std::get<double>(sv), 1.0);
}

TEST(ToSampleValueEnum, StringToUint8ViaEnum) {
    x::json::EnumMap enums = {{"ON", 1.0}, {"OFF", 0.0}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("OFF"), x::telem::UINT8_T, {}, &enums)
    );
    ASSERT_EQ(std::get<uint8_t>(sv), 0);
}

TEST(ToSampleValueEnum, StringToInt32ViaEnum) {
    x::json::EnumMap enums = {{"AUTO", 0}, {"MANUAL", 1}, {"STANDBY", 2}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("STANDBY"), x::telem::INT32_T, {}, &enums)
    );
    ASSERT_EQ(std::get<int32_t>(sv), 2);
}

/// @brief it should keep an int64 enum value past 2^53 exact.
TEST(ToSampleValueEnum, Int64Past2To53ViaEnum) {
    constexpr int64_t past = (int64_t(1) << 53) + 1;
    x::json::EnumMap enums = {{"BIG", past}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("BIG"), x::telem::INT64_T, {}, &enums)
    );
    ASSERT_EQ(std::get<int64_t>(sv), past);
}

/// @brief it should keep the largest uint64 enum value exact.
TEST(ToSampleValueEnum, Uint64MaxViaEnum) {
    constexpr auto max = std::numeric_limits<uint64_t>::max();
    x::json::EnumMap enums = {{"MAX", max}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("MAX"), x::telem::UINT64_T, {}, &enums)
    );
    ASSERT_EQ(std::get<uint64_t>(sv), max);
}

/// @brief it should read a label as NaN for a NaN enum value.
TEST(ToSampleValueEnum, NaNViaEnum) {
    x::json::EnumMap enums = {{"FAULT", std::numeric_limits<double>::quiet_NaN()}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("FAULT"), x::telem::FLOAT32_T, {}, &enums)
    );
    ASSERT_TRUE(std::isnan(std::get<float>(sv)));
}

/// @brief it should read a label as its string enum value.
TEST(ToSampleValueEnum, StringViaEnum) {
    x::json::EnumMap enums = {{"ON", std::string("1")}};
    const auto on = ASSERT_NIL_P(
        x::json::to_sample_value(json("ON"), x::telem::STRING_T, {}, &enums)
    );
    ASSERT_EQ(std::get<std::string>(on), "1");
    const auto off = ASSERT_NIL_P(
        x::json::to_sample_value(json("OFF"), x::telem::STRING_T, {}, &enums)
    );
    ASSERT_EQ(std::get<std::string>(off), "OFF");
}

/// @brief it should reject an enum value that does not fit the target.
TEST(ToSampleValueEnum, EnumValueOutOfRangeFails) {
    x::json::EnumMap enums = {{"BIG", int64_t(300)}};
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("BIG"), x::telem::UINT8_T, {}, &enums),
        x::json::CONVERSION_ERROR
    );
}

/// @brief it should reject a numeric enum value for a string target.
TEST(ToSampleValueEnum, NumericEnumValueToStringFails) {
    x::json::EnumMap enums = {{"ON", 1.0}};
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("ON"), x::telem::STRING_T, {}, &enums),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValueEnum, StringNotInEnumFallsBackToNumericParsing) {
    x::json::EnumMap enums = {{"ON", 1.0}, {"OFF", 0.0}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("42"), x::telem::FLOAT64_T, {}, &enums)
    );
    ASSERT_EQ(std::get<double>(sv), 42.0);
}

TEST(ToSampleValueEnum, StringNotInEnumAndNotNumericFails) {
    x::json::EnumMap enums = {{"ON", 1.0}, {"OFF", 0.0}};
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("UNKNOWN"), x::telem::FLOAT64_T, {}, &enums),
        x::json::CONVERSION_ERROR
    );
}

TEST(ToSampleValueEnum, NullEnumMapUsesNormalConversion) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("42"), x::telem::FLOAT64_T, {}, nullptr)
    );
    ASSERT_EQ(std::get<double>(sv), 42.0);
}

TEST(ToSampleValueEnum, NumberBypassesEnumMap) {
    x::json::EnumMap enums = {{"ON", 1.0}, {"OFF", 0.0}};
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(99), x::telem::INT32_T, {}, &enums)
    );
    ASSERT_EQ(std::get<int32_t>(sv), 99);
}
