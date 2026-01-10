// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <string>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/test/test.h"

namespace synnax::arc {
std::mt19937 gen_rand = random_generator(std::move("Arc Tests"));

std::string random_arc_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + "_" + std::to_string(dis(gen_rand));
}

/// @brief it should create an Arc program and assign it a non-zero key.
TEST(TestArc, testCreate) {
    const auto client = new_test_client();
    auto arc = Arc("test_arc");
    arc.text.raw = "// Simple Arc program";

    ASSERT_NIL(client.arcs.create(arc));
    ASSERT_EQ(arc.name, "test_arc");
    ASSERT_FALSE(arc.key.empty());
}

/// @brief it should create an Arc program using the convenience method.
TEST(TestArc, testCreateConvenience) {
    const auto client = new_test_client();
    auto [arc, err] = client.arcs.create("convenience_arc");

    ASSERT_NIL(err);
    ASSERT_EQ(arc.name, "convenience_arc");
    ASSERT_FALSE(arc.key.empty());
}

/// @brief it should create multiple Arc programs.
TEST(TestArc, testCreateMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        Arc("arc1"),
        Arc("arc2"),
        Arc("arc3"),
    };

    ASSERT_NIL(client.arcs.create(arcs));

    for (const auto &arc: arcs) {
        ASSERT_FALSE(arc.key.empty());
    }
    ASSERT_EQ(arcs[0].name, "arc1");
    ASSERT_EQ(arcs[1].name, "arc2");
    ASSERT_EQ(arcs[2].name, "arc3");
}

/// @brief it should retrieve an Arc program by name.
TEST(TestArc, testRetrieveByName) {
    const auto client = new_test_client();
    auto name = random_arc_name("retrieve_test");
    auto created = Arc(name);
    ASSERT_NIL(client.arcs.create(created));

    auto [retrieved, err] = client.arcs.retrieve_by_name(name);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, created.key);
    ASSERT_EQ(retrieved.name, name);
}

/// @brief it should retrieve an Arc program by key.
TEST(TestArc, testRetrieveByKey) {
    const auto client = new_test_client();
    auto created = Arc("key_test");
    ASSERT_NIL(client.arcs.create(created));

    auto [retrieved, err] = client.arcs.retrieve_by_key(created.key);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, created.key);
    ASSERT_EQ(retrieved.name, "key_test");
}

/// @brief it should retrieve multiple Arc programs by names.
TEST(TestArc, testRetrieveMany) {
    const auto client = new_test_client();
    auto name1 = random_arc_name("multi1");
    auto name2 = random_arc_name("multi2");
    auto arcs = std::vector<Arc>{
        Arc(name1),
        Arc(name2),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    auto [retrieved, err] = client.arcs.retrieve({name1, name2});

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should retrieve multiple Arc programs by keys.
TEST(TestArc, testRetrieveByKeys) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        Arc("keys1"),
        Arc("keys2"),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<std::string> keys = {arcs[0].key, arcs[1].key};
    auto [retrieved, err] = client.arcs.retrieve_by_keys(keys);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should delete an Arc program by key.
TEST(TestArc, testDelete) {
    const auto client = new_test_client();
    auto arc = Arc("delete_test");
    ASSERT_NIL(client.arcs.create(arc));

    ASSERT_NIL(client.arcs.delete_arc(arc.key));

    // Verify it's deleted
    auto [_, err] = client.arcs.retrieve_by_key(arc.key);
    ASSERT_FALSE(err.ok());
}

/// @brief it should delete multiple Arc programs by keys.
TEST(TestArc, testDeleteMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        Arc("delete1"),
        Arc("delete2"),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<std::string> keys = {arcs[0].key, arcs[1].key};
    ASSERT_NIL(client.arcs.delete_arc(keys));

    // Verify they're deleted by trying to retrieve - should fail
    auto [retrieved, err] = client.arcs.retrieve_by_keys(keys);
    // Server returns error when arcs don't exist
    ASSERT_FALSE(err.ok());
}

/// @brief it should handle the module field correctly.
TEST(TestArc, testModuleField) {
    const auto client = new_test_client();
    auto arc = Arc("module_test");
    arc.text.raw = "// Test program";

    ASSERT_NIL(client.arcs.create(arc));

    auto [retrieved, err] = client.arcs.retrieve_by_key(arc.key);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, arc.key);
    // Module field exists but WASM is empty when not compiled
    ASSERT_TRUE(retrieved.module.wasm.empty());
}

/// @brief it should compile an Arc program when retrieved with compile=true.
/// This test mirrors the Go test in arc/go/go_test.go that verifies calc.arc compiles.
TEST(TestArc, testRetrieveWithCompile) {
    const auto client = new_test_client();

    // Create the channels referenced in calc.arc
    auto ox_pt_1 = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("ox_pt_1"),
        x::telem::FLOAT32_T,
        true
    ));
    auto ox_pt_doubled = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("ox_pt_doubled"),
        x::telem::FLOAT32_T,
        true
    ));

    // Create the Arc with calc.arc content
    // This matches arc/go/testdata/calc.arc
    auto arc = Arc(random_arc_name("compile_test"));
    std::string calc_arc_text = R"(
func calc(val f32) f32 {
    return val * 2
}

)" + ox_pt_1.name + " -> calc{} -> " +
                                ox_pt_doubled.name + R"(
)";
    arc.text.raw = calc_arc_text;

    ASSERT_NIL(client.arcs.create(arc));

    // Retrieve with compile=true
    RetrieveOptions options;
    options.compile = true;
    auto [retrieved, err] = client.arcs.retrieve_by_key(arc.key, options);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, arc.key);

    // Verify the module was compiled - should have WASM bytes
    ASSERT_FALSE(retrieved.module.wasm.empty())
        << "Expected WASM bytecode to be present after compilation";

    // Verify correct node structure (same as Go test expectations)
    // 3 nodes: source (on), calc function, sink (write)
    ASSERT_EQ(retrieved.module.nodes.size(), 3)
        << "Expected 3 nodes: source, calc, sink";

    // First node: source channel (on)
    ASSERT_EQ(retrieved.module.nodes[0].type, "on");
    ASSERT_GT(retrieved.module.nodes[0].channels.read.count(ox_pt_1.key), 0)
        << "First node should read from ox_pt_1 channel";
    ASSERT_EQ(retrieved.module.nodes[0].outputs.size(), 1);

    // Second node: calc function
    ASSERT_EQ(retrieved.module.nodes[1].type, "calc");

    // Third node: sink channel (write)
    ASSERT_EQ(retrieved.module.nodes[2].type, "write");
    ASSERT_GT(retrieved.module.nodes[2].channels.write.count(ox_pt_doubled.key), 0)
        << "Third node should write to ox_pt_doubled channel";
    ASSERT_EQ(retrieved.module.nodes[2].inputs.size(), 1);

    // Verify edges (2 edges connecting the 3 nodes)
    ASSERT_EQ(retrieved.module.edges.size(), 2)
        << "Expected 2 edges connecting the nodes";
}
}
