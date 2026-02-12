// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "x/cpp/string/put.h"

TEST(Put, Width2Zero) {
    char buf[2];
    x::strings::put(buf, 0, 2);
    EXPECT_EQ(std::string(buf, 2), "00");
}

TEST(Put, Width2SingleDigit) {
    char buf[2];
    x::strings::put(buf, 7, 2);
    EXPECT_EQ(std::string(buf, 2), "07");
}

TEST(Put, Width2TwoDigits) {
    char buf[2];
    x::strings::put(buf, 42, 2);
    EXPECT_EQ(std::string(buf, 2), "42");
}

TEST(Put, Width2Max) {
    char buf[2];
    x::strings::put(buf, 99, 2);
    EXPECT_EQ(std::string(buf, 2), "99");
}

TEST(Put, Width4Zero) {
    char buf[4];
    x::strings::put(buf, 0, 4);
    EXPECT_EQ(std::string(buf, 4), "0000");
}

TEST(Put, Width4Year) {
    char buf[4];
    x::strings::put(buf, 2026, 4);
    EXPECT_EQ(std::string(buf, 4), "2026");
}

TEST(Put, Width4SmallValue) {
    char buf[4];
    x::strings::put(buf, 5, 4);
    EXPECT_EQ(std::string(buf, 4), "0005");
}

TEST(Put, Width4Max) {
    char buf[4];
    x::strings::put(buf, 9999, 4);
    EXPECT_EQ(std::string(buf, 4), "9999");
}

TEST(Put, Width9Zero) {
    char buf[9];
    x::strings::put(buf, 0, 9);
    EXPECT_EQ(std::string(buf, 9), "000000000");
}

TEST(Put, Width9One) {
    char buf[9];
    x::strings::put(buf, 1, 9);
    EXPECT_EQ(std::string(buf, 9), "000000001");
}

TEST(Put, Width9Nanoseconds) {
    char buf[9];
    x::strings::put(buf, 123456789, 9);
    EXPECT_EQ(std::string(buf, 9), "123456789");
}

TEST(Put, Width9Max) {
    char buf[9];
    x::strings::put(buf, 999999999, 9);
    EXPECT_EQ(std::string(buf, 9), "999999999");
}

TEST(Put, Width1) {
    char buf[1];
    x::strings::put(buf, 5, 1);
    EXPECT_EQ(buf[0], '5');
}
