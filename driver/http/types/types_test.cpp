// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

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

/// @brief it should parse QUERY.
TEST(ParseMethodTest, ParsesQUERY) {
    x::json::Parser p(x::json::json{{"method", "QUERY"}});
    EXPECT_EQ(parse_method(p, "method"), Method::QUERY_METHOD);
    EXPECT_TRUE(p.ok());
}

/// @brief it should refuse TRACE and CONNECT as validation errors.
TEST(ParseMethodTest, RejectsTRACEAndCONNECT) {
    ASSERT_OCCURRED_AS_P(parse_method("TRACE"), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(parse_method("CONNECT"), x::errors::VALIDATION);
}

TEST(ParseMethodTest, UnknownMethodErrors) {
    x::json::Parser p(x::json::json{{"method", "INVALID"}});
    // should default to GET
    EXPECT_EQ(parse_method(p, "method"), Method::GET);
    EXPECT_FALSE(p.ok());
}

TEST(ParseMethodTest, MissingFieldErrors) {
    x::json::Parser p(x::json::json::object());
    // should default to GET
    EXPECT_EQ(parse_method(p, "method"), Method::GET);
    EXPECT_FALSE(p.ok());
}

TEST(ToStringTest, GET) {
    EXPECT_STREQ(to_string(Method::GET), "GET");
}

TEST(ToStringTest, HEAD) {
    EXPECT_STREQ(to_string(Method::HEAD), "HEAD");
}

TEST(ToStringTest, POST) {
    EXPECT_STREQ(to_string(Method::POST), "POST");
}

TEST(ToStringTest, PUT) {
    EXPECT_STREQ(to_string(Method::PUT), "PUT");
}

TEST(ToStringTest, DEL) {
    EXPECT_STREQ(to_string(Method::DEL), "DELETE");
}

TEST(ToStringTest, PATCH) {
    EXPECT_STREQ(to_string(Method::PATCH), "PATCH");
}

TEST(ToStringTest, OPTIONS) {
    EXPECT_STREQ(to_string(Method::OPTIONS), "OPTIONS");
}

/// @brief it should return QUERY for QUERY_METHOD.
TEST(ToStringTest, QUERY) {
    EXPECT_STREQ(to_string(Method::QUERY_METHOD), "QUERY");
}

TEST(ToStringTest, InvalidMethodThrows) {
    EXPECT_THROW(to_string(static_cast<Method>(999)), std::invalid_argument);
}

TEST(HasRequestBodyTest, POST) {
    EXPECT_TRUE(has_request_body(Method::POST));
}

TEST(HasRequestBodyTest, PUT) {
    EXPECT_TRUE(has_request_body(Method::PUT));
}

TEST(HasRequestBodyTest, PATCH) {
    EXPECT_TRUE(has_request_body(Method::PATCH));
}

TEST(HasRequestBodyTest, OPTIONS) {
    EXPECT_TRUE(has_request_body(Method::OPTIONS));
}

TEST(HasRequestBodyTest, GET) {
    EXPECT_FALSE(has_request_body(Method::GET));
}

TEST(HasRequestBodyTest, HEAD) {
    EXPECT_FALSE(has_request_body(Method::HEAD));
}

TEST(HasRequestBodyTest, DEL) {
    EXPECT_FALSE(has_request_body(Method::DEL));
}

/// @brief it should send a body with QUERY.
TEST(HasRequestBodyTest, QUERY) {
    EXPECT_TRUE(has_request_body(Method::QUERY_METHOD));
}

TEST(HasResponseBodyTest, GET) {
    EXPECT_TRUE(has_response_body(Method::GET));
}

TEST(HasResponseBodyTest, POST) {
    EXPECT_TRUE(has_response_body(Method::POST));
}

TEST(HasResponseBodyTest, PUT) {
    EXPECT_TRUE(has_response_body(Method::PUT));
}

TEST(HasResponseBodyTest, DEL) {
    EXPECT_TRUE(has_response_body(Method::DEL));
}

TEST(HasResponseBodyTest, PATCH) {
    EXPECT_TRUE(has_response_body(Method::PATCH));
}

TEST(HasResponseBodyTest, OPTIONS) {
    EXPECT_TRUE(has_response_body(Method::OPTIONS));
}

/// @brief it should read a response body for QUERY.
TEST(HasResponseBodyTest, QUERY) {
    EXPECT_TRUE(has_response_body(Method::QUERY_METHOD));
}

TEST(HasResponseBodyTest, HEAD) {
    EXPECT_FALSE(has_response_body(Method::HEAD));
}

/// @brief it should escape line breaks in a short body and keep the rest.
TEST(BodyExcerptTest, EscapesLineBreaks) {
    const Response resp{.status_code = 500, .body = "a\nb\r\nc"};
    EXPECT_EQ(resp.body_excerpt(), "a\\nb\\r\\nc");
}

/// @brief it should cut a long body at the size limit and end it in "...".
TEST(BodyExcerptTest, CutsLongBody) {
    const Response resp{.status_code = 500, .body = std::string(600, 'a')};
    EXPECT_EQ(resp.body_excerpt(), std::string(MAX_BODY_EXCERPT_SIZE, 'a') + "...");
}

/// @brief it should cut before a multi-byte character that crosses the size limit.
TEST(BodyExcerptTest, CutsAtCharacterBoundary) {
    const Response resp{
        .status_code = 500,
        .body = std::string(MAX_BODY_EXCERPT_SIZE - 1, 'a') + "\xC3\xA9" + "tail",
    };
    EXPECT_EQ(resp.body_excerpt(), std::string(MAX_BODY_EXCERPT_SIZE - 1, 'a') + "...");
}
