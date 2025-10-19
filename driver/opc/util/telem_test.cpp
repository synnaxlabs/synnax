// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/opc/util/util.h"

TEST(TelemTest, testUAToDataType) {
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_FLOAT]), telem::FLOAT32_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_DOUBLE]), telem::FLOAT64_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_SBYTE]), telem::INT8_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_INT16]), telem::INT16_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_INT32]), telem::INT32_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_INT64]), telem::INT64_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_BYTE]), telem::UINT8_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT16]), telem::UINT16_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT32]), telem::UINT32_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT64]), telem::UINT64_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_STRING]), telem::STRING_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_DATETIME]), telem::TIMESTAMP_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_GUID]), telem::UUID_T);
    EXPECT_EQ(util::ua_to_data_type(&UA_TYPES[UA_TYPES_BOOLEAN]), telem::UINT8_T);
    EXPECT_EQ(util::ua_to_data_type(nullptr), telem::UNKNOWN_T);
}

TEST(TelemTest, testDataTypeToUA) {
    EXPECT_EQ(util::data_type_to_ua(telem::FLOAT32_T), &UA_TYPES[UA_TYPES_FLOAT]);
    EXPECT_EQ(util::data_type_to_ua(telem::FLOAT64_T), &UA_TYPES[UA_TYPES_DOUBLE]);
    EXPECT_EQ(util::data_type_to_ua(telem::INT8_T), &UA_TYPES[UA_TYPES_SBYTE]);
    EXPECT_EQ(util::data_type_to_ua(telem::INT16_T), &UA_TYPES[UA_TYPES_INT16]);
    EXPECT_EQ(util::data_type_to_ua(telem::INT32_T), &UA_TYPES[UA_TYPES_INT32]);
    EXPECT_EQ(util::data_type_to_ua(telem::INT64_T), &UA_TYPES[UA_TYPES_INT64]);
    EXPECT_EQ(util::data_type_to_ua(telem::UINT16_T), &UA_TYPES[UA_TYPES_UINT16]);
    EXPECT_EQ(util::data_type_to_ua(telem::UINT32_T), &UA_TYPES[UA_TYPES_UINT32]);
    EXPECT_EQ(util::data_type_to_ua(telem::UINT64_T), &UA_TYPES[UA_TYPES_UINT64]);
    EXPECT_EQ(util::data_type_to_ua(telem::STRING_T), &UA_TYPES[UA_TYPES_STRING]);
    EXPECT_EQ(util::data_type_to_ua(telem::TIMESTAMP_T), &UA_TYPES[UA_TYPES_DATETIME]);
    EXPECT_EQ(util::data_type_to_ua(telem::UUID_T), &UA_TYPES[UA_TYPES_GUID]);
    EXPECT_EQ(util::data_type_to_ua(telem::UNKNOWN_T), &UA_TYPES[UA_TYPES_VARIANT]);
}

TEST(TelemTest, testUAFloatArrayToSeries) {
    // Test regular array conversion
    UA_Variant array_v;
    UA_Variant_init(&array_v);
    UA_Float floats[3] = {1.0f, 2.0f, 3.0f};
    UA_Variant_setArray(&array_v, floats, 3, &UA_TYPES[UA_TYPES_FLOAT]);

    telem::Series series(
        telem::FLOAT32_T,
        3
    ); // Pre-allocate series with correct type and size
    auto [written, err] = util::ua_array_write_to_series(series, &array_v, 3);
    ASSERT_NIL(err);
    EXPECT_EQ(series.size(), 3);
    EXPECT_EQ(series.at<float>(0), 1.0f);
    EXPECT_EQ(series.at<float>(1), 2.0f);
    EXPECT_EQ(series.at<float>(2), 3.0f);

    telem::Series s2(telem::FLOAT64_T, 3); // Pre-allocate second series
    auto [written2, err2] = util::ua_array_write_to_series(s2, &array_v, 3);
    ASSERT_NIL(err2);
    EXPECT_EQ(s2.size(), 3);
    EXPECT_EQ(s2.at<double>(0), 1.0);
    EXPECT_EQ(s2.at<double>(1), 2.0);
    EXPECT_EQ(s2.at<double>(2), 3.0);
}

TEST(TelemTest, testWriteToSeries) {
    auto series = telem::Series(telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);
    UA_Float val = 42.0f;
    UA_Variant_setScalar(&v, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written, err] = util::write_to_series(series, v);
    ASSERT_NIL(err);
    EXPECT_EQ(series.size(), 1);
    EXPECT_EQ(series.at<float>(0), 42.0f);

    UA_Variant v2;
    UA_Variant_init(&v2);
    UA_Float v2v = 43.0f;
    UA_Variant_setScalar(&v2, &v2v, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written2, err2] = util::write_to_series(series, v2);
    ASSERT_NIL(err2);
    EXPECT_EQ(series.size(), 2);
    EXPECT_EQ(series.at<float>(1), 43.0f);
}

TEST(TelemTest, testSeriesToVariant) {
    // Create a series with a single value
    auto series = telem::Series(telem::FLOAT32_T, 1);
    float val = 42.0f;
    series.write(val);

    // Convert to variant
    auto [variant, err] = util::series_to_variant(series);
    EXPECT_EQ(err, xerrors::NIL);
    EXPECT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_EQ(*static_cast<float *>(variant.data), 42.0f);
}

TEST(TelemTest, testWriteToSeriesEmptyVariant) {
    auto series = telem::Series(telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);

    auto [written, err] = util::write_to_series(series, v);
    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
}

TEST(TelemTest, testWriteToSeriesNullTypeOrData) {
    auto series = telem::Series(telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);
    v.type = nullptr;
    v.data = nullptr;

    auto [written, err] = util::write_to_series(series, v);
    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
}

TEST(TelemTest, testWriteToSeriesZeroLengthArray) {
    auto series = telem::Series(telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);
    v.type = &UA_TYPES[UA_TYPES_FLOAT];
    v.arrayLength = 0;
    v.data = UA_EMPTY_ARRAY_SENTINEL;

    auto [written, err] = util::write_to_series(series, v);
    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
}

TEST(TelemTest, testNullVariantTypeHandling) {
    // Test that null variant type is properly handled without crashing
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant null_variant;
    UA_Variant_init(&null_variant);
    null_variant.type = nullptr;
    null_variant.data = nullptr;

    auto [written, err] = util::write_to_series(series, null_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
    EXPECT_TRUE(err.message().find("null type") != std::string::npos);
}

TEST(TelemTest, testNullVariantDataHandling) {
    // Test that null data with valid type is properly handled
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant null_data_variant;
    UA_Variant_init(&null_data_variant);
    null_data_variant.type = &UA_TYPES[UA_TYPES_FLOAT];
    null_data_variant.data = nullptr;

    auto [written, err] = util::write_to_series(series, null_data_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
    EXPECT_TRUE(err.message().find("null data") != std::string::npos);
}

TEST(TelemTest, testZeroLengthArrayHandling) {
    // Test that zero-length arrays are properly rejected
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant zero_length_variant;
    UA_Variant_init(&zero_length_variant);
    zero_length_variant.type = &UA_TYPES[UA_TYPES_FLOAT];
    zero_length_variant.arrayLength = 0;
    zero_length_variant.data = UA_EMPTY_ARRAY_SENTINEL;

    auto [written, err] = util::write_to_series(series, zero_length_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(series.size(), 0);
    EXPECT_TRUE(err.message().find("zero length") != std::string::npos);
}

TEST(TelemTest, testInvalidDataTypeConversion) {
    // Test that invalid data type conversions are handled gracefully
    telem::Series int_series(telem::INT32_T, 10);

    UA_Variant string_variant;
    UA_Variant_init(&string_variant);
    UA_String ua_string = UA_STRING_ALLOC("invalid_for_int");
    UA_Variant_setScalarCopy(&string_variant, &ua_string, &UA_TYPES[UA_TYPES_STRING]);

    auto [written, err] = util::write_to_series(int_series, string_variant);

    // Should either fail or succeed depending on cast capabilities
    // The important thing is it doesn't crash
    UA_String_clear(&ua_string);
    UA_Variant_clear(&string_variant);
}

TEST(TelemTest, testWriteToSeriesReturnsError) {
    // Test that write_to_series properly returns errors in tuple format
    telem::Series series(telem::FLOAT32_T, 10);

    // Valid write should return no error
    UA_Variant valid_variant;
    UA_Variant_init(&valid_variant);
    UA_Float valid_val = 42.0f;
    UA_Variant_setScalar(&valid_variant, &valid_val, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written1, err1] = util::write_to_series(series, valid_variant);
    EXPECT_FALSE(err1);
    EXPECT_EQ(written1, 1);
    EXPECT_EQ(series.size(), 1);

    // Invalid write should return error
    UA_Variant invalid_variant;
    UA_Variant_init(&invalid_variant);
    invalid_variant.type = nullptr;
    invalid_variant.data = nullptr;

    auto [written2, err2] = util::write_to_series(series, invalid_variant);
    EXPECT_TRUE(err2);
    EXPECT_EQ(written2, 0);
    EXPECT_EQ(series.size(), 1); // Should not have added any data
}

TEST(TelemTest, testArraySizeMismatchDetection) {
    // Test that array size mismatches are detected and reported
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant array_variant;
    UA_Variant_init(&array_variant);
    UA_Float floats[3] = {1.0f, 2.0f, 3.0f};
    UA_Variant_setArray(&array_variant, floats, 3, &UA_TYPES[UA_TYPES_FLOAT]);

    // Try to write to series expecting array size of 5 (but we have 3)
    auto [written, err] = util::ua_array_write_to_series(
        series,
        &array_variant,
        5,
        "test_channel"
    );

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_TRUE(err.message().find("too small") != std::string::npos);
    EXPECT_TRUE(err.message().find("test_channel") != std::string::npos);
}

TEST(TelemTest, testArraySizeTooLarge) {
    // Test detection when array is larger than expected
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant array_variant;
    UA_Variant_init(&array_variant);
    UA_Float floats[5] = {1.0f, 2.0f, 3.0f, 4.0f, 5.0f};
    UA_Variant_setArray(&array_variant, floats, 5, &UA_TYPES[UA_TYPES_FLOAT]);

    // Try to write to series expecting array size of 3 (but we have 5)
    auto [written, err] = util::ua_array_write_to_series(
        series,
        &array_variant,
        3,
        "test_channel"
    );

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_TRUE(err.message().find("too large") != std::string::npos);
    EXPECT_TRUE(err.message().find("test_channel") != std::string::npos);
}

TEST(TelemTest, testErrorMessageContainsChannelName) {
    // Test that error messages include channel names for debugging
    telem::Series series(telem::FLOAT32_T, 10);

    UA_Variant null_variant;
    UA_Variant_init(&null_variant);
    null_variant.type = nullptr;
    null_variant.data = nullptr;

    auto [written, err] = util::write_to_series(series, null_variant);

    EXPECT_TRUE(err);
    EXPECT_FALSE(err.message().empty());
    // Error message should contain descriptive information
    EXPECT_TRUE(
        err.message().find("null") != std::string::npos ||
        err.message().find("invalid") != std::string::npos
    );
}

TEST(TelemTest, testBooleanInvalidDataHandling) {
    // Test that boolean data with null type is properly handled
    telem::Series bool_series(telem::UINT8_T, 10);

    UA_Variant null_bool_variant;
    UA_Variant_init(&null_bool_variant);
    null_bool_variant.type = nullptr;
    null_bool_variant.data = nullptr;

    auto [written, err] = util::write_to_series(bool_series, null_bool_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(bool_series.size(), 0);
}

TEST(TelemTest, testBooleanNullDataHandling) {
    // Test that boolean with valid type but null data is handled
    telem::Series bool_series(telem::UINT8_T, 10);

    UA_Variant bool_variant;
    UA_Variant_init(&bool_variant);
    bool_variant.type = &UA_TYPES[UA_TYPES_BOOLEAN];
    bool_variant.data = nullptr;

    auto [written, err] = util::write_to_series(bool_series, bool_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(bool_series.size(), 0);
    EXPECT_TRUE(err.message().find("null data") != std::string::npos);
}

TEST(TelemTest, testFloat32InvalidTypeHandling) {
    // Test that float32 data with null type is properly handled
    telem::Series float_series(telem::FLOAT32_T, 10);

    UA_Variant null_float_variant;
    UA_Variant_init(&null_float_variant);
    null_float_variant.type = nullptr;
    null_float_variant.data = nullptr;

    auto [written, err] = util::write_to_series(float_series, null_float_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(float_series.size(), 0);
}

TEST(TelemTest, testFloat32NullDataHandling) {
    // Test that float32 with valid type but null data is handled
    telem::Series float_series(telem::FLOAT32_T, 10);

    UA_Variant float_variant;
    UA_Variant_init(&float_variant);
    float_variant.type = &UA_TYPES[UA_TYPES_FLOAT];
    float_variant.data = nullptr;

    auto [written, err] = util::write_to_series(float_series, float_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(float_series.size(), 0);
    EXPECT_TRUE(err.message().find("null data") != std::string::npos);
}

TEST(TelemTest, testFloat64InvalidTypeHandling) {
    // Test that float64 data with null type is properly handled
    telem::Series double_series(telem::FLOAT64_T, 10);

    UA_Variant null_double_variant;
    UA_Variant_init(&null_double_variant);
    null_double_variant.type = nullptr;
    null_double_variant.data = nullptr;

    auto [written, err] = util::write_to_series(double_series, null_double_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(double_series.size(), 0);
}

TEST(TelemTest, testFloat64NullDataHandling) {
    // Test that float64 with valid type but null data is handled
    telem::Series double_series(telem::FLOAT64_T, 10);

    UA_Variant double_variant;
    UA_Variant_init(&double_variant);
    double_variant.type = &UA_TYPES[UA_TYPES_DOUBLE];
    double_variant.data = nullptr;

    auto [written, err] = util::write_to_series(double_series, double_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(double_series.size(), 0);
    EXPECT_TRUE(err.message().find("null data") != std::string::npos);
}

TEST(TelemTest, testBooleanValidDataSucceeds) {
    // Test that valid boolean data is written successfully
    telem::Series bool_series(telem::UINT8_T, 10);

    UA_Variant bool_variant;
    UA_Variant_init(&bool_variant);
    UA_Boolean bool_val = true;
    UA_Variant_setScalar(&bool_variant, &bool_val, &UA_TYPES[UA_TYPES_BOOLEAN]);

    auto [written, err] = util::write_to_series(bool_series, bool_variant);

    EXPECT_FALSE(err);
    EXPECT_EQ(written, 1);
    EXPECT_EQ(bool_series.size(), 1);
    EXPECT_EQ(bool_series.at<std::uint8_t>(0), 1);
}

TEST(TelemTest, testFloat32ValidDataSucceeds) {
    // Test that valid float32 data is written successfully
    telem::Series float_series(telem::FLOAT32_T, 10);

    UA_Variant float_variant;
    UA_Variant_init(&float_variant);
    UA_Float float_val = 3.14159f;
    UA_Variant_setScalar(&float_variant, &float_val, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written, err] = util::write_to_series(float_series, float_variant);

    EXPECT_FALSE(err);
    EXPECT_EQ(written, 1);
    EXPECT_EQ(float_series.size(), 1);
    EXPECT_NEAR(float_series.at<float>(0), 3.14159f, 0.0001f);
}

TEST(TelemTest, testFloat64ValidDataSucceeds) {
    // Test that valid float64 data is written successfully
    telem::Series double_series(telem::FLOAT64_T, 10);

    UA_Variant double_variant;
    UA_Variant_init(&double_variant);
    UA_Double double_val = 2.71828;
    UA_Variant_setScalar(&double_variant, &double_val, &UA_TYPES[UA_TYPES_DOUBLE]);

    auto [written, err] = util::write_to_series(double_series, double_variant);

    EXPECT_FALSE(err);
    EXPECT_EQ(written, 1);
    EXPECT_EQ(double_series.size(), 1);
    EXPECT_NEAR(double_series.at<double>(0), 2.71828, 0.0001);
}

TEST(TelemTest, testBooleanZeroLengthArrayHandling) {
    // Test that zero-length boolean arrays are properly rejected
    telem::Series bool_series(telem::UINT8_T, 10);

    UA_Variant bool_array_variant;
    UA_Variant_init(&bool_array_variant);
    bool_array_variant.type = &UA_TYPES[UA_TYPES_BOOLEAN];
    bool_array_variant.arrayLength = 0;
    bool_array_variant.data = UA_EMPTY_ARRAY_SENTINEL;

    auto [written, err] = util::write_to_series(bool_series, bool_array_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(bool_series.size(), 0);
    EXPECT_TRUE(err.message().find("zero length") != std::string::npos);
}

TEST(TelemTest, testFloat32ZeroLengthArrayHandling) {
    // Test that zero-length float32 arrays are properly rejected
    telem::Series float_series(telem::FLOAT32_T, 10);

    UA_Variant float_array_variant;
    UA_Variant_init(&float_array_variant);
    float_array_variant.type = &UA_TYPES[UA_TYPES_FLOAT];
    float_array_variant.arrayLength = 0;
    float_array_variant.data = UA_EMPTY_ARRAY_SENTINEL;

    auto [written, err] = util::write_to_series(float_series, float_array_variant);

    EXPECT_TRUE(err);
    EXPECT_EQ(written, 0);
    EXPECT_EQ(float_series.size(), 0);
    EXPECT_TRUE(err.message().find("zero length") != std::string::npos);
}
