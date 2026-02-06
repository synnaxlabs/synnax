// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/convert.h"
#include "x/cpp/xtest/xtest.h"

using json = nlohmann::json;

// ========================== resolve_read_converter ==========================
//
// --- JSON Number → Synnax Numeric ---
// [x] resolved (Number, float64) converts json 42.5 → Series with double
// [x] resolved (Number, float32) converts json 42.5 → Series with float
// [x] resolved (Number, int64) converts json 7 → Series with int64_t
// [x] resolved (Number, int32) converts json 7 → Series with int32_t
// [x] resolved (Number, int16) converts json 7 → Series with int16_t
// [x] resolved (Number, int8) converts json 7 → Series with int8_t
// [x] resolved (Number, uint64) converts json 7 → Series with uint64_t
// [x] resolved (Number, uint32) converts json 7 → Series with uint32_t
// [x] resolved (Number, uint16) converts json 7 → Series with uint16_t
// [x] resolved (Number, uint8) converts json 7 → Series with uint8_t
//
// --- JSON Number → Synnax Numeric (strict truncation) ---
// [x] resolved (Number, int64, strict=false) with json 3.7 succeeds, writes 3
// [x] resolved (Number, int64, strict=true) with json 3.7 returns error
// [x] resolved (Number, uint8, strict=true) with json 300 returns error (overflow)
// [x] resolved (Number, uint8, strict=true) with json -1 returns error (underflow)
//
// --- JSON Number → Synnax String ---
// [x] resolved (Number, string) converts json 42.5 → Series with "42.5"
// [x] resolved (Number, string) converts json 7 → Series with "7"
//
// --- JSON String → Synnax String ---
// [x] resolved (String, string) converts json "hello" → Series with "hello"
//
// --- JSON String → Synnax Numeric ---
// [x] resolve (String, float64) returns error at resolve time
//
// --- JSON Boolean → Synnax Numeric ---
// [x] resolved (Boolean, int64) converts json true → Series with int64_t(1)
// [x] resolved (Boolean, int64) converts json false → Series with int64_t(0)
// [x] resolved (Boolean, float64) converts json true → Series with double(1.0)
// [x] resolved (Boolean, uint8) converts json false → Series with uint8_t(0)
//
// --- JSON Boolean → Synnax String ---
// [x] resolved (Boolean, string) converts json true → Series with "true"
// [x] resolved (Boolean, string) converts json false → Series with "false"
//
// --- Unsupported conversions ---
// [x] resolve (Number, UUID) returns error at resolve time
// [x] resolve (Number, JSON) returns error at resolve time

// --- Number → Numeric ---

TEST(ResolveReadConverter, NumberToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::FLOAT64_T)
    );
    const auto series = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_DOUBLE_EQ(series.at<double>(0), 42.5);
}

TEST(ResolveReadConverter, NumberToFloat32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::FLOAT32_T)
    );
    const auto [series, err] = converter(json(42.5));
    ASSERT_NIL(err);
    ASSERT_FLOAT_EQ(series.at<float>(0), 42.5f);
}

TEST(ResolveReadConverter, NumberToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int64_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT32_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int32_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT16_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int16_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT8_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int8_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT64_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<uint64_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT32_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<uint32_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT16_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<uint16_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<uint8_t>(0), 7);
}

// --- Strict truncation ---

TEST(ResolveReadConverter, NumberToInt64NonStrictTruncation) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T, false)
    );
    const auto [series, err] = converter(json(3.7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int64_t>(0), 3);
}

TEST(ResolveReadConverter, NumberToInt64StrictTruncationError) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T, true)
    );
    auto [series, write_err] = converter(json(3.7));
    ASSERT_OCCURRED_AS(write_err, xjson::TRUNCATION_ERROR);
}

TEST(ResolveReadConverter, NumberToUint8StrictOverflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, true)
    );
    auto [series, write_err] = converter(json(300));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERROR);
}

TEST(ResolveReadConverter, NumberToUint8StrictUnderflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, true)
    );
    auto [series, write_err] = converter(json(-1));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERROR);
}

// --- Number → String ---

TEST(ResolveReadConverter, NumberToStringDecimal) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::STRING_T)
    );
    const auto [series, err] = converter(json(42.5));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<std::string>(0), "42.5");
}

TEST(ResolveReadConverter, NumberToStringInteger) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::STRING_T)
    );
    const auto [series, err] = converter(json(7));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<std::string>(0), "7");
}

// --- String → String ---

TEST(ResolveReadConverter, StringToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::String, telem::STRING_T)
    );
    const auto [series, err] = converter(json("hello"));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<std::string>(0), "hello");
}

// --- String → Numeric (unsupported) ---

TEST(ResolveReadConverter, StringToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::String, telem::FLOAT64_T),
        xjson::UNSUPPORTED_ERROR
    );
}

// --- Boolean → Numeric ---

TEST(ResolveReadConverter, BooleanTrueToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::INT64_T)
    );
    const auto [series, err] = converter(json(true));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int64_t>(0), 1);
}

TEST(ResolveReadConverter, BooleanFalseToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::INT64_T)
    );
    const auto [series, err] = converter(json(false));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<int64_t>(0), 0);
}

TEST(ResolveReadConverter, BooleanTrueToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::FLOAT64_T)
    );
    const auto [series, err] = converter(json(true));
    ASSERT_NIL(err);
    ASSERT_DOUBLE_EQ(series.at<double>(0), 1.0);
}

TEST(ResolveReadConverter, BooleanFalseToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::UINT8_T)
    );
    const auto [series, err] = converter(json(false));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<uint8_t>(0), 0);
}

// --- Boolean → String ---

TEST(ResolveReadConverter, BooleanTrueToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::STRING_T)
    );
    const auto [series, err] = converter(json(true));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<std::string>(0), "true");
}

TEST(ResolveReadConverter, BooleanFalseToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::STRING_T)
    );
    const auto [series, err] = converter(json(false));
    ASSERT_NIL(err);
    ASSERT_EQ(series.at<std::string>(0), "false");
}

// --- Unsupported target types ---

TEST(ResolveReadConverter, NumberToUUIDError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UUID_T),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ResolveReadConverter, NumberToJSONError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::JSON_T),
        xjson::UNSUPPORTED_ERROR
    );
}

// --- from_sample_value: Numeric → Number ---

TEST(FromSampleValue, Float64ToNumber) {
    const auto value = 42.5;
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Float64WithoutDecimalToNumber) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(42.0), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(42));
}

TEST(FromSampleValue, Float64WithLongDecimalToNumber) {
    const auto value = 1.2345689012;
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Int64ToNumber) {
    const auto value = int64_t(-743984);
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Int64NegativeToNumber) {
    const auto value = int64_t(-743984);
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Uint8ZeroToNumber) {
    const auto value = uint8_t(0);
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Uint8ToNumber) {
    const auto value = uint8_t(255);
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::Number)
    );
    ASSERT_EQ(result, json(value));
}

// --- Numeric → String ---

TEST(FromSampleValue, Float64ToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(42.5), xjson::Type::String)
    );
    ASSERT_EQ(result, json("42.5"));
}

TEST(FromSampleValue, Float64WithLongDecimalToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(1.2345689012), xjson::Type::String)
    );
    ASSERT_EQ(result, json("1.2345689012"));
}

TEST(FromSampleValue, Float64WithoutDecimalToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(42.0), xjson::Type::String)
    );
    ASSERT_EQ(result, json("42"));
}

TEST(FromSampleValue, Int64ToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(int64_t(7)), xjson::Type::String)
    );
    ASSERT_EQ(result, json("7"));
}

TEST(FromSampleValue, Int64NegativeToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(int64_t(-743984)),
            xjson::Type::String
        )
    );
    ASSERT_EQ(result, json("-743984"));
}

TEST(FromSampleValue, Uint8ToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(uint8_t(255)), xjson::Type::String)
    );
    ASSERT_EQ(result, json("255"));
}

TEST(FromSampleValue, Uint8ZeroToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(uint8_t(0)), xjson::Type::String)
    );
    ASSERT_EQ(result, json("0"));
}

// --- Numeric → Boolean ---

TEST(FromSampleValue, Int64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(int64_t(0)), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Int64OneToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(int64_t(1)), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Int64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(int64_t(-743984)),
            xjson::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(0.0), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Float64PositiveToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(42.5), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(-743984.0), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Uint8ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(uint8_t(0)), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Uint8NonZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(uint8_t(255)), xjson::Type::Boolean)
    );
    ASSERT_EQ(result, json(true));
}

// --- String → String ---

TEST(FromSampleValue, StringToString) {
    const auto value = std::string("hello");
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::String)
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, StringWithZeroLengthToString) {
    const auto value = std::string("");
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(telem::SampleValue(value), xjson::Type::String)
    );
    ASSERT_EQ(result, json(value));
}

// --- Unsupported from_sample_value conversions ---

TEST(FromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(std::string("hello")),
            xjson::Type::Number
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(std::string("hello")),
            xjson::Type::Boolean
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToNumberError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(telem::TimeStamp(1000000000)),
            xjson::Type::Number
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToStringError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(telem::TimeStamp(1000000000)),
            xjson::Type::String
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(telem::TimeStamp(1000000000)),
            xjson::Type::Boolean
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(FromTimestamp, UnixNanosecond) {
    // 10^9 seconds since epoch in nanoseconds
    const int64_t value = 1000000000000000000;
    const auto ts = telem::TimeStamp(value);
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixNanosecond),
        json(value)
    );
}

TEST(FromTimestamp, UnixNanosecondZero) {
    const int64_t value = 0;
    ASSERT_EQ(
        xjson::from_timestamp(
            telem::TimeStamp(value),
            xjson::TimeFormat::UnixNanosecond
        ),
        json(value)
    );
}

TEST(FromTimestamp, UnixNanosecondSubSecond) {
    const int64_t value = 1000000000123456789;
    const auto ts = telem::TimeStamp(value);
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixNanosecond),
        json(value)
    );
}

TEST(FromTimestamp, UnixNanosecondNegative) {
    const int64_t value = -1500000001;
    ASSERT_EQ(
        xjson::from_timestamp(
            telem::TimeStamp(value),
            xjson::TimeFormat::UnixNanosecond
        ),
        json(value)
    );
}

TEST(FromTimestamp, UnixMicrosecond) {
    const auto value = int64_t(1000000000000000000);
    const auto ts = telem::TimeStamp(value);
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMicrosecond),
        json(value / 1000)
    );
}

TEST(FromTimestamp, UnixMicrosecondZero) {
    ASSERT_EQ(
        xjson::from_timestamp(telem::TimeStamp(0), xjson::TimeFormat::UnixMicrosecond),
        json(int64_t(0))
    );
}

TEST(FromTimestamp, UnixMicrosecondFloorsSubMicrosecond) {
    // 789 nanoseconds floored away
    const auto ts = telem::TimeStamp(int64_t(1000000000123456789));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMicrosecond),
        json(int64_t(1000000000123456))
    );
}

TEST(FromTimestamp, UnixMicrosecondNegativeFloors) {
    // -1500000001 ns → floor(-1500000.001) = -1500001 (not -1500000)
    const auto ts = telem::TimeStamp(int64_t(-1500000001));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMicrosecond),
        json(int64_t(-1500001))
    );
}

TEST(FromTimestamp, UnixMillisecond) {
    const int64_t value = 1000000000000000000;
    const auto ts = telem::TimeStamp(value);
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMillisecond),
        json(value / 1000000)
    );
}

TEST(FromTimestamp, UnixMillisecondZero) {
    ASSERT_EQ(
        xjson::from_timestamp(telem::TimeStamp(0), xjson::TimeFormat::UnixMillisecond),
        json(int64_t(0))
    );
}

TEST(FromTimestamp, UnixMillisecondFloorsSubMillisecond) {
    // 456789 nanoseconds floored away
    const auto ts = telem::TimeStamp(int64_t(1000000000123456789));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMillisecond),
        json(int64_t(1000000000123))
    );
}

TEST(FromTimestamp, UnixMillisecondNegativeFloors) {
    // -1500000001 ns → floor(-1500.000001) = -1501 (not -1500)
    const auto ts = telem::TimeStamp(int64_t(-1500000001));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixMillisecond),
        json(int64_t(-1501))
    );
}

TEST(FromTimestamp, UnixSecondInt) {
    const int64_t value = 1000000000000000000;
    const auto ts = telem::TimeStamp(value);
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondInt),
        json(value / 1000000000)
    );
}

TEST(FromTimestamp, UnixSecondIntZero) {
    ASSERT_EQ(
        xjson::from_timestamp(telem::TimeStamp(0), xjson::TimeFormat::UnixSecondInt),
        json(int64_t(0))
    );
}

TEST(FromTimestamp, UnixSecondIntFloorsSubSecond) {
    const auto ts = telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondInt),
        json(int64_t(1000000000))
    );
}

TEST(FromTimestamp, UnixSecondIntNegativeFloors) {
    // -1500000001 ns → floor(-1.500000001) = -2 (not -1)
    const auto ts = telem::TimeStamp(int64_t(-1500000001));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondInt),
        json(int64_t(-2))
    );
}

TEST(FromTimestamp, UnixSecondFloat) {
    const auto ts = telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondFloat),
        json(1000000000)
    );
}

TEST(FromTimestamp, UnixSecondFloatZero) {
    ASSERT_EQ(
        xjson::from_timestamp(telem::TimeStamp(0), xjson::TimeFormat::UnixSecondFloat),
        json(0.0)
    );
}

TEST(FromTimestamp, UnixSecondFloatPreservesSubSecond) {
    // 0.5 seconds preserved as float
    const auto ts = telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondFloat),
        json(1000000000.5)
    );
}

TEST(FromTimestamp, UnixSecondFloatNegative) {
    const auto ts = telem::TimeStamp(int64_t(-1500000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::UnixSecondFloat),
        json(-1.5)
    );
}

TEST(FromTimestamp, ISO8601Epoch) {
    ASSERT_EQ(
        xjson::from_timestamp(telem::TimeStamp(0), xjson::TimeFormat::ISO8601),
        json("1970-01-01T00:00:00Z")
    );
}

TEST(FromTimestamp, ISO8601) {
    // 10^9 seconds = 2001-09-09T01:46:40Z
    const auto ts = telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40Z")
    );
}

TEST(FromTimestamp, ISO8601WithSubSecond) {
    // 10^9 seconds + 500ms
    const auto ts = telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40.5Z")
    );
}

TEST(FromTimestamp, ISO8601Negative) {
    // -10^9 seconds from epoch = 1938-04-24T22:13:20Z
    const auto ts = telem::TimeStamp(int64_t(-1000000000000000000));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::ISO8601),
        json("1938-04-24T22:13:20Z")
    );
}

TEST(FromTimestamp, ISO8601WithNanosecondPrecision) {
    const auto ts = telem::TimeStamp(int64_t(1000000000000000001));
    ASSERT_EQ(
        xjson::from_timestamp(ts, xjson::TimeFormat::ISO8601),
        json("2001-09-09T01:46:40.000000001Z")
    );
}

TEST(CheckFromSampleValue, Float64ToNumberOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::Number));
}

TEST(CheckFromSampleValue, Float64ToStringOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::String));
}

TEST(CheckFromSampleValue, Float64ToBooleanOK) {
    ASSERT_FALSE(
        xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::Boolean)
    );
}

TEST(CheckFromSampleValue, Int64ToNumberOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::INT64_T, xjson::Type::Number));
}

TEST(CheckFromSampleValue, Int64ToStringOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::INT64_T, xjson::Type::String));
}

TEST(CheckFromSampleValue, Int64ToBooleanOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::INT64_T, xjson::Type::Boolean));
}

TEST(CheckFromSampleValue, Uint8ToNumberOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::UINT8_T, xjson::Type::Number));
}

TEST(CheckFromSampleValue, Uint8ToStringOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::UINT8_T, xjson::Type::String));
}

TEST(CheckFromSampleValue, Uint8ToBooleanOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::UINT8_T, xjson::Type::Boolean));
}

TEST(CheckFromSampleValue, StringToStringOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::STRING_T, xjson::Type::String));
}

TEST(CheckFromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::STRING_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::STRING_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::TIMESTAMP_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::TIMESTAMP_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::TIMESTAMP_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::JSON_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::JSON_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::JSON_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ZeroValue, Number) {
    ASSERT_EQ(xjson::zero_value(xjson::Type::Number), 0);
}

TEST(ZeroValue, String) {
    ASSERT_EQ(xjson::zero_value(xjson::Type::String), "");
}

TEST(ZeroValue, Boolean) {
    ASSERT_EQ(xjson::zero_value(xjson::Type::Boolean), false);
}
