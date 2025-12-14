// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"

namespace telem {
/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testConstructor) {
    const auto ts = TimeStamp(5);
    ASSERT_EQ(ts.nanoseconds(), 5);
}

/// @brief it should add two timestamps together.
TEST(TimeStampTests, testAddition) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.nanoseconds(), 10);
}

/// @brief it should subtract two timestamps.
TEST(TimeStampTests, testSubtraction) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.nanoseconds(), 0);
}

/// @brief it should multiply two timestamps.
TEST(TimeStampTests, testMultiplication) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts * ts2;
    ASSERT_EQ(ts3.nanoseconds(), 25);
}

/// @brief it should divide two timestamps.
TEST(TimeStampTests, testDivision) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    const auto ts3 = ts / ts2;
    ASSERT_EQ(ts3.nanoseconds(), 1);
}

/// @brief it should compare two timestamps for equality.
TEST(TimeStampTests, testEquality) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts == ts2);
}

/// @brief it should compare two timestamps for inequality.
TEST(TimeStampTests, testInequality) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts != ts2);
}

/// @brief it should compare two timestamps with less than.
TEST(TimeStampTests, testLessThan) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(6);
    ASSERT_TRUE(ts < ts2);
}

/// @brief it should compare two timestamps with less than or equal.
TEST(TimeStampTests, testLessThanEqual) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts <= ts2);
}

/// @brief it should compare two timestamps with greater than.
TEST(TimeStampTests, testGreaterThan) {
    const auto ts = TimeStamp(6);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts > ts2);
}

/// @brief it should compare two timestamps with greater than or equal.
TEST(TimeStampTests, testGreaterThanEqual) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ASSERT_TRUE(ts >= ts2);
}

/// @brief it should calculate modulo of two timestamps.
TEST(TimeStampTests, testModulo) {
    const auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(2);
    const auto ts3 = ts % ts2;
    ASSERT_EQ(ts3.nanoseconds(), 1);
}

/// @brief it should add and assign timestamps.
TEST(TimeStampTests, testAdditionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts += ts2;
    ASSERT_EQ(ts.nanoseconds(), 10);
}

/// @brief it should subtract and assign timestamps.
TEST(TimeStampTests, testSubtractionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts -= ts2;
    ASSERT_EQ(ts.nanoseconds(), 0);
}

/// @brief it should multiply and assign timestamps.
TEST(TimeStampTests, testMultiplicationAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts *= ts2;
    ASSERT_EQ(ts.nanoseconds(), 25);
}

/// @brief it should divide and assign timestamps.
TEST(TimeStampTests, testDivisionAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(5);
    ts /= ts2;
    ASSERT_EQ(ts.nanoseconds(), 1);
}

/// @brief it should calculate modulo and assign timestamps.
TEST(TimeStampTests, testModuloAssignment) {
    auto ts = TimeStamp(5);
    const auto ts2 = TimeStamp(2);
    ts %= ts2;
    ASSERT_EQ(ts.nanoseconds(), 1);
}

////////////////////////////////////////////////////////////
// TimeSpan Tests
////////////////////////////////////////////////////////////

/// @brief it should initialize a timespan from a long.
TEST(TimeSpanTests, testConstructor) {
    const auto ts = TimeSpan(5);
    ASSERT_EQ(ts.nanoseconds(), 5);
}

/// @brief it should add two timespans together.
TEST(TimeSpanTests, testAddition) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts + ts2;
    ASSERT_EQ(ts3.nanoseconds(), 10);
}

/// @brief it should subtract two timespans.
TEST(TimeSpanTests, testSubtraction) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    const auto ts3 = ts - ts2;
    ASSERT_EQ(ts3.nanoseconds(), 0);
}

/// @brief it should multiply two timespans.
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

/// @brief it should divide two timespans.
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

/// @brief it should compare two timespans for equality.
TEST(TimeSpanTests, testEquality) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts == ts2);
}

/// @brief it should compare two timespans for inequality.
TEST(TimeSpanTests, testInequality) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts != ts2);
}

/// @brief it should compare two timespans with less than.
TEST(TimeSpanTests, testLessThan) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(6);
    ASSERT_TRUE(ts < ts2);
}

/// @brief it should compare two timespans with less than or equal.
TEST(TimeSpanTests, testLessThanEqual) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts <= ts2);
}

/// @brief it should compare two timespans with greater than.
TEST(TimeSpanTests, testGreaterThan) {
    const auto ts = TimeSpan(6);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts > ts2);
}

/// @brief it should compare two timespans with greater than or equal.
TEST(TimeSpanTests, testGreaterThanEqual) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(5);
    ASSERT_TRUE(ts >= ts2);
}

/// @brief it should calculate modulo of two timespans.
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

/// @brief it should truncate a timespan to a given unit.
TEST(TimeSpanTests, testTruncate) {
    const auto ts = telem::SECOND * 5 + telem::MICROSECOND * 10;
    const auto ts3 = ts.truncate(telem::SECOND);
    ASSERT_EQ(ts3, telem::SECOND * 5);
}

/// @brief it should return the original timespan when truncating to zero.
TEST(TimeSpanTests, testTruncateZeroTarget) {
    const auto ts = telem::SECOND * 5;
    const auto target = TimeSpan(0);
    const auto ts3 = ts.truncate(target);
    ASSERT_EQ(ts3, ts);
}

/// @brief it should calculate the delta between two timespans.
TEST(TimeSpanTests, testDelta) {
    const auto ts = TimeSpan(5);
    const auto ts2 = TimeSpan(2);
    const auto ts3 = ts.delta(ts2);
    ASSERT_EQ(ts3.nanoseconds(), 3);
}

/// @brief it should perform arithmetic operations with scalars.
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

/// @brief it should perform compound assignment operations with scalars.
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

/// @brief it should convert a timespan to a human-readable string.
TEST(TimeSpanTests, testToString) {
    const auto ts = TimeSpan(
        DAY + HOUR + MINUTE + SECOND + MILLISECOND + MICROSECOND + 1
    ); // 1 day, 1 hour, 1 minute, 1 second, 1ms, 1us, 1ns
    const auto str = ts.to_string();
    ASSERT_EQ(str, "1d 1h 1m 1s 1ms 1us 1ns");

    // Test zero case
    const auto zero = TimeSpan(0);
    ASSERT_EQ(zero.to_string(), "0ns");
}

/// @brief it should convert a timespan to a std::chrono duration.
TEST(TimeSpanTests, testChronoConversion) {
    const auto ts = TimeSpan(SECOND);
    const auto chrono_duration = ts.chrono();
    ASSERT_EQ(chrono_duration.count(), SECOND.nanoseconds());
}

/// @brief it should return a zero timespan from the static method.
TEST(TimeSpanTests, testZeroStatic) {
    const auto zero = TimeSpan(0);
    ASSERT_EQ(zero.nanoseconds(), 0);
}

/// @brief it should return the absolute value of a timespan.
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

/// @brief it should check if a timestamp is contained within a time range.
TEST(TimeRangeTests, testContains) {
    const auto tr = TimeRange(5, 10);
    const auto ts = TimeStamp(7);
    ASSERT_TRUE(tr.contains(ts));
}

/// @brief it should check if a time range is contained within another time range.
TEST(TimeRangeTests, testContainsRange) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(6, 9);
    ASSERT_TRUE(tr.contains(tr2));
}

/// @brief it should return true for equal time ranges.
TEST(TimeRangeTests, testEqualOperatorEqual) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(5, 10);
    ASSERT_TRUE(tr == tr2);
}

/// @brief it should return false for unequal time ranges.
TEST(TimeRangeTests, testEqualOperatorNotEqual) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(5, 11);
    ASSERT_FALSE(tr == tr2);
}

/// @brief it should return false for the not-equal operator on equal time ranges.
TEST(TimeRangetests, testNotEqualOperatorEqual) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(5, 10);
    ASSERT_FALSE(tr != tr2);
}

/// @brief it should return true for the not-equal operator on unequal time ranges.
TEST(TimeRangeTests, testNotEqualOperatorNotEqual) {
    const auto tr = TimeRange(5, 10);
    const auto tr2 = TimeRange(5, 11);
    ASSERT_TRUE(tr != tr2);
}

////////////////////////////////////////////////////////////
// Rate Tests
////////////////////////////////////////////////////////////

/// @brief it should calculate the period from a rate.
TEST(RateTests, testPeriod) {
    const auto r = Rate(1);
    ASSERT_EQ(r.period(), telem::SECOND);
    const auto r2 = Rate(2);
    ASSERT_EQ(r2.period(), telem::SECOND / 2);
}

/// @brief it should initialize a rate from a frequency.
TEST(RateTests, testContructor) {
    const auto r = Rate(5);
    ASSERT_EQ(r.hz(), 5);
}

/// @brief it should add two rates together.
TEST(RateTests, testAddition) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r + r2;
    ASSERT_EQ(r3.hz(), 10);
}

/// @brief it should subtract two rates.
TEST(RateTests, testSubtraction) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto r3 = r - r2;
    ASSERT_EQ(r3.hz(), 0);
}

/// @brief it should multiply two rates.
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

/// @brief it should divide two rates.
TEST(RateTests, testDivision) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    const auto multiple = r / r2;
    ASSERT_EQ(multiple, 1);

    const auto r4 = Rate(5);
    const auto r5 = r4 / 5;
    ASSERT_EQ(r5.hz(), 1);
}

/// @brief it should compare two rates for equality.
TEST(RateTests, testEquality) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r == r2);
}

/// @brief it should compare two rates for inequality.
TEST(RateTests, testInequality) {
    const auto r = Rate(5);
    const auto r2 = Rate(6);
    ASSERT_TRUE(r != r2);
}

/// @brief it should compare two rates with less than.
TEST(RateTests, testLessThan) {
    const auto r = Rate(5);
    const auto r2 = Rate(6);
    ASSERT_TRUE(r < r2);
}

/// @brief it should compare two rates with less than or equal.
TEST(RateTests, testLessThanEqual) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r <= r2);
}

/// @brief it should compare two rates with greater than.
TEST(RateTests, testGreaterThan) {
    const auto r = Rate(6);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r > r2);
}

/// @brief it should compare two rates with greater than or equal.
TEST(RateTests, testGreaterThanEqual) {
    const auto r = Rate(5);
    const auto r2 = Rate(5);
    ASSERT_TRUE(r >= r2);
}

/// @brief it should calculate periods for various frequencies.
TEST(RateTests, testPeriodVariousFrequencies) {
    // Test common frequencies
    ASSERT_EQ(Rate(1).period(), SECOND); // 1 Hz = 1s
    ASSERT_EQ(Rate(2).period(), SECOND / 2); // 2 Hz = 500ms
    ASSERT_EQ(Rate(5).period(), SECOND / 5); // 5 Hz = 200ms
    ASSERT_EQ(Rate(10).period(), SECOND / 10); // 10 Hz = 100ms
    ASSERT_EQ(Rate(20).period(), SECOND / 20); // 20 Hz = 50ms
    ASSERT_EQ(Rate(50).period(), SECOND / 50); // 50 Hz = 20ms
    ASSERT_EQ(Rate(100).period(), SECOND / 100); // 100 Hz = 10ms
    ASSERT_EQ(Rate(1000).period(), SECOND / 1000); // 1kHz = 1ms

    // Verify actual time values
    ASSERT_EQ(Rate(20).period().milliseconds(), 50); // 20 Hz should be 50ms
    ASSERT_EQ(Rate(50).period().milliseconds(), 20); // 50 Hz should be 20ms
    ASSERT_EQ(Rate(100).period().milliseconds(), 10); // 100 Hz should be 10ms
    ASSERT_EQ(Rate(1000).period().milliseconds(), 1); // 1kHz should be 1ms
}

/// @brief Test that telem::Rate can be streamed with << operator
TEST(RateTests, testRateStreamOperator) {
    const Rate rate_25(25.0);
    std::ostringstream oss;
    oss << rate_25;
    EXPECT_EQ(oss.str(), "25 Hz");

    const Rate rate_100(100.5);
    std::ostringstream oss2;
    oss2 << rate_100;
    EXPECT_EQ(oss2.str(), "100.5 Hz");

    const Rate rate_zero(0.0);
    std::ostringstream oss3;
    oss3 << rate_zero;
    EXPECT_EQ(oss3.str(), "0 Hz");
}

/// @brief Test that Rate operator<< works in error messages
TEST(RateTests, testRateInErrorMessage) {
    const Rate configured_rate(25.0);
    std::ostringstream msg;
    msg << "configured sample rate (" << configured_rate << ") is below device minimum";
    EXPECT_TRUE(msg.str().find("25 Hz") != std::string::npos);
    EXPECT_FALSE(msg.str().find(".hz()") != std::string::npos);
}

////////////////////////////////////////////////////////////
// DataType Tests
////////////////////////////////////////////////////////////

class DataTypeTests : public ::testing::Test {};

struct TypeTestCase {
    telem::DataType expected;
    std::function<DataType()> infer_fn;
};

class DataTypeInferTest : public testing::TestWithParam<TypeTestCase> {};

/// @brief it should infer the correct data type for built-in types.
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

/// @brief it should allow overriding the inferred data type.
TEST(DataTypeTests, testInferOveride) {
    const auto dt = DataType::infer<int8_t>(INT16_T);
    ASSERT_EQ(dt, INT16_T);
}

/// @brief it should return the name of a data type.
TEST(DataTypeTests, testName) {
    const auto dt = telem::FLOAT32_T;
    ASSERT_EQ(dt.name(), "float32");
}

/// @brief it should return the byte density for each data type.
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
    ASSERT_EQ(telem::TIMESTAMP_T.density(), 8);
    ASSERT_EQ(telem::UUID_T.density(), 16);
    ASSERT_EQ(telem::STRING_T.density(), 0);
    ASSERT_EQ(telem::JSON_T.density(), 0);
}

/// @brief it should identify variable-length data types.
TEST(DataTypeTests, testIsVariable) {
    ASSERT_TRUE(telem::STRING_T.is_variable());
    ASSERT_TRUE(telem::JSON_T.is_variable());
    ASSERT_FALSE(telem::FLOAT32_T.is_variable());
    ASSERT_FALSE(telem::INT64_T.is_variable());
}

/// @brief it should check if a data type matches a set of types.
TEST(DataTypeTests, testMatches) {
    const auto empty = telem::UNKNOWN_T;
    const auto dt = telem::FLOAT32_T;

    const std::vector types = {telem::FLOAT32_T, telem::FLOAT64_T};
    ASSERT_TRUE(dt.matches(types));

    const std::vector non_matching = {telem::INT32_T, telem::INT64_T};
    ASSERT_FALSE(dt.matches(non_matching));
}

/// @brief it should compare two data types for equality.
TEST(DataTypeTests, testEquality) {
    const auto dt1 = telem::FLOAT32_T;
    const auto dt2 = telem::FLOAT32_T;
    const auto dt3 = telem::FLOAT64_T;

    ASSERT_TRUE(dt1 == dt2);
    ASSERT_FALSE(dt1 == dt3);
}

/// @brief it should compare two data types for inequality.
TEST(DataTypeTests, testInequality) {
    const auto dt1 = telem::FLOAT32_T;
    const auto dt2 = telem::FLOAT32_T;
    const auto dt3 = telem::FLOAT64_T;

    ASSERT_FALSE(dt1 != dt2);
    ASSERT_TRUE(dt1 != dt3);
}

/// @brief it should stream a data type to an output stream.
TEST(DataTypeTests, testStreamOperator) {
    const auto dt = telem::FLOAT32_T;
    std::stringstream ss;
    ss << dt;
    ASSERT_EQ(ss.str(), "float32");
}

/// @brief it should return the domain index from an alignment.
TEST(AlignmentTests, testDomainIndex) {
    const telem::Alignment a(1, 0);
    ASSERT_EQ(a.domain_index(), 1);
}

/// @brief it should return the sample index from an alignment.
TEST(AlignmentTests, testSampleIndex) {
    const telem::Alignment a(0, 1);
    ASSERT_EQ(a.sample_index(), 1);
}

/// @brief it should construct an alignment from a uint64.
TEST(AlignmentTests, testConstructionFromUint64) {
    const telem::Alignment a(20);
    ASSERT_EQ(a.domain_index(), 0);
    ASSERT_EQ(a.sample_index(), 20);
}

/// @brief it should compare two alignments for equality.
TEST(AlignmentTests, testEquality) {
    auto a = telem::Alignment(1, 2);
    auto b = telem::Alignment(1, 2);
    auto c = telem::Alignment(2, 1);
    ASSERT_TRUE(a == b);
    ASSERT_FALSE(a != b);
    ASSERT_FALSE(a == c);
    ASSERT_TRUE(a != c);
}

/// @brief it should compare an alignment with a uint64 value.
TEST(AlignmentTests, testUint64Equality) {
    auto a = telem::Alignment(1, 2);
    ASSERT_TRUE(a == 4294967298);
    ASSERT_FALSE(a != 4294967298);
    ASSERT_FALSE(a == 4294967292);
    ASSERT_TRUE(a != 4294967294);
}

////////////////////////////////////////////////////////////
// to_string Tests
////////////////////////////////////////////////////////////

/// @brief it should convert a double to a string.
TEST(ToStringTests, testDoubleConversion) {
    constexpr SampleValue VALUE = 123.456;
    ASSERT_EQ(to_string(VALUE), "123.456000");
}

/// @brief it should convert a float to a string.
TEST(ToStringTests, testFloatConversion) {
    constexpr SampleValue VALUE = 123.456f;
    ASSERT_EQ(to_string(VALUE), "123.456001");
}

/// @brief it should convert an int64 to a string.
TEST(ToStringTests, testInt64Conversion) {
    constexpr SampleValue VALUE = static_cast<int64_t>(123456789);
    ASSERT_EQ(to_string(VALUE), "123456789");
}

/// @brief it should convert an int32 to a string.
TEST(ToStringTests, testInt32Conversion) {
    constexpr SampleValue VALUE = static_cast<int32_t>(123456);
    ASSERT_EQ(to_string(VALUE), "123456");
}

/// @brief it should convert an int16 to a string.
TEST(ToStringTests, testInt16Conversion) {
    constexpr SampleValue VALUE = static_cast<int16_t>(12345);
    ASSERT_EQ(to_string(VALUE), "12345");
}

/// @brief it should convert an int8 to a string.
TEST(ToStringTests, testInt8Conversion) {
    constexpr SampleValue VALUE = static_cast<int8_t>(123);
    ASSERT_EQ(to_string(VALUE), "123");
}

/// @brief it should convert a uint64 to a string.
TEST(ToStringTests, testUint64Conversion) {
    constexpr SampleValue VALUE = static_cast<uint64_t>(123456789);
    ASSERT_EQ(to_string(VALUE), "123456789");
}

/// @brief it should convert a uint32 to a string.
TEST(ToStringTests, testUint32Conversion) {
    constexpr SampleValue VALUE = static_cast<uint32_t>(123456);
    ASSERT_EQ(to_string(VALUE), "123456");
}

/// @brief it should convert a uint16 to a string.
TEST(ToStringTests, testUint16Conversion) {
    constexpr SampleValue VALUE = static_cast<uint16_t>(12345);
    ASSERT_EQ(to_string(VALUE), "12345");
}

/// @brief it should convert a uint8 to a string.
TEST(ToStringTests, testUint8Conversion) {
    constexpr SampleValue VALUE = static_cast<uint8_t>(123);
    ASSERT_EQ(to_string(VALUE), "123");
}

/// @brief it should convert a timestamp to a string.
TEST(ToStringTests, testTimeStampConversion) {
    const SampleValue value = TimeStamp(1234567890123456789);
    ASSERT_EQ(to_string(value), "1234567890123456789");
}

/// @brief it should return a string unchanged.
TEST(ToStringTests, testStringConversion) {
    constexpr SampleValue VALUE = std::string("hello world");
    ASSERT_EQ(to_string(VALUE), "hello world");
}

/// @brief it should convert negative numbers to strings.
TEST(ToStringTests, testNegativeNumbers) {
    constexpr SampleValue NEG_INT64 = static_cast<int64_t>(-123456789);
    ASSERT_EQ(to_string(NEG_INT64), "-123456789");

    constexpr SampleValue NEG_INT32 = static_cast<int32_t>(-123456);
    ASSERT_EQ(to_string(NEG_INT32), "-123456");

    constexpr SampleValue NEG_INT16 = static_cast<int16_t>(-12345);
    ASSERT_EQ(to_string(NEG_INT16), "-12345");

    constexpr SampleValue NEG_INT8 = static_cast<int8_t>(-123);
    ASSERT_EQ(to_string(NEG_INT8), "-123");

    constexpr SampleValue NEG_DOUBLE = -123.456;
    ASSERT_EQ(to_string(NEG_DOUBLE), "-123.456000");

    constexpr SampleValue NEG_FLOAT = -123.456f;
    ASSERT_EQ(to_string(NEG_FLOAT), "-123.456001");
}

/// @brief it should convert zero values to strings.
TEST(ToStringTests, testZeroValues) {
    constexpr SampleValue ZERO_INT64 = static_cast<int64_t>(0);
    ASSERT_EQ(to_string(ZERO_INT64), "0");

    constexpr SampleValue ZERO_DOUBLE = 0.0;
    ASSERT_EQ(to_string(ZERO_DOUBLE), "0.000000");

    constexpr SampleValue ZERO_FLOAT = 0.0f;
    ASSERT_EQ(to_string(ZERO_FLOAT), "0.000000");

    const SampleValue zero_timestamp = TimeStamp(0);
    ASSERT_EQ(to_string(zero_timestamp), "0");
}

/// @brief it should handle empty strings.
TEST(ToStringTests, testEmptyString) {
    const SampleValue value = std::string("");
    ASSERT_EQ(to_string(value), "");
}
}
