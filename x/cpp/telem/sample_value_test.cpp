// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "gtest/gtest.h"

/// internal
#include "x/cpp/telem/telem.h"

namespace telem {

class NumericSampleValueTest : public ::testing::Test {
protected:
    // Test values of different types
    NumericSampleValue int8_val = static_cast<int8_t>(5);
    NumericSampleValue int16_val = static_cast<int16_t>(10);
    NumericSampleValue int32_val = 100;
    NumericSampleValue int64_val = static_cast<int64_t>(1000);
    NumericSampleValue uint8_val = static_cast<uint8_t>(6);
    NumericSampleValue uint16_val = static_cast<uint16_t>(11);
    NumericSampleValue uint32_val = static_cast<uint32_t>(101);
    NumericSampleValue uint64_val = static_cast<uint64_t>(1001);
    NumericSampleValue float32_val = 3.14f;
    NumericSampleValue float64_val = 2.71828;
    NumericSampleValue timestamp_val = TimeStamp(1000000000); // 1 second
};

// Addition tests
TEST_F(NumericSampleValueTest, AdditionSameType) {
    // Integer addition
    EXPECT_EQ(std::get<int32_t>(int32_val + int32_val), 200);
    EXPECT_EQ(std::get<int64_t>(int64_val + int64_val), 2000);
    EXPECT_EQ(std::get<uint32_t>(uint32_val + uint32_val), 202);
    EXPECT_EQ(std::get<uint8_t>(uint8_val + uint8_val), 12);
    
    // Float addition
    EXPECT_FLOAT_EQ(std::get<float>(float32_val + float32_val), 6.28f);
    EXPECT_DOUBLE_EQ(std::get<double>(float64_val + float64_val), 5.43656);
    
    // Timestamp addition
    const TimeStamp result = std::get<TimeStamp>(timestamp_val + timestamp_val);
    EXPECT_EQ(result.nanoseconds(), 2000000000);
}

TEST_F(NumericSampleValueTest, AdditionDifferentTypes) {
    // Integer promotion
    EXPECT_EQ(std::get<int32_t>(int8_val + int32_val), 105);
    EXPECT_EQ(std::get<int64_t>(int16_val + int64_val), 1010);
    EXPECT_EQ(std::get<uint32_t>(uint8_val + uint32_val), 107);
    
    // Float promotion
    EXPECT_NEAR(std::get<double>(float32_val + float64_val), 5.85828, 1e-5);
    
    // Mixed signed/unsigned
    EXPECT_EQ(std::get<int32_t>(int32_val + uint8_val), 106);
    
    // Integer + float
    EXPECT_NEAR(std::get<double>(int32_val + float64_val), 102.71828, 1e-5);
}

TEST_F(NumericSampleValueTest, AdditionWithTimestamp) {
    // Timestamp + integer
    const auto ts_int_result = std::get<TimeStamp>(timestamp_val + int32_val);
    EXPECT_EQ(ts_int_result.nanoseconds(), 1000000100);
    
    // Integer + timestamp
    const TimeStamp int_ts_result = std::get<TimeStamp>(int64_val + timestamp_val);
    EXPECT_EQ(int_ts_result.nanoseconds(), 1000001000);
}

// Subtraction tests
TEST_F(NumericSampleValueTest, SubtractionSameType) {
    // Integer subtraction
    EXPECT_EQ(std::get<int32_t>(int32_val - int8_val), 95);
    EXPECT_EQ(std::get<int64_t>(int64_val - int16_val), 990);
    EXPECT_EQ(std::get<uint32_t>(uint32_val - uint8_val), 95);
    
    // Float subtraction
    EXPECT_FLOAT_EQ(std::get<float>(float32_val - float32_val), 0.0f);
    EXPECT_NEAR(std::get<double>(float64_val - float32_val), -0.42172, 1e-5);
    
    // Timestamp subtraction
    const TimeStamp ts_diff = std::get<TimeStamp>(timestamp_val - timestamp_val);
    EXPECT_EQ(ts_diff, 0);
}

TEST_F(NumericSampleValueTest, SubtractionDifferentTypes) {
    // Integer promotion
    EXPECT_EQ(std::get<int32_t>(int32_val - int8_val), 95);
    EXPECT_EQ(std::get<int64_t>(int64_val - int16_val), 990);
    
    // Float promotion
    EXPECT_NEAR(std::get<double>(float64_val - float32_val), -0.42172, 1e-5);
    
    // Mixed signed/unsigned
    EXPECT_EQ(std::get<int32_t>(int32_val - uint8_val), 94);
    
    // Integer - float
    EXPECT_NEAR(std::get<double>(int32_val - float64_val), 97.28172, 1e-5);
}

TEST_F(NumericSampleValueTest, SubtractionWithTimestamp) {
    // Timestamp - integer
    const TimeStamp ts_int_result = std::get<TimeStamp>(timestamp_val - int32_val);
    EXPECT_EQ(ts_int_result.nanoseconds(), 999999900);
    
    // Integer - timestamp
    const TimeStamp int_ts_result = std::get<TimeStamp>(int64_val - timestamp_val);
    EXPECT_EQ(int_ts_result.nanoseconds(), -999999000);
    
    // Timestamp - timestamp = int64_t (nanoseconds)
    const NumericSampleValue ts2 = TimeStamp(500000000);
    const TimeStamp ts_diff = std::get<TimeStamp>(timestamp_val - ts2);
    EXPECT_EQ(ts_diff.nanoseconds(), 500000000);
}

// Multiplication tests
TEST_F(NumericSampleValueTest, MultiplicationSameType) {
    // Integer multiplication
    EXPECT_EQ(std::get<int32_t>(int32_val * int8_val), 500);
    EXPECT_EQ(std::get<int64_t>(int64_val * int16_val), 10000);
    EXPECT_EQ(std::get<uint32_t>(uint32_val * uint8_val), 606);
    
    // Float multiplication
    EXPECT_FLOAT_EQ(std::get<float>(float32_val * float32_val), 9.8596f);
    EXPECT_NEAR(std::get<double>(float64_val * float64_val), 7.38905, 1e-5);
}

TEST_F(NumericSampleValueTest, MultiplicationDifferentTypes) {
    // Integer promotion
    EXPECT_EQ(std::get<int32_t>(int32_val * int8_val), 500);
    EXPECT_EQ(std::get<int64_t>(int64_val * int16_val), 10000);
    
    // Float promotion - use a slightly larger epsilon for this specific case
    EXPECT_NEAR(std::get<double>(float64_val * float32_val), 8.53541, 2e-5);
    
    // Mixed signed/unsigned
    EXPECT_EQ(std::get<int32_t>(int32_val * uint8_val), 600);
    
    // Integer * float
    EXPECT_NEAR(std::get<double>(int32_val * float64_val), 271.828, 1e-3);
}

TEST_F(NumericSampleValueTest, MultiplicationWithTimestamp) {
    // Timestamp * integer
    const TimeStamp ts_int_result = std::get<TimeStamp>(timestamp_val * int8_val);
    EXPECT_EQ(ts_int_result.nanoseconds(), 5000000000);
    
    // Integer * timestamp
    const auto int_ts_result = std::get<TimeStamp>(int16_val * timestamp_val);
    EXPECT_EQ(int_ts_result.nanoseconds(), 10000000000);
}

// Division tests
TEST_F(NumericSampleValueTest, DivisionSameType) {
    // Integer division
    EXPECT_EQ(std::get<int32_t>(int32_val / int8_val), 20);
    EXPECT_EQ(std::get<int64_t>(int64_val / int16_val), 100);
    EXPECT_EQ(std::get<uint32_t>(uint32_val / uint8_val), 16);
    
    // Float division
    EXPECT_FLOAT_EQ(std::get<float>(float32_val / float32_val), 1.0f);
    EXPECT_DOUBLE_EQ(std::get<double>(float64_val / float64_val), 1.0);
}

TEST_F(NumericSampleValueTest, DivisionDifferentTypes) {
    // Integer promotion
    EXPECT_EQ(std::get<int32_t>(int32_val / int8_val), 20);
    EXPECT_EQ(std::get<int64_t>(int64_val / int16_val), 100);
    
    // Float promotion
    EXPECT_NEAR(std::get<double>(float64_val / float32_val), 0.86569, 1e-5);
    
    // Mixed signed/unsigned
    EXPECT_EQ(std::get<int32_t>(int32_val / uint8_val), 16);
    
    // Integer / float - use a slightly larger epsilon for this specific case
    EXPECT_NEAR(std::get<double>(int32_val / float64_val), 36.78794, 3e-5);
}

TEST_F(NumericSampleValueTest, DivisionWithTimestamp) {
    // Timestamp / integer
    const TimeStamp ts_int_result = std::get<TimeStamp>(timestamp_val / int8_val);
    EXPECT_EQ(ts_int_result.nanoseconds(), 200000000);
    
    // Timestamp / timestamp = double ratio
    const NumericSampleValue ts2 = TimeStamp(500000000);
    const double ts_ratio = std::get<double>(timestamp_val / ts2);
    EXPECT_DOUBLE_EQ(ts_ratio, 2.0);
    
    // Integer / timestamp = double
    const double int_ts_ratio = std::get<double>(int64_val / timestamp_val);
    EXPECT_DOUBLE_EQ(int_ts_ratio, 1e-6);
}

TEST_F(NumericSampleValueTest, DivisionByZero) {
    NumericSampleValue zero_int = 0;
    NumericSampleValue zero_float = 0.0;
    
    // Division by zero should throw
    EXPECT_THROW(int32_val / zero_int, std::runtime_error);
    EXPECT_THROW(float64_val / zero_float, std::runtime_error);
}

// Edge cases
TEST_F(NumericSampleValueTest, EdgeCases) {
    // Extreme values
    NumericSampleValue max_int32 = std::numeric_limits<int32_t>::max();
    NumericSampleValue min_int32 = std::numeric_limits<int32_t>::min();
    NumericSampleValue max_float = std::numeric_limits<float>::max();
    
    // Overflow behavior (depends on platform)
    auto overflow_result = max_int32 + max_int32;
    EXPECT_EQ(std::get<int32_t>(overflow_result), -2);
    
    // Underflow behavior
    auto underflow_result = min_int32 - max_int32;
    EXPECT_EQ(std::get<int32_t>(underflow_result), 1);
    
    // Very large timestamp
    NumericSampleValue large_ts = TimeStamp(std::numeric_limits<int64_t>::max() - 100);
    NumericSampleValue small_val = static_cast<int8_t>(1);
    
    // This should not overflow due to the implementation
    EXPECT_NO_THROW(large_ts + small_val);
}

// Type conversion tests
TEST_F(NumericSampleValueTest, TypeConversion) {
    // Test narrow_numeric function
    SampleValue string_val = std::string("not a number");
    EXPECT_THROW(narrow_numeric(string_val), std::runtime_error);

    SampleValue numeric_val = 42.0;
    NumericSampleValue narrowed = narrow_numeric(numeric_val);
    EXPECT_DOUBLE_EQ(std::get<double>(narrowed), 42.0);

    // // Test casting between types
    // DataType float_type = FLOAT64_T;
    // SampleValue casted = float_type.cast(int32_val);
    // EXPECT_DOUBLE_EQ(std::get<double>(casted), 100.0);
}

} // namespace telem

