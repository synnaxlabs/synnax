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

#include "x/cpp/hash/xxhash.h"

/// @brief it should match the xxHash specification reference vector for empty
/// input.
TEST(XXHash64, testEmptyInput) {
    EXPECT_EQ(x::hash::xxhash64_hex(""), "ef46db3751d8e999");
}

/// @brief it should hash every input length class: stripe, 8-byte, 4-byte, and
/// tail paths.
TEST(XXHash64, testAllLengthClasses) {
    const std::string long_input = "abcdefghijklmnopqrstuvwxyz0123456789"
                                   "ABCDEFGHIJKLMNOPQ";
    for (size_t i = 0; i < long_input.size(); ++i)
        EXPECT_EQ(x::hash::xxhash64_hex(long_input.substr(0, i)).size(), 16);
}

/// @brief it should match the Go and TypeScript golden vectors for canonical
/// config strings.
TEST(XXHash64, testCrossLanguageGoldenVectors) {
    EXPECT_EQ(x::hash::xxhash64_hex("{}"), "2e1472b57af294d1");
    EXPECT_EQ(
        x::hash::xxhash64_hex(R"({"host":"localhost","port":8080,"rate":50})"),
        "2de66015b3bdded8"
    );
}
