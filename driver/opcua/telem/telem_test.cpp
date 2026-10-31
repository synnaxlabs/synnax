// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/telem/telem.h"

namespace driver::opcua::telem {
/// @brief it should convert UA types to telem data types.
TEST(TelemTest, testUAToDataType) {
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_FLOAT]), x::telem::FLOAT32_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_DOUBLE]), x::telem::FLOAT64_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_SBYTE]), x::telem::INT8_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_INT16]), x::telem::INT16_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_INT32]), x::telem::INT32_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_INT64]), x::telem::INT64_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_BYTE]), x::telem::UINT8_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_UINT16]), x::telem::UINT16_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_UINT32]), x::telem::UINT32_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_UINT64]), x::telem::UINT64_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_STRING]), x::telem::STRING_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_DATETIME]), x::telem::TIMESTAMP_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_GUID]), x::telem::UUID_T);
    EXPECT_EQ(ua_to_data_type(&UA_TYPES[UA_TYPES_BOOLEAN]), x::telem::UINT8_T);
    EXPECT_EQ(ua_to_data_type(nullptr), x::telem::UNKNOWN_T);
}

/// @brief it should convert telem data types to UA types.
TEST(TelemTest, testDataTypeToUA) {
    EXPECT_EQ(data_type_to_ua(x::telem::FLOAT32_T), &UA_TYPES[UA_TYPES_FLOAT]);
    EXPECT_EQ(data_type_to_ua(x::telem::FLOAT64_T), &UA_TYPES[UA_TYPES_DOUBLE]);
    EXPECT_EQ(data_type_to_ua(x::telem::INT8_T), &UA_TYPES[UA_TYPES_SBYTE]);
    EXPECT_EQ(data_type_to_ua(x::telem::INT16_T), &UA_TYPES[UA_TYPES_INT16]);
    EXPECT_EQ(data_type_to_ua(x::telem::INT32_T), &UA_TYPES[UA_TYPES_INT32]);
    EXPECT_EQ(data_type_to_ua(x::telem::INT64_T), &UA_TYPES[UA_TYPES_INT64]);
    EXPECT_EQ(data_type_to_ua(x::telem::UINT16_T), &UA_TYPES[UA_TYPES_UINT16]);
    EXPECT_EQ(data_type_to_ua(x::telem::UINT32_T), &UA_TYPES[UA_TYPES_UINT32]);
    EXPECT_EQ(data_type_to_ua(x::telem::UINT64_T), &UA_TYPES[UA_TYPES_UINT64]);
    EXPECT_EQ(data_type_to_ua(x::telem::STRING_T), &UA_TYPES[UA_TYPES_STRING]);
    EXPECT_EQ(data_type_to_ua(x::telem::TIMESTAMP_T), &UA_TYPES[UA_TYPES_DATETIME]);
    EXPECT_EQ(data_type_to_ua(x::telem::UUID_T), &UA_TYPES[UA_TYPES_GUID]);
    EXPECT_EQ(data_type_to_ua(x::telem::UNKNOWN_T), &UA_TYPES[UA_TYPES_VARIANT]);
}

/// @brief it should convert UA float arrays to telem series.
TEST(TelemTest, testUAFloatArrayToSeries) {
    UA_Variant array_v;
    UA_Variant_init(&array_v);
    UA_Float floats[3] = {1.0f, 2.0f, 3.0f};
    UA_Variant_setArray(&array_v, floats, 3, &UA_TYPES[UA_TYPES_FLOAT]);

    x::telem::Series series(x::telem::FLOAT32_T, 3);
    auto written = ASSERT_NIL_P(ua_array_write_to_series(series, &array_v, 3));
    EXPECT_EQ(written, 3);
    EXPECT_EQ(series.size(), 3);
    EXPECT_EQ(series.at<float>(0), 1.0f);
    EXPECT_EQ(series.at<float>(1), 2.0f);
    EXPECT_EQ(series.at<float>(2), 3.0f);

    x::telem::Series s2(x::telem::FLOAT64_T, 3);
    auto written2 = ASSERT_NIL_P(ua_array_write_to_series(s2, &array_v, 3));
    EXPECT_EQ(written2, 3);
    EXPECT_EQ(s2.size(), 3);
    EXPECT_EQ(s2.at<double>(0), 1.0);
    EXPECT_EQ(s2.at<double>(1), 2.0);
    EXPECT_EQ(s2.at<double>(2), 3.0);
}

/// @brief it should convert telem series to UA variant.
TEST(TelemTest, testSeriesToVariant) {
    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    float val = 42.0f;
    series.write(val);

    auto variant = ASSERT_NIL_P(series_to_variant(series));
    EXPECT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_EQ(*static_cast<float *>(variant.data), 42.0f);

    UA_Variant_clear(&variant);
}

/// @brief it should read a String node into a string sample.
TEST(TelemTest, ToSampleString) {
    UA_String str = UA_STRING_STATIC("hello");
    UA_Variant v;
    UA_Variant_setScalar(&v, &str, &UA_TYPES[UA_TYPES_STRING]);
    const auto sample = ASSERT_NIL_P(to_sample(v, x::telem::STRING_T));
    EXPECT_EQ(std::get<std::string>(sample), "hello");
}

/// @brief it should convert a DateTime to nanoseconds since 1970.
TEST(TelemTest, ToSampleDateTime) {
    UA_DateTime t = UA_DATETIME_UNIX_EPOCH + 15 * UA_DATETIME_SEC;
    UA_Variant v;
    UA_Variant_setScalar(&v, &t, &UA_TYPES[UA_TYPES_DATETIME]);
    const auto sample = ASSERT_NIL_P(to_sample(v, x::telem::TIMESTAMP_T));
    EXPECT_EQ(
        std::get<x::telem::TimeStamp>(sample),
        x::telem::TimeStamp(15 * x::telem::SECOND)
    );
}

/// @brief it should reject a DateTime that a timestamp cannot hold, such as the zero
/// DateTime that servers send for an unset value.
TEST(TelemTest, ToSampleDateTimeOutOfRange) {
    UA_DateTime t = 0;
    UA_Variant v;
    UA_Variant_setScalar(&v, &t, &UA_TYPES[UA_TYPES_DATETIME]);
    const auto err = to_sample(v, x::telem::TIMESTAMP_T).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "the DateTime is outside the years 1677 to 2262 that a timestamp holds"
    );
}

/// @brief it should reject an empty variant.
TEST(TelemTest, ToSampleEmpty) {
    UA_Variant v;
    UA_Variant_init(&v);
    const auto err = to_sample(v, x::telem::FLOAT32_T).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the value is empty");
}

/// @brief it should reject a type that it cannot convert.
TEST(TelemTest, ToSampleUnsupportedType) {
    UA_LocalizedText text{
        .locale = UA_STRING_STATIC("en"),
        .text = UA_STRING_STATIC("text")
    };
    UA_Variant v;
    UA_Variant_setScalar(&v, &text, &UA_TYPES[UA_TYPES_LOCALIZEDTEXT]);
    const auto err = to_sample(v, x::telem::FLOAT32_T).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "cannot convert a LocalizedText to float32");
}

/// @brief it should write a String array to a string series.
TEST(TelemTest, StringArrayToSeries) {
    UA_String strs[2] = {UA_STRING_STATIC("a"), UA_STRING_STATIC("bc")};
    UA_Variant v;
    UA_Variant_setArray(&v, strs, 2, &UA_TYPES[UA_TYPES_STRING]);
    x::telem::Series series(x::telem::STRING_T, 2);
    EXPECT_EQ(ASSERT_NIL_P(ua_array_write_to_series(series, &v, 2)), 2);
    EXPECT_EQ(series.at<std::string>(0), "a");
    EXPECT_EQ(series.at<std::string>(1), "bc");
}

/// @brief it should write nothing when one element of a DateTime array is out of range.
TEST(TelemTest, DateTimeArrayWritesNothingOnBadElement) {
    UA_DateTime times[2] = {UA_DATETIME_UNIX_EPOCH, 0};
    UA_Variant v;
    UA_Variant_setArray(&v, times, 2, &UA_TYPES[UA_TYPES_DATETIME]);
    x::telem::Series series(x::telem::TIMESTAMP_T, 2);
    const auto err = ua_array_write_to_series(series, &v, 2).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "the DateTime is outside the years 1677 to 2262 that a timestamp holds"
    );
    EXPECT_EQ(series.size(), 0);
}

/// @brief it should write a string sample as an OPC UA String.
TEST(TelemTest, StringSeriesToVariant) {
    x::telem::Series series(std::string("hello"));
    auto variant = ASSERT_NIL_P(series_to_variant(series));
    ASSERT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_STRING]));
    const auto *str = static_cast<UA_String *>(variant.data);
    EXPECT_EQ(std::string(reinterpret_cast<char *>(str->data), str->length), "hello");
    UA_Variant_clear(&variant);
}

/// @brief it should write a timestamp sample as an OPC UA DateTime.
TEST(TelemTest, TimestampSeriesToVariant) {
    x::telem::Series series(x::telem::TimeStamp(15 * x::telem::SECOND));
    auto variant = ASSERT_NIL_P(series_to_variant(series));
    ASSERT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_DATETIME]));
    EXPECT_EQ(
        *static_cast<UA_DateTime *>(variant.data),
        UA_DATETIME_UNIX_EPOCH + 15 * UA_DATETIME_SEC
    );
    UA_Variant_clear(&variant);
}
}
