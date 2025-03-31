// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"

TEST(URLTest, DefaultConstruction) {
    freighter::URL url;
    EXPECT_TRUE(url.ip.empty());
    EXPECT_EQ(url.port, 0);
    EXPECT_TRUE(url.path.empty());
}

TEST(URLTest, ExplicitConstruction) {
    freighter::URL url("127.0.0.1", 8080, "/api/v1");
    EXPECT_EQ(url.ip, "127.0.0.1");
    EXPECT_EQ(url.port, 8080);
    EXPECT_EQ(url.path, "/api/v1/");  // Note: join_paths adds trailing slash
}

TEST(URLTest, StringParsing) {
    freighter::URL url("localhost:8080/api/v1");
    EXPECT_EQ(url.ip, "localhost");
    EXPECT_EQ(url.port, 8080);
    EXPECT_EQ(url.path, "/api/v1/");

    freighter::URL simple("127.0.0.1:8080");
    EXPECT_EQ(simple.ip, "127.0.0.1");
    EXPECT_EQ(simple.port, 8080);
    EXPECT_TRUE(simple.path.empty());
}

TEST(URLTest, EmptyStringParsing) {
    freighter::URL url("");
    EXPECT_TRUE(url.ip.empty());
    EXPECT_EQ(url.port, 0);
    EXPECT_TRUE(url.path.empty());
}

TEST(URLTest, ChildURLs) {
    freighter::URL parent("api.example.com", 443, "/v1");

    // Test adding child path
    auto child1 = parent.child("users");
    EXPECT_EQ(child1.ip, "api.example.com");
    EXPECT_EQ(child1.port, 443);
    EXPECT_EQ(child1.path, "/v1/users/");

    // Test adding child with leading slash
    auto child2 = parent.child("/posts");
    EXPECT_EQ(child2.path, "/v1/posts/");

    // Test adding child to empty parent path
    freighter::URL parent2("api.example.com", 443, "");
    auto child3 = parent2.child("users");
    EXPECT_EQ(child3.path, "/users/");

    // Test adding empty child path
    auto child4 = parent.child("");
    EXPECT_EQ(child4.path, "/v1/");
}

TEST(URLTest, ToString) {
    freighter::URL url("example.com", 8080, "/api/v1");
    EXPECT_EQ(url.to_string(), "example.com:8080/api/v1/");

    freighter::URL simple("localhost", 80, "");
    EXPECT_EQ(simple.to_string(), "localhost:80/");
}

TEST(URLTest, HostAddress) {
    freighter::URL url("example.com", 8080, "/api/v1");
    EXPECT_EQ(url.host_address(), "example.com:8080");
}

TEST(URLTest, PathNormalization) {
    // Test that paths are properly normalized with slashes
    freighter::URL url1("localhost", 8080, "api/v1");
    EXPECT_EQ(url1.path, "/api/v1/");

    freighter::URL url2("localhost", 8080, "/api/v1/");
    EXPECT_EQ(url2.path, "/api/v1/");

    freighter::URL url3("localhost", 8080, "/api/v1");
    EXPECT_EQ(url3.path, "/api/v1/");
}

TEST(URLTest, EmptyAndInvalidStringConstruction) {
    freighter::URL empty("");
    EXPECT_TRUE(empty.ip.empty());
    EXPECT_EQ(empty.port, 0);
    EXPECT_TRUE(empty.path.empty());

    freighter::URL no_port("localhost");
    EXPECT_EQ(no_port.ip, "localhost");
    EXPECT_EQ(no_port.port, 0);
    EXPECT_TRUE(no_port.path.empty());

    freighter::URL invalid_port("localhost:abc");
    EXPECT_EQ(invalid_port.ip, "localhost");
    EXPECT_EQ(invalid_port.port, 0);
    EXPECT_TRUE(invalid_port.path.empty());
}
