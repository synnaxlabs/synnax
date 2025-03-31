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

namespace telem {
/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testConstructor) {
    const auto ts = TimeStamp(5);
    ASSERT_EQ(ts.nanoseconds(), 5);
}

TEST(TimeStampTests, testAddition) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.nanoseconds(), 10);
}

TEST(TimeStampTests, testSubtraction) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.nanoseconds(), 0);
}

TEST(TimeStampTests, testMultiplication) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.nanoseconds(), 25);
}

TEST(TimeStampTests, testDivision) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.nanoseconds(), 1);
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
    ASSERT_EQ(ts3.nanoseconds(), 1);
}

TEST(TimeStampTests, testAdditionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts += ts2;
    ASSERT_EQ(ts.nanoseconds(), 10);
}

TEST(TimeStampTests, testSubtractionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts -= ts2;
    ASSERT_EQ(ts.nanoseconds(), 0);
}

TEST(TimeStampTests, testMultiplicationAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts *= ts2;
    ASSERT_EQ(ts.nanoseconds(), 25);
}

TEST(TimeStampTests, testDivisionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts /= ts2;
    ASSERT_EQ(ts.nanoseconds(), 1);
}

TEST(TimeStampTests, testModuloAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(2);
    ts %= ts2;
    ASSERT_EQ(ts.nanoseconds(), 1);
}

////////////////////////////////////////////////////////////
// TimeSpan Tests
////////////////////////////////////////////////////////////

TEST(TimeSpanTests, testConstructor) {
    const auto ts = TimeSpan(5);
    ASSERT_EQ(ts.nanoseconds(), 5);
}

TEST(TimeSpanTests, testAddition) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.nanoseconds(), 10);
}

TEST(TimeSpanTests, testSubtraction) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.nanoseconds(), 0);
}

TEST(TimeSpanTests, testMultiplication) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.nanoseconds(), 25);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = ts4 * 5;
    ASSERT_EQ(ts5.nanoseconds(), 25);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = 5 * ts6;
    ASSERT_EQ(ts7.nanoseconds(), 25);

    const auto ts8 = TimeSpan(5);
    const auto ts9 = ts8 * 5.0;
    ASSERT_EQ(ts9.nanoseconds(), 25);

    const auto ts10 = TimeSpan(5);
    const auto ts11 = ts10 * 5.0f;
    ASSERT_EQ(ts11.nanoseconds(), 25);
}

TEST(TimeSpanTests, testDivision) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.nanoseconds(), 1);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = ts4 / 5;
    ASSERT_EQ(ts5.nanoseconds(), 1);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = 5 / ts6;
    ASSERT_EQ(ts7.nanoseconds(), 1);
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
    ASSERT_EQ(ts3.nanoseconds(), 1);

    const auto ts4 = TimeSpan(5);
    const auto ts5 = 2 % ts4;
    ASSERT_EQ(ts5.nanoseconds(), 2);

    const auto ts6 = TimeSpan(5);
    const auto ts7 = ts6 % 2;
    ASSERT_EQ(ts7.nanoseconds(), 1);
}

TEST(TimeSpanTests, testTruncate) {
    const auto ts = telem::SECOND * 5 + telem::MICROSECOND * 10;
    const auto ts3 = ts.truncate(telem::SECOND);
    ASSERT_EQ(ts3, telem::SECOND * 5);
}

TEST(TimeSpanTests, testTruncateZeroTarget) {
    const auto ts = telem::SECOND * 5;
    const auto target = TimeSpan(0);
    const auto ts3 = ts.truncate(target);
    ASSERT_EQ(ts3, ts);
}

TEST(TimeSpanTests, testDelta) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(2);
    const auto ts3 = ts.delta(ts2);
    ASSERT_EQ(ts3.nanoseconds(), 3);
}

TEST(TimeSpanTests, testScalarOperations) {
    const auto ts = TimeSpan(10);
    
    // Addition with scalar
    const auto ts2 = ts + 5;
    ASSERT_EQ(ts2.nanoseconds(), 15);
    const auto ts3 = 5 + ts;
    ASSERT_EQ(ts3.nanoseconds(), 15);

    // Subtraction with scalar
    const auto ts4 = ts - 5;
    ASSERT_EQ(ts4.nanoseconds(), 5);
    const auto ts5 = 15 - ts;
    ASSERT_EQ(ts5.nanoseconds(), 5);
}

TEST(TimeSpanTests, testScalarAssignments) {
    auto ts = TimeSpan(10);
    
    ts += 5;
    ASSERT_EQ(ts.nanoseconds(), 15);
    
    ts -= 5;
    ASSERT_EQ(ts.nanoseconds(), 10);
    
    ts *= 2;
    ASSERT_EQ(ts.nanoseconds(), 20);
    
    ts /= 2;
    ASSERT_EQ(ts.nanoseconds(), 10);
    
    ts %= 3;
    ASSERT_EQ(ts.nanoseconds(), 1);
}

TEST(TimeSpanTests, testToString) {
    const auto ts = TimeSpan(_priv::DAY + _priv::HOUR + _priv::MINUTE + 
                            _priv::SECOND + _priv::MILLISECOND + 
                            _priv::MICROSECOND + 1); // 1 day, 1 hour, 1 minute, 1 second, 1ms, 1us, 1ns
    const auto str = ts.to_string();
    ASSERT_EQ(str, "1d 1h 1m 1s 1ms 1us 1ns");
    
    // Test zero case
    const auto zero = TimeSpan(0);
    ASSERT_EQ(zero.to_string(), "0ns");
}

TEST(TimeSpanTests, testChronoConversion) {
    const auto ts = TimeSpan(_priv::SECOND);
    const auto chrono_duration = ts.chrono();
    ASSERT_EQ(chrono_duration.count(), _priv::SECOND);
}

TEST(TimeSpanTests, testZeroStatic) {
    const auto zero = TimeSpan::ZERO();
    ASSERT_EQ(zero.nanoseconds(), 0);
}

TEST(TimeSpanTests, testAbs) {
    const auto positive = TimeSpan(5);
    ASSERT_EQ(positive.abs().nanoseconds(), 5);
    
    const auto negative = TimeSpan(-5);
    ASSERT_EQ(negative.abs().nanoseconds(), 5);
    
    const auto zero = TimeSpan(0);
    ASSERT_EQ(zero.abs().nanoseconds(), 0);
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
    ASSERT_EQ(r.period(), telem::SECOND);
    const auto r2 = Rate(2);
    ASSERT_EQ(r2.period(), telem::SECOND / 2);
}

TEST(RateTests, testContructor) {
    const auto r = Rate(5);
    ASSERT_EQ(r.hz(), 5);
}

TEST(RateTests, testAddition) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r + r2;
    ASSERT_EQ(r3.hz(), 10);
}

TEST(RateTests, testSubtraction) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r - r2;
    ASSERT_EQ(r3.hz(), 0);
}

TEST(RateTests, testMultiplication) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r * r2;
    ASSERT_EQ(r3.hz(), 25);

    const auto r4 = Rate(5);
    const auto r5 = r4 * 5;
    ASSERT_EQ(r5.hz(), 25);

    const auto r6 = Rate(5);
    const auto r7 = 5 * r6;
    ASSERT_EQ(r7.hz(), 25);

    const auto r8 = Rate(5);
    const auto r9 = r8 * 5.0;
    ASSERT_EQ(r9.hz(), 25);

    const auto r10 = Rate(5);
    const auto r11 = r10 * 5.0f;
    ASSERT_EQ(r11.hz(), 25);

    const auto r12 = Rate(5);
    const auto r13 = r12 * 5.0l;
    ASSERT_EQ(r13.hz(), 25);
}

TEST(RateTests, testDivision) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto multiple = r / r2;
    ASSERT_EQ(multiple, 1);

    const auto r4 = Rate(5);
    const auto r5 = r4 / 5;
    ASSERT_EQ(r5.hz(), 1);
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

TEST(RateTests, testPeriodVariousFrequencies) {
    // Test common frequencies
    ASSERT_EQ(Rate(1).period(), SECOND);          // 1 Hz = 1s
    ASSERT_EQ(Rate(2).period(), SECOND / 2);      // 2 Hz = 500ms
    ASSERT_EQ(Rate(5).period(), SECOND / 5);      // 5 Hz = 200ms
    ASSERT_EQ(Rate(10).period(), SECOND / 10);    // 10 Hz = 100ms
    ASSERT_EQ(Rate(20).period(), SECOND / 20);    // 20 Hz = 50ms
    ASSERT_EQ(Rate(50).period(), SECOND / 50);    // 50 Hz = 20ms
    ASSERT_EQ(Rate(100).period(), SECOND / 100);  // 100 Hz = 10ms
    ASSERT_EQ(Rate(1000).period(), SECOND / 1000); // 1kHz = 1ms
    
    // Verify actual time values
    ASSERT_EQ(Rate(20).period().milliseconds(), 50);   // 20 Hz should be 50ms
    ASSERT_EQ(Rate(50).period().milliseconds(), 20);   // 50 Hz should be 20ms
    ASSERT_EQ(Rate(100).period().milliseconds(), 10);  // 100 Hz should be 10ms
    ASSERT_EQ(Rate(1000).period().milliseconds(), 1);  // 1kHz should be 1ms
}

////////////////////////////////////////////////////////////
// DataType Tests
////////////////////////////////////////////////////////////

class DataTypeTests : public ::testing::Test {
};

struct TypeTestCase {
    telem::DataType expected;
    std::function<DataType()> inferFn;
};

class DataTypeInferTest : public testing::TestWithParam<TypeTestCase> {
};

TEST_P(DataTypeInferTest, testInferBuiltInTypes) {
    const auto &[expected, infer_fn] = GetParam();
    const auto dt = infer_fn();
    ASSERT_EQ(dt, expected);
}

INSTANTIATE_TEST_SUITE_P(
    DataTypes,
    DataTypeInferTest,
    testing::Values(
        TypeTestCase{telem::INT8_T, []() { return DataType::infer<int8_t>(); }},
        TypeTestCase{telem::UINT8_T, []() { return DataType::infer<uint8_t>(); }},
        TypeTestCase{telem::INT16_T, []() { return DataType::infer<int16_t>(); }},
        TypeTestCase{telem::UINT16_T, []() { return DataType::infer<uint16_t>(); }},
        TypeTestCase{telem::INT32_T, []() { return DataType::infer<int32_t>(); }},
        TypeTestCase{telem::UINT32_T, []() { return DataType::infer<uint32_t>(); }},
        TypeTestCase{telem::INT64_T, []() { return DataType::infer<int64_t>(); }},
        TypeTestCase{telem::UINT64_T, []() { return DataType::infer<uint64_t>(); }},
        TypeTestCase{telem::FLOAT32_T, []() { return DataType::infer<float>(); }},
        TypeTestCase{telem::FLOAT64_T, []() { return DataType::infer<double>(); }},
        TypeTestCase{telem::TIMESTAMP_T, []() { return DataType::infer<TimeStamp>(); }},
        TypeTestCase{telem::STRING_T, []() { return DataType::infer<std::string>(); }}
    )
);

TEST(DataTypeTests, testInferOveride) {
    const auto dt = DataType::infer<int8_t>(INT16_T);
    ASSERT_EQ(dt, INT16_T);
}

TEST(DataTypeTests, testName) {
    const auto dt = telem::FLOAT32_T;
    ASSERT_EQ(dt.name(), "float32");
}

TEST(DataTypeTests, testDensity) {
    ASSERT_EQ(telem::FLOAT64_T.density(), 8);
    ASSERT_EQ(telem::FLOAT32_T.density(), 4);
    ASSERT_EQ(telem::INT8_T.density(), 1);
    ASSERT_EQ(telem::INT16_T.density(), 2);
    ASSERT_EQ(telem::INT32_T.density(), 4);
    ASSERT_EQ(telem::INT64_T.density(), 8);
    ASSERT_EQ(telem::UINT8_T.density(), 1);
    ASSERT_EQ(telem::UINT16_T.density(), 2);
    ASSERT_EQ(telem::UINT32_T.density(), 4);
    ASSERT_EQ(telem::UINT64_T.density(), 8);
    ASSERT_EQ(telem::UINT128_T.density(), 16);
    ASSERT_EQ(telem::TIMESTAMP_T.density(), 8);
    ASSERT_EQ(telem::UUID_T.density(), 16);
    ASSERT_EQ(telem::STRING_T.density(), 0);
    ASSERT_EQ(telem::JSON_T.density(), 0);
}

TEST(DataTypeTests, testIsVariable) {
    ASSERT_TRUE(telem::STRING_T.is_variable());
    ASSERT_TRUE(telem::JSON_T.is_variable());
    ASSERT_FALSE(telem::FLOAT32_T.is_variable());
    ASSERT_FALSE(telem::INT64_T.is_variable());
}

TEST(DataTypeTests, testMatches) {
    const auto empty = telem::UNKNOWN_T;
    const auto dt = telem::FLOAT32_T;

    const std::vector types = {telem::FLOAT32_T, telem::FLOAT64_T};
    ASSERT_TRUE(dt.matches(types));

    const std::vector non_matching = {telem::INT32_T, telem::INT64_T};
    ASSERT_FALSE(dt.matches(non_matching));
}

TEST(DataTypeTests, testEquality) {
    const auto dt1 = telem::FLOAT32_T;
    const auto dt2 = telem::FLOAT32_T;
    const auto dt3 = telem::FLOAT64_T;

    ASSERT_TRUE(dt1 == dt2);
    ASSERT_FALSE(dt1 == dt3);
}

TEST(DataTypeTests, testInequality) {
    const auto dt1 = telem::FLOAT32_T;
    const auto dt2 = telem::FLOAT32_T;
    const auto dt3 = telem::FLOAT64_T;

    ASSERT_FALSE(dt1 != dt2);
    ASSERT_TRUE(dt1 != dt3);
}

TEST(DataTypeTests, testStreamOperator) {
    const auto dt = telem::FLOAT32_T;
    std::stringstream ss;
    ss << dt;
    ASSERT_EQ(ss.str(), "float32");
}
}