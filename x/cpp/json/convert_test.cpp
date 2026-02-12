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

#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

using json = nlohmann::json;

// --- Number → Numeric ---

TEST(ToSampleValue, NumberToFloat64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::FLOAT64_T)
    );
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 42.5);
}

TEST(ToSampleValue, NumberToFloat32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::FLOAT32_T)
    );
    ASSERT_FLOAT_EQ(std::get<float>(sv), 42.5f);
}

TEST(ToSampleValue, NumberToInt64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::INT64_T)
    );
    ASSERT_EQ(std::get<int64_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::INT32_T)
    );
    ASSERT_EQ(std::get<int32_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt16) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::INT16_T)
    );
    ASSERT_EQ(std::get<int16_t>(sv), 7);
}

TEST(ToSampleValue, NumberToInt8) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::INT8_T)
    );
    ASSERT_EQ(std::get<int8_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint64) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::UINT64_T)
    );
    ASSERT_EQ(std::get<uint64_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint32) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::UINT32_T)
    );
    ASSERT_EQ(std::get<uint32_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint16) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::UINT16_T)
    );
    ASSERT_EQ(std::get<uint16_t>(sv), 7);
}

TEST(ToSampleValue, NumberToUint8) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::UINT8_T)
    );
    ASSERT_EQ(std::get<uint8_t>(sv), 7);
}

// --- Strict truncation ---

TEST(ToSampleValue, NumberToInt64NonStrictTruncation) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(3.7), x::telem::INT64_T, {.strict = false})
    );
    ASSERT_EQ(std::get<int64_t>(sv), 3);
}

TEST(ToSampleValue, NumberToInt64StrictTruncationError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(3.7), x::telem::INT64_T, {.strict = true}),
        x::json::TRUNCATION_ERROR
    );
}

TEST(ToSampleValue, NumberToUint8StrictOverflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(300), x::telem::UINT8_T, {.strict = true}),
        x::json::OVERFLOW_ERROR
    );
}

TEST(ToSampleValue, NumberToUint8StrictUnderflow) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(-1), x::telem::UINT8_T, {.strict = true}),
        x::json::OVERFLOW_ERROR
    );
}

// --- Number/Boolean/String → String ---

TEST(ToSampleValue, NumberToStringDecimal) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(42.5), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "42.5");
}

TEST(ToSampleValue, NumberToStringInteger) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json(7), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "7");
}

TEST(ToSampleValue, StringToString) {
    const auto sv = ASSERT_NIL_P(
        x::json::to_sample_value(json("hello"), x::telem::STRING_T)
    );
    ASSERT_EQ(std::get<std::string>(sv), "hello");
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

// --- String → Numeric (unsupported) ---

TEST(ToSampleValue, StringToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json("hello"), x::telem::FLOAT64_T),
        x::json::UNSUPPORTED_ERROR
    );
}

// --- Boolean → Numeric ---

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

// --- Unsupported target types ---

TEST(ToSampleValue, NumberToUUIDError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(42), x::telem::UUID_T),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, NumberToJSONError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(42), x::telem::JSON_T),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, NumberToBytesError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(42), x::telem::BYTES_T),
        x::json::UNSUPPORTED_ERROR
    );
}

// --- Timestamps ---

TEST(ToSampleValue, NumberToTimestampNanosecond) {
    const int64_t value = 1000000000000000000;
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(value), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixNanosecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(value));
}

TEST(ToSampleValue, NumberToTimestampMicrosecond) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(int64_t(1000000)), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixMicrosecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1000000000));
}

TEST(ToSampleValue, NumberToTimestampMillisecond) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(int64_t(1500)), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixMillisecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, NumberToTimestampSecondInteger) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(int64_t(1000000000)), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixSecond}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, NumberToTimestampSecondDecimal) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(1.5), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixSecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500000000));
}

TEST(ToSampleValue, NumberToTimestampMillisecondDecimal) {
    // 1500.5 ms = 1500500000 ns
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(1500.5), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixMillisecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1500500000));
}

TEST(ToSampleValue, NumberToTimestampMicrosecondDecimal) {
    // 1000000.5 us = 1000000500 ns
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json(1000000.5), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::UnixMicrosecond}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(1000000500));
}

TEST(ToSampleValue, NumberToTimestampISO8601Error) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(42), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampUnixNanosecondError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::UnixNanosecond}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40.5Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000500000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601WithOffset) {
    // 02:46:40.5+01:00 = 01:46:40.5 UTC
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T02:46:40.5+01:00"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000500000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601WithoutSubSecond) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

// --- ISO8601 parsing edge cases ---

TEST(ToSampleValue, StringToTimestampISO8601Epoch) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("1970-01-01T00:00:00Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(0));
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondThreeDigits) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40.123Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondSixDigits) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40.123456Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SubSecondNineDigits) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40.123456789Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456789)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601ExcessDigitsTruncated) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40.1234567891111Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000123456789)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NegativeOffset) {
    // 00:46:40-01:00 = 01:46:40 UTC
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T00:46:40-01:00"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NegativeOffsetCrossesMidnight) {
    // 23:46:40-05:00 on Sep 8 = 04:46:40 UTC on Sep 9
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-08T23:46:40-05:00"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    const int64_t expected =
        1000000000000000000 + int64_t(3) * 3600 * 1000000000;
    ASSERT_EQ(std::get<x::telem::TimeStamp>(sv), x::telem::TimeStamp(expected));
}

TEST(ToSampleValue, StringToTimestampISO8601ExplicitPlusZero) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40+00:00"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MinusZero) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40-00:00"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LowercaseZ) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T01:46:40z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LowercaseT) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09t01:46:40Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SpaceSeparator) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09 01:46:40Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LeapYearFeb29) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2000-02-29T00:00:00Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(int64_t(11016) * 86400 * 1000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601PreEpoch) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("1969-12-31T23:59:59Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(-1000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601PreEpochWithFraction) {
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("1969-12-31T23:59:59.5Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(-500000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601HalfHourOffset) {
    // India: +05:30
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T07:16:40+05:30"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    ASSERT_EQ(
        std::get<x::telem::TimeStamp>(sv),
        x::telem::TimeStamp(1000000000000000000)
    );
}

TEST(ToSampleValue, StringToTimestampISO8601LeapSecondAllowed) {
    // second=60 is allowed per RFC 3339 for leap seconds
    const auto sv = ASSERT_NIL_P(x::json::to_sample_value(
        json("2001-09-09T23:59:60Z"), x::telem::TIMESTAMP_T,
        {.time_format = x::json::TimeFormat::ISO8601}
    ));
    (void)sv;
}

// --- Invalid ISO8601 inputs ---

TEST(ToSampleValue, StringToTimestampISO8601EmptyString) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json(""), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601TooShort) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:4"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MissingTimezone) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601BadSeparator) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09X01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601NonDigitYear) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("20X1-09-09T01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MonthZero) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-00-09T01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MonthThirteen) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-13-09T01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601DayZero) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-00T01:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601HourTwentyFour) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T24:46:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601MinuteSixty) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:60:40Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601SecondSixtyOne) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:61Z"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601InvalidTimezoneChar) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40X"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601OffsetMissingColon) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40+0100"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601OffsetTruncated) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09T01:46:40+01"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, StringToTimestampISO8601JustDate) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(
            json("2001-09-09"), x::telem::TIMESTAMP_T,
            {.time_format = x::json::TimeFormat::ISO8601}
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(ToSampleValue, BooleanToTimestampError) {
    ASSERT_OCCURRED_AS_P(
        x::json::to_sample_value(json(true), x::telem::TIMESTAMP_T),
        x::json::UNSUPPORTED_ERROR
    );
}

// ==================== from_sample_value ====================

TEST(FromSampleValue, Float64ToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(42.5), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(42.5));
}

TEST(FromSampleValue, Float64WithoutDecimalToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(42.0), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(42));
}

TEST(FromSampleValue, Float64WithLongDecimalToNumber) {
    const auto value = 1.2345689012;
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(value), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Int64ToNumber) {
    const auto value = int64_t(-743984);
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(value), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(value));
}

TEST(FromSampleValue, Uint8ZeroToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(0));
}

TEST(FromSampleValue, Uint8ToNumber) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)), x::json::Type::Number
        )
    );
    ASSERT_EQ(result, json(255));
}

TEST(FromSampleValue, Float64ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(42.5), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("42.5"));
}

TEST(FromSampleValue, Float64WithLongDecimalToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(1.2345689012), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("1.2345689012"));
}

TEST(FromSampleValue, Float64WithoutDecimalToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(42.0), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("42"));
}

TEST(FromSampleValue, Int64ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(7)), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("7"));
}

TEST(FromSampleValue, Int64NegativeToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(-743984)), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("-743984"));
}

TEST(FromSampleValue, Uint8ToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("255"));
}

TEST(FromSampleValue, Uint8ZeroToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("0"));
}

TEST(FromSampleValue, Int64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(0)), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Int64OneToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(1)), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Int64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(int64_t(-743984)), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(0.0), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Float64PositiveToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(42.5), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Float64NegativeToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(-743984.0), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, Uint8ZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(0)), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(false));
}

TEST(FromSampleValue, Uint8NonZeroToBoolean) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(uint8_t(255)), x::json::Type::Boolean
        )
    );
    ASSERT_EQ(result, json(true));
}

TEST(FromSampleValue, StringToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json("hello"));
}

TEST(FromSampleValue, StringWithZeroLengthToString) {
    const auto result = ASSERT_NIL_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("")), x::json::Type::String
        )
    );
    ASSERT_EQ(result, json(""));
}

TEST(FromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")), x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(std::string("hello")), x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToNumberError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToStringError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(FromSampleValue, TimeStampToBooleanError) {
    ASSERT_OCCURRED_AS_P(
        x::json::from_sample_value(
            x::telem::SampleValue(x::telem::TimeStamp(1000000000)),
            x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
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
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond)
            .get<double>(),
        1000000000000000.0
    );
}

TEST(FromTimestamp, UnixMicrosecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0), x::json::TimeFormat::UnixMicrosecond
        ).get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixMicrosecondPreservesSubMicrosecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000500));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond)
            .get<double>(),
        1000000.5
    );
}

TEST(FromTimestamp, UnixMicrosecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMicrosecond)
            .get<double>(),
        -1500000.0
    );
}

TEST(FromTimestamp, UnixMillisecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond)
            .get<double>(),
        1000000000000.0
    );
}

TEST(FromTimestamp, UnixMillisecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0), x::json::TimeFormat::UnixMillisecond
        ).get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixMillisecondPreservesSubMillisecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1500500000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond)
            .get<double>(),
        1500.5
    );
}

TEST(FromTimestamp, UnixMillisecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixMillisecond)
            .get<double>(),
        -1500.0
    );
}

TEST(FromTimestamp, UnixSecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000000000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond)
            .get<double>(),
        1000000000.0
    );
}

TEST(FromTimestamp, UnixSecondZero) {
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0), x::json::TimeFormat::UnixSecond
        ).get<double>(),
        0.0
    );
}

TEST(FromTimestamp, UnixSecondPreservesSubSecond) {
    const auto ts = x::telem::TimeStamp(int64_t(1000000000500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond)
            .get<double>(),
        1000000000.5
    );
}

TEST(FromTimestamp, UnixSecondNegative) {
    const auto ts = x::telem::TimeStamp(int64_t(-1500000000));
    ASSERT_DOUBLE_EQ(
        x::json::from_timestamp(ts, x::json::TimeFormat::UnixSecond)
            .get<double>(),
        -1.5
    );
}

TEST(FromTimestamp, ISO8601Epoch) {
    ASSERT_EQ(
        x::json::from_timestamp(
            x::telem::TimeStamp(0), x::json::TimeFormat::ISO8601
        ),
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

// ==================== check_from_sample_value ====================

TEST(CheckFromSampleValue, Float64ToNumberOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::FLOAT64_T, x::json::Type::Number
    ));
}

TEST(CheckFromSampleValue, Float64ToStringOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::FLOAT64_T, x::json::Type::String
    ));
}

TEST(CheckFromSampleValue, Float64ToBooleanOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::FLOAT64_T, x::json::Type::Boolean
    ));
}

TEST(CheckFromSampleValue, Int64ToNumberOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::INT64_T, x::json::Type::Number
    ));
}

TEST(CheckFromSampleValue, Int64ToStringOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::INT64_T, x::json::Type::String
    ));
}

TEST(CheckFromSampleValue, Int64ToBooleanOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::INT64_T, x::json::Type::Boolean
    ));
}

TEST(CheckFromSampleValue, Uint8ToNumberOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::UINT8_T, x::json::Type::Number
    ));
}

TEST(CheckFromSampleValue, Uint8ToStringOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::UINT8_T, x::json::Type::String
    ));
}

TEST(CheckFromSampleValue, Uint8ToBooleanOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::UINT8_T, x::json::Type::Boolean
    ));
}

TEST(CheckFromSampleValue, StringToStringOK) {
    ASSERT_FALSE(x::json::check_from_sample_value(
        x::telem::STRING_T, x::json::Type::String
    ));
}

TEST(CheckFromSampleValue, StringToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::STRING_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, StringToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::STRING_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::TIMESTAMP_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::TIMESTAMP_T, x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, TimestampToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::TIMESTAMP_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UUID_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UUID_T, x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UUIDToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UUID_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::BYTES_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::BYTES_T, x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, BytesToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::BYTES_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UNKNOWN_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UNKNOWN_T, x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, UnknownToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::UNKNOWN_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToNumberError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::JSON_T, x::json::Type::Number
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToStringError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::JSON_T, x::json::Type::String
        ),
        x::json::UNSUPPORTED_ERROR
    );
}

TEST(CheckFromSampleValue, JSONToBooleanError) {
    ASSERT_OCCURRED_AS(
        x::json::check_from_sample_value(
            x::telem::JSON_T, x::json::Type::Boolean
        ),
        x::json::UNSUPPORTED_ERROR
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
