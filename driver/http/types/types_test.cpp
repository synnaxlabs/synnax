// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/http/types/types.h"

using namespace driver::http;

TEST(ParseMethodTest, ParsesGET) {
    x::json::Parser p(x::json::json{{"method", "GET"}});
    EXPECT_EQ(parse_method(p, "method"), Method::GET);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesPOST) {
    x::json::Parser p(x::json::json{{"method", "POST"}});
    EXPECT_EQ(parse_method(p, "method"), Method::POST);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesPUT) {
    x::json::Parser p(x::json::json{{"method", "PUT"}});
    EXPECT_EQ(parse_method(p, "method"), Method::PUT);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesDELETE) {
    x::json::Parser p(x::json::json{{"method", "DELETE"}});
    EXPECT_EQ(parse_method(p, "method"), Method::DEL);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesPATCH) {
    x::json::Parser p(x::json::json{{"method", "PATCH"}});
    EXPECT_EQ(parse_method(p, "method"), Method::PATCH);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesHEAD) {
    x::json::Parser p(x::json::json{{"method", "HEAD"}});
    EXPECT_EQ(parse_method(p, "method"), Method::HEAD);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesOPTIONS) {
    x::json::Parser p(x::json::json{{"method", "OPTIONS"}});
    EXPECT_EQ(parse_method(p, "method"), Method::OPTIONS);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesTRACE) {
    x::json::Parser p(x::json::json{{"method", "TRACE"}});
    EXPECT_EQ(parse_method(p, "method"), Method::TRACE);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, ParsesCONNECT) {
    x::json::Parser p(x::json::json{{"method", "CONNECT"}});
    EXPECT_EQ(parse_method(p, "method"), Method::CONNECT);
    EXPECT_TRUE(p.ok());
}

TEST(ParseMethodTest, UnknownMethodErrors) {
    x::json::Parser p(x::json::json{{"method", "INVALID"}});
    parse_method(p, "method");
    EXPECT_FALSE(p.ok());
}

TEST(ParseMethodTest, MissingFieldErrors) {
    x::json::Parser p(x::json::json::object());
    parse_method(p, "method");
    EXPECT_FALSE(p.ok());
}
