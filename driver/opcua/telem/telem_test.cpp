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
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_FLOAT]), x::telem::FLOAT32_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_DOUBLE]), x::telem::FLOAT64_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_SBYTE]), x::telem::INT8_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_INT16]), x::telem::INT16_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_INT32]), x::telem::INT32_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_INT64]), x::telem::INT64_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_BYTE]), x::telem::UINT8_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_UINT16]), x::telem::UINT16_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_UINT32]), x::telem::UINT32_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_UINT64]), x::telem::UINT64_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_STRING]), x::telem::STRING_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_DATETIME]), x::telem::TIMESTAMP_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_GUID]), x::telem::UUID_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_BOOLEAN]), x::telem::BOOLEAN_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_BYTESTRING]), x::telem::BYTES_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_DURATION]), x::telem::FLOAT64_T);
    EXPECT_EQ(to_data_type(&UA_TYPES[UA_TYPES_SERVERSTATE]), x::telem::INT32_T);
    EXPECT_EQ(to_data_type(nullptr), x::telem::UNKNOWN_T);
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
    EXPECT_EQ(data_type_to_ua(x::telem::UINT8_T), &UA_TYPES[UA_TYPES_BYTE]);
    EXPECT_EQ(data_type_to_ua(x::telem::BOOLEAN_T), &UA_TYPES[UA_TYPES_BOOLEAN]);
    EXPECT_EQ(data_type_to_ua(x::telem::BYTES_T), &UA_TYPES[UA_TYPES_BYTESTRING]);
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

    auto variant = ASSERT_NIL_P(to_variant(series, nullptr));
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
    auto variant = ASSERT_NIL_P(to_variant(series, nullptr));
    ASSERT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_STRING]));
    const auto *str = static_cast<UA_String *>(variant.data);
    EXPECT_EQ(std::string(reinterpret_cast<char *>(str->data), str->length), "hello");
    UA_Variant_clear(&variant);
}

/// @brief it should write a timestamp sample as an OPC UA DateTime.
TEST(TelemTest, TimestampSeriesToVariant) {
    x::telem::Series series(x::telem::TimeStamp(15 * x::telem::SECOND));
    auto variant = ASSERT_NIL_P(to_variant(series, nullptr));
    ASSERT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_DATETIME]));
    EXPECT_EQ(
        *static_cast<UA_DateTime *>(variant.data),
        UA_DATETIME_UNIX_EPOCH + 15 * UA_DATETIME_SEC
    );
    UA_Variant_clear(&variant);
}

/// @brief it should convert the DataType of a node to the data type of its channel.
TEST(TelemTest, ToDataType) {
    EXPECT_EQ(to_data_type(UA_NODEID_NUMERIC(0, UA_NS0ID_DOUBLE)), x::telem::FLOAT64_T);
    EXPECT_EQ(
        to_data_type(UA_NODEID_NUMERIC(0, UA_NS0ID_BOOLEAN)),
        x::telem::BOOLEAN_T
    );
    EXPECT_EQ(
        to_data_type(UA_NODEID_NUMERIC(0, UA_NS0ID_UTCTIME)),
        x::telem::TIMESTAMP_T
    );
    EXPECT_EQ(
        to_data_type(UA_NODEID_NUMERIC(0, UA_NS0ID_BASEDATATYPE)),
        x::telem::UNKNOWN_T
    );
    EXPECT_EQ(
        to_data_type(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVERSTATUSDATATYPE)),
        x::telem::UNKNOWN_T
    );
    EXPECT_EQ(to_data_type(UA_NODEID_NUMERIC(2, 3001)), x::telem::UNKNOWN_T);
}

/// @brief it should read a Boolean node into a boolean sample.
TEST(TelemTest, ToSampleBoolean) {
    UA_Boolean value = true;
    UA_Variant v;
    UA_Variant_setScalar(&v, &value, &UA_TYPES[UA_TYPES_BOOLEAN]);
    EXPECT_EQ(std::get<uint8_t>(ASSERT_NIL_P(to_sample(v, x::telem::BOOLEAN_T))), 1);
    EXPECT_EQ(std::get<uint8_t>(ASSERT_NIL_P(to_sample(v, x::telem::UINT8_T))), 1);
}

/// @brief it should read a Guid into a uuid series and into its text.
TEST(TelemTest, GuidToUUID) {
    UA_Guid guid;
    const auto text = std::string("09087e75-8e5e-499b-954f-f2a9603db28a");
    ASSERT_EQ(
        UA_Guid_parse(&guid, UA_STRING(const_cast<char *>(text.c_str()))),
        UA_STATUSCODE_GOOD
    );
    UA_Variant v;
    UA_Variant_setScalar(&v, &guid, &UA_TYPES[UA_TYPES_GUID]);
    const auto uuid = ASSERT_NIL_P(to_sample(v, x::telem::UUID_T));
    const auto series = to_series(x::telem::UUID_T, {uuid});
    EXPECT_EQ(series.uuids()[0].to_string(), text);
    const auto sample = ASSERT_NIL_P(to_sample(v, x::telem::STRING_T));
    EXPECT_EQ(std::get<std::string>(sample), text);
    const auto round_trip = ASSERT_NIL_P(to_variant(series, nullptr));
    ASSERT_TRUE(UA_Variant_hasScalarType(&round_trip, &UA_TYPES[UA_TYPES_GUID]));
    EXPECT_TRUE(UA_Guid_equal(static_cast<UA_Guid *>(round_trip.data), &guid));
    UA_Variant_clear(const_cast<UA_Variant *>(&round_trip));
}

/// @brief it should read a ByteString array into a bytes series.
TEST(TelemTest, ByteStringArrayToBytes) {
    UA_ByteString bytes[2] = {UA_STRING_STATIC("ab"), UA_STRING_STATIC("c")};
    UA_Variant v;
    UA_Variant_setArray(&v, bytes, 2, &UA_TYPES[UA_TYPES_BYTESTRING]);
    x::telem::Series series(x::telem::BYTES_T, 2);
    EXPECT_EQ(ASSERT_NIL_P(ua_array_write_to_series(series, &v, 2)), 2);
    EXPECT_EQ(series.data_type(), x::telem::BYTES_T);
    EXPECT_EQ(series.size(), 2);
    auto variant = ASSERT_NIL_P(to_variant(series, nullptr));
    ASSERT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_BYTESTRING]));
    const UA_ByteString last = UA_STRING_STATIC("c");
    EXPECT_TRUE(UA_ByteString_equal(static_cast<UA_ByteString *>(variant.data), &last));
    UA_Variant_clear(&variant);
}

/// @brief it should reject a ByteString for a channel that is not bytes.
TEST(TelemTest, ByteStringToStringRejected) {
    UA_ByteString bytes = UA_STRING_STATIC("ab");
    UA_Variant v;
    UA_Variant_setScalar(&v, &bytes, &UA_TYPES[UA_TYPES_BYTESTRING]);
    const auto err = to_sample(v, x::telem::STRING_T).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "cannot convert a ByteString to string");
}

namespace {
/// @returns the variant of the command converted for a node of type target.
UA_Variant convert(const x::telem::Series &command, const UA_DataType &target) {
    auto [v, err] = to_variant(command, &target);
    EXPECT_FALSE(err) << err;
    return v;
}

/// @returns the error of converting the command for a node of type target.
x::errors::Error reject(const x::telem::Series &command, const UA_DataType &target) {
    auto [v, err] = to_variant(command, &target);
    UA_Variant_clear(&v);
    return err;
}
}

/// @brief it should write a uint8 or boolean command to a Boolean node.
TEST(TelemTest, ToVariantBoolean) {
    const auto &boolean = UA_TYPES[UA_TYPES_BOOLEAN];
    for (const auto &command:
         {x::telem::Series(static_cast<uint8_t>(1), x::telem::UINT8_T),
          x::telem::Series(static_cast<uint8_t>(1), x::telem::BOOLEAN_T),
          x::telem::Series(1.0),
          x::telem::Series(std::string("TRUE"))}) {
        auto v = convert(command, boolean);
        ASSERT_TRUE(UA_Variant_hasScalarType(&v, &boolean));
        EXPECT_TRUE(*static_cast<UA_Boolean *>(v.data));
        UA_Variant_clear(&v);
    }
    auto err = reject(x::telem::Series(2.0), boolean);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "2 is out of range for a Boolean node");
    err = reject(x::telem::Series(std::string("maybe")), boolean);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "\"maybe\" is not a valid Boolean");
}

/// @brief it should write a whole number in range to an integer node.
TEST(TelemTest, ToVariantInteger) {
    const auto &byte = UA_TYPES[UA_TYPES_BYTE];
    auto v = convert(x::telem::Series(7.0), byte);
    ASSERT_TRUE(UA_Variant_hasScalarType(&v, &byte));
    EXPECT_EQ(*static_cast<UA_Byte *>(v.data), 7);
    UA_Variant_clear(&v);
    const auto &int64 = UA_TYPES[UA_TYPES_INT64];
    v = convert(x::telem::Series(std::string("-123")), int64);
    EXPECT_EQ(*static_cast<UA_Int64 *>(v.data), -123);
    UA_Variant_clear(&v);
}

/// @brief it should not write a value that the integer type of the node cannot hold.
TEST(TelemTest, ToVariantIntegerRejected) {
    const auto &byte = UA_TYPES[UA_TYPES_BYTE];
    auto err = reject(x::telem::Series(300.0), byte);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "300 is out of range for a Byte node");
    err = reject(x::telem::Series(1.5), byte);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "1.5 is not a whole number, so a Byte node cannot hold it");
    err = reject(x::telem::Series(static_cast<int32_t>(-1)), byte);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "-1 is out of range for a Byte node");
    err = reject(x::telem::Series(std::string("12a")), UA_TYPES[UA_TYPES_INT64]);
    EXPECT_EQ(err.data, "\"12a\" is not a valid Int64");
}

/// @brief it should convert a command to a Float or Double node.
TEST(TelemTest, ToVariantFloat) {
    const auto &dbl = UA_TYPES[UA_TYPES_DOUBLE];
    auto v = convert(x::telem::Series(static_cast<int64_t>(5)), dbl);
    EXPECT_EQ(*static_cast<UA_Double *>(v.data), 5.0);
    UA_Variant_clear(&v);
    v = convert(x::telem::Series(std::string("2.5")), dbl);
    EXPECT_EQ(*static_cast<UA_Double *>(v.data), 2.5);
    UA_Variant_clear(&v);
    const auto err = reject(x::telem::Series(1e40), UA_TYPES[UA_TYPES_FLOAT]);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "1e+40 is out of range for a Float node");
}

/// @brief it should write a number to a String node as the shortest text that reads
/// back as the same number.
TEST(TelemTest, ToVariantString) {
    const auto &str = UA_TYPES[UA_TYPES_STRING];
    const auto text = [&](const x::telem::Series &s) {
        auto v = convert(s, str);
        const auto *ua = static_cast<UA_String *>(v.data);
        std::string out(reinterpret_cast<char *>(ua->data), ua->length);
        UA_Variant_clear(&v);
        return out;
    };
    EXPECT_EQ(text(x::telem::Series(1.5)), "1.5");
    EXPECT_EQ(text(x::telem::Series(1.23456789)), "1.23456789");
    EXPECT_EQ(text(x::telem::Series(1234567.0)), "1234567");
    EXPECT_EQ(text(x::telem::Series(0.1f)), "0.1");
}

/// @brief it should write the builtin type of a subtype or an enumeration.
TEST(TelemTest, ToVariantSubtypes) {
    auto v = convert(x::telem::Series(2.0), UA_TYPES[UA_TYPES_DURATION]);
    EXPECT_TRUE(UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_DOUBLE]));
    UA_Variant_clear(&v);
    v = convert(
        x::telem::Series(static_cast<int32_t>(3)),
        UA_TYPES[UA_TYPES_SERVERSTATE]
    );
    ASSERT_TRUE(UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_INT32]));
    EXPECT_EQ(*static_cast<UA_Int32 *>(v.data), 3);
    UA_Variant_clear(&v);
}

/// @brief it should write the type that matches the command to a node of BaseDataType.
TEST(TelemTest, ToVariantBaseDataType) {
    auto v = convert(x::telem::Series(2.0), UA_TYPES[UA_TYPES_VARIANT]);
    EXPECT_TRUE(UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_DOUBLE]));
    UA_Variant_clear(&v);
}

/// @brief it should reject a command to a structure node, and an empty command.
TEST(TelemTest, ToVariantRejectsStructuresAndEmptyCommands) {
    auto err = reject(x::telem::Series(1.0), UA_TYPES[UA_TYPES_SERVERSTATUSDATATYPE]);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "cannot write to a ServerStatusDataType node");
    err = reject(x::telem::Series(x::telem::FLOAT64_T, 1), UA_TYPES[UA_TYPES_DOUBLE]);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the command has no value");
}
}
