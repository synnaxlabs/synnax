// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/telem/telem.h"

#include "gtest/gtest.h"

using namespace telem;

////////////////////////////////////////////////////////////
// TimeStamp Tests
////////////////////////////////////////////////////////////

/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testConstructor) {
    const auto ts = TimeStamp(5);
    ASSERT_EQ(ts.value, 5);
}

TEST(TimeStampTests, testAddition) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.value, 10);
}

TEST(TimeStampTests, testSubtraction) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.value, 0);
}

TEST(TimeStampTests, testMultiplication) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.value, 25);
}

TEST(TimeStampTests, testDivision) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.value, 1);
}

TEST(TimeStampTests, testEquality) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts == ts2);
}

TEST(TimeStampTests, testInequality) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts != ts2);
}

TEST(TimeStampTests, testLessThan) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts < ts2);
}

TEST(TimeStampTests, testLessThanEqual) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts <= ts2);
}

TEST(TimeStampTests, testGreaterThan) {
    const auto ts = TimeStamp(6);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts > ts2);
}

TEST(TimeStampTests, testGreaterThanEqual) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts >= ts2);
}

TEST(TimeStampTests, testModulo) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(2);
    const auto ts3 = ts % ts2;
    ASSERT_EQ(ts3.value, 1);
}

TEST(TimeStampTests, testAdditionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts += ts2;
    ASSERT_EQ(ts.value, 10);
}

TEST(TimeStampTests, testSubtractionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts -= ts2;
    ASSERT_EQ(ts.value, 0);
}

TEST(TimeStampTests, testMultiplicationAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts *= ts2;
    ASSERT_EQ(ts.value, 25);
}

TEST(TimeStampTests, testDivisionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts /= ts2;
    ASSERT_EQ(ts.value, 1);
}

TEST(TimeStampTests, testModuloAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(2);
    ts %= ts2;
    ASSERT_EQ(ts.value, 1);
}

////////////////////////////////////////////////////////////
// TimeSpan Tests
////////////////////////////////////////////////////////////

TEST(TimeSpanTests, testConstructor) {
    const auto ts = TimeSpan(5);
    ASSERT_EQ(ts.value, 5);
}

TEST(TimeSpanTests, testAddition) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.value, 10);
}

TEST(TimeSpanTests, testSubtraction) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.value, 0);
}

TEST(TimeSpanTests, testMultiplication) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.value, 25);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = ts4 * 5;
    ASSERT_EQ(ts5.value, 25);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = 5 * ts6;
    ASSERT_EQ(ts7.value, 25);

    const auto ts8 = TimeSpan(5);
    const auto ts9 = ts8 * 5.0;
    ASSERT_EQ(ts9.value, 25);

    const auto ts10 = TimeSpan(5);
    const auto ts11 = ts10 * 5.0f;
    ASSERT_EQ(ts11.value, 25);
}

TEST(TimeSpanTests, testDivision) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.value, 1);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = ts4 / 5;
    ASSERT_EQ(ts5.value, 1);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = 5 / ts6;
    ASSERT_EQ(ts7.value, 1);
}

TEST(TimeSpanTests, testEquality) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts == ts2);
}

TEST(TimeSpanTests, testInequality) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts != ts2);
}

TEST(TimeSpanTests, testLessThan) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts < ts2);
}

TEST(TimeSpanTests, testLessThanEqual) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts <= ts2);
}

TEST(TimeSpanTests, testGreaterThan) {
    const auto ts = TimeSpan(6);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts > ts2);
}

TEST(TimeSpanTests, testGreaterThanEqual) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts >= ts2);
}

TEST(TimeSpanTests, testModulo) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(2);
    const auto ts3 = ts % ts2;
    ASSERT_EQ(ts3.value, 1);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = 2 % ts4;
    ASSERT_EQ(ts5.value, 2);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = ts6 % 2;
    ASSERT_EQ(ts7.value, 1);
}

TEST(TimeSpanTests, testTruncate) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(2);
    const auto ts3 = ts.truncate(ts2);
    ASSERT_EQ(ts3.value, 4);
}

TEST(TimeSpanTests, testDelta) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(2);
    const auto ts3 = ts.delta(ts2);
    ASSERT_EQ(ts3.value, 3);
}

////////////////////////////////////////////////////////////
// TimeRange Tests
////////////////////////////////////////////////////////////

TEST(TimeRangeTests, testContains) {
    const auto tr = TimeRange(5, 10);
    const auto ts = TimeStamp(7);
    ASSERT_TRUE(tr.contains(ts));
}

TEST(TimeRangeTests, testContainsRange) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(6, 9);
    ASSERT_TRUE(tr.contains(tr2));
}

TEST(TimeRangeTests, testEquality) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(5, 10);
    ASSERT_TRUE(tr == tr2);
}

////////////////////////////////////////////////////////////
// Rate Tests
////////////////////////////////////////////////////////////

TEST(RateTests, testPeriod) {
    const auto r = Rate(1);
    ASSERT_EQ(r.period().value, telem::SECOND.value);
    const auto r2 = Rate(2);
    ASSERT_EQ(r2.period().value, telem::SECOND.value / 2);
}

TEST(RateTests, testContructor) {
    const auto r = Rate(5);
    ASSERT_EQ(r.value, 5);
}

TEST(RateTests, testAddition) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r + r2;
    ASSERT_EQ(r3.value, 10);
}

TEST(RateTests, testSubtraction) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r - r2;
    ASSERT_EQ(r3.value, 0);
}

TEST(RateTests, testMultiplication) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r * r2;
    ASSERT_EQ(r3.value, 25);

    const auto r4 = Rate(5);
    const auto r5 = r4 * 5;
    ASSERT_EQ(r5.value, 25);

    const auto r6 = Rate(5);
    const auto r7 = 5 * r6;
    ASSERT_EQ(r7.value, 25);

    const auto r8 = Rate(5);
    const auto r9 = r8 * 5.0;
    ASSERT_EQ(r9.value, 25);

    const auto r10 = Rate(5);
    const auto r11 = r10 * 5.0f;
    ASSERT_EQ(r11.value, 25);

    const auto r12 = Rate(5);
    const auto r13 = r12 * 5.0l;
    ASSERT_EQ(r13.value, 25);
}

TEST(RateTests, testDivision) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r / r2;
    ASSERT_EQ(r3.value, 1);

    const auto r4 = Rate(5);
    const auto r5 = r4 / 5;
    ASSERT_EQ(r5.value, 1);
}

TEST(RateTests, testEquality) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r == r2);
}

TEST(RateTests, testInequality) {
    const auto r = Rate(5);
    const auto r2 = Rate(6);
    ASSERT_TRUE(r != r2);
}

TEST(RateTests, testLessThan) {
    const auto r = Rate(5);
    const auto r2 = Rate(6);
    ASSERT_TRUE(r < r2);
}

TEST(RateTests, testLessThanEqual) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r <= r2);
}

TEST(RateTests, testGreaterThan) {
    const auto r = Rate(6);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r > r2);
}

TEST(RateTests, testGreaterThanEqual) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r >= r2);
}

////////////////////////////////////////////////////////////
// DataType Tests
////////////////////////////////////////////////////////////

class DataTypeTests : public ::testing::Test {
};

struct TypeTestCase {
    std::string expected;
    std::function<DataType()> inferFn;
};

class DataTypeInferTest : public testing::TestWithParam<TypeTestCase> {
};

TEST_P(DataTypeInferTest, testInferBuiltInTypes) {
    const auto &[expected, infer_fn] = GetParam();
    const auto dt = infer_fn();
    ASSERT_EQ(dt.value, expected);
}

INSTANTIATE_TEST_SUITE_P(
    DataTypes,
    DataTypeInferTest,
    testing::Values(
        TypeTestCase{"int8", []() { return DataType::infer<int8_t>(); }},
        TypeTestCase{"uint8", []() { return DataType::infer<uint8_t>(); }},
        TypeTestCase{"int16", []() { return DataType::infer<int16_t>(); }},
        TypeTestCase{"uint16", []() { return DataType::infer<uint16_t>(); }},
        TypeTestCase{"int32", []() { return DataType::infer<int32_t>(); }},
        TypeTestCase{"uint32", []() { return DataType::infer<uint32_t>(); }},
        TypeTestCase{"int64", []() { return DataType::infer<int64_t>(); }},
        TypeTestCase{"uint64", []() { return DataType::infer<uint64_t>(); }},
        TypeTestCase{"float32", []() { return DataType::infer<float>(); }},
        TypeTestCase{"float64", []() { return DataType::infer<double>(); }},
        TypeTestCase{"timestamp", []() { return DataType::infer<TimeStamp>(); }},
        TypeTestCase{"string", []() { return DataType::infer<std::string>(); }}
    )
);

TEST(DataTypeTests, testInferOveride) {
    const auto dt = DataType::infer<int8_t>(INT16_T);
    ASSERT_EQ(dt, INT16_T);
}

TEST(DataTypeTests, testName) {
    const auto dt = DataType("float32");
    ASSERT_EQ(dt.name(), "float32");
}

TEST(DataTypeTests, testDensity) {
    ASSERT_EQ(DataType("").density(), 0);
    ASSERT_EQ(DataType("float64").density(), 8);
    ASSERT_EQ(DataType("float32").density(), 4);
    ASSERT_EQ(DataType("int8").density(), 1);
    ASSERT_EQ(DataType("int16").density(), 2);
    ASSERT_EQ(DataType("int32").density(), 4);
    ASSERT_EQ(DataType("int64").density(), 8);
    ASSERT_EQ(DataType("uint8").density(), 1);
    ASSERT_EQ(DataType("uint16").density(), 2);
    ASSERT_EQ(DataType("uint32").density(), 4);
    ASSERT_EQ(DataType("uint64").density(), 8);
    ASSERT_EQ(DataType("uint128").density(), 16);
    ASSERT_EQ(DataType("timestamp").density(), 8);
    ASSERT_EQ(DataType("uuid").density(), 16);
    ASSERT_EQ(DataType("string").density(), 0);
    ASSERT_EQ(DataType("json").density(), 0);
}

TEST(DataTypeTests, testIsVariable) {
    ASSERT_TRUE(DataType("string").is_variable());
    ASSERT_TRUE(DataType("json").is_variable());
    ASSERT_FALSE(DataType("float32").is_variable());
    ASSERT_FALSE(DataType("int64").is_variable());
}

TEST(DataTypeTests, testMatches) {
    // Test empty data type matches anything
    const auto empty = DataType("");
    ASSERT_TRUE(empty.matches(DataType("float32")));
    ASSERT_TRUE(empty.matches("float32"));

    // Test exact matches
    const auto dt = DataType("float32");
    ASSERT_TRUE(dt.matches(DataType("float32")));
    ASSERT_TRUE(dt.matches("float32"));
    ASSERT_FALSE(dt.matches(DataType("float64")));
    ASSERT_FALSE(dt.matches("float64"));

    // Test vector matches
    std::vector<DataType> types = {DataType("float32"), DataType("float64")};
    ASSERT_TRUE(dt.matches(types));

    std::vector<DataType> non_matching = {DataType("int32"), DataType("int64")};
    ASSERT_FALSE(dt.matches(non_matching));
}

TEST(DataTypeTests, testEquality) {
    const auto dt1 = DataType("float32");
    const auto dt2 = DataType("float32");
    const auto dt3 = DataType("float64");

    ASSERT_TRUE(dt1 == dt2);
    ASSERT_FALSE(dt1 == dt3);
}

TEST(DataTypeTests, testInequality) {
    const auto dt1 = DataType("float32");
    const auto dt2 = DataType("float32");
    const auto dt3 = DataType("float64");

    ASSERT_FALSE(dt1 != dt2);
    ASSERT_TRUE(dt1 != dt3);
}

TEST(DataTypeTests, testStreamOperator) {
    const auto dt = DataType("float32");
    std::stringstream ss;
    ss << dt;
    ASSERT_EQ(ss.str(), "float32");
}