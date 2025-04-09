// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <iostream>

/// external
#include "gtest/gtest.h"

/// internal
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

/// protos
#include "x/cpp/loop/loop.h"
#include "x/go/telem/x/go/telem/telem.pb.h"


template<typename T>
class NumericSeriesTest : public ::testing::Test {
protected:
    void validate_vec_ctor(
        const std::vector<T> &vals,
        const telem::DataType &expected_type
    ) {
        const telem::Series s{vals};
        ASSERT_EQ(s.data_type(), expected_type);
        const auto v = s.values<T>();
        ASSERT_EQ(v.size(), vals.size());
        for (size_t i = 0; i < vals.size(); i++)
            ASSERT_EQ(v[i], vals[i]);
    }

    void validate_single_value_ctor(const T value) {
        const auto s = telem::Series(value);
        ASSERT_EQ(s.data_type(), telem::DataType::infer<T>());
        ASSERT_EQ(s.size(), 1);
        ASSERT_EQ(s.byte_size(), sizeof(T));
        const auto v = s.values<T>();
        ASSERT_EQ(v[0], value);
        ASSERT_EQ(s.at<T>(0), value);
    }


    void validate_sample_value_ctor(const T value) {
        telem::SampleValue val = value;
        telem::Series s(val);
        ASSERT_EQ(s.data_type(), telem::DataType::infer<T>());
        ASSERT_EQ(s.size(), 1);
        ASSERT_EQ(s.at<T>(0), value);
    }
};

using NumericTypes = ::testing::Types<
    uint8_t,
    uint16_t,
    uint32_t,
    uint64_t,
    int8_t,
    int16_t,
    int32_t,
    int64_t,
    float,
    double>;

TYPED_TEST_SUITE(NumericSeriesTest, NumericTypes);

/// @brief it should correctly construct the series from a ve
TYPED_TEST(NumericSeriesTest, testNumericVectorConstruction) {
    std::vector<TypeParam> vals;
    if constexpr (std::is_floating_point_v<TypeParam>)
        vals = {1.0, 2.0, 3.0, 4.0, 5.0};
    else
        vals = {1, 2, 3, 4, 5};
    this->validate_vec_ctor(vals, telem::DataType::infer<TypeParam>());
}

/// @brief it should correctly construct a series from a single numeric value.
TYPED_TEST(NumericSeriesTest, testSingleValueConstruction) {
    if constexpr (std::is_floating_point_v<TypeParam>)
        this->validate_single_value_ctor(TypeParam{1.0});
    else
        this->validate_single_value_ctor(TypeParam{1});
}

/// @brief it should correctly construct a series from a single sample value.
TYPED_TEST(NumericSeriesTest, testSampleValueConstruction) {
    if constexpr (std::is_floating_point_v<TypeParam>)
        this->validate_sample_value_ctor(TypeParam{42.5});
    else
        this->validate_sample_value_ctor(TypeParam{42});
}

//// @brief it should correctly initialize and parse a string series.
TEST(TestSeries, testStringVectorConstruction) {
    const std::vector<std::string> vals = {"hello", "world"};
    const telem::Series s{vals};
    ASSERT_EQ(s.data_type(), telem::STRING_T);
    ASSERT_EQ(s.size(), 2);
    ASSERT_EQ(s.byte_size(), 12);
    const auto v = s.strings();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

/// @brief it should correctly construct a series from a vector of timestamps.
TEST(TestSeries, testTimeStampVectorConstruction) {
    const std::vector<telem::TimeStamp> vals = {
        telem::TimeStamp(telem::MILLISECOND * 1),
        telem::TimeStamp(telem::MILLISECOND * 2),
        telem::TimeStamp(telem::MILLISECOND * 3)
    };
    const telem::Series s{vals};
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), 3);
    ASSERT_EQ(s.byte_size(), 24);
    ASSERT_EQ(s.at<int64_t>(0), telem::MILLISECOND.nanoseconds());
    ASSERT_EQ(s.at<int64_t>(1), telem::MILLISECOND.nanoseconds() * 2);
    ASSERT_EQ(s.at<int64_t>(2), telem::MILLISECOND.nanoseconds() * 3);
}

/// @brief it should correclty construct a series from a signle string.
TEST(TestSeries, testStringConstruction) {
    const std::string val = "hello";
    const telem::Series s{val};
    ASSERT_EQ(s.data_type(), telem::STRING_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 6);
    const auto v = s.strings();
    ASSERT_EQ(v[0], val);
}

/// @brief it should correctly construct a series from a single JSON string.
TEST(TestSeries, testJSONStringConstruction) {
    const std::string raw = R"({ "key": "abc" })";
    const telem::Series s(raw, telem::JSON_T);
    ASSERT_EQ(s.data_type(), telem::JSON_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 17);
    const auto v = s.strings();
    ASSERT_EQ(v[0], raw);
}

/// @brief it should correctly construct a series from a timestamp.
TEST(TestSeries, testTimestampConstruction) {
    const telem::Series s(telem::TimeStamp(100));
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::uint64_t>();
    ASSERT_EQ(v[0], 100);
}

/// @brief it should correctly construct a series the current time.
TEST(TestSeries, testTimestampNowConstruction) {
    const auto now = telem::TimeStamp::now();
    const telem::Series s(now);
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::int64_t>();
    ASSERT_EQ(v[0], now.nanoseconds());
}

// Special cases that can't be handled by the typed test
TEST(TestSeries, testSampleValueConstructionTimeStamp) {
    telem::TimeStamp ts(1000);
    telem::SampleValue val = ts;
    telem::Series s(val);
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<uint64_t>(0), 1000);
}

/// @brief it should correclty construct the series from a string sample value.
TEST(TestSeries, testSampleValueConstructionString) {
    telem::SampleValue val = std::string("test");
    telem::Series s(val);
    ASSERT_EQ(s.data_type(), telem::STRING_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<std::string>(0), "test");
}

/// @brief it should correclty construct a series from an inline vector.
TEST(TestSeries, testInlineVectorConstruction) {
    const auto s = telem::Series(std::vector<float>{1, 2, 3});
    ASSERT_EQ(s.data_type(), telem::FLOAT32_T);
    ASSERT_EQ(s.size(), 3);
    ASSERT_EQ(s.cap(), 3);
    ASSERT_EQ(s.at<float>(0), 1);
    ASSERT_EQ(s.at<float>(1), 2);
    ASSERT_EQ(s.at<float>(2), 3);
}

//// @brief it should correctly serialize and deserialize the series from protoubuf.
TEST(TestSeries, testProto) {
    const std::vector<uint16_t> vals = {1, 2, 3, 4, 5};
    const telem::Series s{vals};
    const auto s2 = new telem::PBSeries();
    s.to_proto(s2);
    const telem::Series s3{*s2};
    const auto v = s3.values<std::uint16_t>();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
    delete s2;
}

TEST(TestSeries, testConstructionSingleValue) {
    constexpr std::uint64_t value = 1;
    const auto s = telem::Series(value);
    ASSERT_EQ(s.data_type(), telem::UINT64_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::uint64_t>();
    ASSERT_EQ(v[0], 1);
    ASSERT_EQ(s.at<std::uint64_t>(0), value);
}

TEST(TestSeries, testProtoVariable) {
    const std::vector<std::string> vals = {"hello", "world22"};
    const telem::Series s{vals};
    const auto s2 = new telem::PBSeries();
    s.to_proto(s2);
    const telem::Series s3{*s2};
    const auto v = s3.strings();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

/// @brief it should correctly return the value at a particular index for a fixed
/// density data type.
TEST(TestSeries, testAtFixed) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const telem::Series s{vals};
    ASSERT_EQ(s.at<uint8_t>(0), 1);
    ASSERT_EQ(s.at<uint8_t>(1), 2);
    ASSERT_EQ(s.at<uint8_t>(2), 3);
    ASSERT_EQ(s.at<uint8_t>(3), 4);
    ASSERT_EQ(s.at<uint8_t>(4), 5);
}

/// @brief it should correctly return the value at a particular index for a variable
/// length data type.
TEST(TestSeries, testAtVar) {
    const std::vector<std::string> vals = {"hello", "world"};
    const telem::Series s{vals};
    const auto v = s.at<std::string>(0);
    ASSERT_EQ(v, "hello");
    const auto v2 = s.at<std::string>(1);
    ASSERT_EQ(v2, "world");
}

/// @brief it should allocate a series with a fixed capacity.
TEST(TestSeries, testAllocation) {
    const telem::Series s{telem::UINT32_T, 5};
    ASSERT_EQ(s.data_type(), telem::UINT32_T);
    ASSERT_EQ(s.size(), 0);
    ASSERT_EQ(s.cap(), 5);
    ASSERT_EQ(s.byte_size(), 0);
    ASSERT_EQ(s.byte_cap(), 20);
}

/// @brief it should correctly write a value to the series.
TEST(TestSeries, testWrite) {
    telem::Series s{telem::UINT32_T, 5};
    std::uint32_t value = 1;
    ASSERT_EQ(s.write(value), 1);
    value++;
    ASSERT_EQ(s.write(value), 1);
    value++;
    ASSERT_EQ(s.write(value), 1);
    value++;
    ASSERT_EQ(s.write(value), 1);
    value++;
    ASSERT_EQ(s.write(value), 1);
    value++;
    ASSERT_EQ(s.write(value), 0);
    ASSERT_EQ(s.size(), 5);
    ASSERT_EQ(s.at<std::uint32_t>(0), 1);
    ASSERT_EQ(s.at<std::uint32_t>(1), 2);
    ASSERT_EQ(s.at<std::uint32_t>(2), 3);
    ASSERT_EQ(s.at<std::uint32_t>(3), 4);
    ASSERT_EQ(s.at<std::uint32_t>(4), 5);
}

/// @brief it should correctly write a vector of values to the series.
TEST(TestSeries, testWriteVector) {
    telem::Series s{telem::FLOAT32_T, 5};
    const std::vector<float> values = {1.0, 2.0, 3.0, 4.0, 5.0};
    ASSERT_EQ(s.write(values), 5);
    ASSERT_EQ(s.write(values), 0);
    ASSERT_EQ(s.size(), 5);
    const auto v = s.values<float>();
    ASSERT_EQ(s.at<float>(1), 2.0);
    for (size_t i = 0; i < values.size(); i++)
        ASSERT_EQ(v[i], values[i]);
}

/// @brief it should correclty print out the series.
TEST(TestSeries, testOstreamOperatorForAllTypes) {
    // Refactored tests to match the new format "Series(type: TYPE, size: SIZE, cap:
    // CAP, data: [DATA ])"
    telem::Series s_uint32{telem::UINT32_T, 3};
    for (std::uint32_t i = 1; i <= 3; ++i)
        s_uint32.write(i);
    std::ostringstream oss_uint32;
    oss_uint32 << s_uint32;
    ASSERT_EQ(
        oss_uint32.str(),
        "Series(type: uint32, size: 3, cap: 3, data: [1 2 3 ])"
    );

    telem::Series s_float32{telem::FLOAT32_T, 3};
    for (float i = 1.5f; i <= 3.5f; i += 1.0f)
        s_float32.write(i);
    std::ostringstream oss_float32;
    oss_float32 << s_float32;
    ASSERT_EQ(
        oss_float32.str(),
        "Series(type: float32, size: 3, cap: 3, data: [1.5 2.5 3.5 ])"
    );

    telem::Series s_int32{telem::INT32_T, 3};
    for (int i = -1; i >= -3; --i)
        s_int32.write(i);
    std::ostringstream oss_int32;
    oss_int32 << s_int32;
    ASSERT_EQ(
        oss_int32.str(),
        "Series(type: int32, size: 3, cap: 3, data: [-1 -2 -3 ])"
    );

    telem::Series s_uint64{telem::UINT64_T, 3};
    for (std::uint64_t i = 1; i <= 3; ++i)
        s_uint64.write(i);
    std::ostringstream oss_uint64;
    oss_uint64 << s_uint64;
    ASSERT_EQ(
        oss_uint64.str(),
        "Series(type: uint64, size: 3, cap: 3, data: [1 2 3 ])"
    );

    telem::Series s_int64{telem::INT64_T, 3};
    for (std::int64_t i = -1; i >= -3; --i)
        s_int64.write(i);
    std::ostringstream oss_int64;
    oss_int64 << s_int64;
    ASSERT_EQ(
        oss_int64.str(),
        "Series(type: int64, size: 3, cap: 3, data: [-1 -2 -3 ])"
    );

    telem::Series s_float64{telem::FLOAT64_T, 3};
    for (double i = 1.5; i <= 3.5; i += 1.0)
        s_float64.write(i);
    std::ostringstream oss_float64;
    oss_float64 << s_float64;
    ASSERT_EQ(
        oss_float64.str(),
        "Series(type: float64, size: 3, cap: 3, data: [1.5 2.5 3.5 ])"
    );
    telem::Series s_uint8{telem::UINT8_T, 3};
    for (std::uint8_t i = 1; i <= 3; ++i)
        s_uint8.write(i);
    std::ostringstream oss_uint8;
    oss_uint8 << s_uint8;
    ASSERT_EQ(oss_uint8.str(), "Series(type: uint8, size: 3, cap: 3, data: [1 2 3 ])");
}

class SeriesAtTest : public ::testing::Test {
protected:
    template<typename T>
    void validateAt(
        const telem::Series &s,
        const std::vector<T> &vals,
        const telem::DataType &expected_type
    ) {
        ASSERT_EQ(s.data_type(), expected_type)
            << "Expected data type " << expected_type << " but got " << s.data_type();

        for (size_t i = 0; i < vals.size(); i++) {
            if constexpr (std::is_floating_point_v<T>)
                ASSERT_DOUBLE_EQ(s.at<T>(i), vals[i]);
            else
                ASSERT_EQ(s.at<T>(i), vals[i]);
        }
    }
};

TEST_F(SeriesAtTest, testAtUInt8) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const telem::Series s{vals};
    validateAt(s, vals, telem::UINT8_T);
}

TEST_F(SeriesAtTest, testAtUInt32) {
    const std::vector<uint32_t> vals = {100000, 200000, 300000};
    const telem::Series s{vals};
    validateAt(s, vals, telem::UINT32_T);
}

TEST_F(SeriesAtTest, testAtUInt64) {
    const std::vector<uint64_t> vals = {1000000000ULL, 2000000000ULL, 3000000000ULL};
    const telem::Series s{vals};
    validateAt(s, vals, telem::UINT64_T);
}

TEST_F(SeriesAtTest, testAtInt32) {
    const std::vector<int32_t> vals = {-100000, 0, 100000};
    const telem::Series s{vals};
    validateAt(s, vals, telem::INT32_T);
}

TEST_F(SeriesAtTest, testAtInt64) {
    const std::vector<int64_t> vals = {-1000000000LL, 0, 1000000000LL};
    const telem::Series s{vals};
    validateAt(s, vals, telem::INT64_T);
}

TEST_F(SeriesAtTest, testAtFloat32) {
    const std::vector<float> vals = {-1.5f, 0.0f, 1.5f};
    const telem::Series s{vals};
    validateAt(s, vals, telem::FLOAT32_T);
}

TEST_F(SeriesAtTest, testAtFloat64) {
    const std::vector<double> vals = {-1.5, 0.0, 1.5};
    const telem::Series s{vals};
    validateAt(s, vals, telem::FLOAT64_T);
}

TEST_F(SeriesAtTest, testAtTimestamp) {
    const std::vector<telem::TimeStamp> vals = {
        telem::TimeStamp(1000),
        telem::TimeStamp(2000),
        telem::TimeStamp(3000)
    };
    const auto s = telem::Series(vals);
    telem::SampleValue sample = s.at(0);
    ASSERT_EQ(std::get<telem::TimeStamp>(sample).nanoseconds(), 1000);
}

TEST(TestSeries, testJSONValueConstruction) {
    // Test with a simple JSON object
    json obj = {{"key", "value"}};
    telem::Series s1(obj);
    ASSERT_EQ(s1.data_type(), telem::JSON_T);
    ASSERT_EQ(s1.size(), 1);
    auto v1 = s1.strings();
    ASSERT_EQ(v1[0], obj.dump());

    // Test with a more complex JSON object
    json complex_obj = {
        {"string", "hello"},
        {"number", 42},
        {"array", {1, 2, 3}},
        {"nested", {{"a", 1}, {"b", 2}}}
    };
    telem::Series s2(complex_obj);
    ASSERT_EQ(s2.data_type(), telem::JSON_T);
    ASSERT_EQ(s2.size(), 1);
    auto v2 = s2.strings();
    ASSERT_EQ(v2[0], complex_obj.dump());

    // Test with a JSON array
    json arr = json::array({1, 2, 3});
    telem::Series s3(arr);
    ASSERT_EQ(s3.data_type(), telem::JSON_T);
    ASSERT_EQ(s3.size(), 1);
    auto v3 = s3.strings();
    ASSERT_EQ(v3[0], arr.dump());
}

TEST(TestSeries, testDeepCopy) {
    telem::Series s1{telem::UINT32_T, 3};
    s1.write(1);
    s1.write(2);
    s1.write(3);

    const telem::Series s2 = s1.deep_copy();
    ASSERT_EQ(s2.size(), 3);
    ASSERT_EQ(s2.at<std::uint32_t>(0), 1);
    ASSERT_EQ(s2.at<std::uint32_t>(1), 2);
    ASSERT_EQ(s2.at<std::uint32_t>(2), 3);
    ASSERT_EQ(s2.data_type(), telem::UINT32_T);
    ASSERT_EQ(s2.byte_size(), s1.byte_size());
    ASSERT_EQ(s2.cap(), s1.cap());
}

TEST(TestSeries, testDeepCopyVariableDataType) {
    const telem::Series s1{std::vector<std::string>{"hello", "world", "test"}};
    ASSERT_EQ(s1.size(), 3);
    const telem::Series s2 = s1.deep_copy();
    ASSERT_EQ(s2.size(), 3);
    ASSERT_EQ(s2.at<std::string>(0), "hello");
    ;
    ASSERT_EQ(s2.at<std::string>(1), "world");
    ASSERT_EQ(s2.at<std::string>(2), "test");
    ASSERT_EQ(s2.data_type(), telem::STRING_T);
    ASSERT_EQ(s2.byte_size(), s1.byte_size());
    ASSERT_EQ(s2.cap(), s1.cap());
}

TEST(TestSeriesLinspace, BasicEvenSpacing) {
    const auto start = telem::TimeStamp(100);
    const auto end = telem::TimeStamp(500);
    constexpr size_t count = 5;
    const auto s = telem::Series::linspace(start, end, count);
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), count);
    const auto values = s.values<uint64_t>();
    ASSERT_EQ(values[0], 100);
    ASSERT_EQ(values[1], 180);
    ASSERT_EQ(values[2], 260);
    ASSERT_EQ(values[3], 340);
    ASSERT_EQ(values[4], 420);
}

TEST(TestSeriesLinspace, SinglePoint) {
    const auto start = telem::TimeStamp(100);
    const auto end = telem::TimeStamp(500);
    const auto s = telem::Series::linspace(start, end, 1);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<uint64_t>(0), 100); // Should be starting value
}

TEST(TestSeriesLinspace, LargeTimestamps) {
    const auto start = telem::TimeStamp(1000000000000ULL);
    const auto end = telem::TimeStamp(1000000001000ULL);
    constexpr size_t count = 3;
    const auto s = telem::Series::linspace(start, end, count);
    const auto values = s.values<uint64_t>();
    ASSERT_EQ(values[0], 1000000000000ULL);
    ASSERT_EQ(values[1], 1000000000333ULL);
    ASSERT_EQ(values[2], 1000000000666ULL);
}

TEST(TestSeriesLinspace, EqualStartEnd) {
    const auto timestamp = telem::TimeStamp(100);
    const auto s = telem::Series::linspace(timestamp, timestamp, 5);
    const auto values = s.values<uint64_t>();
    for (size_t i = 0; i < 5; i++)
        ASSERT_EQ(values[i], 100);
}

TEST(TestSeriesLinspace, ZeroCount) {
    const auto start = telem::TimeStamp(100);
    const auto end = telem::TimeStamp(500);
    const auto s = telem::Series::linspace(start, end, 0);
    ASSERT_EQ(s.data_type(), telem::TIMESTAMP_T);
    ASSERT_EQ(s.size(), 0);
    ASSERT_EQ(s.byte_size(), 0);
}

const std::vector<uint8_t> UINT8_DATA = {1, 2, 3, 4, 5};
const std::vector<uint16_t> UINT16_DATA = {1, 2, 3, 4, 5};
const std::vector<uint32_t> UINT32_DATA = {1, 2, 3, 4, 5};
const std::vector<uint64_t> UINT64_DATA = {1, 2, 3, 4, 5};
const std::vector<int8_t> INT8_DATA = {1, 2, 3, 4, 5};
const std::vector<int16_t> INT16_DATA = {1, 2, 3, 4, 5};
const std::vector<int32_t> INT32_DATA = {1, 2, 3, 4, 5};
const std::vector<int64_t> INT64_DATA = {1, 2, 3, 4, 5};
const std::vector<float> FLOAT32_DATA = {1.0f, 2.0f, 3.0f, 4.0f, 5.0f};
const std::vector<double> FLOAT64_DATA = {1.0, 2.0, 3.0, 4.0, 5.0};

#define TEST_ALL_CASTS_FROM_SOURCE(SOURCE_TYPE, SOURCE_DATA)                           \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::UINT8_T, SOURCE_DATA.data(), SOURCE_DATA.size())    \
            .values<uint8_t>(),                                                        \
        UINT8_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::UINT16_T, SOURCE_DATA.data(), SOURCE_DATA.size())   \
            .values<uint16_t>(),                                                       \
        UINT16_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::UINT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())   \
            .values<uint32_t>(),                                                       \
        UINT32_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::UINT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())   \
            .values<uint64_t>(),                                                       \
        UINT64_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::INT8_T, SOURCE_DATA.data(), SOURCE_DATA.size())     \
            .values<int8_t>(),                                                         \
        INT8_DATA                                                                      \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::INT16_T, SOURCE_DATA.data(), SOURCE_DATA.size())    \
            .values<int16_t>(),                                                        \
        INT16_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::INT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())    \
            .values<int32_t>(),                                                        \
        INT32_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::INT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())    \
            .values<int64_t>(),                                                        \
        INT64_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::FLOAT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())  \
            .values<float>(),                                                          \
        FLOAT32_DATA                                                                   \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        telem::Series::cast(telem::FLOAT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())  \
            .values<double>(),                                                         \
        FLOAT64_DATA                                                                   \
    )

TEST(TestSeries, testCast) {
    TEST_ALL_CASTS_FROM_SOURCE(uint8_t, UINT8_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(uint16_t, UINT16_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(uint32_t, UINT32_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(uint64_t, UINT64_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(int8_t, INT8_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(int16_t, INT16_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(int32_t, INT32_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(int64_t, INT64_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(float, FLOAT32_DATA);
    TEST_ALL_CASTS_FROM_SOURCE(double, FLOAT64_DATA);
}

#define TEST_CAST_FROM_VOID_POINTER(SOURCE_TYPE, SOURCE_DATA)                          \
    do {                                                                               \
        auto const_void_ptr = static_cast<const void *>(SOURCE_DATA.data());           \
        auto source_type = telem::DataType::infer<SOURCE_TYPE>();                      \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::UINT8_T,                                                        \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<uint8_t>(),                                                    \
            UINT8_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::UINT16_T,                                                       \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<uint16_t>(),                                                   \
            UINT16_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::UINT32_T,                                                       \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<uint32_t>(),                                                   \
            UINT32_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::UINT64_T,                                                       \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<uint64_t>(),                                                   \
            UINT64_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::INT8_T,                                                         \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<int8_t>(),                                                     \
            INT8_DATA                                                                  \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::INT16_T,                                                        \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<int16_t>(),                                                    \
            INT16_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::INT32_T,                                                        \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<int32_t>(),                                                    \
            INT32_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::INT64_T,                                                        \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<int64_t>(),                                                    \
            INT64_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::FLOAT32_T,                                                      \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<float>(),                                                      \
            FLOAT32_DATA                                                               \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            telem::Series::cast(                                                       \
                telem::FLOAT64_T,                                                      \
                const_void_ptr,                                                        \
                SOURCE_DATA.size(),                                                    \
                source_type                                                            \
            )                                                                          \
                .values<double>(),                                                     \
            FLOAT64_DATA                                                               \
        );                                                                             \
    } while (0)

TEST(TestSeries, testCastVoidPointer) {
    TEST_CAST_FROM_VOID_POINTER(uint8_t, UINT8_DATA);
    TEST_CAST_FROM_VOID_POINTER(uint16_t, UINT16_DATA);
    TEST_CAST_FROM_VOID_POINTER(uint32_t, UINT32_DATA);
    TEST_CAST_FROM_VOID_POINTER(uint64_t, UINT64_DATA);
    TEST_CAST_FROM_VOID_POINTER(int8_t, INT8_DATA);
    TEST_CAST_FROM_VOID_POINTER(int16_t, INT16_DATA);
    TEST_CAST_FROM_VOID_POINTER(int32_t, INT32_DATA);
    TEST_CAST_FROM_VOID_POINTER(int64_t, INT64_DATA);
    TEST_CAST_FROM_VOID_POINTER(float, FLOAT32_DATA);
    TEST_CAST_FROM_VOID_POINTER(double, FLOAT64_DATA);
}

TEST(TestSeriesInplace, testAddInplace) {
    // Test with integer type
    std::vector<int32_t> int_data = {1, 2, 3, 4, 5};
    telem::Series int_series(int_data);
    int_series.add_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {3, 4, 5, 6, 7};
    ASSERT_EQ(int_result, expected_int);

    // Test with floating point type
    std::vector<float> float_data = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    telem::Series float_series(float_data);
    float_series.add_inplace(1.5f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {3.0f, 4.0f, 5.0f, 6.0f, 7.0f};
    ASSERT_EQ(float_result, expected_float);
}

TEST(TestSeriesInplace, testSubInplace) {
    // Test with integer type
    std::vector<int32_t> int_data = {5, 6, 7, 8, 9};
    telem::Series int_series(int_data);
    int_series.sub_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {3, 4, 5, 6, 7};
    ASSERT_EQ(int_result, expected_int);

    // Test with floating point type
    std::vector<float> float_data = {3.5f, 4.5f, 5.5f, 6.5f, 7.5f};
    telem::Series float_series(float_data);
    float_series.sub_inplace(1.5f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {2.0f, 3.0f, 4.0f, 5.0f, 6.0f};
    ASSERT_EQ(float_result, expected_float);
}

TEST(TestSeriesInplace, testMultiplyInplace) {
    // Test with integer type
    std::vector<int32_t> int_data = {1, 2, 3, 4, 5};
    telem::Series int_series(int_data);
    int_series.multiply_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {2, 4, 6, 8, 10};
    ASSERT_EQ(int_result, expected_int);

    // Test with floating point type
    std::vector<float> float_data = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    telem::Series float_series(float_data);
    float_series.multiply_inplace(2.0f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {3.0f, 5.0f, 7.0f, 9.0f, 11.0f};
    ASSERT_EQ(float_result, expected_float);
}

TEST(TestSeriesInplace, testDivideInplace) {
    // Test with integer type
    std::vector<int32_t> int_data = {2, 4, 6, 8, 10};
    telem::Series int_series(int_data);
    int_series.divide_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {1, 2, 3, 4, 5};
    ASSERT_EQ(int_result, expected_int);

    // Test with floating point type
    std::vector<float> float_data = {3.0f, 5.0f, 7.0f, 9.0f, 11.0f};
    telem::Series float_series(float_data);
    float_series.divide_inplace(2.0f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    ASSERT_EQ(float_result, expected_float);

    // Test division by zero
    telem::Series zero_test(std::vector<int32_t>{1, 2, 3});
    ASSERT_THROW(zero_test.divide_inplace(0), std::runtime_error);
}

// Test all operations with different numeric types
TEST(TestSeriesInplace, testMultipleTypes) {
    // Test uint8_t
    std::vector<uint8_t> uint8_data = {1, 2, 3, 4, 5};
    telem::Series uint8_series(uint8_data);
    uint8_series.add_inplace(1);
    uint8_series.multiply_inplace(2);
    uint8_series.sub_inplace(2);
    uint8_series.divide_inplace(2);
    auto uint8_result = uint8_series.values<uint8_t>();
    std::vector<uint8_t> expected_uint8 = {1, 2, 3, 4, 5};
    ASSERT_EQ(uint8_result, expected_uint8);

    // Test double
    std::vector<double> double_data = {1.0, 2.0, 3.0, 4.0, 5.0};
    telem::Series double_series(double_data);
    double_series.add_inplace(1.0);
    double_series.multiply_inplace(2.0);
    double_series.sub_inplace(2.0);
    double_series.divide_inplace(2.0);
    auto double_result = double_series.values<double>();
    std::vector<double> expected_double = {1.0, 2.0, 3.0, 4.0, 5.0};
    ASSERT_EQ(double_result, expected_double);
}

TEST(TestSeries, testJSONVectorConstruction) {
    // Test with simple JSON objects
    std::vector<json> simple_values = {
        json{{"key1", "value1"}},
        json{{"key2", "value2"}}
    };
    telem::Series s1(simple_values);
    ASSERT_EQ(s1.data_type(), telem::JSON_T);
    ASSERT_EQ(s1.size(), 2);
    auto strings1 = s1.strings();
    ASSERT_EQ(strings1[0], R"({"key1":"value1"})");
    ASSERT_EQ(strings1[1], R"({"key2":"value2"})");

    // Test with mixed JSON types
    std::vector<json> complex_values = {
        json{{"string", "hello"}},
        json{{"number", 42}},
        json::array({1, 2, 3}),
        json{{"nested", {{"a", 1}, {"b", 2}}}}
    };
    telem::Series s2(complex_values);
    ASSERT_EQ(s2.data_type(), telem::JSON_T);
    ASSERT_EQ(s2.size(), 4);
    auto strings2 = s2.strings();
    ASSERT_EQ(strings2[0], R"({"string":"hello"})");
    ASSERT_EQ(strings2[1], R"({"number":42})");
    ASSERT_EQ(strings2[2], R"([1,2,3])");
    ASSERT_EQ(strings2[3], R"({"nested":{"a":1,"b":2}})");

    // Test with empty vector
    std::vector<json> empty_values;
    telem::Series s3(empty_values);
    ASSERT_EQ(s3.data_type(), telem::JSON_T);
    ASSERT_EQ(s3.size(), 0);
    ASSERT_EQ(s3.byte_size(), 0);
}
