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

TEST(ResolveReadConverter, NumberToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::FLOAT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 42.5);
}

TEST(ResolveReadConverter, NumberToFloat32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::FLOAT32_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_FLOAT_EQ(std::get<float>(sv), 42.5f);
}

TEST(ResolveReadConverter, NumberToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<int64_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToInt32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT32_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<int32_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToInt16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT16_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<int16_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToInt8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT8_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<int8_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToUint64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<uint64_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToUint32) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT32_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<uint32_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToUint16) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT16_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<uint16_t>(sv), 7);
}

TEST(ResolveReadConverter, NumberToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<uint8_t>(sv), 7);
}

// --- Strict truncation ---

TEST(ResolveReadConverter, NumberToInt64NonStrictTruncation) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T, {.strict = false})
    );
    const auto sv = ASSERT_NIL_P(converter(json(3.7)));
    ASSERT_EQ(std::get<int64_t>(sv), 3);
}

TEST(ResolveReadConverter, NumberToInt64StrictTruncationError) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::INT64_T, {.strict = true})
    );
    auto [sv, write_err] = converter(json(3.7));
    ASSERT_OCCURRED_AS(write_err, xjson::TRUNCATION_ERROR);
}

// TEST(ResolveReadConverter, NumberToUint8NonStrictOverflow) {
//     const auto converter = ASSERT_NIL_P(
//         xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, {.strict = false})
//     );
//     const auto sv = ASSERT_NIL_P(converter(json(300)));
//     ASSERT_EQ(std::get<uint8_t>(sv), 255);
// }

// TEST(ResolveReadConverter, NumberToUint8NonStrictUnderflow) {
//     const auto converter = ASSERT_NIL_P(
//         xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, {.strict = false})
//     );
//     const auto sv = ASSERT_NIL_P(converter(json(-1)));
//     ASSERT_EQ(std::get<uint8_t>(sv), 0);
// }

TEST(ResolveReadConverter, NumberToUint8StrictOverflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, {.strict = true})
    );
    auto [sv, write_err] = converter(json(300));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERROR);
}

TEST(ResolveReadConverter, NumberToUint8StrictUnderflow) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::UINT8_T, {.strict = true})
    );
    auto [sv, write_err] = converter(json(-1));
    ASSERT_OCCURRED_AS(write_err, xjson::OVERFLOW_ERROR);
}

TEST(ResolveReadConverter, NumberToStringDecimal) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::STRING_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(42.5)));
    ASSERT_EQ(std::get<std::string>(sv), "42.5");
}

TEST(ResolveReadConverter, NumberToStringInteger) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::STRING_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(7)));
    ASSERT_EQ(std::get<std::string>(sv), "7");
}

TEST(ResolveReadConverter, StringToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::String, telem::STRING_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json("hello")));
    ASSERT_EQ(std::get<std::string>(sv), "hello");
}

TEST(ResolveReadConverter, StringToFloat64Error) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::String, telem::FLOAT64_T),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ResolveReadConverter, BooleanTrueToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::INT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(true)));
    ASSERT_EQ(std::get<int64_t>(sv), 1);
}

TEST(ResolveReadConverter, BooleanFalseToInt64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::INT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(std::get<int64_t>(sv), 0);
}

TEST(ResolveReadConverter, BooleanTrueToFloat64) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::FLOAT64_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(true)));
    ASSERT_DOUBLE_EQ(std::get<double>(sv), 1.0);
}

TEST(ResolveReadConverter, BooleanFalseToUint8) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::UINT8_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(std::get<uint8_t>(sv), 0);
}

TEST(ResolveReadConverter, BooleanTrueToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::STRING_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(true)));
    ASSERT_EQ(std::get<std::string>(sv), "true");
}

TEST(ResolveReadConverter, BooleanFalseToString) {
    const auto converter = ASSERT_NIL_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::STRING_T)
    );
    const auto sv = ASSERT_NIL_P(converter(json(false)));
    ASSERT_EQ(std::get<std::string>(sv), "false");
}

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

TEST(ResolveReadConverter, NumberToBytesError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Number, telem::BYTES_T),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ResolveReadConverter, NumberToTimestampNanosecond) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::Number, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::UnixNanosecond}
    ));
    const int64_t value =1000000000000000000;
    const auto sv = ASSERT_NIL_P(converter(json(value)));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(value));
}

TEST(ResolveReadConverter, NumberToTimestampMicrosecond) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::Number, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::UnixMicrosecond}
    ));
    const auto sv = ASSERT_NIL_P(converter(json(int64_t(1000000))));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000));
}

TEST(ResolveReadConverter, NumberToTimestampMillisecond) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::Number, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::UnixMillisecond}
    ));
    const auto sv = ASSERT_NIL_P(converter(json(int64_t(1500))));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1500000000));
}

TEST(ResolveReadConverter, NumberToTimestampSecondInt) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::Number, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::UnixSecondInt}
    ));
    const auto sv = ASSERT_NIL_P(converter(json(int64_t(1000000000))));
    ASSERT_EQ(
        std::get<telem::TimeStamp>(sv),
        telem::TimeStamp(1000000000000000000)
    );
}

TEST(ResolveReadConverter, NumberToTimestampSecondFloat) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::Number, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::UnixSecondFloat}
    ));
    const auto sv = ASSERT_NIL_P(converter(json(1.5)));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1500000000));
}

TEST(ResolveReadConverter, NumberToTimestampISO8601Error) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(
            xjson::Type::Number, telem::TIMESTAMP_T,
            {.time_format = xjson::TimeFormat::ISO8601}
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ResolveReadConverter, StringToTimestampUnixNanosecondError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(
            xjson::Type::String, telem::TIMESTAMP_T,
            {.time_format = xjson::TimeFormat::UnixNanosecond}
        ),
        xjson::UNSUPPORTED_ERROR
    );
}

TEST(ResolveReadConverter, StringToTimestampISO8601) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40.5Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000500000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601WithOffset) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    // 02:46:40.5+01:00 = 01:46:40.5 UTC = same instant as .5Z
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T02:46:40.5+01:00")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000500000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601WithoutSubSecond) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

// --- ISO8601 parsing edge cases ---

TEST(ResolveReadConverter, StringToTimestampISO8601Epoch) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("1970-01-01T00:00:00Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(0));
}

TEST(ResolveReadConverter, StringToTimestampISO8601SubSecondThreeDigits) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40.123Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000123000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601SubSecondSixDigits) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40.123456Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000123456000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601SubSecondNineDigits) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40.123456789Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000123456789));
}

TEST(ResolveReadConverter, StringToTimestampISO8601ExcessDigitsTruncated) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40.1234567891111Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000123456789));
}

TEST(ResolveReadConverter, StringToTimestampISO8601NegativeOffset) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    // 00:46:40-01:00 = 01:46:40 UTC
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T00:46:40-01:00")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601NegativeOffsetCrossesMidnight) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    // 23:46:40-05:00 on Sep 8 = 04:46:40 UTC on Sep 9
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-08T23:46:40-05:00")));
    const int64_t expected = 1000000000000000000 + int64_t(3) * 3600 * 1000000000;
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(expected));
}

TEST(ResolveReadConverter, StringToTimestampISO8601ExplicitPlusZero) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40+00:00")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601MinusZero) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40-00:00")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601LowercaseZ) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T01:46:40z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601LowercaseT) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09t01:46:40Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601SpaceSeparator) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09 01:46:40Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601LeapYearFeb29) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("2000-02-29T00:00:00Z")));
    ASSERT_EQ(
        std::get<telem::TimeStamp>(sv),
        telem::TimeStamp(int64_t(11016) * 86400 * 1000000000)
    );
}

TEST(ResolveReadConverter, StringToTimestampISO8601PreEpoch) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("1969-12-31T23:59:59Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(-1000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601PreEpochWithFraction) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    const auto sv = ASSERT_NIL_P(converter(json("1969-12-31T23:59:59.5Z")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(-500000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601HalfHourOffset) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    // India: +05:30
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T07:16:40+05:30")));
    ASSERT_EQ(std::get<telem::TimeStamp>(sv), telem::TimeStamp(1000000000000000000));
}

TEST(ResolveReadConverter, StringToTimestampISO8601LeapSecondAllowed) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    // second=60 is allowed per RFC 3339 for leap seconds
    const auto sv = ASSERT_NIL_P(converter(json("2001-09-09T23:59:60Z")));
    (void)sv; // just verify it parsed without error
}

// --- Invalid ISO8601 inputs ---

TEST(ResolveReadConverter, StringToTimestampISO8601EmptyString) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json(""));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601TooShort) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:4"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601MissingTimezone) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:40"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601BadSeparator) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09X01:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601NonDigitYear) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("20X1-09-09T01:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601MonthZero) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-00-09T01:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601MonthThirteen) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-13-09T01:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601DayZero) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-00T01:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601HourTwentyFour) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T24:46:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601MinuteSixty) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:60:40Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601SecondSixtyOne) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:61Z"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601InvalidTimezoneChar) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:40X"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601OffsetMissingColon) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:40+0100"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601OffsetTruncated) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09T01:46:40+01"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, StringToTimestampISO8601JustDate) {
    const auto converter = ASSERT_NIL_P(xjson::resolve_read_converter(
        xjson::Type::String, telem::TIMESTAMP_T,
        {.time_format = xjson::TimeFormat::ISO8601}
    ));
    auto [sv, err] = converter(json("2001-09-09"));
    ASSERT_OCCURRED_AS(err, xjson::UNSUPPORTED_ERROR);
}

TEST(ResolveReadConverter, BooleanToTimestampError) {
    ASSERT_OCCURRED_AS_P(
        xjson::resolve_read_converter(xjson::Type::Boolean, telem::TIMESTAMP_T),
        xjson::UNSUPPORTED_ERROR
    );
}

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
