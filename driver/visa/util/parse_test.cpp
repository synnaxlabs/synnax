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

#include "driver/visa/util/parse.h"
#include "driver/visa/channels.h"

using namespace visa::util;
using namespace visa::channel;

TEST(ParseTest, testTrim) {
    EXPECT_EQ(trim("  hello  "), "hello");
    EXPECT_EQ(trim("\t\n hello \r\n"), "hello");
    EXPECT_EQ(trim("   "), "");
    EXPECT_EQ(trim(""), "");
    EXPECT_EQ(trim("no-spaces"), "no-spaces");
}

TEST(ParseTest, testParseFloatValid) {
    auto [val1, err1] = parse_float("3.14159");
    ASSERT_NIL(err1);
    EXPECT_NEAR(val1, 3.14159, 0.0001);

    auto [val2, err2] = parse_float("  -42.5  ");
    ASSERT_NIL(err2);
    EXPECT_NEAR(val2, -42.5, 0.0001);

    auto [val3, err3] = parse_float("1.23e-4");
    ASSERT_NIL(err3);
    EXPECT_NEAR(val3, 1.23e-4, 1e-7);

    auto [val4, err4] = parse_float("0.0");
    ASSERT_NIL(err4);
    EXPECT_NEAR(val4, 0.0, 0.0001);
}

TEST(ParseTest, testParseFloatInvalid) {
    auto [val1, err1] = parse_float("");
    ASSERT_TRUE(err1);
    EXPECT_TRUE(err1.message().find("empty") != std::string::npos);

    auto [val2, err2] = parse_float("not-a-number");
    ASSERT_TRUE(err2);

    auto [val3, err3] = parse_float("  ");
    ASSERT_TRUE(err3);
}

TEST(ParseTest, testParseIntValid) {
    auto [val1, err1] = parse_int("42");
    ASSERT_NIL(err1);
    EXPECT_EQ(val1, 42);

    auto [val2, err2] = parse_int("  -123  ");
    ASSERT_NIL(err2);
    EXPECT_EQ(val2, -123);

    auto [val3, err3] = parse_int("0");
    ASSERT_NIL(err3);
    EXPECT_EQ(val3, 0);

    auto [val4, err4] = parse_int("9223372036854775807"); // INT64_MAX
    ASSERT_NIL(err4);
    EXPECT_EQ(val4, 9223372036854775807LL);
}

TEST(ParseTest, testParseIntInvalid) {
    auto [val1, err1] = parse_int("");
    ASSERT_TRUE(err1);

    auto [val2, err2] = parse_int("3.14");
    ASSERT_TRUE(err2);

    auto [val3, err3] = parse_int("abc");
    ASSERT_TRUE(err3);
}

TEST(ParseTest, testParseBoolValid) {
    auto [val1, err1] = parse_bool("1");
    ASSERT_NIL(err1);
    EXPECT_TRUE(val1);

    auto [val2, err2] = parse_bool("0");
    ASSERT_NIL(err2);
    EXPECT_FALSE(val2);

    auto [val3, err3] = parse_bool("  ON  ");
    ASSERT_NIL(err3);
    EXPECT_TRUE(val3);

    auto [val4, err4] = parse_bool("off");
    ASSERT_NIL(err4);
    EXPECT_FALSE(val4);

    auto [val5, err5] = parse_bool("TRUE");
    ASSERT_NIL(err5);
    EXPECT_TRUE(val5);

    auto [val6, err6] = parse_bool("false");
    ASSERT_NIL(err6);
    EXPECT_FALSE(val6);

    auto [val7, err7] = parse_bool("yes");
    ASSERT_NIL(err7);
    EXPECT_TRUE(val7);

    auto [val8, err8] = parse_bool("NO");
    ASSERT_NIL(err8);
    EXPECT_FALSE(val8);
}

TEST(ParseTest, testParseBoolInvalid) {
    auto [val1, err1] = parse_bool("");
    ASSERT_TRUE(err1);

    auto [val2, err2] = parse_bool("maybe");
    ASSERT_TRUE(err2);

    auto [val3, err3] = parse_bool("2");
    ASSERT_TRUE(err3);
}

TEST(ParseTest, testParseFloatArrayValid) {
    auto [vals1, err1] = parse_float_array("1.0,2.0,3.0");
    ASSERT_NIL(err1);
    ASSERT_EQ(vals1.size(), 3);
    EXPECT_NEAR(vals1[0], 1.0, 0.001);
    EXPECT_NEAR(vals1[1], 2.0, 0.001);
    EXPECT_NEAR(vals1[2], 3.0, 0.001);

    auto [vals2, err2] = parse_float_array("  -1.5 ,  2.5 ,  -3.5  ");
    ASSERT_NIL(err2);
    ASSERT_EQ(vals2.size(), 3);
    EXPECT_NEAR(vals2[0], -1.5, 0.001);
    EXPECT_NEAR(vals2[1], 2.5, 0.001);
    EXPECT_NEAR(vals2[2], -3.5, 0.001);

    auto [vals3, err3] = parse_float_array("42");
    ASSERT_NIL(err3);
    ASSERT_EQ(vals3.size(), 1);
    EXPECT_NEAR(vals3[0], 42.0, 0.001);
}

TEST(ParseTest, testParseFloatArrayInvalid) {
    auto [vals1, err1] = parse_float_array("");
    ASSERT_TRUE(err1);

    auto [vals2, err2] = parse_float_array("1.0,bad,3.0");
    ASSERT_TRUE(err2);

    auto [vals3, err3] = parse_float_array("  ");
    ASSERT_TRUE(err3);
}

TEST(ParseTest, testParseBinaryHeaderValid) {
    // #2<len><data> format: #<digit count><length>
    auto [len1, err1] = parse_binary_header("#210data here");
    ASSERT_NIL(err1);
    EXPECT_EQ(len1, 10);

    auto [len2, err2] = parse_binary_header("#3100data...");
    ASSERT_NIL(err2);
    EXPECT_EQ(len2, 100);

    auto [len3, err3] = parse_binary_header("#15data");
    ASSERT_NIL(err3);
    EXPECT_EQ(len3, 5);
}

TEST(ParseTest, testParseBinaryHeaderInvalid) {
    auto [len1, err1] = parse_binary_header("no hash");
    ASSERT_TRUE(err1);
    EXPECT_TRUE(err1.message().find("#") != std::string::npos);

    auto [len2, err2] = parse_binary_header("#");
    ASSERT_TRUE(err2);

    auto [len3, err3] = parse_binary_header("#0");
    ASSERT_TRUE(err3);

    auto [len4, err4] = parse_binary_header("#21"); // Too short for claimed digits
    ASSERT_TRUE(err4);

    auto [len5, err5] = parse_binary_header("#abc");
    ASSERT_TRUE(err5);
}

TEST(ParseTest, testParseResponseFloat) {
    InputChannel ch(
        1,
        "MEAS:VOLT?",
        ResponseFormat::FLOAT,
        telem::FLOAT64_T
    );

    auto [series, err] = parse_response("3.14159", ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 1);
    EXPECT_NEAR(series.at<double>(0), 3.14159, 0.0001);
}

TEST(ParseTest, testParseResponseInteger) {
    InputChannel ch(
        1,
        "MEAS:COUNT?",
        ResponseFormat::INTEGER,
        telem::INT64_T
    );

    auto [series, err] = parse_response("42", ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 1);
    EXPECT_EQ(series.at<int64_t>(0), 42);
}

TEST(ParseTest, testParseResponseString) {
    InputChannel ch(
        1,
        "SYST:ERR?",
        ResponseFormat::STRING,
        telem::STRING_T
    );

    auto [series, err] = parse_response("  No Error  ", ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 1);
    EXPECT_EQ(series.at<std::string>(0), "No Error");
}

TEST(ParseTest, testParseResponseBoolean) {
    InputChannel ch(
        1,
        "OUTP:STAT?",
        ResponseFormat::BOOLEAN,
        telem::UINT8_T
    );

    auto [series1, err1] = parse_response("1", ch);
    ASSERT_NIL(err1);
    ASSERT_EQ(series1.size(), 1);
    EXPECT_EQ(series1.at<uint8_t>(0), 1);

    auto [series2, err2] = parse_response("OFF", ch);
    ASSERT_NIL(err2);
    ASSERT_EQ(series2.size(), 1);
    EXPECT_EQ(series2.at<uint8_t>(0), 0);
}

TEST(ParseTest, testParseResponseFloatArray) {
    InputChannel ch(
        1,
        "TRAC:DATA?",
        ResponseFormat::FLOAT_ARRAY,
        telem::FLOAT64_T,
        ",",
        3 // Expected length
    );

    auto [series, err] = parse_response("1.0,2.0,3.0", ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 3);
    EXPECT_NEAR(series.at<double>(0), 1.0, 0.001);
    EXPECT_NEAR(series.at<double>(1), 2.0, 0.001);
    EXPECT_NEAR(series.at<double>(2), 3.0, 0.001);
}

TEST(ParseTest, testParseResponseFloatArrayLengthMismatch) {
    InputChannel ch(
        1,
        "TRAC:DATA?",
        ResponseFormat::FLOAT_ARRAY,
        telem::FLOAT64_T,
        ",",
        5 // Expected 5, but get 3
    );

    auto [series, err] = parse_response("1.0,2.0,3.0", ch);
    ASSERT_TRUE(err);
    EXPECT_TRUE(err.message().find("mismatch") != std::string::npos);
}

TEST(ParseTest, testParseResponseFloatArrayNoLengthCheck) {
    InputChannel ch(
        1,
        "TRAC:DATA?",
        ResponseFormat::FLOAT_ARRAY,
        telem::FLOAT64_T,
        ",",
        0 // No length check
    );

    auto [series, err] = parse_response("1.0,2.0", ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 2);
}

TEST(ParseTest, testParseResponseBinaryBlock) {
    // Binary block: #<digit count><length><data>
    // #15<5 bytes of data>
    std::string response = "#15";
    response += std::string("\x01\x02\x03\x04\x05", 5);

    InputChannel ch(
        1,
        "CURV?",
        ResponseFormat::BINARY_BLOCK,
        telem::UINT8_T
    );

    auto [series, err] = parse_response(response, ch);
    ASSERT_NIL(err);
    ASSERT_EQ(series.size(), 5);
    EXPECT_EQ(series.at<uint8_t>(0), 1);
    EXPECT_EQ(series.at<uint8_t>(1), 2);
    EXPECT_EQ(series.at<uint8_t>(2), 3);
    EXPECT_EQ(series.at<uint8_t>(3), 4);
    EXPECT_EQ(series.at<uint8_t>(4), 5);
}

TEST(ParseTest, testParseResponseBinaryBlockTruncated) {
    // Claims 10 bytes but only provides 5
    std::string response = "#210";
    response += std::string("\x01\x02\x03\x04\x05", 5);

    InputChannel ch(
        1,
        "CURV?",
        ResponseFormat::BINARY_BLOCK,
        telem::UINT8_T
    );

    auto [series, err] = parse_response(response, ch);
    ASSERT_TRUE(err);
    EXPECT_TRUE(err.message().find("truncated") != std::string::npos);
}

TEST(ParseTest, testParseResponseInvalidFormat) {
    InputChannel ch(
        1,
        "MEAS?",
        ResponseFormat::FLOAT,
        telem::FLOAT64_T
    );

    auto [series, err] = parse_response("not-a-float", ch);
    ASSERT_TRUE(err);
}
