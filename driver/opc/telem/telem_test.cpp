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

#include "driver/opc/telem/telem.h"

/// @brief it should convert UA types to telem data types.
TEST(TelemTest, testUAToDataType) {
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_FLOAT]), telem::FLOAT32_T);
    EXPECT_EQ(
        opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_DOUBLE]),
        telem::FLOAT64_T
    );
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_SBYTE]), telem::INT8_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT16]), telem::INT16_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT32]), telem::INT32_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_INT64]), telem::INT64_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_BYTE]), telem::UINT8_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT16]), telem::UINT16_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT32]), telem::UINT32_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_UINT64]), telem::UINT64_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_STRING]), telem::STRING_T);
    EXPECT_EQ(
        opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_DATETIME]),
        telem::TIMESTAMP_T
    );
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_GUID]), telem::UUID_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(&UA_TYPES[UA_TYPES_BOOLEAN]), telem::UINT8_T);
    EXPECT_EQ(opc::telem::ua_to_data_type(nullptr), telem::UNKNOWN_T);
}

/// @brief it should convert telem data types to UA types.
TEST(TelemTest, testDataTypeToUA) {
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::FLOAT32_T), &UA_TYPES[UA_TYPES_FLOAT]);
    EXPECT_EQ(
        opc::telem::data_type_to_ua(telem::FLOAT64_T),
        &UA_TYPES[UA_TYPES_DOUBLE]
    );
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::INT8_T), &UA_TYPES[UA_TYPES_SBYTE]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::INT16_T), &UA_TYPES[UA_TYPES_INT16]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::INT32_T), &UA_TYPES[UA_TYPES_INT32]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::INT64_T), &UA_TYPES[UA_TYPES_INT64]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::UINT16_T), &UA_TYPES[UA_TYPES_UINT16]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::UINT32_T), &UA_TYPES[UA_TYPES_UINT32]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::UINT64_T), &UA_TYPES[UA_TYPES_UINT64]);
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::STRING_T), &UA_TYPES[UA_TYPES_STRING]);
    EXPECT_EQ(
        opc::telem::data_type_to_ua(telem::TIMESTAMP_T),
        &UA_TYPES[UA_TYPES_DATETIME]
    );
    EXPECT_EQ(opc::telem::data_type_to_ua(telem::UUID_T), &UA_TYPES[UA_TYPES_GUID]);
    EXPECT_EQ(
        opc::telem::data_type_to_ua(telem::UNKNOWN_T),
        &UA_TYPES[UA_TYPES_VARIANT]
    );
}

/// @brief it should convert UA float arrays to telem series.
TEST(TelemTest, testUAFloatArrayToSeries) {
    UA_Variant array_v;
    UA_Variant_init(&array_v);
    UA_Float floats[3] = {1.0f, 2.0f, 3.0f};
    UA_Variant_setArray(&array_v, floats, 3, &UA_TYPES[UA_TYPES_FLOAT]);

    telem::Series series(telem::FLOAT32_T, 3);
    auto [written, err] = opc::telem::ua_array_write_to_series(series, &array_v, 3);
    ASSERT_NIL(err);
    EXPECT_EQ(series.size(), 3);
    EXPECT_EQ(series.at<float>(0), 1.0f);
    EXPECT_EQ(series.at<float>(1), 2.0f);
    EXPECT_EQ(series.at<float>(2), 3.0f);

    telem::Series s2(telem::FLOAT64_T, 3);
    auto [written2, err2] = opc::telem::ua_array_write_to_series(s2, &array_v, 3);
    ASSERT_NIL(err2);
    EXPECT_EQ(s2.size(), 3);
    EXPECT_EQ(s2.at<double>(0), 1.0);
    EXPECT_EQ(s2.at<double>(1), 2.0);
    EXPECT_EQ(s2.at<double>(2), 3.0);
}

/// @brief it should write UA variant values to telem series.
TEST(TelemTest, testWriteToSeries) {
    auto series = telem::Series(telem::FLOAT32_T, 10);

    UA_Variant v;
    UA_Variant_init(&v);
    UA_Float val = 42.0f;
    UA_Variant_setScalar(&v, &val, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written, err] = opc::telem::write_to_series(series, v);
    ASSERT_NIL(err);
    EXPECT_EQ(series.size(), 1);
    EXPECT_EQ(series.at<float>(0), 42.0f);

    UA_Variant v2;
    UA_Variant_init(&v2);
    UA_Float v2v = 43.0f;
    UA_Variant_setScalar(&v2, &v2v, &UA_TYPES[UA_TYPES_FLOAT]);

    auto [written2, err2] = opc::telem::write_to_series(series, v2);
    ASSERT_NIL(err2);
    EXPECT_EQ(series.size(), 2);
    EXPECT_EQ(series.at<float>(1), 43.0f);
}

/// @brief it should convert telem series to UA variant.
TEST(TelemTest, testSeriesToVariant) {
    auto series = telem::Series(telem::FLOAT32_T, 1);
    float val = 42.0f;
    series.write(val);

    auto [variant, err] = opc::telem::series_to_variant(series);
    EXPECT_EQ(err, xerrors::NIL);
    EXPECT_TRUE(UA_Variant_hasScalarType(&variant, &UA_TYPES[UA_TYPES_FLOAT]));
    EXPECT_EQ(*static_cast<float *>(variant.data), 42.0f);

    UA_Variant_clear(&variant);
}
