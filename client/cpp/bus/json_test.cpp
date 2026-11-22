// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/bus/json.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::bus {
/// @brief it should read and write a length size as its schema value.
TEST(Framing, LengthSizeRoundTrip) {
    const x::json::Parser parser(
        x::json::json{{"type", "sync"}, {"sync", "qg=="}, {"length_size", 4}}
    );
    const auto framing = parse_framing(parser);
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(std::holds_alternative<SyncFraming>(framing));
    const auto &sync = std::get<SyncFraming>(framing);
    EXPECT_EQ(sync.sync, std::vector<std::uint8_t>{0xAA});
    EXPECT_EQ(sync.length_size, LengthSize::Four);
    EXPECT_EQ(sync.to_json()["length_size"], 4);
}
}
