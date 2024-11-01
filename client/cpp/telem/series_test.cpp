// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>
#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

/*

///// @brief create basic int series.
TEST(TestSeries, testConstruction) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const synnax::Series s{vals};
    ASSERT_EQ(s.data_type, synnax::SY_UINT8);
    const auto v = s.values<std::uint8_t>();
    ASSERT_EQ(v.size(), vals.size());
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

//// @brief it should correctly initialize and parse a string series.
TEST(TestSeries, testStringVectorConstruction) {
    const std::vector<std::string> vals = {"hello", "world"};
    const Series s{vals};
    ASSERT_EQ(s.data_type, synnax::STRING);
    ASSERT_EQ(s.size, 2);
    ASSERT_EQ(s.byteSize(), 12);
    const auto v = s.strings();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
}

TEST(TestSeries, testStringConstruction) {
    const std::string val = "hello";
    const Series s{val};
    ASSERT_EQ(s.data_type, synnax::STRING);
    ASSERT_EQ(s.size, 1);
    ASSERT_EQ(s.byteSize(), 6);
    const auto v = s.strings();
    ASSERT_EQ(v[0], val);
}

TEST(TestSeries, testJSONConstruction) {
    const std::string raw = R"({ "key": "abc" })";
    const Series s(raw, JSON);
    ASSERT_EQ(s.data_type, synnax::JSON);
    ASSERT_EQ(s.size, 1);
    ASSERT_EQ(s.byteSize(), 17);
    const auto v = s.strings();
    ASSERT_EQ(v[0], raw);
}

//// @brief it should correctly serialize and deserialize the series from protoubuf.
TEST(TestSeries, testProto) {
    const std::vector<uint16_t> vals = {1, 2, 3, 4, 5};
    const Series s{vals};
    const auto s2 = new telem::PBSeries();
    s.to_proto(s2);
    const Series s3{*s2};
    const auto v = s3.values<std::uint16_t>();
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i]);
    delete s2;
}

TEST(TestSeries, testProtoVariable) {
    const std::vector<std::string> vals = {"hello", "world22"};
    const Series s{vals};
    const auto s2 = new telem::PBSeries();
    s.to_proto(s2);
    const Series s3{*s2};
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

/// @brief it should correclty return the value at a particular index for a variable
/// length data type.
TEST(TestSeries, testAtVar) {
    const std::vector<std::string> vals = {"hello", "world"};
    const Series s{vals};
    std::string value;
    s.at(0, value);
    ASSERT_EQ(value, "hello");
    s.at(1, value);
    ASSERT_EQ(value, "world");
}

TEST(TestSeries, testAllocation) {
    const Series s{synnax::UINT32, 5};
    ASSERT_EQ(s.data_type, synnax::UINT32);
    ASSERT_EQ(s.size, 0);
    ASSERT_EQ(s.cap, 5);
    ASSERT_EQ(s.byteSize(), 0);
    ASSERT_EQ(s.byteCap(), 20);
}

TEST(TestSeries, testWrite) {
    Series s{synnax::UINT32, 5};
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
    ASSERT_EQ(s.size, 5);
    ASSERT_EQ(s.at<std::uint32_t>(0), 1);
    ASSERT_EQ(s.at<std::uint32_t>(1), 2);
    ASSERT_EQ(s.at<std::uint32_t>(2), 3);
    ASSERT_EQ(s.at<std::uint32_t>(3), 4);
    ASSERT_EQ(s.at<std::uint32_t>(4), 5);
}

TEST(TestSeries, testWriteVector) {
    Series s{synnax::FLOAT32, 5};
    const std::vector<float> values = {1.0, 2.0, 3.0, 4.0, 5.0};
    ASSERT_EQ(s.write(values), 5);
    ASSERT_EQ(s.write(values), 0);
    ASSERT_EQ(s.size, 5);
    const auto v = s.values<float>();
    ASSERT_EQ(s.at<float>(1), 2.0);
    for (size_t i = 0; i < values.size(); i++)
        ASSERT_EQ(v[i], values[i]);
}


TEST(TestSeries, testOstreamOperatorForAllTypes) {
    // Refactored tests to match the new format "Series(type: TYPE, size: SIZE, cap: CAP, data: [DATA ])"

    Series s_uint32{synnax::UINT32, 3};
    for (std::uint32_t i = 1; i <= 3; ++i) {
        s_uint32.write(i);
    }
    std::ostringstream oss_uint32;
    oss_uint32 << s_uint32;
    ASSERT_EQ(oss_uint32.str(),
              "Series(type: uint32, size: 3, cap: 3, data: [1 2 3 ])");

    Series s_float32{synnax::FLOAT32, 3};
    for (float i = 1.5f; i <= 3.5f; i += 1.0f) {
        s_float32.write(i);
    }
    std::ostringstream oss_float32;
    oss_float32 << s_float32;
    ASSERT_EQ(oss_float32.str(),
              "Series(type: float32, size: 3, cap: 3, data: [1.5 2.5 3.5 ])");

    Series s_int32{synnax::INT32, 3};
    for (int i = -1; i >= -3; --i) {
        s_int32.write(i);
    }
    std::ostringstream oss_int32;
    oss_int32 << s_int32;
    ASSERT_EQ(oss_int32.str(),
              "Series(type: int32, size: 3, cap: 3, data: [-1 -2 -3 ])");

    Series s_uint64{synnax::UINT64, 3};
    for (std::uint64_t i = 1; i <= 3; ++i) {
        s_uint64.write(i);
    }
    std::ostringstream oss_uint64;
    oss_uint64 << s_uint64;
    ASSERT_EQ(oss_uint64.str(),
              "Series(type: uint64, size: 3, cap: 3, data: [1 2 3 ])");

    Series s_int64{synnax::INT64, 3};
    for (std::int64_t i = -1; i >= -3; --i) {
        s_int64.write(i);
    }
    std::ostringstream oss_int64;
    oss_int64 << s_int64;
    ASSERT_EQ(oss_int64.str(),
              "Series(type: int64, size: 3, cap: 3, data: [-1 -2 -3 ])");

    Series s_float64{synnax::FLOAT64, 3};
    for (double i = 1.5; i <= 3.5; i += 1.0) {
        s_float64.write(i);
    }
    std::ostringstream oss_float64;
    oss_float64 << s_float64;
    ASSERT_EQ(oss_float64.str(),
              "Series(type: float64, size: 3, cap: 3, data: [1.5 2.5 3.5 ])");

    Series s_uint8{synnax::SY_UINT8, 3};
    for (std::uint8_t i = 1; i <= 3; ++i) {
        s_uint8.write(i);
    }
    std::ostringstream oss_uint8;
    oss_uint8 << s_uint8;
    ASSERT_EQ(oss_uint8.str(), "Series(type: uint8, size: 3, cap: 3, data: [1 2 3 ])");
}
*/

///// @brief test_transform_
TEST(TestSeries, test_transform_inplace) {
    std::vector<double> vals = {1.0, 2.0, 3.0, 4.0, 5.0};
    synnax::Series s{vals};
    ASSERT_EQ(s.data_type, synnax::FLOAT64);

    s.transform_inplace<double>([](double x) { return x * 2; });
    const auto v = s.values<double>();
    ASSERT_EQ(v.size(), vals.size());
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v[i], vals[i] * 2);

    vals = std::vector<double>({2.0, 4.0, 6.0, 8.0, 10.0});

    // now try a lienar transformation
    s.transform_inplace<double>([](double x) { return (3*x + 1); });
    const auto v2 = s.values<double>();
    ASSERT_EQ(v2.size(), vals.size());
    for (size_t i = 0; i < vals.size(); i++)
        ASSERT_EQ(v2[i], 3*vals[i] + 1);
}