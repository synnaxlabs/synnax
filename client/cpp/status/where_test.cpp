// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/status/status.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::status {
namespace {
x::json::json parse_json(const char *encoded) {
    return x::json::json::parse(encoded);
}
}

/// @brief it should read plain values and a single label in a designated initializer.
TEST(WhereTest, testPlainValues) {
    const auto label = ASSERT_NIL_P(
        x::uuid::UUID::parse("6d61ca8e-d9b1-11f1-806d-fdce102162f0")
    );
    const Where w{.key = "a", .variant = {"error", "warning"}, .labels = label};
    const auto j = w.to_json();
    EXPECT_EQ(j["key"], parse_json(R"({"eq": "a"})"));
    EXPECT_EQ(j["variant"], parse_json(R"({"in": ["error", "warning"]})"));
    EXPECT_EQ(w.labels, std::vector{label});
}

/// @brief it should combine filters with || and !.
TEST(WhereTest, testOrNot) {
    const Where error{.variant = x::filter::Set<std::string>{.eq = "error"}};
    const Where a{.key = "a"};
    EXPECT_EQ((error || !a).to_json(), parse_json(R"({"or": [
        {"variant": {"eq": "error"}},
        {"not": {"key": {"eq": "a"}}}
    ]})"));
}

/// @brief it should combine filters with &&.
TEST(WhereTest, testAnd) {
    const Where w = Where{.key = {"a", "b"}} && Where{.variant = "error"};
    EXPECT_EQ(w.to_json(), parse_json(R"({"and": [
        {"key": {"in": ["a", "b"]}},
        {"variant": {"eq": "error"}}
    ]})"));
}

/// @brief it should keep every line through JSON and protobuf.
TEST(WhereTest, testRoundTrip) {
    const auto label = ASSERT_NIL_P(
        x::uuid::UUID::parse("7650083a-d9b1-11f1-80f2-8b8f25ffcaa3")
    );
    const Where w = Where{.key = "a", .labels = {label}} || !Where{.variant = "error"};
    const auto expected = w.to_json();
    EXPECT_EQ(Where::parse(x::json::Parser(expected)).to_json(), expected);
    const auto pb = ASSERT_NIL_P(w.to_proto());
    EXPECT_EQ(ASSERT_NIL_P(Where::from_proto(pb)).to_json(), expected);
}
}
