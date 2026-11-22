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
#include <string_view>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/field.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

namespace x::json {
namespace {
constexpr std::int64_t PAST_2_TO_53 = (std::int64_t{1} << 53) + 1;
constexpr auto INT64_MIN_VALUE = std::numeric_limits<std::int64_t>::min();
constexpr auto INT64_MAX_VALUE = std::numeric_limits<std::int64_t>::max();
constexpr auto UINT64_MAX_VALUE = std::numeric_limits<std::uint64_t>::max();
constexpr double NaN = std::numeric_limits<double>::quiet_NaN();

std::vector<Enum> on_off() {
    return {{"OFF", 0.0}, {"ON", 1.0}};
}

/// @brief returns the series that a Decoder for field appends raw to.
telem::Series decode(Field field, const std::string_view raw) {
    telem::Series dst(field.data_type, 2);
    auto d = ASSERT_NIL_P(Decoder::create(std::move(field)));
    EXPECT_NIL(d.append(raw, dst));
    EXPECT_EQ(dst.size(), 1);
    return dst;
}

/// @brief returns the error that a Decoder for field gives for raw, checking that it
/// appends nothing.
errors::Error decode_error(Field field, const std::string_view raw) {
    telem::Series dst(field.data_type, 2);
    auto d = ASSERT_NIL_P(Decoder::create(std::move(field)));
    auto err = d.append(raw, dst);
    EXPECT_EQ(dst.size(), 0);
    return err;
}

/// @brief returns the error that creating a Decoder for field gives.
errors::Error create_error(Field field) {
    return Decoder::create(std::move(field)).second;
}

/// @brief returns the series that a Decoder for field appends v to.
telem::Series decode_float(Field field, const double v) {
    telem::Series dst(field.data_type, 2);
    const auto d = ASSERT_NIL_P(Decoder::create(std::move(field)));
    EXPECT_NIL(d.append_float(v, dst));
    return dst;
}

/// @brief returns the error that a Decoder for field gives for v.
errors::Error decode_float_error(Field field, const double v) {
    telem::Series dst(field.data_type, 2);
    const auto d = ASSERT_NIL_P(Decoder::create(std::move(field)));
    return d.append_float(v, dst);
}
}

/// @brief it should convert a float.
TEST(Decoder, ConvertsFloat) {
    ASSERT_EQ(decode({.data_type = telem::FLOAT64_T}, "1.5").at<double>(0), 1.5);
}

/// @brief it should convert a float32.
TEST(Decoder, ConvertsFloat32) {
    ASSERT_EQ(decode({.data_type = telem::FLOAT32_T}, "0.25").at<float>(0), 0.25F);
}

/// @brief it should convert an int64 past the precision of a double.
TEST(Decoder, ConvertsInt64PastFloatPrecision) {
    ASSERT_EQ(
        decode({.data_type = telem::INT64_T}, "9007199254740993").at<std::int64_t>(0),
        PAST_2_TO_53
    );
}

/// @brief it should convert the largest uint64.
TEST(Decoder, ConvertsLargestUint64) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT64_T}, "18446744073709551615")
            .at<std::uint64_t>(0),
        UINT64_MAX_VALUE
    );
}

/// @brief it should convert a negative int8.
TEST(Decoder, ConvertsNegativeInt8) {
    ASSERT_EQ(
        decode({.data_type = telem::INT8_T}, "-2").at<std::int8_t>(0),
        std::int8_t{-2}
    );
}

/// @brief it should convert a whole float into an integer.
TEST(Decoder, ConvertsWholeFloatToInteger) {
    ASSERT_EQ(
        decode({.data_type = telem::INT16_T}, "-3.0").at<std::int16_t>(0),
        std::int16_t{-3}
    );
}

/// @brief it should convert a whole decimal past the precision of a double.
TEST(Decoder, ConvertsWholeDecimalPastFloatPrecision) {
    ASSERT_EQ(
        decode({.data_type = telem::INT64_T}, "9007199254740993.0").at<std::int64_t>(0),
        PAST_2_TO_53
    );
}

/// @brief it should convert an exponent past the precision of a double.
TEST(Decoder, ConvertsExponentPastFloatPrecision) {
    ASSERT_EQ(
        decode({.data_type = telem::INT64_T}, "9007199254740993e0").at<std::int64_t>(0),
        PAST_2_TO_53
    );
}

/// @brief it should convert a whole number with a negative exponent.
TEST(Decoder, ConvertsNegativeExponentToWholeNumber) {
    ASSERT_EQ(decode({.data_type = telem::INT32_T}, "4000e-3").at<std::int32_t>(0), 4);
}

/// @brief it should convert the largest uint64 with an exponent.
TEST(Decoder, ConvertsLargestUint64WithExponent) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT64_T}, "1.8446744073709551615e19")
            .at<std::uint64_t>(0),
        UINT64_MAX_VALUE
    );
}

/// @brief it should convert the smallest int64 with a decimal point.
TEST(Decoder, ConvertsSmallestInt64WithDecimalPoint) {
    ASSERT_EQ(
        decode({.data_type = telem::INT64_T}, "-9223372036854775808.0")
            .at<std::int64_t>(0),
        INT64_MIN_VALUE
    );
}

/// @brief it should convert a negative zero into an unsigned integer.
TEST(Decoder, ConvertsNegativeZeroToUnsigned) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT8_T}, "-0.0").at<std::uint8_t>(0),
        std::uint8_t{0}
    );
}

/// @brief it should convert true into a number.
TEST(Decoder, ConvertsTrueToNumber) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT8_T}, "true").at<std::uint8_t>(0),
        std::uint8_t{1}
    );
}

/// @brief it should convert a number other than 0 into true.
TEST(Decoder, ConvertsNumberToBoolean) {
    ASSERT_EQ(
        decode({.data_type = telem::BOOLEAN_T}, "2").at<std::uint8_t>(0),
        std::uint8_t{1}
    );
}

/// @brief it should convert an enum label.
TEST(Decoder, ConvertsEnumLabel) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT8_T, .enums = on_off()}, R"("ON")")
            .at<std::uint8_t>(0),
        std::uint8_t{1}
    );
}

/// @brief it should convert an enum label that holds an escape.
TEST(Decoder, ConvertsEscapedEnumLabel) {
    ASSERT_EQ(
        decode({.data_type = telem::UINT8_T, .enums = on_off()}, R"("\u004fN")")
            .at<std::uint8_t>(0),
        std::uint8_t{1}
    );
}

/// @brief it should convert an int64 enum past 2^53.
TEST(Decoder, ConvertsInt64EnumPast2To53) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::INT64_T,
             .enums = {{"BIG", std::int64_t{PAST_2_TO_53}}}},
            R"("BIG")"
        )
            .at<std::int64_t>(0),
        PAST_2_TO_53
    );
}

/// @brief it should convert the largest uint64 enum.
TEST(Decoder, ConvertsLargestUint64Enum) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::UINT64_T, .enums = {{"MAX", UINT64_MAX_VALUE}}},
            R"("MAX")"
        )
            .at<std::uint64_t>(0),
        UINT64_MAX_VALUE
    );
}

/// @brief it should convert a boolean enum into a number.
TEST(Decoder, ConvertsBooleanEnumToNumber) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::INT8_T, .enums = {{"ON", std::uint8_t{1}}}},
            R"("ON")"
        )
            .at<std::int8_t>(0),
        std::int8_t{1}
    );
}

/// @brief it should convert an enum label whose number is outside the range.
TEST(Decoder, ConvertsEnumOutsideRange) {
    const auto s = decode(
        {.data_type = telem::FLOAT64_T,
         .enums = {{"OFF", -1.0}, {"FAULT", NaN}},
         .range = telem::UINT64_T},
        R"("OFF")"
    );
    ASSERT_EQ(s.at<double>(0), -1.0);
}

/// @brief it should convert a NaN enum label in an integer range.
TEST(Decoder, ConvertsNaNEnumInRange) {
    const auto s = decode(
        {.data_type = telem::FLOAT64_T,
         .enums = {{"FAULT", NaN}},
         .range = telem::INT64_T},
        R"("FAULT")"
    );
    ASSERT_TRUE(std::isnan(s.at<double>(0)));
}

/// @brief it should reject a repeated name in a large JSON object.
TEST(Decoder, RejectsRepeatedNameInLargeJSONObject) {
    std::string raw = "{";
    for (int i = 0; i < 50000; i++)
        raw += "\"" + std::to_string(i) + "\":0,";
    raw += R"("7":0})";
    const auto err = decode_error({.data_type = telem::JSON_T}, raw);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, R"(duplicate object member name "7")");
}

/// @brief it should convert an integer in range into a float.
TEST(Decoder, ConvertsIntegerInRangeToFloat) {
    ASSERT_EQ(
        decode({.data_type = telem::FLOAT64_T, .range = telem::INT64_T}, "-3")
            .at<double>(0),
        -3.0
    );
}

/// @brief it should convert a whole decimal in range into a float32.
TEST(Decoder, ConvertsWholeDecimalInRangeToFloat32) {
    ASSERT_EQ(
        decode({.data_type = telem::FLOAT32_T, .range = telem::UINT64_T}, "7.0")
            .at<float>(0),
        7.0F
    );
}

/// @brief it should convert true in an unsigned range.
TEST(Decoder, ConvertsTrueInUnsignedRange) {
    ASSERT_EQ(
        decode({.data_type = telem::FLOAT64_T, .range = telem::UINT64_T}, "true")
            .at<double>(0),
        1.0
    );
}

/// @brief it should convert an ISO 8601 timestamp.
TEST(Decoder, ConvertsISO8601) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("2026-01-02T03:04:05.000000006Z")"
        )
            .at<std::int64_t>(0),
        1767323045000000006
    );
}

/// @brief it should convert an ISO 8601 timestamp with a space and a lowercase zone.
TEST(Decoder, ConvertsISO8601WithSpaceAndLowercaseZone) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("2026-01-02 03:04:05z")"
        )
            .at<std::int64_t>(0),
        1767323045000000000
    );
}

/// @brief it should convert an ISO 8601 timestamp with a zone offset.
TEST(Decoder, ConvertsISO8601WithOffset) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("2026-01-02T03:04:05.5+01:00")"
        )
            .at<std::int64_t>(0),
        1767319445500000000
    );
}

/// @brief it should truncate an ISO 8601 fraction past the nanosecond.
TEST(Decoder, TruncatesISO8601PastNanoseconds) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("2026-01-02T03:04:05.1234567891Z")"
        )
            .at<std::int64_t>(0),
        1767323045123456789
    );
}

/// @brief it should convert the smallest ISO 8601 timestamp.
TEST(Decoder, ConvertsSmallestISO8601) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("1677-09-21T00:12:43.145224192Z")"
        )
            .at<std::int64_t>(0),
        INT64_MIN_VALUE
    );
}

/// @brief it should convert the largest ISO 8601 timestamp.
TEST(Decoder, ConvertsLargestISO8601) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
            R"("2262-04-11T23:47:16.854775807Z")"
        )
            .at<std::int64_t>(0),
        INT64_MAX_VALUE
    );
}

/// @brief it should convert Unix seconds with every digit to the nanosecond.
TEST(Decoder, ConvertsUnixSecondsToNanosecond) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::UnixSecond},
            "1767323045.123456789"
        )
            .at<std::int64_t>(0),
        1767323045123456789
    );
}

/// @brief it should convert Unix milliseconds in a string.
TEST(Decoder, ConvertsUnixMillisecondsInString) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T,
             .time_format = TimeFormat::UnixMillisecond},
            R"("1767323045123")"
        )
            .at<std::int64_t>(0),
        1767323045123000000
    );
}

/// @brief it should convert Unix microseconds with an exponent.
TEST(Decoder, ConvertsUnixMicrosecondsWithExponent) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T,
             .time_format = TimeFormat::UnixMicrosecond},
            "1.5e3"
        )
            .at<std::int64_t>(0),
        1500000
    );
}

/// @brief it should convert negative Unix seconds.
TEST(Decoder, ConvertsNegativeUnixSeconds) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::UnixSecond},
            "-1.5"
        )
            .at<std::int64_t>(0),
        -1500000000
    );
}

/// @brief it should truncate a fraction of a nanosecond.
TEST(Decoder, TruncatesFractionOfNanosecond) {
    ASSERT_EQ(
        decode(
            {.data_type = telem::TIMESTAMP_T,
             .time_format = TimeFormat::UnixNanosecond},
            "10.9"
        )
            .at<std::int64_t>(0),
        10
    );
}

/// @brief it should convert a NaN enum.
TEST(Decoder, ConvertsNaNEnum) {
    const auto s = decode(
        {.data_type = telem::FLOAT32_T, .enums = {{"FAULT", NaN}}},
        R"("FAULT")"
    );
    ASSERT_TRUE(std::isnan(s.at<float>(0)));
}

/// @brief it should keep the sign of a negative zero for a float.
TEST(Decoder, KeepsNegativeZeroForFloat) {
    ASSERT_TRUE(
        std::signbit(decode({.data_type = telem::FLOAT64_T}, "-0.0").at<double>(0))
    );
}

/// @brief it should convert a string enum.
TEST(Decoder, ConvertsStringEnum) {
    const auto s = decode(
        {.data_type = telem::STRING_T, .enums = {{"ON", "1"}}},
        R"("ON")"
    );
    ASSERT_EQ(s.strings(), std::vector<std::string>{"1"});
}

/// @brief it should keep a string that is not an enum label.
TEST(Decoder, KeepsStringThatIsNotEnumLabel) {
    const auto s = decode(
        {.data_type = telem::STRING_T, .enums = {{"ON", "1"}}},
        R"("OFF")"
    );
    ASSERT_EQ(s.strings(), std::vector<std::string>{"OFF"});
}

/// @brief it should decode the escapes of a string.
TEST(Decoder, ConvertsString) {
    const auto s = decode({.data_type = telem::STRING_T}, R"("a\nb")");
    ASSERT_EQ(s.strings(), std::vector<std::string>{"a\nb"});
}

/// @brief it should compact JSON but keep its spelling.
TEST(Decoder, CompactsJSON) {
    const auto s = decode(
        {.data_type = telem::JSON_T},
        R"({ "z" : 1.0, "a" : [ 1 , 2 ], "s": " {x} " })"
    );
    ASSERT_EQ(
        s.strings(),
        std::vector<std::string>{R"({"z":1.0,"a":[1,2],"s":" {x} "})"}
    );
}

/// @brief it should allow repeated names in sibling objects of JSON.
TEST(Decoder, AllowsRepeatedNamesInSiblingJSONObjects) {
    const auto s = decode({.data_type = telem::JSON_T}, R"([{"a":1},{"a":2}])");
    ASSERT_EQ(s.strings(), std::vector<std::string>{R"([{"a":1},{"a":2}])"});
}

/// @brief it should allow repeated strings in an array of JSON.
TEST(Decoder, AllowsRepeatedStringsInJSONArray) {
    const auto s = decode({.data_type = telem::JSON_T}, R"({"a":["a","a"],"b":"a"})");
    ASSERT_EQ(s.strings(), std::vector<std::string>{R"({"a":["a","a"],"b":"a"})"});
}

/// @brief it should keep a scalar as JSON as it is.
TEST(Decoder, KeepsScalarAsJSON) {
    const auto s = decode({.data_type = telem::JSON_T}, "1e999");
    ASSERT_EQ(s.strings(), std::vector<std::string>{"1e999"});
}

/// @brief it should convert a UUID.
TEST(Decoder, ConvertsUUID) {
    const auto id = ASSERT_NIL_P(
        uuid::UUID::parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
    );
    const auto s = decode({.data_type = telem::UUID_T}, "\"" + id.to_string() + "\"");
    ASSERT_EQ(s.uuids(), std::vector<uuid::UUID>{id});
}

/// @brief it should decode base64, including an escaped slash.
TEST(Decoder, ConvertsBase64) {
    const auto s = decode({.data_type = telem::BYTES_T}, R"("AP8\/")");
    ASSERT_EQ(s.strings(), std::vector<std::string>{std::string("\x00\xff\x3f", 3)});
}

/// @brief it should convert an empty string into empty bytes.
TEST(Decoder, ConvertsEmptyBytes) {
    const auto s = decode({.data_type = telem::BYTES_T}, R"("")");
    ASSERT_EQ(s.strings(), std::vector<std::string>{""});
}

/// @brief it should append samples one after another.
TEST(Decoder, AppendsSamplesInOrder) {
    auto d = ASSERT_NIL_P(Decoder::create({.data_type = telem::STRING_T}));
    telem::Series dst(telem::STRING_T, 0);
    ASSERT_NIL(d.append(R"("a")", dst));
    ASSERT_NIL(d.append(R"("bc")", dst));
    ASSERT_EQ(dst.strings(), (std::vector<std::string>{"a", "bc"}));
}

/// @brief it should reject a number for a string.
TEST(Decoder, RejectsNumberForString) {
    const auto err = decode_error({.data_type = telem::STRING_T}, "1");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a string, got a number");
}

/// @brief it should reject a string that is no enum label for a number.
TEST(Decoder, RejectsStringForNumber) {
    const auto err = decode_error({.data_type = telem::FLOAT64_T}, R"("1")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "unknown enum label \"1\"");
}

/// @brief it should reject null for a number.
TEST(Decoder, RejectsNullForNumber) {
    const auto err = decode_error({.data_type = telem::FLOAT64_T}, "null");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a number or boolean, got null");
}

/// @brief it should reject a number past an int8.
TEST(Decoder, RejectsNumberPastInt8) {
    const auto err = decode_error({.data_type = telem::INT8_T}, "128");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "128 is out of range for int8");
}

/// @brief it should reject a negative number for an unsigned integer.
TEST(Decoder, RejectsNegativeForUnsigned) {
    const auto err = decode_error({.data_type = telem::UINT8_T}, "-1");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "-1 is out of range for uint8");
}

/// @brief it should reject a fraction for an integer.
TEST(Decoder, RejectsFractionForInteger) {
    const auto err = decode_error({.data_type = telem::INT32_T}, "1.5");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1.5 is not an integer");
}

/// @brief it should reject a fraction that a double rounds away.
TEST(Decoder, RejectsFractionThatFloatRoundsAway) {
    const auto err = decode_error({.data_type = telem::INT64_T}, "9007199254740993.5");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "9007199254740993.5 is not an integer");
}

/// @brief it should reject a whole decimal past an int64.
TEST(Decoder, RejectsWholeDecimalPastInt64) {
    const auto err = decode_error({.data_type = telem::INT64_T}, "1e19");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "10000000000000000000 is out of range for int64");
}

/// @brief it should reject a number past a float32.
TEST(Decoder, RejectsFloatPastFloat32) {
    const auto err = decode_error({.data_type = telem::FLOAT32_T}, "1e39");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1e39 overflows float32");
}

/// @brief it should reject an object for a UUID.
TEST(Decoder, RejectsObjectForUUID) {
    const auto err = decode_error({.data_type = telem::UUID_T}, "{}");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a UUID string, got an object");
}

/// @brief it should reject text that is not a UUID.
TEST(Decoder, RejectsInvalidUUID) {
    const auto err = decode_error({.data_type = telem::UUID_T}, R"("x")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid UUID \"x\"");
}

/// @brief it should reject a number for an ISO 8601 timestamp.
TEST(Decoder, RejectsNumberForISO8601) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
        "1"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected an ISO 8601 string, got a number");
}

/// @brief it should reject text that is not ISO 8601.
TEST(Decoder, RejectsInvalidISO8601) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
        R"("now")"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid ISO 8601 timestamp \"now\"");
}

/// @brief it should reject a day past the end of its month.
TEST(Decoder, RejectsDayPastMonth) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
        R"("2026-02-29T00:00:00Z")"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid ISO 8601 timestamp \"2026-02-29T00:00:00Z\"");
}

/// @brief it should reject an ISO 8601 timestamp with no zone.
TEST(Decoder, RejectsISO8601WithoutZone) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
        R"("2026-01-02T03:04:05")"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid ISO 8601 timestamp \"2026-01-02T03:04:05\"");
}

/// @brief it should reject an ISO 8601 timestamp before the smallest timestamp.
TEST(Decoder, RejectsISO8601PastTimestamp) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::ISO8601},
        R"("1677-09-21T00:12:43.145224191Z")"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "\"1677-09-21T00:12:43.145224191Z\" is out of range for a timestamp"
    );
}

/// @brief it should reject a boolean for Unix seconds.
TEST(Decoder, RejectsBooleanForUnixSeconds) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::UnixSecond},
        "true"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a number, got a boolean");
}

/// @brief it should reject text that is not a number for Unix seconds.
TEST(Decoder, RejectsWordForUnixSeconds) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::UnixSecond},
        R"("soon")"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid number \"soon\"");
}

/// @brief it should reject Unix seconds past the range of a timestamp.
TEST(Decoder, RejectsUnixSecondsPastTimestamp) {
    const auto err = decode_error(
        {.data_type = telem::TIMESTAMP_T, .time_format = TimeFormat::UnixSecond},
        "1e12"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1e12 is out of range for a unix_sec timestamp");
}

/// @brief it should reject JSON that repeats a member name.
TEST(Decoder, RejectsRepeatedJSONMember) {
    const auto err = decode_error({.data_type = telem::JSON_T}, R"({"a":1,"a":2})");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "duplicate object member name \"a\"");
}

/// @brief it should reject JSON that repeats a member name in a nested object.
TEST(Decoder, RejectsRepeatedNestedJSONMember) {
    const auto err = decode_error(
        {.data_type = telem::JSON_T},
        R"({"a":[{"b":1,"b":2}]})"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "duplicate object member name \"b\"");
}

/// @brief it should reject a number for bytes.
TEST(Decoder, RejectsNumberForBytes) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, "1");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "expected a base64 string, got a number");
}

/// @brief it should reject text that is not base64.
TEST(Decoder, RejectsInvalidBase64) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, R"("a*b=")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "not canonical base64");
}

/// @brief it should reject base64 without padding.
TEST(Decoder, RejectsUnpaddedBase64) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, R"("YQ")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "not canonical base64");
}

/// @brief it should reject base64 with padding bits set.
TEST(Decoder, RejectsBase64PaddingBits) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, R"("YR==")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "not canonical base64");
}

/// @brief it should reject base64 with a line feed.
TEST(Decoder, RejectsBase64LineFeed) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, R"("Y\nQ==")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "base64 must not hold a line break");
}

/// @brief it should reject base64 with a carriage return.
TEST(Decoder, RejectsBase64CarriageReturn) {
    const auto err = decode_error({.data_type = telem::BYTES_T}, R"("YQ==\r")");
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "base64 must not hold a line break");
}

/// @brief it should reject a fraction that a double rounds away from an integer range.
TEST(Decoder, RejectsRoundedFractionInRange) {
    const auto err = decode_error(
        {.data_type = telem::FLOAT64_T, .range = telem::INT64_T},
        "9007199254740993.5"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "9007199254740993.5 is not an integer");
}

/// @brief it should reject a negative number outside an unsigned range.
TEST(Decoder, RejectsNegativeOutsideUnsignedRange) {
    const auto err = decode_error(
        {.data_type = telem::INT64_T, .range = telem::UINT64_T},
        "-1"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "-1 is out of range for uint64");
}

/// @brief it should reject a whole decimal past an integer range.
TEST(Decoder, RejectsWholeDecimalPastRange) {
    const auto err = decode_error(
        {.data_type = telem::FLOAT64_T, .range = telem::INT64_T},
        "1e19"
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "10000000000000000000 is out of range for int64");
}

/// @brief it should reject a data type that no JSON value converts to.
TEST(Decoder, RejectsUnknownDataType) {
    const auto err = create_error({.data_type = telem::UNKNOWN_T});
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "cannot convert a JSON value to " + telem::UNKNOWN_T.name());
}

/// @brief it should reject a range that is not an integer data type.
TEST(Decoder, RejectsFloatRange) {
    const auto err = create_error(
        {.data_type = telem::FLOAT64_T, .range = telem::FLOAT32_T}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "range float32 is not an integer data type");
}

/// @brief it should reject a fraction enum for an integer.
TEST(Decoder, RejectsFractionEnumForInteger) {
    const auto err = create_error(
        {.data_type = telem::UINT8_T, .enums = {{"HALF", 0.5}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"HALF\": 0.5 is not an integer");
}

/// @brief it should reject an enum past the data type.
TEST(Decoder, RejectsEnumPastDataType) {
    const auto err = create_error(
        {.data_type = telem::INT8_T, .enums = {{"BIG", std::int64_t{128}}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"BIG\": 128 is out of range for int8");
}

/// @brief it should reject a NaN enum for an integer.
TEST(Decoder, RejectsNaNEnumForInteger) {
    const auto err = create_error(
        {.data_type = telem::INT32_T, .enums = {{"FAULT", NaN}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"FAULT\": NaN is not an integer");
}

/// @brief it should reject a fraction enum for a boolean.
TEST(Decoder, RejectsFractionEnumForBoolean) {
    const auto err = create_error(
        {.data_type = telem::BOOLEAN_T, .enums = {{"HALF", 0.5}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"HALF\": 0.5 is not a boolean");
}

/// @brief it should reject a negative enum for a boolean.
TEST(Decoder, RejectsNegativeEnumForBoolean) {
    const auto err = create_error(
        {.data_type = telem::BOOLEAN_T, .enums = {{"ERR", std::int64_t{-1}}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"ERR\": -1 is not a boolean");
}

/// @brief it should reject a NaN enum for a boolean.
TEST(Decoder, RejectsNaNEnumForBoolean) {
    const auto err = create_error(
        {.data_type = telem::BOOLEAN_T, .enums = {{"FAULT", NaN}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "enum label \"FAULT\": NaN is not a boolean");
}

/// @brief it should reject a number enum for a string.
TEST(Decoder, RejectsNumberEnumForString) {
    const auto err = create_error(
        {.data_type = telem::STRING_T, .enums = {{"ON", 1.0}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "enum label \"ON\": a float64 value does not convert to string"
    );
}

/// @brief it should reject a string enum for a number.
TEST(Decoder, RejectsStringEnumForNumber) {
    const auto err = create_error(
        {.data_type = telem::FLOAT64_T, .enums = {{"ON", "1"}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "enum label \"ON\": a string value does not convert to float64"
    );
}

/// @brief it should reject an enum for a timestamp.
TEST(Decoder, RejectsEnumForTimestamp) {
    const auto err = create_error(
        {.data_type = telem::TIMESTAMP_T, .enums = {{"EPOCH", std::int64_t{0}}}}
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "enum label \"EPOCH\": a int64 value does not convert to timestamp"
    );
}

/// @brief it should convert a double into a float64.
TEST(DecoderAppendFloat, ConvertsFloat64) {
    ASSERT_EQ(decode_float({.data_type = telem::FLOAT64_T}, 0.1).at<double>(0), 0.1);
}

/// @brief it should convert a double into a float32.
TEST(DecoderAppendFloat, ConvertsFloat32) {
    ASSERT_EQ(decode_float({.data_type = telem::FLOAT32_T}, 0.25).at<float>(0), 0.25F);
}

/// @brief it should convert a whole double into an integer.
TEST(DecoderAppendFloat, ConvertsWholeFloatToInteger) {
    ASSERT_EQ(
        decode_float({.data_type = telem::INT16_T}, -3.0).at<std::int16_t>(0),
        std::int16_t{-3}
    );
}

/// @brief it should convert a whole double in an integer range.
TEST(DecoderAppendFloat, ConvertsWholeFloatInRange) {
    const auto s = decode_float(
        {.data_type = telem::FLOAT64_T, .range = telem::INT64_T},
        4.0
    );
    ASSERT_EQ(s.at<double>(0), 4.0);
}

/// @brief it should reject infinity.
TEST(DecoderAppendFloat, RejectsInfinity) {
    const auto err = decode_float_error(
        {.data_type = telem::FLOAT64_T},
        std::numeric_limits<double>::infinity()
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "Infinity is not finite");
}

/// @brief it should reject NaN.
TEST(DecoderAppendFloat, RejectsNaN) {
    const auto err = decode_float_error({.data_type = telem::FLOAT32_T}, NaN);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "NaN is not finite");
}

/// @brief it should reject a double past a float32.
TEST(DecoderAppendFloat, RejectsFloatPastFloat32) {
    const auto err = decode_float_error({.data_type = telem::FLOAT32_T}, 1e39);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1e+39 overflows float32");
}

/// @brief it should reject a fraction for an integer.
TEST(DecoderAppendFloat, RejectsFractionForInteger) {
    const auto err = decode_float_error({.data_type = telem::INT32_T}, 1.5);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "1.5 is not an integer");
}

/// @brief it should reject a fraction outside an integer range.
TEST(DecoderAppendFloat, RejectsFractionOutsideRange) {
    const auto err = decode_float_error(
        {.data_type = telem::FLOAT64_T, .range = telem::INT64_T},
        2.5
    );
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "2.5 is not an integer");
}

/// @brief it should reject a double for a string.
TEST(DecoderAppendFloat, RejectsFloatForString) {
    const auto err = decode_float_error({.data_type = telem::STRING_T}, 1.0);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "cannot convert a float to string");
}
}
