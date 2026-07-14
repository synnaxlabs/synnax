// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>

#include "gtest/gtest.h"

#include "x/cpp/hash/xxhash.h"
#include "x/cpp/json/canonical.h"

using x::json::canonical;
using x::json::json;

static std::string must_canonical(const json &j) {
    auto [out, err] = canonical(j);
    EXPECT_FALSE(err) << err;
    return out;
}

/// @brief it should format numbers exactly like JSON.stringify.
TEST(CanonicalJSON, testES6Numbers) {
    EXPECT_EQ(must_canonical(1.0), "1");
    EXPECT_EQ(must_canonical(-0.0), "0");
    EXPECT_EQ(must_canonical(0.1), "0.1");
    EXPECT_EQ(must_canonical(2.5), "2.5");
    EXPECT_EQ(must_canonical(0.001), "0.001");
    EXPECT_EQ(must_canonical(0.000001), "0.000001");
    EXPECT_EQ(must_canonical(1e-7), "1e-7");
    EXPECT_EQ(must_canonical(1e20), "100000000000000000000");
    EXPECT_EQ(must_canonical(1e21), "1e+21");
    EXPECT_EQ(must_canonical(1.5e300), "1.5e+300");
    EXPECT_EQ(must_canonical(5e-324), "5e-324");
    EXPECT_EQ(must_canonical(123456789012345680000.0), "123456789012345680000");
    EXPECT_EQ(must_canonical(-42.75), "-42.75");
    EXPECT_EQ(must_canonical(50.0), "50");
    EXPECT_EQ(must_canonical(int64_t(50000)), "50000");
}

/// @brief it should sort object keys and escape strings like the Go and
/// TypeScript implementations.
TEST(CanonicalJSON, testObjectsAndStrings) {
    EXPECT_EQ(must_canonical(json::object()), "{}");
    EXPECT_EQ(must_canonical(json::array()), "[]");
    EXPECT_EQ(must_canonical(nullptr), "null");
    EXPECT_EQ(
        must_canonical(json{{"b", 2.5}, {"a", 1}, {"g", json{{"z", 1}, {"a", "x"}}}}),
        R"({"a":1,"b":2.5,"g":{"a":"x","z":1}})"
    );
    EXPECT_EQ(
        must_canonical(json{{"name", "ch\"1\""}, {"notes", "héllo⚡ <&> \n\ttab"}}),
        "{\"name\":\"ch\\\"1\\\"\",\"notes\":\"héllo⚡ <&> \\n\\ttab\"}"
    );
    EXPECT_EQ(must_canonical(std::string("\x01\x1f")), "\"\\u0001\\u001f\"");
}

/// @brief it should produce the shared cross-language golden hashes.
TEST(CanonicalJSON, testCrossLanguageGoldenHashes) {
    EXPECT_EQ(
        x::hash::xxhash64_hex(must_canonical(json::object())),
        "2e1472b57af294d1"
    );
    EXPECT_EQ(
        x::hash::xxhash64_hex(
            must_canonical(json{{"rate", 50.0}, {"port", 8080}, {"host", "localhost"}})
        ),
        "2de66015b3bdded8"
    );
    EXPECT_EQ(
        x::hash::xxhash64_hex(must_canonical(
            json{
                {"enabled", true},
                {"channels",
                 json::array(
                     {json{{"key", 12}, {"name", "ch\"1\""}, {"scale", 0.001}}}
                 )},
                {"notes", "héllo⚡"}
            }
        )),
        "811ef1fc462a59f2"
    );
}

/// @brief it should reject values with no JSON representation.
TEST(CanonicalJSON, testRejectsNonJSONValues) {
    auto [out, err] = canonical(std::numeric_limits<double>::quiet_NaN());
    EXPECT_TRUE(err);
    auto [out2, err2] = canonical(std::numeric_limits<double>::infinity());
    EXPECT_TRUE(err2);
}
