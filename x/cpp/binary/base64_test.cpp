// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "x/cpp/binary/base64.h"

TEST(Base64Decode, testEmptyString) {
    const auto result = binary::decode_base64("");
    ASSERT_EQ(result.size(), 0);
}

TEST(Base64Decode, testSingleByte) {
    const auto result = binary::decode_base64("QQ==");
    ASSERT_EQ(result.size(), 1);
    ASSERT_EQ(result[0], 'A');
}

TEST(Base64Decode, testTwoBytes) {
    const auto result = binary::decode_base64("QUI=");
    ASSERT_EQ(result.size(), 2);
    ASSERT_EQ(result[0], 'A');
    ASSERT_EQ(result[1], 'B');
}

TEST(Base64Decode, testThreeBytes) {
    const auto result = binary::decode_base64("QUJD");
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result[0], 'A');
    ASSERT_EQ(result[1], 'B');
    ASSERT_EQ(result[2], 'C');
}

TEST(Base64Decode, testFourBytes) {
    const auto result = binary::decode_base64("QUJDRA==");
    ASSERT_EQ(result.size(), 4);
    ASSERT_EQ(result[0], 'A');
    ASSERT_EQ(result[1], 'B');
    ASSERT_EQ(result[2], 'C');
    ASSERT_EQ(result[3], 'D');
}

TEST(Base64Decode, testRFC4648TestVector1) {
    const auto result = binary::decode_base64("Zg==");
    ASSERT_EQ(result.size(), 1);
    ASSERT_EQ(result[0], 'f');
}

TEST(Base64Decode, testRFC4648TestVector2) {
    const auto result = binary::decode_base64("Zm8=");
    ASSERT_EQ(result.size(), 2);
    ASSERT_EQ(result[0], 'f');
    ASSERT_EQ(result[1], 'o');
}

TEST(Base64Decode, testRFC4648TestVector3) {
    const auto result = binary::decode_base64("Zm9v");
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result[0], 'f');
    ASSERT_EQ(result[1], 'o');
    ASSERT_EQ(result[2], 'o');
}

TEST(Base64Decode, testRFC4648TestVector4) {
    const auto result = binary::decode_base64("Zm9vYg==");
    ASSERT_EQ(result.size(), 4);
    ASSERT_EQ(result[0], 'f');
    ASSERT_EQ(result[1], 'o');
    ASSERT_EQ(result[2], 'o');
    ASSERT_EQ(result[3], 'b');
}

TEST(Base64Decode, testRFC4648TestVector5) {
    const auto result = binary::decode_base64("Zm9vYmE=");
    ASSERT_EQ(result.size(), 5);
    ASSERT_EQ(result[0], 'f');
    ASSERT_EQ(result[1], 'o');
    ASSERT_EQ(result[2], 'o');
    ASSERT_EQ(result[3], 'b');
    ASSERT_EQ(result[4], 'a');
}

TEST(Base64Decode, testRFC4648TestVector6) {
    const auto result = binary::decode_base64("Zm9vYmFy");
    ASSERT_EQ(result.size(), 6);
    ASSERT_EQ(result[0], 'f');
    ASSERT_EQ(result[1], 'o');
    ASSERT_EQ(result[2], 'o');
    ASSERT_EQ(result[3], 'b');
    ASSERT_EQ(result[4], 'a');
    ASSERT_EQ(result[5], 'r');
}

TEST(Base64Decode, testAllZeros) {
    const auto result = binary::decode_base64("AAAA");
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result[0], 0x00);
    ASSERT_EQ(result[1], 0x00);
    ASSERT_EQ(result[2], 0x00);
}

TEST(Base64Decode, testAllOnes) {
    const auto result = binary::decode_base64("////");
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result[0], 0xFF);
    ASSERT_EQ(result[1], 0xFF);
    ASSERT_EQ(result[2], 0xFF);
}

TEST(Base64Decode, testBinaryPattern1) {
    const auto result = binary::decode_base64("EjRWeA==");
    ASSERT_EQ(result.size(), 4);
    ASSERT_EQ(result[0], 0x12);
    ASSERT_EQ(result[1], 0x34);
    ASSERT_EQ(result[2], 0x56);
    ASSERT_EQ(result[3], 0x78);
}

TEST(Base64Decode, testBinaryPattern2) {
    const auto result = binary::decode_base64("EjRWeJCrze8=");
    ASSERT_EQ(result.size(), 8);
    ASSERT_EQ(result[0], 0x12);
    ASSERT_EQ(result[1], 0x34);
    ASSERT_EQ(result[2], 0x56);
    ASSERT_EQ(result[3], 0x78);
    ASSERT_EQ(result[4], 0x90);
    ASSERT_EQ(result[5], 0xAB);
    ASSERT_EQ(result[6], 0xCD);
    ASSERT_EQ(result[7], 0xEF);
}

TEST(Base64Decode, testLongerText) {
    const auto result = binary::decode_base64("SGVsbG8gV29ybGQh");
    ASSERT_EQ(result.size(), 12);
    const std::string expected = "Hello World!";
    for (size_t i = 0; i < expected.size(); i++)
        ASSERT_EQ(result[i], expected[i]);
}

TEST(Base64Decode, testPaddingOneByte) {
    const auto result = binary::decode_base64("YQ==");
    ASSERT_EQ(result.size(), 1);
    ASSERT_EQ(result[0], 'a');
}

TEST(Base64Decode, testPaddingTwoBytes) {
    const auto result = binary::decode_base64("YWI=");
    ASSERT_EQ(result.size(), 2);
    ASSERT_EQ(result[0], 'a');
    ASSERT_EQ(result[1], 'b');
}

TEST(Base64Decode, testNoPadding) {
    const auto result = binary::decode_base64("YWJj");
    ASSERT_EQ(result.size(), 3);
    ASSERT_EQ(result[0], 'a');
    ASSERT_EQ(result[1], 'b');
    ASSERT_EQ(result[2], 'c');
}

TEST(Base64Decode, testUppercaseLetters) {
    const auto result = binary::decode_base64("QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=");
    ASSERT_EQ(result.size(), 26);
    for (size_t i = 0; i < 26; i++)
        ASSERT_EQ(result[i], 'A' + i);
}

TEST(Base64Decode, testLowercaseLetters) {
    const auto result = binary::decode_base64("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=");
    ASSERT_EQ(result.size(), 26);
    for (size_t i = 0; i < 26; i++)
        ASSERT_EQ(result[i], 'a' + i);
}

TEST(Base64Decode, testDigits) {
    const auto result = binary::decode_base64("MDEyMzQ1Njc4OQ==");
    ASSERT_EQ(result.size(), 10);
    for (size_t i = 0; i < 10; i++)
        ASSERT_EQ(result[i], '0' + i);
}

TEST(Base64Decode, testPlusAndSlash) {
    const auto result = binary::decode_base64("+/8=");
    ASSERT_EQ(result.size(), 2);
    ASSERT_EQ(result[0], 0xFB);
    ASSERT_EQ(result[1], 0xFF);
}

TEST(Base64Decode, testSequentialBytes) {
    const auto result = binary::decode_base64("AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=");
    ASSERT_EQ(result.size(), 32);
    for (size_t i = 0; i < 32; i++)
        ASSERT_EQ(result[i], static_cast<uint8_t>(i));
}

TEST(Base64Decode, testMaxByteValue) {
    const auto result = binary::decode_base64("/w==");
    ASSERT_EQ(result.size(), 1);
    ASSERT_EQ(result[0], 0xFF);
}

TEST(Base64Decode, testMultipleBlocks) {
    const auto result = binary::decode_base64("QUJDREVGR0hJSktM");
    ASSERT_EQ(result.size(), 12);
    const std::string expected = "ABCDEFGHIJKL";
    for (size_t i = 0; i < expected.size(); i++)
        ASSERT_EQ(result[i], expected[i]);
}

TEST(Base64Decode, testVariousBytePatterns) {
    const auto test1 = binary::decode_base64("AA==");
    ASSERT_EQ(test1.size(), 1);
    ASSERT_EQ(test1[0], 0x00);
    const auto test2 = binary::decode_base64("AQ==");
    ASSERT_EQ(test2.size(), 1);
    ASSERT_EQ(test2[0], 0x01);
    const auto test3 = binary::decode_base64("Ag==");
    ASSERT_EQ(test3.size(), 1);
    ASSERT_EQ(test3[0], 0x02);
    const auto test4 = binary::decode_base64("/w==");
    ASSERT_EQ(test4.size(), 1);
    ASSERT_EQ(test4[0], 0xFF);
    const auto test5 = binary::decode_base64("AAA=");
    ASSERT_EQ(test5.size(), 2);
    ASSERT_EQ(test5[0], 0x00);
    ASSERT_EQ(test5[1], 0x00);
    const auto test6 = binary::decode_base64("//8=");
    ASSERT_EQ(test6.size(), 2);
    ASSERT_EQ(test6[0], 0xFF);
    ASSERT_EQ(test6[1], 0xFF);
    const auto test7 = binary::decode_base64("AAAA");
    ASSERT_EQ(test7.size(), 3);
    ASSERT_EQ(test7[0], 0x00);
    ASSERT_EQ(test7[1], 0x00);
    ASSERT_EQ(test7[2], 0x00);
    const auto test8 = binary::decode_base64("////");
    ASSERT_EQ(test8.size(), 3);
    ASSERT_EQ(test8[0], 0xFF);
    ASSERT_EQ(test8[1], 0xFF);
    ASSERT_EQ(test8[2], 0xFF);
    const auto test9 = binary::decode_base64("VGVzdA==");
    ASSERT_EQ(test9.size(), 4);
    ASSERT_EQ(test9[0], 'T');
    ASSERT_EQ(test9[1], 'e');
    ASSERT_EQ(test9[2], 's');
    ASSERT_EQ(test9[3], 't');
}

TEST(Base64Decode, testLargeData) {
    std::string large_encoded;
    for (int i = 0; i < 100; i++)
        large_encoded += "QUJDREVGR0hJ";
    const auto result = binary::decode_base64(large_encoded);
    ASSERT_EQ(result.size(), 900);
    for (size_t i = 0; i < result.size(); i++)
        ASSERT_EQ(result[i], static_cast<uint8_t>('A' + (i % 9)));
}