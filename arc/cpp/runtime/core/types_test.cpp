// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <unordered_map>

#include "gtest/gtest.h"

#include "arc/cpp/runtime/core/types.h"

// Test Handle equality
TEST(TypesTest, HandleEquality) {
    arc::ir::Handle h1{"node1", "out"};
    arc::ir::Handle h2{"node1", "out"};
    arc::ir::Handle h3{"node2", "out"};
    arc::ir::Handle h4{"node1", "in"};

    EXPECT_EQ(h1, h2);
    EXPECT_NE(h1, h3);
    EXPECT_NE(h1, h4);
}

// Test Handle hashing for use in unordered_map
TEST(TypesTest, HandleHash) {
    std::unordered_map<arc::ir::Handle, int, arc::ir::Handle::Hasher> map;

    arc::ir::Handle h1{"node1", "out"};
    arc::ir::Handle h2{"node2", "in"};

    map[h1] = 42;
    map[h2] = 100;

    EXPECT_EQ(map[h1], 42);
    EXPECT_EQ(map[h2], 100);
    EXPECT_EQ(map.size(), 2);
}

// Test Edge equality
TEST(TypesTest, EdgeEquality) {
    arc::ir::Edge e1{arc::ir::Handle{"A", "out"}, arc::ir::Handle{"B", "in"}};
    arc::ir::Edge e2{arc::ir::Handle{"A", "out"}, arc::ir::Handle{"B", "in"}};
    arc::ir::Edge e3{arc::ir::Handle{"A", "out"}, arc::ir::Handle{"C", "in"}};

    EXPECT_EQ(e1, e2);
    EXPECT_NE(e1, e3);
}

// Test ValuePair construction
TEST(TypesTest, ValuePair) {
    auto data = std::make_shared<telem::Series>(std::vector<float>{1.0f, 2.0f, 3.0f});
    auto time = std::make_shared<telem::Series>(std::vector<int64_t>{100, 200, 300});

    arc::ValuePair vp{data, time};

    EXPECT_NE(vp.data, nullptr);
    EXPECT_NE(vp.time, nullptr);
    EXPECT_EQ(vp.data->size(), 3);
    EXPECT_EQ(vp.time->size(), 3);
}

// Test NodeMetadata construction
TEST(TypesTest, NodeMetadata) {
    arc::NodeMetadata meta{"test_node"};

    meta.type = "binary_op";
    meta.input_params = {"left", "right"};
    meta.output_params = {"result"};
    meta.read_channels = {1, 2};
    meta.write_channels = {3};

    EXPECT_EQ(meta.key, "test_node");
    EXPECT_EQ(meta.type, "binary_op");
    EXPECT_EQ(meta.input_params.size(), 2);
    EXPECT_EQ(meta.output_params.size(), 1);
    EXPECT_EQ(meta.read_channels.size(), 2);
    EXPECT_EQ(meta.write_channels.size(), 1);
}
