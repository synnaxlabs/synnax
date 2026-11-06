// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <random>
#include <string>
#include <variant>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::arc {
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): one seed per suite,
// printed so that a failing run can be replayed.
std::mt19937 gen_rand = random_generator("Arc Tests");
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

std::string random_arc_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + "_" + std::to_string(dis(gen_rand));
}

/// @brief it should create an Arc program and assign it a non-zero key.
TEST(TestArc, testCreate) {
    const auto client = new_test_client();
    const auto name = random_arc_name("test_arc");
    Arc arc = AutomationArc{{.name = name}};
    std::get<AutomationArc>(arc).source = TextSource{
        .text = {.raw = "// Simple Arc program"}
    };

    ASSERT_NIL(client.arcs.create(arc));
    ASSERT_EQ(base(arc).name, name);
    ASSERT_NE(base(arc).key, x::uuid::NIL);
}

/// @brief it should create an Arc program using the convenience method.
TEST(TestArc, testCreateConvenience) {
    const auto client = new_test_client();
    const auto name = random_arc_name("convenience_arc");
    auto arc = ASSERT_NIL_P(client.arcs.create(name));
    ASSERT_EQ(base(arc).name, name);
    ASSERT_NE(base(arc).key, x::uuid::NIL);
}

/// @brief it should create multiple Arc programs.
TEST(TestArc, testCreateMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        AutomationArc{{.name = random_arc_name("arc1")}},
        AutomationArc{{.name = random_arc_name("arc2")}},
        AutomationArc{{.name = random_arc_name("arc3")}},
    };

    ASSERT_NIL(client.arcs.create(arcs));

    for (const auto &arc: arcs)
        ASSERT_NE(base(arc).key, x::uuid::NIL);
    ASSERT_TRUE(base(arcs[0]).name.starts_with("arc1_"));
    ASSERT_TRUE(base(arcs[1]).name.starts_with("arc2_"));
    ASSERT_TRUE(base(arcs[2]).name.starts_with("arc3_"));
}

/// @brief it should create a module and retrieve it as a module.
TEST(TestArc, testModule) {
    const auto client = new_test_client();
    Arc arc = ModuleArc{{.name = random_arc_name("module_test")}};
    ASSERT_NIL(client.arcs.create(arc));
    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_key(
        base(arc).key,
        RetrieveOptions{.include_program = true}
    ));
    ASSERT_TRUE(std::holds_alternative<ModuleArc>(retrieved));
}

/// @brief it should retrieve an Arc program by name.
TEST(TestArc, testRetrieveByName) {
    const auto client = new_test_client();
    auto name = random_arc_name("retrieve_test");
    Arc created = AutomationArc{{.name = name}};
    ASSERT_NIL(client.arcs.create(created));

    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_name(name));
    ASSERT_EQ(base(retrieved).key, base(created).key);
    ASSERT_EQ(base(retrieved).name, name);
}

/// @brief it should retrieve an Arc program by key.
TEST(TestArc, testRetrieveByKey) {
    const auto client = new_test_client();
    const auto name = random_arc_name("key_test");
    Arc created = AutomationArc{{.name = name}};
    ASSERT_NIL(client.arcs.create(created));

    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_key(base(created).key));
    ASSERT_EQ(base(retrieved).key, base(created).key);
    ASSERT_EQ(base(retrieved).name, name);
}

/// @brief it should retrieve multiple Arc programs by names.
TEST(TestArc, testRetrieveMany) {
    const auto client = new_test_client();
    auto name1 = random_arc_name("multi1");
    auto name2 = random_arc_name("multi2");
    auto arcs = std::vector<Arc>{
        AutomationArc{{.name = name1}},
        AutomationArc{{.name = name2}}
    };
    ASSERT_NIL(client.arcs.create(arcs));

    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve({name1, name2}));
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should retrieve multiple Arc programs by keys.
TEST(TestArc, testRetrieveByKeys) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        AutomationArc{{.name = random_arc_name("keys1")}},
        AutomationArc{{.name = random_arc_name("keys2")}},
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<x::uuid::UUID> keys = {base(arcs[0]).key, base(arcs[1]).key};
    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_keys(keys));
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should retrieve the Arc programs that match a filter tree.
TEST(TestArc, testRetrieveWhere) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        AutomationArc{{.name = random_arc_name("where_a")}},
        AutomationArc{{.name = random_arc_name("where_b")}},
    };
    ASSERT_NIL(client.arcs.create(arcs));
    const auto retrieved = ASSERT_NIL_P(client.arcs.retrieve(
        RetrieveRequest{
            .where = Where{
                .not_ = x::mem::indirect<Where>(
                    Where{.key = x::filter::Set<Key>{.eq = base(arcs[0]).key}}
                ),
                .key = x::filter::Set<Key>{
                    .in = std::vector<Key>{base(arcs[0]).key, base(arcs[1]).key},
                },
            },
        }
    ));
    ASSERT_EQ(retrieved.size(), 1);
    ASSERT_EQ(base(retrieved[0]).key, base(arcs[1]).key);
}

/// @brief it should delete an Arc program by key.
TEST(TestArc, testDelete) {
    const auto client = new_test_client();
    Arc arc = AutomationArc{{.name = random_arc_name("delete_test")}};
    ASSERT_NIL(client.arcs.create(arc));

    ASSERT_NIL(client.arcs.del(base(arc).key));

    ASSERT_OCCURRED_AS_P(
        client.arcs.retrieve_by_key(base(arc).key),
        x::errors::NOT_FOUND
    );
}

/// @brief it should delete multiple Arc programs by keys.
TEST(TestArc, testDeleteMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<Arc>{
        AutomationArc{{.name = random_arc_name("delete1")}},
        AutomationArc{{.name = random_arc_name("delete2")}},
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<x::uuid::UUID> keys = {base(arcs[0]).key, base(arcs[1]).key};
    ASSERT_NIL(client.arcs.del(keys));

    ASSERT_TRUE(ASSERT_NIL_P(client.arcs.retrieve_by_keys(keys)).empty());
}

/// @brief it should handle the program field correctly.
TEST(TestArc, testProgramField) {
    const auto client = new_test_client();
    Arc arc = AutomationArc{{.name = random_arc_name("module_test")}};
    std::get<AutomationArc>(arc).source = TextSource{
        .text = {.raw = "// Test program"}
    };

    ASSERT_NIL(client.arcs.create(arc));

    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_key(base(arc).key));
    ASSERT_EQ(base(retrieved).key, base(arc).key);
    ASSERT_FALSE(std::get<AutomationArc>(retrieved).program.has_value());
}

/// @brief it should compile an Arc program when retrieved with compile=true.
/// This test mirrors the Go test in arc/go/go_test.go that verifies calc.arc compiles.
TEST(TestArc, testRetrieveWithCompile) {
    const auto client = new_test_client();

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

    Arc arc = AutomationArc{{.name = random_arc_name("compile_test")}};
    std::string calc_arc_text = R"(
func calc() {val f32} f32 {
    return val * 2
}

)" + ox_pt_1.name + " -> calc() -> " +
                                ox_pt_doubled.name + R"(
)";
    std::get<AutomationArc>(arc).source = TextSource{.text = {.raw = calc_arc_text}};

    ASSERT_NIL(client.arcs.create(arc));

    RetrieveOptions options;
    options.include_program = true;
    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_key(base(arc).key, options));
    ASSERT_EQ(base(retrieved).key, base(arc).key);
    const auto &program = std::get<AutomationArc>(retrieved).program.value();

    // Verify the module was compiled - should have WASM bytes
    ASSERT_FALSE(program.wasm.empty())
        << "Expected WASM bytecode to be present after compilation";

    // Verify correct node structure (same as Go test expectations)
    // 3 nodes: source (on), calc function, sink (write)
    ASSERT_EQ(program.nodes.size(), 3) << "Expected 3 nodes: source, calc, sink";

    // First node: source channel (on)
    ASSERT_EQ(program.nodes[0].type, "on");
    ASSERT_GT(program.nodes[0].channels.read.count(ox_pt_1.key), 0)
        << "First node should read from ox_pt_1 channel";
    ASSERT_EQ(program.nodes[0].outputs.size(), 1);

    // Second node: calc function
    ASSERT_EQ(program.nodes[1].type, "calc");

    // Third node: sink channel (write)
    ASSERT_EQ(program.nodes[2].type, "write");
    ASSERT_GT(program.nodes[2].channels.write.count(ox_pt_doubled.key), 0)
        << "Third node should write to ox_pt_doubled channel";
    // write node has two inputs: the value input and the channel param.
    ASSERT_EQ(program.nodes[2].inputs.size(), 2);

    // Verify edges (2 edges connecting the 3 nodes)
    ASSERT_EQ(program.edges.size(), 2) << "Expected 2 edges connecting the nodes";
}

/// @brief it should compile an Arc program with an interval node in a sequence.
TEST(TestArc, testIntervalNodeCompiles) {
    const auto client = new_test_client();

    Arc arc = AutomationArc{{.name = random_arc_name("interval_test")}};
    std::get<AutomationArc>(arc).source = TextSource{.text = {.raw = R"(
sequence main {
    stage initial {
        interval(period=5s) => next
    }
    stage end {
    }
}
)"}};

    ASSERT_NIL(client.arcs.create(arc));

    auto retrieved = ASSERT_NIL_P(client.arcs.retrieve_by_key(
        base(arc).key,
        synnax::arc::RetrieveOptions{
            .include_program = true,
        }
    ));
    const auto &program = std::get<AutomationArc>(retrieved).program.value();
    ASSERT_FALSE(program.wasm.empty());

    bool found_interval = false;
    for (const auto &node: program.nodes) {
        if (node.type == "interval") {
            found_interval = true;
            bool found_period = false;
            for (const auto &param: node.inputs)
                if (param.name == "period") found_period = true;
            ASSERT_TRUE(found_period);
            break;
        }
    }
    ASSERT_TRUE(found_interval);

    // The compiled scope tree should contain a sequential "main" scope
    // with two member stages ("initial" and "end"). After the IR-refactor
    // that replaced flat sequences/stages with the composable Scope tree,
    // sequences and stages are no longer top-level struct fields; they
    // live as nested Scopes under program->root.
    const ::arc::ir::Scope *main_scope = nullptr;
    std::function<void(const ::arc::ir::Scope &)> find_main;
    find_main = [&](const ::arc::ir::Scope &scope) {
        if (main_scope != nullptr) return;
        if (scope.key == "main" && scope.mode == ::arc::ir::ScopeMode::Sequential) {
            main_scope = &scope;
            return;
        }
        for (const auto &stratum: scope.strata)
            for (const auto &m: stratum)
                if (m.scope) find_main(*m.scope);
        for (const auto &m: scope.steps)
            if (m.scope) find_main(*m.scope);
    };
    find_main(program.root);
    ASSERT_NE(main_scope, nullptr);
    ASSERT_EQ(main_scope->steps.size(), 2);
}
}
