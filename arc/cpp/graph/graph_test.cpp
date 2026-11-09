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

#include "arc/cpp/graph/json.gen.h"
#include "arc/cpp/graph/types.gen.h"

namespace arc::graph {
/// @brief it should fall back to an empty graph when the object omits every field.
TEST(Graph, ParsesEmptyObject) {
    x::json::Parser parser(x::json::json::object());
    const auto graph = Graph::parse(parser);
    ASSERT_NIL(parser.error());
    EXPECT_TRUE(graph.functions.empty());
    EXPECT_TRUE(graph.edges.empty());
    EXPECT_TRUE(graph.nodes.empty());
    EXPECT_TRUE(graph.inputs.empty());
}

/// @brief it should bind one error for content that is not an object.
TEST(Graph, RejectsNonObjectContent) {
    for (const x::json::json &bad:
         {x::json::json(false), x::json::json("bad"), x::json::json::array()}) {
        x::json::Parser parser(x::json::json{{"graph", bad}});
        parser.field<Graph>("graph");
        ASSERT_EQ(parser.errors->size(), 1);
        EXPECT_EQ(parser.errors->at(0)["message"], "expected an object");
    }
}
}
