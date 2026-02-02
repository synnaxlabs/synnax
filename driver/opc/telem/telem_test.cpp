// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/opc/telem/telem.h"

/// @brief it should convert UA types to telem data types.
TEST(TelemTest, testUAToDataType) {
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_FLOAT]),
        x::telem::FLOAT32_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_DOUBLE]),
        x::telem::FLOAT64_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_SBYTE]),
        x::telem::INT8_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT16]),
        x::telem::INT16_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT32]),
        x::telem::INT32_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT64]),
        x::telem::INT64_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_BYTE]),
        x::telem::UINT8_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT16]),
        x::telem::UINT16_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT32]),
        x::telem::UINT32_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT64]),
        x::telem::UINT64_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_STRING]),
        x::telem::STRING_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_DATETIME]),
        x::telem::TIMESTAMP_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_GUID]),
        x::telem::UUID_T
    );
    EXPECT_EQ(
        driver::opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_BOOLEAN]),
        x::telem::UINT8_T
    );
    EXPECT_EQ(driver::opc::telem::ua_to_data_type(nullptr), x::telem::UNKNOWN_T);
}

/// @brief it should convert telem data types to UA types.
TEST(TelemTest, testDataTypeToUA) {
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::FLOAT32_T),
        &UA_TYPES[UA_TYPES_FLOAT]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::FLOAT64_T),
        &UA_TYPES[UA_TYPES_DOUBLE]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::INT8_T),
        &UA_TYPES[UA_TYPES_SBYTE]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::INT16_T),
        &UA_TYPES[UA_TYPES_INT16]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::INT32_T),
        &UA_TYPES[UA_TYPES_INT32]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::INT64_T),
        &UA_TYPES[UA_TYPES_INT64]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::UINT16_T),
        &UA_TYPES[UA_TYPES_UINT16]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::UINT32_T),
        &UA_TYPES[UA_TYPES_UINT32]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::UINT64_T),
        &UA_TYPES[UA_TYPES_UINT64]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::STRING_T),
        &UA_TYPES[UA_TYPES_STRING]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::TIMESTAMP_T),
        &UA_TYPES[UA_TYPES_DATETIME]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::UUID_T),
        &UA_TYPES[UA_TYPES_GUID]
    );
    EXPECT_EQ(
        driver::opc::telem::data_type_to_ua(x::telem::UNKNOWN_T),
        &UA_TYPES[UA_TYPES_VARIANT]
    );
}

/// @brief it should convert UA float arrays to telem series.
TEST(TelemTest, testUAFloatArrayToSeries) {
    UA_Variant array_v;
    UA_Variant_init(&array_v);
    UA_Float floats[3] = {1.0f, 2.0f, 3.0f};
    UA_Variant_setArray(&array_v, floats, 3, &UA_TYPES[UA_TYPES_FLOAT]);

    x::telem::Series series(x::telem::FLOAT32_T, 3);
    auto written = ASSERT_NIL_P(
        driver::opc::telem::ua_array_write_to_series(series, &array_v, 3)
    );
    EXPECT_EQ(written, 3);
    EXPECT_EQ(series.size(), 3);
    EXPECT_EQ(series.at<float>(0), 1.0f);
    EXPECT_EQ(series.at<float>(1), 2.0f);
    EXPECT_EQ(series.at<float>(2), 3.0f);

    x::telem::Series s2(x::telem::FLOAT64_T, 3);
    auto written2 = ASSERT_NIL_P(
        driver::opc::telem::ua_array_write_to_series(s2, &array_v, 3)
    );
    EXPECT_EQ(written2, 3);
    EXPECT_EQ(s2.size(), 3);
    EXPECT_EQ(s2.at<double>(0), 1.0);
    EXPECT_EQ(s2.at<double>(1), 2.0);
    EXPECT_EQ(s2.at<double>(2), 3.0);
}

/// @brief it should write UA variant values to telem series.
TEST(TelemTest, testWriteToSeries) {
    auto series = x::telem::Series(x::telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);
    UA_Float val = 42.0f;
    UA_Variant_setScalar(&v, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    const auto written = ASSERT_NIL_P(driver::opc::telem::write_to_series(series, v));
    EXPECT_EQ(written, 1);
    EXPECT_EQ(series.size(), 1);
    EXPECT_EQ(series.at<float>(0), 42.0f);

    UA_Variant v2;
    UA_Variant_init(&v2);
    UA_Float v2v = 43.0f;
    UA_Variant_setScalar(&v2, &v2v, &UA_TYPES[UA_TYPES_FLOAT]);

    const auto written2 = ASSERT_NIL_P(driver::opc::telem::write_to_series(series, v2));
    EXPECT_EQ(written2, 1);
    EXPECT_EQ(series.size(), 2);
    EXPECT_EQ(series.at<float>(1), 43.0f);
}

/// @brief it should convert telem series to UA variant.
TEST(TelemTest, testSeriesToVariant) {
    auto series = x::telem::Series(x::telem::FLOAT32_T, 1);
    float val = 42.0f;
    series.write(val);

    auto variant = ASSERT_NIL_P(driver::opc::telem::series_to_variant(series));
    EXPECT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_EQ(*static_cast<float *>(variant.data), 42.0f);

    UA_Variant_clear(&variant);
}
