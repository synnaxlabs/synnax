// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "x/cpp/xjson/convert.h"
#include "x/cpp/xtest/xtest.h"
#include "x/cpp/telem/telem.h"

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
//
// ========================== from_sample_value ==========================
//
// --- Synnax Numeric → JSON Number ---
// [x] double(42.5) → json 42.5
// [x] double(1.2345689012) → json 1.2345689012
// [x] int64_t(-743984) → json -743984
// [x] uint8_t(255) → json 255
//
// --- Synnax Numeric → JSON String ---
// [x] double(42.5) → json "42.5"
// [x] double(1.2345689012) → json "1.2345689012"
// [x] double(42.0) → json "42"
// [x] int64_t(7) → json "7"
// [x] int64_t(-743984) → json "-743984"
//
// --- Synnax Numeric → JSON Boolean ---
// [x] int64_t(0) → json false
// [x] int64_t(1) → json true
// [x] int64_t(-743984) → json true
// [x] double(0.0) → json false
// [x] double(42.5) → json true
// [x] double(-743984.0) → json true
// [x] uint8_t(0) → json false
// [x] uint8_t(255) → json true
//
// --- Synnax String → JSON String ---
// [x] "hello" → json "hello"
// [x] "" → json ""
//
// --- Unsupported from_sample_value conversions ---
// [x] string → Number returns error
// [x] string → Boolean returns error
//
// ========================== check_from_sample_value ==========================
//
// [x] (float64, Number) → OK
// [x] (string, String) → OK
// [x] (string, Number) → UNSUPPORTED_ERR
// [x] (string, Boolean) → UNSUPPORTED_ERR
// [x] (UUID, Number) → UNSUPPORTED_ERR
// [x] (UUID, String) → UNSUPPORTED_ERR
// [x] (UUID, Boolean) → UNSUPPORTED_ERR
// [x] (BYTES, Number) → UNSUPPORTED_ERR
// [x] (BYTES, String) → UNSUPPORTED_ERR
// [x] (BYTES, Boolean) → UNSUPPORTED_ERR
// [x] (UNKNOWN, Number) → UNSUPPORTED_ERR
// [x] (UNKNOWN, String) → UNSUPPORTED_ERR
// [x] (UNKNOWN, Boolean) → UNSUPPORTED_ERR
//
// ========================== zero_value ==========================
//
// [x] Number → json 0
// [x] String → json ""
// [x] Boolean → json false

// --- Number → Numeric ---

TEST(ResolveReadConverter, NumberToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::FLOAT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_DOUBLE_EQ(series.at<double>(0), 42.5);
}

TEST(ResolveReadConverter, NumberToFloat32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::FLOAT32_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_FLOAT_EQ(series.at<float>(0), 42.5f);
}

TEST(ResolveReadConverter, NumberToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<int64_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT32_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<int32_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT16_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<int16_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToInt8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT8_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<int8_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<uint64_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT32_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<uint32_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT16_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<uint16_t>(0), 7);
}

TEST(ResolveReadConverter, NumberToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT8_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<uint8_t>(0), 7);
}

// --- Strict truncation ---

TEST(ResolveReadConverter, NumberToInt64NonStrictTruncation) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT64_T, false
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(3.7)));
    ASSERT_EQ(series.at<int64_t>(0), 3);
}

TEST(ResolveReadConverter, NumberToInt64StrictTruncationError) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::INT64_T, true
        )
    );
    auto [series, write_err] = converter(json(3.7));
    ASSERT_OCCURRED_AS(write_err, xjson::TRUNCATION_ERR);
}

TEST(ResolveReadConverter, NumberToUint8StrictOverflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT8_T, true
        )
    );
    auto [series, write_err] = converter(json(300));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERR);
}

TEST(ResolveReadConverter, NumberToUint8StrictUnderflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::UINT8_T, true
        )
    );
    auto [series, write_err] = converter(json(-1));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERR);
}

// --- Number → String ---

TEST(ResolveReadConverter, NumberToStringDecimal) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::STRING_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_EQ(series.at<std::string>(0), "42.5");
}

TEST(ResolveReadConverter, NumberToStringInteger) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::STRING_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(series.at<std::string>(0), "7");
}

// --- String → String ---

TEST(ResolveReadConverter, StringToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::String, telem::STRING_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json("hello")));
    ASSERT_EQ(series.at<std::string>(0), "hello");
}

// --- String → Numeric (unsupported) ---

TEST(ResolveReadConverter, StringToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::String, telem::FLOAT64_T),
        xjson::UNSUPPORTED_ERR
    );
}

// --- Boolean → Numeric ---

TEST(ResolveReadConverter, BooleanTrueToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::INT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(true)));
    ASSERT_EQ(series.at<int64_t>(0), 1);
}

TEST(ResolveReadConverter, BooleanFalseToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::INT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(series.at<int64_t>(0), 0);
}

TEST(ResolveReadConverter, BooleanTrueToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::FLOAT64_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(true)));
    ASSERT_DOUBLE_EQ(series.at<double>(0), 1.0);
}

TEST(ResolveReadConverter, BooleanFalseToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::UINT8_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(series.at<uint8_t>(0), 0);
}

// --- Boolean → String ---

TEST(ResolveReadConverter, BooleanTrueToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::STRING_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(true)));
    ASSERT_EQ(series.at<std::string>(0), "true");
}

TEST(ResolveReadConverter, BooleanFalseToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(
            xjson::Type::Boolean, telem::STRING_T
        )
    );
    const auto series = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(series.at<std::string>(0), "false");
}

// --- Unsupported target types ---

TEST(ResolveReadConverter, NumberToUUIDError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UUID_T),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(ResolveReadConverter, NumberToJSONError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::JSON_T),
        xjson::UNSUPPORTED_ERR
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
        xjson::from_sample_value(
            telem::SampleValue(1.2345689012), xjson::Type::String
        )
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
        xjson::from_sample_value(
            telem::SampleValue(int64_t(7)), xjson::Type::String
        )
    );
    ASSERT_EQ(result, json("7"));
}

TEST(FromSampleValue, Int64NegativeToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(int64_t(-743984)), xjson::Type::String
        )
    );
    ASSERT_EQ(result, json("-743984"));
}

TEST(FromSampleValue, Uint8ToString) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(uint8_t(255)), xjson::Type::String
        )
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
        xjson::from_sample_value(
            telem::SampleValue(int64_t(0)), xjson::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Int64OneToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(int64_t(1)), xjson::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Int64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(int64_t(-743984)), xjson::Type::Boolean
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
        xjson::from_sample_value(
            telem::SampleValue(-743984.0), xjson::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Uint8ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(uint8_t(0)), xjson::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Uint8NonZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        xjson::from_sample_value(
            telem::SampleValue(uint8_t(255)), xjson::Type::Boolean
        )
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
            telem::SampleValue(std::string("hello")), xjson::Type::Number
        ),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(FromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        xjson::from_sample_value(
            telem::SampleValue(std::string("hello")), xjson::Type::Boolean
        ),
        xjson::UNSUPPORTED_ERR
    );
}

// --- check_from_sample_value ---

TEST(CheckFromSampleValue, Float64ToNumberOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::Number));
}

TEST(CheckFromSampleValue, Float64ToStringOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::String));
}

TEST(CheckFromSampleValue, Float64ToBooleanOK) {
    ASSERT_FALSE(xjson::check_from_sample_value(telem::FLOAT64_T, xjson::Type::Boolean));
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
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::STRING_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UUIDToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UUIDToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UUIDToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UUID_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, BytesToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, BytesToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, BytesToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::BYTES_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UnknownToNumberError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::Number),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UnknownToStringError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::String),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(CheckFromSampleValue, UnknownToBooleanError) {
    ASSERT_OCCURRED_AS(
        xjson::check_from_sample_value(telem::UNKNOWN_T, xjson::Type::Boolean),
        xjson::UNSUPPORTED_ERR
    );
}

TEST(ZeroValue, Number) {
    ASSERT_EQ(xjson::zero_value(xjson::Type::Number), 0);
}

TEST(ZeroValue, String) {
    ASSERT_EQ(
        xjson::zero_value(xjson::Type::String),
        ""
    );
}

TEST(ZeroValue, Boolean) {
    ASSERT_EQ(
        xjson::zero_value(xjson::Type::Boolean),
        false
    );
}
