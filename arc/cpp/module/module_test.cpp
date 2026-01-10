// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "arc/cpp/module/module.h"

/// @brief it should correctly round-trip Module through protobuf
TEST(ModuleTest, testModuleProtobufRoundTrip) {
    arc::module::Module original;

    // Add some IR data
    arc::ir::Node node;
    node.key = "test_node";
    node.type = "multiply";
    original.nodes.push_back(node);

    // Add WASM bytecode
    original.wasm = {0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00};

    // Add output memory bases
    original.output_memory_bases["output1"] = 1024;
    original.output_memory_bases["output2"] = 2048;
}
