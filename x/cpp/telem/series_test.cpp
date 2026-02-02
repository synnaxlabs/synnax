// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>

#include "gtest/gtest.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "x/go/telem/pb/telem.pb.h"

namespace x::telem {
template<typename T>
class NumericSeriesTest : public ::testing::Test {
protected:
    void validate_vec_ctor(const std::vector<T> &vals, const DataType &expected_type) {
        const Series s{vals};
        ASSERT_EQ(s.data_type(), expected_type);
        const auto v = s.values<T>();
        ASSERT_EQ(v.size(), vals.size());
        for (size_t i = 0; i < vals.size(); i++)
            ASSERT_EQ(v[i], vals[i]);
    }

    void validate_single_value_ctor(const T value) {
        const auto s = Series(value);
        ASSERT_EQ(s.data_type(), DataType::infer<T>());
        ASSERT_EQ(s.size(), 1);
        ASSERT_EQ(s.byte_size(), sizeof(T));
        const auto v = s.values<T>();
        ASSERT_EQ(v[0], value);
        ASSERT_EQ(s.at<T>(0), value);
    }

    void validate_sample_value_ctor(const T value) {
        SampleValue val = value;
        Series s(val);
        ASSERT_EQ(s.data_type(), DataType::infer<T>());
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

/// @brief it should correctly construct the series from a vector.
TYPED_TEST(NumericSeriesTest, testNumericVectorConstruction) {
    std::vector<TypeParam> vals;
    if constexpr (std::is_floating_point_v<TypeParam>)
        vals = {1.0, 2.0, 3.0, 4.0, 5.0};
    else
        vals = {1, 2, 3, 4, 5};
    this->validate_vec_ctor(vals, DataType::infer<TypeParam>());
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
    const Series s{vals};
    ASSERT_EQ(s.data_type(), STRING_T);
    ASSERT_EQ(s.size(), 2);
    ASSERT_EQ(s.byte_size(), 12);
    const auto v = s.strings();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

/// @brief it should correctly construct a series from a vector of timestamps.
TEST(TestSeries, testTimeStampVectorConstruction) {
    const std::vector<TimeStamp> vals = {
        TimeStamp(MILLISECOND * 1),
        TimeStamp(MILLISECOND * 2),
        TimeStamp(MILLISECOND * 3)
    };
    const Series s{vals};
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
    ASSERT_EQ(s.size(), 3);
    ASSERT_EQ(s.byte_size(), 24);
    ASSERT_EQ(s.at<int64_t>(0), MILLISECOND.nanoseconds());
    ASSERT_EQ(s.at<int64_t>(1), MILLISECOND.nanoseconds() * 2);
    ASSERT_EQ(s.at<int64_t>(2), MILLISECOND.nanoseconds() * 3);
}

/// @brief it should correctly construct a series from a signle string.
TEST(TestSeries, testStringConstruction) {
    const std::string val = "hello";
    const Series s{val};
    ASSERT_EQ(s.data_type(), STRING_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 6);
    const auto v = s.strings();
    ASSERT_EQ(v[0], val);
}

/// @brief it should correctly construct a series from a single json::json string.
TEST(TestSeries, testJSONStringConstruction) {
    const std::string raw = R"({ "key": "abc" })";
    const Series s(raw, x::telem::JSON_T);
    ASSERT_EQ(s.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 17);
    const auto v = s.strings();
    ASSERT_EQ(v[0], raw);
}

/// @brief it should correctly construct a series from a timestamp.
TEST(TestSeries, testTimestampConstruction) {
    const Series s(TimeStamp(100));
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::uint64_t>();
    ASSERT_EQ(v[0], 100);
}

/// @brief it should correctly construct a series at the current time.
TEST(TestSeries, testTimestampNowConstruction) {
    const auto now = TimeStamp::now();
    const Series s(now);
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::int64_t>();
    ASSERT_EQ(v[0], now.nanoseconds());
}

/// @brief it should correctly construct the series from a timestamp sample value.
TEST(TestSeries, testSampleValueConstructionTimeStamp) {
    TimeStamp ts(1000);
    SampleValue val = ts;
    Series s(val);
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<uint64_t>(0), 1000);
}

/// @brief it should correctly construct the series from a string sample value.
TEST(TestSeries, testSampleValueConstructionString) {
    SampleValue val = std::string("test");
    Series s(val);
    ASSERT_EQ(s.data_type(), STRING_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<std::string>(0), "test");
}

/// @brief it should correctly construct a series from an inline vector.
TEST(TestSeries, testInlineVectorConstruction) {
    const auto s = Series(std::vector<float>{1, 2, 3});
    ASSERT_EQ(s.data_type(), FLOAT32_T);
    ASSERT_EQ(s.size(), 3);
    ASSERT_EQ(s.cap(), 3);
    ASSERT_EQ(s.at<float>(0), 1);
    ASSERT_EQ(s.at<float>(1), 2);
    ASSERT_EQ(s.at<float>(2), 3);
}

/// @brief it should correctly construct a series from a single value.
TEST(TestSeries, testConstructionSingleValue) {
    constexpr std::uint64_t value = 1;
    const auto s = Series(value);
    ASSERT_EQ(s.data_type(), UINT64_T);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.byte_size(), 8);
    const auto v = s.values<std::uint64_t>();
    ASSERT_EQ(v[0], 1);
    ASSERT_EQ(s.at<std::uint64_t>(0), value);
}

/// @brief it should construct a variable density series from it's protobuf
/// representation.
TEST(TestSeries, testConstrucitonFromVariableProtoSeries) {
    const std::vector<std::string> vals = {"hello", "world22"};
    const Series s{vals};
    auto s2 = s.to_proto();
    auto [s3, err] = Series::from_proto(s2);
    const auto v = s3.strings();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

/// @brief it should correctly return the value at a particular index for a fixed
/// density data type.
TEST(TestSeries, testAtFixed) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const Series s{vals};
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
    const Series s{vals};
    const auto v = s.at<std::string>(0);
    ASSERT_EQ(v, "hello");
    const auto v2 = s.at<std::string>(1);
    ASSERT_EQ(v2, "world");
}

/// @brief it should allocate a series with a fixed capacity.
TEST(TestSeries, testAllocation) {
    const Series s{UINT32_T, 5};
    ASSERT_EQ(s.data_type(), UINT32_T);
    ASSERT_EQ(s.size(), 0);
    ASSERT_EQ(s.cap(), 5);
    ASSERT_EQ(s.byte_size(), 0);
    ASSERT_EQ(s.byte_cap(), 20);
}

/// @brief it should pre-allocate a variable length series with a fixed capacity instead
/// of a fixed size.
TEST(TestSeries, testAllocationVariable) {
    const Series s{STRING_T, 20};
    ASSERT_EQ(s.data_type(), STRING_T);
    ASSERT_EQ(s.size(), 0);
    ASSERT_EQ(s.cap(), 0);
    ASSERT_EQ(s.byte_size(), 0);
    ASSERT_EQ(s.byte_cap(), 20);
}

/// @brief it should correctly write a value to the series.
TEST(TestSeries, testWrite) {
    Series s{UINT32_T, 5};
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
    Series s{FLOAT32_T, 5};
    const std::vector<float> values = {1.0, 2.0, 3.0, 4.0, 5.0};
    ASSERT_EQ(s.write(values), 5);
    ASSERT_EQ(s.write(values), 0);
    ASSERT_EQ(s.size(), 5);
    const auto v = s.values<float>();
    ASSERT_EQ(s.at<float>(1), 2.0);
    for (size_t i = 0; i < values.size(); i++)
        ASSERT_EQ(v[i], values[i]);
}

/// @brief it should correctly print out the series.
TEST(TestSeries, testOstreamOperatorForAllTypes) {
    // Refactored tests to match the new format "Series(type: TYPE, size: SIZE, cap:
    // CAP, data: [DATA ])"
    Series s_uint32{UINT32_T, 3};
    for (std::uint32_t i = 1; i <= 3; ++i)
        s_uint32.write(i);
    std::ostringstream oss_uint32;
    oss_uint32 << s_uint32;
    ASSERT_EQ(
        oss_uint32.str(),
        "Series(type: uint32, size: 3, cap: 3, data: [1 2 3 ])"
    );

    Series s_float32{FLOAT32_T, 3};
    for (float i = 1.5f; i <= 3.5f; i += 1.0f)
        s_float32.write(i);
    std::ostringstream oss_float32;
    oss_float32 << s_float32;
    ASSERT_EQ(
        oss_float32.str(),
        "Series(type: float32, size: 3, cap: 3, data: [1.5 2.5 3.5 ])"
    );

    Series s_int32{INT32_T, 3};
    for (int i = -1; i >= -3; --i)
        s_int32.write(i);
    std::ostringstream oss_int32;
    oss_int32 << s_int32;
    ASSERT_EQ(
        oss_int32.str(),
        "Series(type: int32, size: 3, cap: 3, data: [-1 -2 -3 ])"
    );

    Series s_uint64{UINT64_T, 3};
    for (std::uint64_t i = 1; i <= 3; ++i)
        s_uint64.write(i);
    std::ostringstream oss_uint64;
    oss_uint64 << s_uint64;
    ASSERT_EQ(
        oss_uint64.str(),
        "Series(type: uint64, size: 3, cap: 3, data: [1 2 3 ])"
    );

    Series s_int64{INT64_T, 3};
    for (std::int64_t i = -1; i >= -3; --i)
        s_int64.write(i);
    std::ostringstream oss_int64;
    oss_int64 << s_int64;
    ASSERT_EQ(
        oss_int64.str(),
        "Series(type: int64, size: 3, cap: 3, data: [-1 -2 -3 ])"
    );

    Series s_float64{FLOAT64_T, 3};
    for (double i = 1.5; i <= 3.5; i += 1.0)
        s_float64.write(i);
    std::ostringstream oss_float64;
    oss_float64 << s_float64;
    ASSERT_EQ(
        oss_float64.str(),
        "Series(type: float64, size: 3, cap: 3, data: [1.5 2.5 3.5 ])"
    );
    Series s_uint8{UINT8_T, 3};
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
        const Series &s,
        const std::vector<T> &vals,
        const DataType &expected_type
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

/// @brief it should retrieve uint8 values at specific indices.
TEST_F(SeriesAtTest, testAtUInt8) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const Series s{vals};
    validateAt(s, vals, UINT8_T);
}

/// @brief it should retrieve uint32 values at specific indices.
TEST_F(SeriesAtTest, testAtUInt32) {
    const std::vector<uint32_t> vals = {100000, 200000, 300000};
    const Series s{vals};
    validateAt(s, vals, UINT32_T);
}

/// @brief it should retrieve uint64 values at specific indices.
TEST_F(SeriesAtTest, testAtUInt64) {
    const std::vector<uint64_t> vals = {1000000000ULL, 2000000000ULL, 3000000000ULL};
    const Series s{vals};
    validateAt(s, vals, UINT64_T);
}

/// @brief it should retrieve int32 values at specific indices.
TEST_F(SeriesAtTest, testAtInt32) {
    const std::vector<int32_t> vals = {-100000, 0, 100000};
    const Series s{vals};
    validateAt(s, vals, INT32_T);
}

/// @brief it should retrieve int64 values at specific indices.
TEST_F(SeriesAtTest, testAtInt64) {
    const std::vector<int64_t> vals = {-1000000000LL, 0, 1000000000LL};
    const Series s{vals};
    validateAt(s, vals, INT64_T);
}

/// @brief it should retrieve float32 values at specific indices.
TEST_F(SeriesAtTest, testAtFloat32) {
    const std::vector vals = {-1.5f, 0.0f, 1.5f};
    const Series s{vals};
    validateAt(s, vals, FLOAT32_T);
}

/// @brief it should retrieve float64 values at specific indices.
TEST_F(SeriesAtTest, testAtFloat64) {
    const std::vector vals = {-1.5, 0.0, 1.5};
    const Series s{vals};
    validateAt(s, vals, FLOAT64_T);
}

/// @brief it should retrieve timestamp values at specific indices.
TEST_F(SeriesAtTest, testAtTimestamp) {
    const std::vector vals = {TimeStamp(1000), TimeStamp(2000), TimeStamp(3000)};
    const auto s = Series(vals);
    SampleValue sample = s.at(0);
    ASSERT_EQ(std::get<TimeStamp>(sample).nanoseconds(), 1000);
}

/// @brief it should construct a series from json::json values.
TEST(TestSeries, testJsonValueConstruction) {
    json::json obj = {{"key", "value"}};
    Series s1(obj);
    ASSERT_EQ(s1.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s1.size(), 1);
    auto v1 = s1.strings();
    ASSERT_EQ(v1[0], obj.dump());

    json::json complex_obj = {
        {"string", "hello"},
        {"number", 42},
        {"array", {1, 2, 3}},
        {"nested", {{"a", 1}, {"b", 2}}}
    };
    Series s2(complex_obj);
    ASSERT_EQ(s2.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s2.size(), 1);
    auto v2 = s2.strings();
    ASSERT_EQ(v2[0], complex_obj.dump());

    json::json arr = json::json::array({1, 2, 3});
    Series s3(arr);
    ASSERT_EQ(s3.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s3.size(), 1);
    auto v3 = s3.strings();
    ASSERT_EQ(v3[0], arr.dump());
}

/// @brief it should deep copy a fixed data type series.
TEST(TestSeries, testDeepCopy) {
    Series s1{UINT32_T, 3};
    s1.write(1);
    s1.write(2);
    s1.write(3);
    s1.alignment = Alignment(5, 10);

    const Series s2 = s1.deep_copy();
    ASSERT_EQ(s2.size(), 3);
    ASSERT_EQ(s2.at<std::uint32_t>(0), 1);
    ASSERT_EQ(s2.at<std::uint32_t>(1), 2);
    ASSERT_EQ(s2.at<std::uint32_t>(2), 3);
    ASSERT_EQ(s2.data_type(), UINT32_T);
    ASSERT_EQ(s2.byte_size(), s1.byte_size());
    ASSERT_EQ(s2.cap(), s1.cap());
    ASSERT_EQ(s2.alignment.uint64(), s1.alignment.uint64());
}

/// @brief it should deep copy a variable data type series.
TEST(TestSeries, testDeepCopyVariableDataType) {
    Series s1{std::vector<std::string>{"hello", "world", "test"}};
    s1.alignment = Alignment(7, 42);
    ASSERT_EQ(s1.size(), 3);
    const Series s2 = s1.deep_copy();
    ASSERT_EQ(s2.size(), 3);
    ASSERT_EQ(s2.at<std::string>(0), "hello");
    ASSERT_EQ(s2.at<std::string>(1), "world");
    ASSERT_EQ(s2.at<std::string>(2), "test");
    ASSERT_EQ(s2.data_type(), STRING_T);
    ASSERT_EQ(s2.byte_size(), s1.byte_size());
    ASSERT_EQ(s2.cap(), s1.cap());
    ASSERT_EQ(s2.alignment.uint64(), s1.alignment.uint64());
}

/// @brief it should preserve alignment when moving a series.
TEST(TestSeries, testMovePreservesAlignment) {
    Series s1{UINT32_T, 3};
    s1.write(1);
    s1.write(2);
    s1.write(3);
    s1.alignment = Alignment(5, 10);

    Series s2 = std::move(s1);
    ASSERT_EQ(s2.size(), 3);
    ASSERT_EQ(s2.at<std::uint32_t>(0), 1);
    ASSERT_EQ(s2.at<std::uint32_t>(1), 2);
    ASSERT_EQ(s2.at<std::uint32_t>(2), 3);
    ASSERT_EQ(s2.data_type(), UINT32_T);
    ASSERT_EQ(s2.alignment.uint64(), Alignment(5, 10).uint64());
}

/// @brief it should generate evenly spaced timestamps.
TEST(TestSeriesLinspace, BasicEvenSpacing) {
    const auto start = TimeStamp(100);
    const auto end = TimeStamp(500);
    constexpr size_t count = 5;
    const auto s = Series::linspace(start, end, count);
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
    ASSERT_EQ(s.size(), count);
    const auto values = s.values<uint64_t>();
    ASSERT_EQ(values[0], 100);
    ASSERT_EQ(values[1], 180);
    ASSERT_EQ(values[2], 260);
    ASSERT_EQ(values[3], 340);
    ASSERT_EQ(values[4], 420);
}

/// @brief it should generate a single point linspace series.
TEST(TestSeriesLinspace, SinglePoint) {
    const auto start = TimeStamp(100);
    const auto end = TimeStamp(500);
    const auto s = Series::linspace(start, end, 1);
    ASSERT_EQ(s.size(), 1);
    ASSERT_EQ(s.at<uint64_t>(0), 100); // Should be starting value
}

/// @brief it should generate linspace with large timestamps.
TEST(TestSeriesLinspace, LargeTimestamps) {
    const auto start = TimeStamp(1000000000000ULL);
    const auto end = TimeStamp(1000000001000ULL);
    constexpr size_t count = 3;
    const auto s = Series::linspace(start, end, count);
    const auto values = s.values<uint64_t>();
    ASSERT_EQ(values[0], 1000000000000ULL);
    ASSERT_EQ(values[1], 1000000000333ULL);
    ASSERT_EQ(values[2], 1000000000666ULL);
}

/// @brief it should generate constant values when start equals end.
TEST(TestSeriesLinspace, EqualStartEnd) {
    const auto timestamp = TimeStamp(100);
    const auto s = Series::linspace(timestamp, timestamp, 5);
    const auto values = s.values<uint64_t>();
    for (size_t i = 0; i < 5; i++)
        ASSERT_EQ(values[i], 100);
}

/// @brief it should generate an empty series with zero count.
TEST(TestSeriesLinspace, ZeroCount) {
    const auto start = TimeStamp(100);
    const auto end = TimeStamp(500);
    const auto s = Series::linspace(start, end, 0);
    ASSERT_EQ(s.data_type(), TIMESTAMP_T);
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
        Series::cast(UINT8_T, SOURCE_DATA.data(), SOURCE_DATA.size())                  \
            .values<uint8_t>(),                                                        \
        UINT8_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(UINT16_T, SOURCE_DATA.data(), SOURCE_DATA.size())                 \
            .values<uint16_t>(),                                                       \
        UINT16_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(UINT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())                 \
            .values<uint32_t>(),                                                       \
        UINT32_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(UINT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())                 \
            .values<uint64_t>(),                                                       \
        UINT64_DATA                                                                    \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(INT8_T, SOURCE_DATA.data(), SOURCE_DATA.size()).values<int8_t>(), \
        INT8_DATA                                                                      \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(INT16_T, SOURCE_DATA.data(), SOURCE_DATA.size())                  \
            .values<int16_t>(),                                                        \
        INT16_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(INT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())                  \
            .values<int32_t>(),                                                        \
        INT32_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(INT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())                  \
            .values<int64_t>(),                                                        \
        INT64_DATA                                                                     \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(FLOAT32_T, SOURCE_DATA.data(), SOURCE_DATA.size())                \
            .values<float>(),                                                          \
        FLOAT32_DATA                                                                   \
    );                                                                                 \
    ASSERT_EQ(                                                                         \
        Series::cast(FLOAT64_T, SOURCE_DATA.data(), SOURCE_DATA.size())                \
            .values<double>(),                                                         \
        FLOAT64_DATA                                                                   \
    )

/// @brief it should cast series data between all numeric types.
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
        auto source_type = DataType::infer<SOURCE_TYPE>();                             \
        ASSERT_EQ(                                                                     \
            Series::cast(UINT8_T, const_void_ptr, SOURCE_DATA.size(), source_type)     \
                .values<uint8_t>(),                                                    \
            UINT8_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(UINT16_T, const_void_ptr, SOURCE_DATA.size(), source_type)    \
                .values<uint16_t>(),                                                   \
            UINT16_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(UINT32_T, const_void_ptr, SOURCE_DATA.size(), source_type)    \
                .values<uint32_t>(),                                                   \
            UINT32_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(UINT64_T, const_void_ptr, SOURCE_DATA.size(), source_type)    \
                .values<uint64_t>(),                                                   \
            UINT64_DATA                                                                \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(INT8_T, const_void_ptr, SOURCE_DATA.size(), source_type)      \
                .values<int8_t>(),                                                     \
            INT8_DATA                                                                  \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(INT16_T, const_void_ptr, SOURCE_DATA.size(), source_type)     \
                .values<int16_t>(),                                                    \
            INT16_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(INT32_T, const_void_ptr, SOURCE_DATA.size(), source_type)     \
                .values<int32_t>(),                                                    \
            INT32_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(INT64_T, const_void_ptr, SOURCE_DATA.size(), source_type)     \
                .values<int64_t>(),                                                    \
            INT64_DATA                                                                 \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(FLOAT32_T, const_void_ptr, SOURCE_DATA.size(), source_type)   \
                .values<float>(),                                                      \
            FLOAT32_DATA                                                               \
        );                                                                             \
        ASSERT_EQ(                                                                     \
            Series::cast(FLOAT64_T, const_void_ptr, SOURCE_DATA.size(), source_type)   \
                .values<double>(),                                                     \
            FLOAT64_DATA                                                               \
        );                                                                             \
    } while (0)

/// @brief it should cast series data from void pointer with source type.
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

/// @brief it should add a scalar value inplace to all series elements.
TEST(TestSeriesInplace, testAddInplace) {
    std::vector int_data = {1, 2, 3, 4, 5};
    Series int_series(int_data);
    int_series.add_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {3, 4, 5, 6, 7};
    ASSERT_EQ(int_result, expected_int);

    std::vector float_data = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    Series float_series(float_data);
    float_series.add_inplace(1.5f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {3.0f, 4.0f, 5.0f, 6.0f, 7.0f};
    ASSERT_EQ(float_result, expected_float);
}

/// @brief it should subtract a scalar value inplace from all series elements.
TEST(TestSeriesInplace, testSubInplace) {
    std::vector int_data = {5, 6, 7, 8, 9};
    Series int_series(int_data);
    int_series.sub_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector<int32_t> expected_int = {3, 4, 5, 6, 7};
    ASSERT_EQ(int_result, expected_int);

    std::vector float_data = {3.5f, 4.5f, 5.5f, 6.5f, 7.5f};
    Series float_series(float_data);
    float_series.sub_inplace(1.5f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {2.0f, 3.0f, 4.0f, 5.0f, 6.0f};
    ASSERT_EQ(float_result, expected_float);
}

/// @brief it should multiply a scalar value inplace to all series elements.
TEST(TestSeriesInplace, testMultiplyInplace) {
    std::vector int_data = {1, 2, 3, 4, 5};
    Series int_series(int_data);
    int_series.multiply_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector expected_int = {2, 4, 6, 8, 10};
    ASSERT_EQ(int_result, expected_int);

    std::vector float_data = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    Series float_series(float_data);
    float_series.multiply_inplace(2.0f);
    auto float_result = float_series.values<float>();
    std::vector expected_float = {3.0f, 5.0f, 7.0f, 9.0f, 11.0f};
    ASSERT_EQ(float_result, expected_float);
}

/// @brief it should divide all series elements inplace by a scalar value.
TEST(TestSeriesInplace, testDivideInplace) {
    std::vector int_data = {2, 4, 6, 8, 10};
    Series int_series(int_data);
    int_series.divide_inplace(2);
    auto int_result = int_series.values<int32_t>();
    std::vector expected_int = {1, 2, 3, 4, 5};
    ASSERT_EQ(int_result, expected_int);

    std::vector float_data = {3.0f, 5.0f, 7.0f, 9.0f, 11.0f};
    Series float_series(float_data);
    float_series.divide_inplace(2.0f);
    auto float_result = float_series.values<float>();
    std::vector<float> expected_float = {1.5f, 2.5f, 3.5f, 4.5f, 5.5f};
    ASSERT_EQ(float_result, expected_float);

    Series zero_test(std::vector{1, 2, 3});
    ASSERT_THROW(zero_test.divide_inplace(0), std::runtime_error);
}

/// @brief it should perform inplace operations on different numeric types.
TEST(TestSeriesInplace, testMultipleTypes) {
    std::vector<uint8_t> uint8_data = {1, 2, 3, 4, 5};
    Series uint8_series(uint8_data);
    uint8_series.add_inplace(1);
    uint8_series.multiply_inplace(2);
    uint8_series.sub_inplace(2);
    uint8_series.divide_inplace(2);
    auto uint8_result = uint8_series.values<uint8_t>();
    std::vector<uint8_t> expected_uint8 = {1, 2, 3, 4, 5};
    ASSERT_EQ(uint8_result, expected_uint8);

    std::vector double_data = {1.0, 2.0, 3.0, 4.0, 5.0};
    Series double_series(double_data);
    double_series.add_inplace(1.0);
    double_series.multiply_inplace(2.0);
    double_series.sub_inplace(2.0);
    double_series.divide_inplace(2.0);
    auto double_result = double_series.values<double>();
    std::vector expected_double = {1.0, 2.0, 3.0, 4.0, 5.0};
    ASSERT_EQ(double_result, expected_double);
}

/// @brief it should construct a series from a vector of json::json values.
TEST(TestSeries, testJSONVectorConstruction) {
    std::vector<json::json> simple_values = {
        json::json{{"key1", "value1"}},
        json::json{{"key2", "value2"}}
    };
    Series s1(simple_values);
    ASSERT_EQ(s1.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s1.size(), 2);
    auto strings1 = s1.strings();
    ASSERT_EQ(strings1[0], R"({"key1":"value1"})");
    ASSERT_EQ(strings1[1], R"({"key2":"value2"})");

    // Test with mixed json::json types including nulls and booleans
    std::vector<json::json> complex_values = {
        json::json{{"string", "hello"}},
        json::json{{"number", 42}},
        json::json{{"null_value", nullptr}},
        json::json{{"bool_value", true}},
        json::json::array({1, 2, 3}),
        json::json{{"nested", {{"a", 1}, {"b", 2}, {"c", false}, {"d", nullptr}}}}
    };
    Series s2(complex_values);
    ASSERT_EQ(s2.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s2.size(), 6);
    auto strings2 = s2.strings();
    ASSERT_EQ(strings2[0], R"({"string":"hello"})");
    ASSERT_EQ(strings2[1], R"({"number":42})");
    ASSERT_EQ(strings2[2], R"({"null_value":null})");
    ASSERT_EQ(strings2[3], R"({"bool_value":true})");
    ASSERT_EQ(strings2[4], R"([1,2,3])");
    ASSERT_EQ(strings2[5], R"({"nested":{"a":1,"b":2,"c":false,"d":null}})");

    // Test with empty vector
    std::vector<json::json> empty_values;
    Series s3(empty_values);
    ASSERT_EQ(s3.data_type(), x::telem::JSON_T);
    ASSERT_EQ(s3.size(), 0);
    ASSERT_EQ(s3.byte_size(), 0);
}

/// @brief it should retrieve json::json values from a series.
TEST(TestSeries, testJSONValuesBasic) {
    std::vector<json::json> input_values = {
        json::json{{"key1", "value1"}},
        json::json{{"key2", 42}},
        json::json{{"null_field", nullptr}},
        json::json{{"bool_true", true}},
        json::json{{"bool_false", false}},
        json::json::array({1, 2, nullptr, true, false}),
        json::json{{"nested", {{"a", 1}, {"b", "test"}, {"c", nullptr}, {"d", true}}}}
    };

    const Series s(input_values);
    const auto output_values = s.json_values();

    ASSERT_EQ(output_values.size(), input_values.size());
    for (size_t i = 0; i < input_values.size(); i++)
        ASSERT_EQ(output_values[i], input_values[i]);
}

/// @brief it should return empty vector for empty json::json series.
TEST(TestSeries, testJSONValuesEmpty) {
    const Series empty_series(std::vector<json::json>{});
    auto empty_values = empty_series.json_values();
    ASSERT_TRUE(empty_values.empty());
}

/// @brief it should throw error when getting json::json values from non-json::json
/// series.
TEST(TestSeries, testJSONValuesErrorOnNonJSON) {
    const Series non_json_series(std::vector<int>{1, 2, 3});
    ASSERT_THROW((void) non_json_series.json_values(), std::runtime_error);
}

/// @brief it should fill series from binary reader with fixed size data.
TEST(TestSeries, testFillFromFixedSize) {
    std::vector<uint32_t> source_data = {1, 2, 3, 4, 5};
    std::vector<uint8_t> binary_data;
    binary::Writer writer(binary_data, source_data.size() * sizeof(uint32_t));
    writer.write(source_data.data(), source_data.size() * sizeof(uint32_t));

    Series series(UINT32_T, 10);
    binary::Reader reader(binary_data);

    size_t bytes_read = series.fill_from(reader);

    ASSERT_EQ(bytes_read, source_data.size() * sizeof(uint32_t));
    ASSERT_EQ(series.size(), source_data.size());
    auto values = series.values<uint32_t>();
    ASSERT_EQ(values, source_data);
}

/// @brief it should fill series from binary reader with string data.
TEST(TestSeries, testFillFromString) {
    std::vector<std::string> source_strings = {"hello", "world", "test"};
    std::vector<uint8_t> binary_data;
    size_t total_size = 0;
    for (const auto &str: source_strings)
        total_size += str.size() + 1; // +1 for newline terminator

    binary::Writer writer(binary_data, total_size);
    for (const auto &str: source_strings) {
        writer.write(str.data(), str.size());
        writer.uint8('\n');
    }

    Series series(STRING_T, total_size);
    binary::Reader reader(binary_data);

    size_t bytes_read = series.fill_from(reader);

    ASSERT_EQ(bytes_read, total_size);
    ASSERT_EQ(series.size(), source_strings.size());
    auto values = series.strings();
    ASSERT_EQ(values, source_strings);
}

/// @brief it should fill series partially when capacity is less than data.
TEST(TestSeries, testFillFromPartial) {
    std::vector<uint16_t> source_data = {1, 2, 3, 4, 5};
    std::vector<uint8_t> binary_data;
    binary::Writer writer(binary_data, source_data.size() * sizeof(uint16_t));
    writer.write(source_data.data(), source_data.size() * sizeof(uint16_t));

    Series series(UINT16_T, 3); // Only space for 3 elements
    binary::Reader reader(binary_data);

    size_t bytes_read = series.fill_from(reader);

    ASSERT_EQ(bytes_read, 3 * sizeof(uint16_t));
    ASSERT_EQ(series.size(), 3);
    auto values = series.values<uint16_t>();
    ASSERT_EQ(values.size(), 3);
    for (size_t i = 0; i < 3; i++)
        ASSERT_EQ(values[i], source_data[i]);
}

/// @brief it should handle empty binary reader for fill_from.
TEST(TestSeries, testFillFromEmpty) {
    std::vector<uint8_t> empty_data;
    binary::Reader reader(empty_data);

    Series series(UINT32_T, 5);
    size_t bytes_read = series.fill_from(reader);

    ASSERT_EQ(bytes_read, 0);
    ASSERT_EQ(series.size(), 0);
}

/// @brief it should fill series from multiple binary reader reads.
TEST(TestSeries, testFillFromMultipleReads) {
    std::vector source_data1 = {1.0f, 2.0f, 3.0f};
    std::vector source_data2 = {4.0f, 5.0f};

    std::vector<uint8_t> binary_data1, binary_data2;
    binary::Writer writer1(binary_data1, source_data1.size() * sizeof(float));
    binary::Writer writer2(binary_data2, source_data2.size() * sizeof(float));

    writer1.write(source_data1.data(), source_data1.size() * sizeof(float));
    writer2.write(source_data2.data(), source_data2.size() * sizeof(float));

    Series series(FLOAT32_T, 5);

    binary::Reader reader1(binary_data1);
    size_t bytes_read1 = series.fill_from(reader1);
    ASSERT_EQ(bytes_read1, source_data1.size() * sizeof(float));
    ASSERT_EQ(series.size(), source_data1.size());

    binary::Reader reader2(binary_data2);
    size_t bytes_read2 = series.fill_from(reader2);
    ASSERT_EQ(bytes_read2, source_data2.size() * sizeof(float));
    ASSERT_EQ(series.size(), source_data1.size() + source_data2.size());

    auto values = series.values<float>();
    std::vector<float> expected;
    expected.insert(expected.end(), source_data1.begin(), source_data1.end());
    expected.insert(expected.end(), source_data2.begin(), source_data2.end());
    ASSERT_EQ(values, expected);
}

TEST(TestSeries, testResizeGrow) {
    Series s(FLOAT32_T, 10);
    s.write(1.0f);
    s.write(2.0f);
    ASSERT_EQ(s.size(), 2);

    s.resize(5);
    ASSERT_EQ(s.size(), 5);
    ASSERT_EQ(s.cap(), 10);
    ASSERT_EQ(s.at<float>(0), 1.0f);
    ASSERT_EQ(s.at<float>(1), 2.0f);
}

TEST(TestSeries, testResizeShrink) {
    Series s(INT32_T, 10);
    for (int i = 0; i < 5; i++) {
        s.write(i);
    }
    ASSERT_EQ(s.size(), 5);

    s.resize(2);
    ASSERT_EQ(s.size(), 2);
    ASSERT_EQ(s.cap(), 10);
    ASSERT_EQ(s.at<int32_t>(0), 0);
    ASSERT_EQ(s.at<int32_t>(1), 1);
}

TEST(TestSeries, testResizeNoOp) {
    Series s(UINT64_T, 10);
    for (int i = 0; i < 5; i++) {
        s.write(static_cast<uint64_t>(i));
    }

    s.resize(5);
    ASSERT_EQ(s.size(), 5);
    ASSERT_EQ(s.cap(), 10);
}

TEST(TestSeries, testResizeExceedsCapacity) {
    Series s(FLOAT64_T, 5);
    s.write(1.0);
    s.write(2.0);
    ASSERT_EQ(s.size(), 2);
    ASSERT_EQ(s.cap(), 5);

    s.resize(10);
    ASSERT_EQ(s.size(), 10);
    ASSERT_EQ(s.cap(), 10);
    ASSERT_EQ(s.at<double>(0), 1.0);
    ASSERT_EQ(s.at<double>(1), 2.0);
}

TEST(TestSeries, testResizeVariableType) {
    Series s(std::vector<std::string>{"hello", "world"});

    ASSERT_THROW(s.resize(1), std::runtime_error);
}

TEST(TestSeries, testResizeToZero) {
    Series s(INT16_T, 10);
    s.write(static_cast<int16_t>(1));
    s.write(static_cast<int16_t>(2));

    s.resize(0);
    ASSERT_EQ(s.size(), 0);
    ASSERT_TRUE(s.empty());
}

/// @brief it should correctly set a SampleValue at an index for numeric types
TEST(TestSeries, testSetSampleValueF32) {
    Series s(FLOAT32_T, 5);
    s.write(1.0f);
    s.write(2.0f);
    s.write(3.0f);
    s.write(4.0f);
    s.write(5.0f);

    // Test setting with different numeric types in SampleValue
    SampleValue val_double = 42.5f;
    s.set(0, val_double);
    ASSERT_EQ(s.at<float>(0), 42.5f);
}

/// @brief it should correctly set a SampleValue at a negative index
TEST(TestSeries, testSetSampleValueNegativeIndex) {
    Series s(INT32_T, 5);
    for (int i = 1; i <= 5; i++)
        s.write(i);

    SampleValue val = 999;
    s.set(-1, val);
    ASSERT_EQ(s.at<int32_t>(4), 999);

    s.set(-3, val);
    ASSERT_EQ(s.at<int32_t>(2), 999);
}

/// @brief it should correctly set a TimeStamp SampleValue
TEST(TestSeries, testSetSampleValueTimestamp) {
    Series s(TIMESTAMP_T, 3);
    s.write(TimeStamp(100));
    s.write(TimeStamp(200));
    s.write(TimeStamp(300));

    SampleValue val = TimeStamp(9999);
    s.set(1, val);
    ASSERT_EQ(s.at<TimeStamp>(1).nanoseconds(), 9999);
}

/// @brief it should throw an error when setting SampleValue on variable-size series
TEST(TestSeries, testSetSampleValueVariableError) {
    Series s(std::vector<std::string>{"hello", "world"});

    SampleValue val = std::string("test");
    ASSERT_THROW(s.set(0, val), std::runtime_error);
}

/// @brief it should throw an error when setting string SampleValue on non-string series
TEST(TestSeries, testSetSampleValueStringError) {
    Series s(INT32_T, 3);
    s.write(1);
    s.write(2);
    s.write(3);

    SampleValue val = std::string("test");
    ASSERT_THROW(s.set(0, val), std::runtime_error);
}

/// @brief it should throw an error when index is out of bounds
TEST(TestSeries, testSetSampleValueOutOfBounds) {
    Series s(UINT32_T, 3);
    s.write(1u);
    s.write(2u);
    s.write(3u);

    SampleValue val = 999u;
    ASSERT_THROW(s.set(5, val), std::runtime_error);
    ASSERT_THROW(s.set(-10, val), std::runtime_error);
}

/// @brief it should work with all numeric data types
TEST(TestSeries, testSetSampleValueAllNumericTypes) {
    // Test uint8_t
    Series s_uint8(UINT8_T, 3);
    for (uint8_t i = 1; i <= 3; i++)
        s_uint8.write(i);
    SampleValue val_uint8 = static_cast<uint8_t>(99);
    s_uint8.set(1, val_uint8);
    ASSERT_EQ(s_uint8.at<uint8_t>(1), 99);

    // Test int64_t
    Series s_int64(INT64_T, 3);
    for (int64_t i = 1; i <= 3; i++)
        s_int64.write(i);
    SampleValue val_int64 = static_cast<int64_t>(123456789);
    s_int64.set(2, val_int64);
    ASSERT_EQ(s_int64.at<int64_t>(2), 123456789);

    // Test float64
    Series s_float64(FLOAT64_T, 3);
    for (int i = 1; i <= 3; i++)
        s_float64.write(static_cast<double>(i));
    SampleValue val_float64 = 3.14159;
    s_float64.set(0, val_float64);
    ASSERT_DOUBLE_EQ(s_float64.at<double>(0), 3.14159);
}

/// @brief Tests Series + Series addition operator.
TEST(SeriesOperators, AdditionSameLength) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{4.0, 5.0, 6.0});
    auto result = a + b;
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result.data_type(), x::telem::FLOAT64_T);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 5.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 7.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 9.0);
}

/// @brief Tests Series - Series subtraction operator.
TEST(SeriesOperators, SubtractionSameLength) {
    auto a = Series(std::vector<double>{10.0, 20.0, 30.0});
    auto b = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = a - b;
    ASSERT_EQ(result.size(), 3);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 9.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 18.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 27.0);
}

/// @brief Tests Series * Series multiplication operator.
TEST(SeriesOperators, MultiplicationSameLength) {
    auto a = Series(std::vector<double>{2.0, 3.0, 4.0});
    auto b = Series(std::vector<double>{5.0, 6.0, 7.0});
    auto result = a * b;
    ASSERT_EQ(result.size(), 3);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 10.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 18.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 28.0);
}

/// @brief Tests Series / Series division operator.
TEST(SeriesOperators, DivisionSameLength) {
    auto a = Series(std::vector<double>{10.0, 20.0, 30.0});
    auto b = Series(std::vector<double>{2.0, 4.0, 5.0});
    auto result = a / b;
    ASSERT_EQ(result.size(), 3);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 5.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 5.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 6.0);
}

/// @brief Tests that length mismatch throws for binary operations.
TEST(SeriesOperators, LengthMismatchThrows) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{4.0, 5.0});
    ASSERT_THROW(a + b, std::runtime_error);
    ASSERT_THROW(a - b, std::runtime_error);
    ASSERT_THROW(a * b, std::runtime_error);
    ASSERT_THROW(a / b, std::runtime_error);
}

/// @brief Tests that type mismatch throws for binary operations.
TEST(SeriesOperators, TypeMismatchThrows) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<int32_t>{4, 5, 6});
    ASSERT_THROW(a + b, std::runtime_error);
    ASSERT_THROW(a - b, std::runtime_error);
    ASSERT_THROW(a * b, std::runtime_error);
    ASSERT_THROW(a / b, std::runtime_error);
}

/// @brief Tests Series + scalar operator.
TEST(SeriesOperators, ScalarAddition) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = a + 10.0;
    ASSERT_EQ(result.size(), 3);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 11.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 12.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 13.0);
    // Original should be unchanged
    ASSERT_DOUBLE_EQ(a.at<double>(0), 1.0);
}

/// @brief Tests scalar + Series operator (commutative).
TEST(SeriesOperators, ScalarOnLeftAddition) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = 10.0 + a;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 11.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 12.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 13.0);
}

/// @brief Tests Series - scalar operator.
TEST(SeriesOperators, ScalarSubtraction) {
    auto a = Series(std::vector<double>{10.0, 20.0, 30.0});
    auto result = a - 5.0;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 5.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 15.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 25.0);
}

/// @brief Tests scalar - Series operator (non-commutative).
TEST(SeriesOperators, ScalarOnLeftSubtraction) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = 10.0 - a;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 9.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 8.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 7.0);
}

/// @brief Tests Series * scalar operator.
TEST(SeriesOperators, ScalarMultiplication) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = a * 3.0;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 3.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 6.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 9.0);
}

/// @brief Tests scalar * Series operator (commutative).
TEST(SeriesOperators, ScalarOnLeftMultiplication) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto result = 3.0 * a;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 3.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 6.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 9.0);
}

/// @brief Tests Series / scalar operator.
TEST(SeriesOperators, ScalarDivision) {
    auto a = Series(std::vector<double>{10.0, 20.0, 30.0});
    auto result = a / 2.0;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 5.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 10.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 15.0);
}

/// @brief Tests scalar / Series operator (non-commutative).
TEST(SeriesOperators, ScalarOnLeftDivision) {
    auto a = Series(std::vector<double>{1.0, 2.0, 4.0});
    auto result = 8.0 / a;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 8.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 4.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 2.0);
}

/// @brief Tests division by zero throws.
TEST(SeriesOperators, DivisionByZeroThrows) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    ASSERT_THROW(a / 0.0, std::runtime_error);
}

/// @brief Tests > comparison operator.
TEST(SeriesOperators, GreaterThanReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 5.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 3.0});
    auto result = a > b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result.at<uint8_t>(0), 0); // 1.0 > 2.0 = false
    ASSERT_EQ(result.at<uint8_t>(1), 1); // 5.0 > 3.0 = true
    ASSERT_EQ(result.at<uint8_t>(2), 0); // 3.0 > 3.0 = false
}

/// @brief Tests < comparison operator.
TEST(SeriesOperators, LessThanReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 5.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 3.0});
    auto result = a < b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 1.0 < 2.0 = true
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 5.0 < 3.0 = false
    ASSERT_EQ(result.at<uint8_t>(2), 0); // 3.0 < 3.0 = false
}

/// @brief Tests >= comparison operator.
TEST(SeriesOperators, GreaterThanOrEqualReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 5.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 3.0});
    auto result = a >= b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 0); // 1.0 >= 2.0 = false
    ASSERT_EQ(result.at<uint8_t>(1), 1); // 5.0 >= 3.0 = true
    ASSERT_EQ(result.at<uint8_t>(2), 1); // 3.0 >= 3.0 = true
}

/// @brief Tests <= comparison operator.
TEST(SeriesOperators, LessThanOrEqualReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 5.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 3.0});
    auto result = a <= b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 1.0 <= 2.0 = true
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 5.0 <= 3.0 = false
    ASSERT_EQ(result.at<uint8_t>(2), 1); // 3.0 <= 3.0 = true
}

/// @brief Tests == comparison operator.
TEST(SeriesOperators, EqualityReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 3.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 4.0});
    auto result = a == b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 0); // 1.0 == 2.0 = false
    ASSERT_EQ(result.at<uint8_t>(1), 1); // 3.0 == 3.0 = true
    ASSERT_EQ(result.at<uint8_t>(2), 0); // 3.0 == 4.0 = false
}

/// @brief Tests != comparison operator.
TEST(SeriesOperators, InequalityReturnsUint8) {
    auto a = Series(std::vector<double>{1.0, 3.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 3.0, 4.0});
    auto result = a != b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 1.0 != 2.0 = true
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 3.0 != 3.0 = false
    ASSERT_EQ(result.at<uint8_t>(2), 1); // 3.0 != 4.0 = true
}

/// @brief Tests comparison operators throw on length mismatch.
TEST(SeriesOperators, ComparisonLengthMismatchThrows) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{4.0, 5.0});
    ASSERT_THROW((void) (a > b), std::runtime_error);
    ASSERT_THROW((void) (a < b), std::runtime_error);
    ASSERT_THROW((void) (a >= b), std::runtime_error);
    ASSERT_THROW((void) (a <= b), std::runtime_error);
    ASSERT_THROW((void) (a == b), std::runtime_error);
    ASSERT_THROW((void) (a != b), std::runtime_error);
}

/// @brief Tests operations with empty series.
TEST(SeriesOperators, EmptySeriesOperations) {
    auto a = Series(x::telem::FLOAT64_T, 0);
    auto b = Series(x::telem::FLOAT64_T, 0);
    auto result = a + b;
    ASSERT_EQ(result.size(), 0);
}

/// @brief Tests operations with single element series.
TEST(SeriesOperators, SingleElementOperations) {
    auto a = Series(std::vector<double>{5.0});
    auto b = Series(std::vector<double>{3.0});
    auto result = a + b;
    ASSERT_EQ(result.size(), 1);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 8.0);
}

/// @brief Tests operations with int32_t type.
TEST(SeriesOperators, Int32Operations) {
    auto a = Series(std::vector<int32_t>{1, 2, 3});
    auto b = Series(std::vector<int32_t>{4, 5, 6});
    auto result = a + b;
    ASSERT_EQ(result.data_type(), x::telem::INT32_T);
    ASSERT_EQ(result.at<int32_t>(0), 5);
    ASSERT_EQ(result.at<int32_t>(1), 7);
    ASSERT_EQ(result.at<int32_t>(2), 9);
}

/// @brief Tests operations with float32 type.
TEST(SeriesOperators, Float32Operations) {
    auto a = Series(std::vector<float>{1.0f, 2.0f, 3.0f});
    auto b = Series(std::vector<float>{4.0f, 5.0f, 6.0f});
    auto result = a + b;
    ASSERT_EQ(result.data_type(), x::telem::FLOAT32_T);
    ASSERT_FLOAT_EQ(result.at<float>(0), 5.0f);
    ASSERT_FLOAT_EQ(result.at<float>(1), 7.0f);
    ASSERT_FLOAT_EQ(result.at<float>(2), 9.0f);
}

/// @brief Tests operations with uint8_t type.
TEST(SeriesOperators, Uint8Operations) {
    auto a = Series(std::vector<uint8_t>{10, 20, 30});
    auto b = Series(std::vector<uint8_t>{5, 10, 15});
    auto result = a + b;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 15);
    ASSERT_EQ(result.at<uint8_t>(1), 30);
    ASSERT_EQ(result.at<uint8_t>(2), 45);
}

/// @brief Tests operations with int64_t type.
TEST(SeriesOperators, Int64Operations) {
    auto a = Series(std::vector<int64_t>{100, 200, 300});
    auto b = Series(std::vector<int64_t>{10, 20, 30});
    auto result = a - b;
    ASSERT_EQ(result.data_type(), x::telem::INT64_T);
    ASSERT_EQ(result.at<int64_t>(0), 90);
    ASSERT_EQ(result.at<int64_t>(1), 180);
    ASSERT_EQ(result.at<int64_t>(2), 270);
}

/// @brief Tests chained operations.
TEST(SeriesOperators, ChainedOperations) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{2.0, 2.0, 2.0});
    // (a + b) * 3 - 1
    auto result = (a + b) * 3.0 - 1.0;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 8.0); // (1+2)*3-1 = 8
    ASSERT_DOUBLE_EQ(result.at<double>(1), 11.0); // (2+2)*3-1 = 11
    ASSERT_DOUBLE_EQ(result.at<double>(2), 14.0); // (3+2)*3-1 = 14
}

/// @brief Tests that original series is not modified by operators.
TEST(SeriesOperators, OriginalUnmodified) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{4.0, 5.0, 6.0});
    auto result = a + b;

    // Original series should be unchanged
    ASSERT_DOUBLE_EQ(a.at<double>(0), 1.0);
    ASSERT_DOUBLE_EQ(a.at<double>(1), 2.0);
    ASSERT_DOUBLE_EQ(a.at<double>(2), 3.0);
    ASSERT_DOUBLE_EQ(b.at<double>(0), 4.0);
    ASSERT_DOUBLE_EQ(b.at<double>(1), 5.0);
    ASSERT_DOUBLE_EQ(b.at<double>(2), 6.0);
}

/// @brief Tests unary negation operator with double.
TEST(SeriesOperators, UnaryNegateFloat64) {
    auto a = Series(std::vector<double>{1.0, -2.0, 3.0, 0.0});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::FLOAT64_T);
    ASSERT_EQ(result.size(), 4);
    ASSERT_DOUBLE_EQ(result.at<double>(0), -1.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 2.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), -3.0);
    ASSERT_DOUBLE_EQ(result.at<double>(3), 0.0);
    // Original should be unchanged
    ASSERT_DOUBLE_EQ(a.at<double>(0), 1.0);
}

/// @brief Tests unary negation operator with float.
TEST(SeriesOperators, UnaryNegateFloat32) {
    auto a = Series(std::vector<float>{1.5f, -2.5f, 3.5f});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::FLOAT32_T);
    ASSERT_FLOAT_EQ(result.at<float>(0), -1.5f);
    ASSERT_FLOAT_EQ(result.at<float>(1), 2.5f);
    ASSERT_FLOAT_EQ(result.at<float>(2), -3.5f);
}

/// @brief Tests unary negation operator with int32.
TEST(SeriesOperators, UnaryNegateInt32) {
    auto a = Series(std::vector<int32_t>{1, -2, 3, 0, -100});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::INT32_T);
    ASSERT_EQ(result.at<int32_t>(0), -1);
    ASSERT_EQ(result.at<int32_t>(1), 2);
    ASSERT_EQ(result.at<int32_t>(2), -3);
    ASSERT_EQ(result.at<int32_t>(3), 0);
    ASSERT_EQ(result.at<int32_t>(4), 100);
}

/// @brief Tests unary negation operator with int64.
TEST(SeriesOperators, UnaryNegateInt64) {
    auto a = Series(std::vector<int64_t>{1000000000LL, -2000000000LL});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::INT64_T);
    ASSERT_EQ(result.at<int64_t>(0), -1000000000LL);
    ASSERT_EQ(result.at<int64_t>(1), 2000000000LL);
}

/// @brief Tests unary negation operator with int16.
TEST(SeriesOperators, UnaryNegateInt16) {
    auto a = Series(std::vector<int16_t>{100, -200, 300});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::INT16_T);
    ASSERT_EQ(result.at<int16_t>(0), -100);
    ASSERT_EQ(result.at<int16_t>(1), 200);
    ASSERT_EQ(result.at<int16_t>(2), -300);
}

/// @brief Tests unary negation operator with int8.
TEST(SeriesOperators, UnaryNegateInt8) {
    auto a = Series(std::vector<int8_t>{10, -20, 30});
    auto result = -a;
    ASSERT_EQ(result.data_type(), x::telem::INT8_T);
    ASSERT_EQ(result.at<int8_t>(0), -10);
    ASSERT_EQ(result.at<int8_t>(1), 20);
    ASSERT_EQ(result.at<int8_t>(2), -30);
}

/// @brief Tests unary negation with empty series.
TEST(SeriesOperators, UnaryNegateEmpty) {
    auto a = Series(x::telem::FLOAT64_T, 0);
    auto result = -a;
    ASSERT_EQ(result.size(), 0);
    ASSERT_EQ(result.data_type(), x::telem::FLOAT64_T);
}

/// @brief Tests unary negation with single element.
TEST(SeriesOperators, UnaryNegateSingleElement) {
    auto a = Series(std::vector<double>{5.0});
    auto result = -a;
    ASSERT_EQ(result.size(), 1);
    ASSERT_DOUBLE_EQ(result.at<double>(0), -5.0);
}

/// @brief Tests bitwise NOT operator with uint8.
TEST(SeriesOperators, BitwiseNotUint8) {
    auto a = Series(std::vector<uint8_t>{0x00, 0xFF, 0x0F, 0xF0, 0xAA});
    auto result = ~a;
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.size(), 5);
    ASSERT_EQ(result.at<uint8_t>(0), 0xFF);
    ASSERT_EQ(result.at<uint8_t>(1), 0x00);
    ASSERT_EQ(result.at<uint8_t>(2), 0xF0);
    ASSERT_EQ(result.at<uint8_t>(3), 0x0F);
    ASSERT_EQ(result.at<uint8_t>(4), 0x55);
}

/// @brief Tests bitwise NOT operator with uint16.
TEST(SeriesOperators, BitwiseNotUint16) {
    auto a = Series(std::vector<uint16_t>{0x0000, 0xFFFF, 0x00FF});
    auto result = ~a;
    ASSERT_EQ(result.data_type(), x::telem::UINT16_T);
    ASSERT_EQ(result.at<uint16_t>(0), 0xFFFF);
    ASSERT_EQ(result.at<uint16_t>(1), 0x0000);
    ASSERT_EQ(result.at<uint16_t>(2), 0xFF00);
}

/// @brief Tests bitwise NOT operator with uint32.
TEST(SeriesOperators, BitwiseNotUint32) {
    auto a = Series(std::vector<uint32_t>{0x00000000, 0xFFFFFFFF, 0x0000FFFF});
    auto result = ~a;
    ASSERT_EQ(result.data_type(), x::telem::UINT32_T);
    ASSERT_EQ(result.at<uint32_t>(0), 0xFFFFFFFF);
    ASSERT_EQ(result.at<uint32_t>(1), 0x00000000);
    ASSERT_EQ(result.at<uint32_t>(2), 0xFFFF0000);
}

/// @brief Tests bitwise NOT operator with uint64.
TEST(SeriesOperators, BitwiseNotUint64) {
    auto a = Series(
        std::vector<uint64_t>{0x0000000000000000ULL, 0xFFFFFFFFFFFFFFFFULL}
    );
    auto result = ~a;
    ASSERT_EQ(result.data_type(), x::telem::UINT64_T);
    ASSERT_EQ(result.at<uint64_t>(0), 0xFFFFFFFFFFFFFFFFULL);
    ASSERT_EQ(result.at<uint64_t>(1), 0x0000000000000000ULL);
}

/// @brief Tests bitwise NOT operator with int32 (two's complement).
TEST(SeriesOperators, BitwiseNotInt32) {
    auto a = Series(std::vector<int32_t>{0, -1, 1});
    auto result = ~a;
    ASSERT_EQ(result.data_type(), x::telem::INT32_T);
    ASSERT_EQ(result.at<int32_t>(0), -1); // ~0 = -1 in two's complement
    ASSERT_EQ(result.at<int32_t>(1), 0); // ~(-1) = 0
    ASSERT_EQ(result.at<int32_t>(2), -2); // ~1 = -2 in two's complement
}

/// @brief Tests bitwise NOT throws for floating-point types.
TEST(SeriesOperators, BitwiseNotFloatThrows) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    ASSERT_THROW(~a, std::runtime_error);
}

/// @brief Tests bitwise NOT throws for float32 types.
TEST(SeriesOperators, BitwiseNotFloat32Throws) {
    auto a = Series(std::vector<float>{1.0f, 2.0f, 3.0f});
    ASSERT_THROW(~a, std::runtime_error);
}

/// @brief Tests double negation returns original values.
TEST(SeriesOperators, DoubleNegation) {
    auto a = Series(std::vector<double>{1.0, -2.0, 3.0});
    auto result = -(-a);
    ASSERT_DOUBLE_EQ(result.at<double>(0), 1.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), -2.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 3.0);
}

/// @brief Tests double bitwise NOT returns original values.
TEST(SeriesOperators, DoubleBitwiseNot) {
    auto a = Series(std::vector<uint8_t>{0x00, 0xFF, 0xAA});
    auto result = ~~a;
    ASSERT_EQ(result.at<uint8_t>(0), 0x00);
    ASSERT_EQ(result.at<uint8_t>(1), 0xFF);
    ASSERT_EQ(result.at<uint8_t>(2), 0xAA);
}

/// @brief Tests logical NOT with uint8 (0 -> 1, non-zero -> 0).
TEST(SeriesOperators, LogicalNotUint8) {
    auto a = Series(std::vector<uint8_t>{0, 1, 255, 0, 42});
    auto result = a.logical_not();
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 0 -> 1
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 1 -> 0
    ASSERT_EQ(result.at<uint8_t>(2), 0); // 255 -> 0
    ASSERT_EQ(result.at<uint8_t>(3), 1); // 0 -> 1
    ASSERT_EQ(result.at<uint8_t>(4), 0); // 42 -> 0
}

/// @brief Tests logical NOT with int32.
TEST(SeriesOperators, LogicalNotInt32) {
    auto a = Series(std::vector<int32_t>{0, 1, -1, 100, 0});
    auto result = a.logical_not();
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 0 -> 1
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 1 -> 0
    ASSERT_EQ(result.at<uint8_t>(2), 0); // -1 -> 0
    ASSERT_EQ(result.at<uint8_t>(3), 0); // 100 -> 0
    ASSERT_EQ(result.at<uint8_t>(4), 1); // 0 -> 1
}

/// @brief Tests logical NOT with float64.
TEST(SeriesOperators, LogicalNotFloat64) {
    auto a = Series(std::vector<double>{0.0, 1.0, -1.0, 0.5, 0.0});
    auto result = a.logical_not();
    ASSERT_EQ(result.data_type(), x::telem::UINT8_T);
    ASSERT_EQ(result.at<uint8_t>(0), 1); // 0.0 -> 1
    ASSERT_EQ(result.at<uint8_t>(1), 0); // 1.0 -> 0
    ASSERT_EQ(result.at<uint8_t>(2), 0); // -1.0 -> 0
    ASSERT_EQ(result.at<uint8_t>(3), 0); // 0.5 -> 0
    ASSERT_EQ(result.at<uint8_t>(4), 1); // 0.0 -> 1
}

/// @brief Tests double logical NOT preserves truthiness.
TEST(SeriesOperators, DoubleLogicalNot) {
    auto a = Series(std::vector<uint8_t>{0, 1, 0, 255});
    auto result = a.logical_not().logical_not();
    // Double NOT should give 0 for false, 1 for true (normalized)
    ASSERT_EQ(result.at<uint8_t>(0), 0); // 0 -> 1 -> 0
    ASSERT_EQ(result.at<uint8_t>(1), 1); // 1 -> 0 -> 1
    ASSERT_EQ(result.at<uint8_t>(2), 0); // 0 -> 1 -> 0
    ASSERT_EQ(result.at<uint8_t>(3), 1); // 255 -> 0 -> 1
}

/// @brief Tests negation can be chained with other operations.
TEST(SeriesOperators, NegationChainedWithOperations) {
    auto a = Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = Series(std::vector<double>{4.0, 5.0, 6.0});
    // -a + b should give {3.0, 3.0, 3.0}
    auto result = (-a) + b;
    ASSERT_DOUBLE_EQ(result.at<double>(0), 3.0);
    ASSERT_DOUBLE_EQ(result.at<double>(1), 3.0);
    ASSERT_DOUBLE_EQ(result.at<double>(2), 3.0);
}
}
