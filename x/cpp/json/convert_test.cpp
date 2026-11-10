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
#include "x/cpp/telem/series.h"
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

// ==================== from_sample_value ====================

TEST(FromSampleValue, Float64ToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(42.5), x::json::Type::Number)
    );
    ASSERT_EQ(result, json(42.5));
}

TEST(FromSampleValue, Float64WithoutDecimalToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(42.0), x::json::Type::Number)
    );
    ASSERT_EQ(result, json(42));
}

TEST(FromSampleValue, Float64WithLongDecimalToNumber) {
    const auto value = 1.2345689012;
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(value), x::json::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Int64ToNumber) {
    const auto value = int64_t(-743984);
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(value), x::json::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Uint8ZeroToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)),
            x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(0));
}

TEST(FromSampleValue, Uint8ToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)),
            x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(255));
}

TEST(FromSampleValue, Float64ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(42.5), x::json::Type::String)
    );
    ASSERT_EQ(result, json("42.5"));
}

TEST(FromSampleValue, Float64WithLongDecimalToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.2345689012),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("1.2345689012"));
}

TEST(FromSampleValue, Float64WithoutDecimalToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(42.0), x::json::Type::String)
    );
    ASSERT_EQ(result, json("42"));
}

TEST(FromSampleValue, Int64ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(7)),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("7"));
}

TEST(FromSampleValue, Int64NegativeToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(-743984)),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("-743984"));
}

TEST(FromSampleValue, Uint8ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("255"));
}

TEST(FromSampleValue, Uint8ZeroToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("0"));
}

TEST(FromSampleValue, Int64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(0)),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Int64OneToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(1)),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Int64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(-743984)),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(0.0), x::json::Type::Boolean)
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Float64PositiveToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(x::telem::SampleValue(42.5), x::json::Type::Boolean)
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(-743984.0),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Uint8ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Uint8NonZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, NonFiniteFloatToNumberErrors) {
    for (const double v:
         {std::numeric_limits<double>::quiet_NaN(),
          std::numeric_limits<double>::infinity(),
          -std::numeric_limits<double>::infinity()})
        ASSERT_OCCURRED_AS_P(
            x::json::from_sample_value(x::telem::SampleValue(v), x::json::Type::Number),
            x::json::CONVERSION_ERROR
        );
}

TEST(FromSampleValue, Float32NaNToNumberErrors) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<float>::quiet_NaN()),
            x::json::Type::Number
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, NaNToBooleanErrors) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<double>::quiet_NaN()),
            x::json::Type::Boolean
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, InfinityToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<double>::infinity()),
            x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, StringToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("hello"));
}

TEST(FromSampleValue, StringWithZeroLengthToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("")),
            x::json::Type::String
        )
    );
    ASSERT_EQ(result, json(""));
}

TEST(FromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")),
            x::json::Type::Number
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")),
            x::json::Type::Boolean
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, TimeStampToNumberError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::Number
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, TimeStampToStringError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::String
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, TimeStampToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::Boolean
        ),
        x::json::CONVERSION_ERROR
    );
}

TEST(FromSampleValue, EnumMatchFloat64) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}, {0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.0),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("ON"));
}

TEST(FromSampleValue, EnumMatchZeroFloat64) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}, {0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(0.0),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("OFF"));
}

TEST(FromSampleValue, EnumMatchUint8) {
    const x::json::ReverseEnumMap enums = {{1, "ON"}, {0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(1)),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("ON"));
}

TEST(FromSampleValue, EnumMatchInt64) {
    const x::json::ReverseEnumMap enums = {{2, "HIGH"}, {1, "MEDIUM"}, {0, "LOW"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(2)),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("HIGH"));
}

TEST(FromSampleValue, EnumNoMatchFallsThrough) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}, {0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(99.0),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("99"));
}

/// @brief it should write an unmatched NaN as text.
TEST(FromSampleValue, NaNWithNoEnumToString) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}, {0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<double>::quiet_NaN()),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("NaN"));
}

/// @brief it should write the infinities as text.
TEST(FromSampleValue, InfinityToString) {
    const auto pos = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<float>::infinity()),
            x::json::Type::String
        )
    );
    ASSERT_EQ(pos, json("Infinity"));
    const auto neg = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(-std::numeric_limits<double>::infinity()),
            x::json::Type::String
        )
    );
    ASSERT_EQ(neg, json("-Infinity"));
}

/// @brief it should write a matched sample as its label for a number target.
TEST(FromSampleValue, EnumMatchForNumberTarget) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}, {0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.0),
            x::json::Type::Number,
            &enums
        )
    );
    ASSERT_EQ(result, json("ON"));
}

/// @brief it should tell apart int64 enum values that a double rounds together.
TEST(FromSampleValue, EnumMatchInt64Past2To53) {
    constexpr int64_t past = (int64_t(1) << 53) + 1;
    const x::json::ReverseEnumMap enums = {
        {int64_t(past - 1), "LOW"},
        {int64_t(past), "HIGH"},
    };
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(past),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("HIGH"));
}

/// @brief it should tell apart uint64 enum values near the maximum.
TEST(FromSampleValue, EnumMatchUint64Max) {
    constexpr auto max = std::numeric_limits<uint64_t>::max();
    const x::json::ReverseEnumMap enums = {
        {uint64_t(max - 1), "LOW"},
        {max, "HIGH"},
    };
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(max),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("HIGH"));
}

/// @brief it should not match a double entry to an integer it only rounds to.
TEST(FromSampleValue, EnumFloatEntryMatchesIntegerExactly) {
    constexpr int64_t past = (int64_t(1) << 53) + 1;
    const x::json::ReverseEnumMap enums = {{static_cast<double>(past), "ROUNDED"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(past),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("9007199254740993"));
}

/// @brief it should match an integer sample to a whole float entry.
TEST(FromSampleValue, EnumMatchUint8ToFloatEntry) {
    const x::json::ReverseEnumMap enums = {{1.0, "ON"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(1)),
            x::json::Type::Boolean,
            &enums
        )
    );
    ASSERT_EQ(result, json("ON"));
}

/// @brief it should match any NaN to a NaN entry.
TEST(FromSampleValue, EnumMatchNaN) {
    const x::json::ReverseEnumMap enums = {
        {std::numeric_limits<double>::quiet_NaN(), "FAULT"},
    };
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<float>::signaling_NaN()),
            x::json::Type::Number,
            &enums
        )
    );
    ASSERT_EQ(result, json("FAULT"));
}

/// @brief it should match negative zero to a zero entry.
TEST(FromSampleValue, EnumMatchNegativeZero) {
    const x::json::ReverseEnumMap enums = {{0.0, "OFF"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(-0.0),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("OFF"));
}

/// @brief it should narrow a double entry to the type of a float32 sample.
TEST(FromSampleValue, EnumMatchFloat32Narrowed) {
    const x::json::ReverseEnumMap enums = {{0.1, "TENTH"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(0.1f),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("TENTH"));
}

/// @brief it should not match a double entry past the float32 range to infinity.
TEST(FromSampleValue, EnumEntryPastFloat32DoesNotMatchInfinity) {
    const x::json::ReverseEnumMap enums = {{1e39, "BIG"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::numeric_limits<float>::infinity()),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("Infinity"));
}

/// @brief it should write a string sample as its enum label.
TEST(FromSampleValue, EnumMatchString) {
    const x::json::ReverseEnumMap enums = {{std::string("1"), "ON"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("1")),
            x::json::Type::String,
            &enums
        )
    );
    ASSERT_EQ(result, json("ON"));
}

/// @brief it should not match a string entry to a number.
TEST(FromSampleValue, EnumStringEntryDoesNotMatchNumber) {
    const x::json::ReverseEnumMap enums = {{std::string("1"), "ON"}};
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.0),
            x::json::Type::Number,
            &enums
        )
    );
    ASSERT_EQ(result, json(1.0));
}

TEST(FromSampleValue, EnumNullptrBehavesNormally) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.0),
            x::json::Type::String,
            nullptr
        )
    );
    ASSERT_EQ(result, json("1"));
}

// ==================== from_series ====================

/// @brief it should convert a JSON sample to the value that its text holds.
TEST(FromSeries, JSON) {
    const x::telem::Series s(
        std::vector<std::string>{"1", R"({"a":[1,"b",null]})"},
        x::telem::JSON_T
    );
    ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, 0)), json(1));
    ASSERT_EQ(
        ASSERT_NIL_P(x::json::from_series(s, -1)),
        json::parse(R"({"a":[1,"b",null]})")
    );
}

/// @brief it should reject a JSON sample whose text is not valid JSON.
TEST(FromSeries, InvalidJSON) {
    const x::telem::Series s(std::string("{"), x::telem::JSON_T);
    ASSERT_OCCURRED_AS_P(x::json::from_series(s, 0), x::json::CONVERSION_ERROR);
}

/// @brief it should convert a UUID sample to its text.
TEST(FromSeries, UUID) {
    const auto a = x::uuid::create();
    const auto b = x::uuid::create();
    x::telem::Series s(x::telem::UUID_T, 2);
    s.write(a.data(), 1);
    s.write(b.data(), 1);
    ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, 0)), json(a.to_string()));
    ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, -1)), json(b.to_string()));
}

/// @brief it should convert a bytes sample to padded base64 text.
TEST(FromSeries, Bytes) {
    const x::telem::Series s(
        std::vector<std::string>{std::string("\x00\xff\x01\x02", 4), ""},
        x::telem::BYTES_T
    );
    ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, 0)), json("AP8BAg=="));
    ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, 1)), json(""));
}

/// @brief it should throw on an index past the last sample.
TEST(FromSeries, OutOfBounds) {
    const x::telem::Series bytes(std::vector<std::string>{}, x::telem::BYTES_T);
    ASSERT_THROW((void) x::json::from_series(bytes, 0), std::runtime_error);
    const x::telem::Series text(std::string("1"), x::telem::JSON_T);
    ASSERT_THROW((void) x::json::from_series(text, 1), std::runtime_error);
}

/// @brief it should reject a series of another data type.
TEST(FromSeries, OtherDataType) {
    for (const auto &s: {x::telem::Series(1.0), x::telem::Series(std::string("a"))})
        ASSERT_OCCURRED_AS_P(x::json::from_series(s, 0), x::json::CONVERSION_ERROR);
}

/// @brief it should convert JSON, UUID, and bytes values to samples and back.
TEST(FromSeries, RoundTrip) {
    const std::vector<std::pair<x::telem::DataType, json>> cases = {
        {x::telem::JSON_T, json::parse(R"({"a":[1,2.5,"c",true,null]})")},
        {x::telem::JSON_T, json("text")},
        {x::telem::UUID_T, json("6ba7b810-9dad-11d1-80b4-00c04fd430c8")},
        {x::telem::BYTES_T, json("SGVsbG8sIFdvcmxkIQ==")},
    };
    for (const auto &[dt, value]: cases) {
        const auto sv = ASSERT_NIL_P(x::json::to_sample_value(value, dt));
        const x::telem::Series s(sv, dt);
        ASSERT_EQ(s.data_type(), dt);
        ASSERT_EQ(ASSERT_NIL_P(x::json::from_series(s, 0)), value);
    }
}

// ==================== from_timestamp ====================

TEST(FromTimestamp, UnixNanosecond) {
    const int64_t value = 1000000000000000000;
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(value),
            x::json::TimeFormat::UnixNanosecond
        ),
        json(value)
    );
}

TEST(FromTimestamp, UnixNanosecondZero) {
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0),
            x::json::TimeFormat::UnixNanosecond
        ),
        json(int64_t(0))
    );
}

TEST(FromTimestamp, UnixNanosecondSubSecond) {
    const int64_t value = 1000000000123456789;
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(value),
            x::json::TimeFormat::UnixNanosecond
        ),
        json(value)
    );
}

TEST(FromTimestamp, UnixNanosecondNegative) {
    const int64_t value = -1500000001;
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(value),
            x::json::TimeFormat::UnixNanosecond
        ),
        json(value)
    );
}

TEST(FromTimestamp, UnixMicrosecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond).get<double>(),
        1000000000000000.0
    );
}

TEST(FromTimestamp, UnixMicrosecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0),
            x::json::TimeFormat::UnixMicrosecond
        )
            .get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixMicrosecondPreservesSubMicrosecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000500));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond).get<double>(),
        1000000.5
    );
}

TEST(FromTimestamp, UnixMicrosecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond).get<double>(),
        -1500000.0
    );
}

/// @brief it should write a whole count of units past 2^53 as an exact integer.
TEST(FromTimestamp, UnixMicrosecondWholePast2To53) {
    constexpr int64_t past = (int64_t(1) << 53) + 1;
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(past * 1000),
            x::json::TimeFormat::UnixMicrosecond
        ),
        json(past)
    );
}

/// @brief it should round a fraction once, from its exact decimal value.
TEST(FromTimestamp, UnixMicrosecondRoundsOnce) {
    const auto ts = x::telem::TimeStamp(int64_t(1767323045123456789));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond).get<double>(),
        std::stod("1767323045123456.789")
    );
}

/// @brief it should pad the fraction of a negative timestamp with zeros.
TEST(FromTimestamp, UnixSecondNegativePaddedFraction) {
    const auto ts = x::telem::TimeStamp(int64_t(-1000000001));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond).get<double>(),
        -1.000000001
    );
}

TEST(FromTimestamp, UnixMillisecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond).get<double>(),
        1000000000000.0
    );
}

TEST(FromTimestamp, UnixMillisecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0),
            x::json::TimeFormat::UnixMillisecond
        )
            .get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixMillisecondPreservesSubMillisecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1500500000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond).get<double>(),
        1500.5
    );
}

TEST(FromTimestamp, UnixMillisecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond).get<double>(),
        -1500.0
    );
}

TEST(FromTimestamp, UnixSecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond).get<double>(),
        1000000000.0
    );
}

TEST(FromTimestamp, UnixSecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(x::telem::TimeStamp(0), x::json::TimeFormat::UnixSecond)
            .get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixSecondPreservesSubSecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond).get<double>(),
        1000000000.5
    );
}

TEST(FromTimestamp, UnixSecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond).get<double>(),
        -1.5
    );
}

TEST(FromTimestamp, ISO8601Epoch) {
    ASSERT_EQ(
        x::json::from_timestamp(x::telem::TimeStamp(0), x::json::TimeFormat::ISO8601),
        json("1970-01-01T00:00:00Z")
    );
}

TEST(FromTimestamp, ISO8601) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40Z")
    );
}

TEST(FromTimestamp, ISO8601WithSubSecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40.5Z")
    );
}

TEST(FromTimestamp, ISO8601Negative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1000000000000000000));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::ISO8601),
        json("1938-04-24T22:13:20Z")
    );
}

TEST(FromTimestamp, ISO8601WithNanosecondPrecision) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000001));
    ASSERT_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40.000000001Z")
    );
}

// ==================== check_to_sample_value ====================

TEST(CheckToSampleValue, Float64) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::FLOAT64_T));
}

TEST(CheckToSampleValue, Float32) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::FLOAT32_T));
}

TEST(CheckToSampleValue, Int64) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::INT64_T));
}

TEST(CheckToSampleValue, Int32) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::INT32_T));
}

TEST(CheckToSampleValue, Int16) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::INT16_T));
}

TEST(CheckToSampleValue, Int8) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::INT8_T));
}

TEST(CheckToSampleValue, Uint64) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::UINT64_T));
}

TEST(CheckToSampleValue, Uint32) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::UINT32_T));
}

TEST(CheckToSampleValue, Uint16) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::UINT16_T));
}

TEST(CheckToSampleValue, Uint8) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::UINT8_T));
}

TEST(CheckToSampleValue, Timestamp) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::TIMESTAMP_T));
}

TEST(CheckToSampleValue, String) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::STRING_T));
}

TEST(CheckToSampleValue, UUID) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::UUID_T));
}

TEST(CheckToSampleValue, JSON) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::JSON_T));
}

TEST(CheckToSampleValue, Bytes) {
    ASSERT_TRUE(x::json::check_to_sample_value(x::telem::BYTES_T));
}

TEST(CheckToSampleValue, Unknown) {
    ASSERT_FALSE(x::json::check_to_sample_value(x::telem::UNKNOWN_T));
}

// ==================== check_from_sample_value ====================

TEST(CheckFromSampleValue, Float64ToNumberOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::FLOAT64_T, x::json::Type::Number)
    );
}

TEST(CheckFromSampleValue, Float64ToStringOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::FLOAT64_T, x::json::Type::String)
    );
}

TEST(CheckFromSampleValue, Float64ToBooleanOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::FLOAT64_T, x::json::Type::Boolean)
    );
}

TEST(CheckFromSampleValue, Int64ToNumberOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::INT64_T, x::json::Type::Number)
    );
}

TEST(CheckFromSampleValue, Int64ToStringOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::INT64_T, x::json::Type::String)
    );
}

TEST(CheckFromSampleValue, Int64ToBooleanOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::INT64_T, x::json::Type::Boolean)
    );
}

TEST(CheckFromSampleValue, Uint8ToNumberOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::UINT8_T, x::json::Type::Number)
    );
}

TEST(CheckFromSampleValue, Uint8ToStringOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::UINT8_T, x::json::Type::String)
    );
}

TEST(CheckFromSampleValue, Uint8ToBooleanOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::UINT8_T, x::json::Type::Boolean)
    );
}

TEST(CheckFromSampleValue, StringToStringOK) {
    ASSERT_NIL(
        x::json::check_from_sample_value(x::telem::STRING_T, x::json::Type::String)
    );
}

TEST(CheckFromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::STRING_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::STRING_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::TIMESTAMP_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::TIMESTAMP_T, x::json::Type::String),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::TIMESTAMP_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UUID_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UUID_T, x::json::Type::String),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UUID_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::BYTES_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::BYTES_T, x::json::Type::String),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::BYTES_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UNKNOWN_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UNKNOWN_T, x::json::Type::String),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::UNKNOWN_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::JSON_T, x::json::Type::Number),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::JSON_T, x::json::Type::String),
        x::json::CONVERSION_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(x::telem::JSON_T, x::json::Type::Boolean),
        x::json::CONVERSION_ERROR
    );
}

// ==================== zero_value ====================

TEST(ZeroValue, Number) {
    ASSERT_EQ(x::json::zero_value(x::json::Type::Number), 0);
}

TEST(ZeroValue, String) {
    ASSERT_EQ(x::json::zero_value(x::json::Type::String), "");
}

TEST(ZeroValue, Boolean) {
    ASSERT_EQ(x::json::zero_value(x::json::Type::Boolean), false);
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
