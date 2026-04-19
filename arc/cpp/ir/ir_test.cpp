// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"

namespace arc::ir {

/// @brief it should correctly round-trip Handle through protobuf
TEST(IRTest, testHandleProtobufRoundTrip) {
    const Handle original("node1", "param1");
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto reconstructed = ASSERT_NIL_P(Handle::from_proto(pb));
    ASSERT_EQ(reconstructed.node, "node1");
    ASSERT_EQ(reconstructed.param, "param1");
}

/// @brief it should correctly round-trip Edge through protobuf
TEST(IRTest, testEdgeProtobufRoundTrip) {
    Handle src("src_node", "output");
    Handle tgt("tgt_node", "input");
    Edge original(src, tgt);
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto reconstructed = ASSERT_NIL_P(Edge::from_proto(pb));
    ASSERT_EQ(reconstructed.source.node, "src_node");
    ASSERT_EQ(reconstructed.source.param, "output");
    ASSERT_EQ(reconstructed.target.node, "tgt_node");
    ASSERT_EQ(reconstructed.target.param, "input");
}

/// @brief it should correctly round-trip Channels through protobuf
TEST(IRTest, testChannelsProtobufRoundTrip) {
    types::Channels original;
    original.read[1] = "channel_a";
    original.read[2] = "channel_b";
    original.write[3] = "channel_c";
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto reconstructed = ASSERT_NIL_P(types::Channels::from_proto(pb));
    ASSERT_EQ(reconstructed.read.size(), 2);
    ASSERT_EQ(reconstructed.read.at(1), "channel_a");
    ASSERT_EQ(reconstructed.read.at(2), "channel_b");
    ASSERT_EQ(reconstructed.write.size(), 1);
    ASSERT_EQ(reconstructed.write.at(3), "channel_c");
}

/// @brief it should correctly round-trip Param through protobuf
TEST(IRTest, testParamProtobufRoundTrip) {
    types::Param original;
    original.name = "test_param";
    original.type.kind = types::Kind::F64;
    original.value = 42.5;
    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto reconstructed = ASSERT_NIL_P(types::Param::from_proto(pb));
    ASSERT_EQ(reconstructed.name, "test_param");
    ASSERT_EQ(reconstructed.type.kind, types::Kind::F64);
    ASSERT_DOUBLE_EQ(reconstructed.value.get<double>(), 42.5);
}

/// @brief it should round-trip a complete IR (including a sequential Scope
/// with transitions) through protobuf
TEST(IRTest, testIRProtobufRoundTrip) {
    IR original;

    Function fn;
    fn.key = "test_func";
    fn.channels.read[1] = "chan1";
    original.functions.push_back(fn);

    Node node;
    node.key = "init";
    node.type = "add";
    original.nodes.push_back(node);
    Node node2;
    node2.key = "run";
    node2.type = "add";
    original.nodes.push_back(node2);

    Edge edge(Handle("init", "out"), Handle("run", "in"));
    original.edges.push_back(edge);

    // Build a parallel+always root whose stratum 0 contains a sequential
    // gated child scope with two steps and an exit transition.
    Scope main;
    main.key = "main";
    main.mode = ScopeMode::Sequential;
    main.liveness = Liveness::Gated;
    main.steps.push_back(node_member("init"));
    main.steps.push_back(node_member("run"));
    Transition t;
    t.on = Handle("run", "done");
    main.transitions.push_back(t);

    original.root.mode = ScopeMode::Parallel;
    original.root.liveness = Liveness::Always;
    original.root.strata.push_back(Members{scope_member(std::move(main))});

    const auto pb = ASSERT_NIL_P(original.to_proto());
    const auto reconstructed = ASSERT_NIL_P(IR::from_proto(pb));
    ASSERT_EQ(reconstructed.functions.size(), 1);
    ASSERT_EQ(reconstructed.functions[0].key, "test_func");
    ASSERT_EQ(reconstructed.nodes.size(), 2);
    ASSERT_EQ(reconstructed.edges.size(), 1);
    ASSERT_EQ(reconstructed.root.mode, ScopeMode::Parallel);
    ASSERT_EQ(reconstructed.root.liveness, Liveness::Always);
    ASSERT_EQ(reconstructed.root.strata.size(), 1);
    const auto &m = reconstructed.root.strata[0][0];
    ASSERT_NE(m.scope, nullptr);
    ASSERT_EQ(m.scope->mode, ScopeMode::Sequential);
    ASSERT_EQ(m.scope->steps.size(), 2);
    ASSERT_EQ(m.scope->transitions.size(), 1);
    // An exit transition leaves target_key unset (nullopt).
    ASSERT_FALSE(m.scope->transitions[0].target_key.has_value());
}

/// @brief it should access nodes by key using node()
TEST(IRTest, testNodeAccess) {
    IR ir;
    Node n1;
    n1.key = "node_a";
    n1.type = "add";
    Node n2;
    n2.key = "node_b";
    n2.type = "multiply";
    ir.nodes.push_back(n1);
    ir.nodes.push_back(n2);

    ASSERT_EQ(ir.node("node_a").type, "add");
    ASSERT_EQ(ir.node("node_b").type, "multiply");
}

/// @brief it should access functions by key using function()
TEST(IRTest, testFunctionAccess) {
    IR ir;
    Function f1;
    f1.key = "func_x";
    Function f2;
    f2.key = "func_y";
    ir.functions.push_back(f1);
    ir.functions.push_back(f2);

    ASSERT_EQ(ir.function("func_x").key, "func_x");
    ASSERT_EQ(ir.function("func_y").key, "func_y");
}

/// @brief it should find edges by target handle using edge_to()
TEST(IRTest, testEdgeTo) {
    IR ir;
    Handle src1("node_a", "output");
    Handle tgt1("node_b", "input");
    Handle src2("node_c", "out");
    Handle tgt2("node_d", "in");
    ir.edges.emplace_back(src1, tgt1);
    ir.edges.emplace_back(src2, tgt2);

    auto edge1 = ir.edge_to(tgt1);
    ASSERT_TRUE(edge1.has_value());
    ASSERT_EQ(edge1->source.node, "node_a");
    ASSERT_EQ(edge1->source.param, "output");

    auto edge2 = ir.edge_to(tgt2);
    ASSERT_TRUE(edge2.has_value());
    ASSERT_EQ(edge2->source.node, "node_c");

    Handle missing("missing", "input");
    const auto no_edge = ir.edge_to(missing);
    ASSERT_FALSE(no_edge.has_value());
}

/// @brief it should return edges grouped by output param using edges_from()
TEST(IRTest, testEdgesFrom) {
    IR ir;
    ir.edges.emplace_back(Handle("node_a", "output"), Handle("node_b", "in1"));
    ir.edges.emplace_back(Handle("node_a", "output"), Handle("node_c", "in2"));
    ir.edges.emplace_back(Handle("node_a", "other"), Handle("node_d", "in3"));
    ir.edges.emplace_back(Handle("node_x", "out"), Handle("node_y", "in"));

    auto edges = ir.edges_from("node_a");
    ASSERT_EQ(edges.size(), 2);
    ASSERT_EQ(edges["output"].size(), 2);
    ASSERT_EQ(edges["other"].size(), 1);

    auto no_edges = ir.edges_from("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should return all edges into a node using edges_to()
TEST(IRTest, testEdgesTo) {
    IR ir;
    ir.edges.emplace_back(Handle("node_a", "out"), Handle("node_target", "in1"));
    ir.edges.emplace_back(Handle("node_b", "out"), Handle("node_target", "in2"));
    ir.edges.emplace_back(Handle("node_c", "out"), Handle("node_other", "in"));

    const auto edges = ir.edges_to("node_target");
    ASSERT_EQ(edges.size(), 2);

    const auto no_edges = ir.edges_to("nonexistent");
    ASSERT_TRUE(no_edges.empty());
}

/// @brief it should format a Handle as "node.param"
TEST(IRTest, testHandleToString) {
    const Handle h("node_a", "output");
    ASSERT_EQ(h.to_string(), "node_a.output");
}

/// @brief it should format a continuous Edge
TEST(IRTest, testEdgeToStringContinuous) {
    const Edge e(Handle("a", "out"), Handle("b", "in"), EdgeKind::Continuous);
    const auto str = e.to_string();
    ASSERT_EQ(str, "a.out -> b.in (continuous)");
}

/// @brief it should format a conditional Edge
TEST(IRTest, testEdgeToStringConditional) {
    const Edge e(Handle("a", "out"), Handle("b", "in"), EdgeKind::Conditional);
    const auto str = e.to_string();
    ASSERT_EQ(str, "a.out => b.in (conditional)");
}

/// @brief it should format a Transition with a step-key target
TEST(IRTest, testTransitionToStringStepKey) {
    Transition t;
    t.on = Handle("n", "out");
    t.target_key = "next";
    const auto str = t.to_string();
    ASSERT_NE(str.find("on n/out"), std::string::npos);
    ASSERT_NE(str.find("=> next"), std::string::npos);
}

/// @brief it should format a Transition with an exit (unset) target
TEST(IRTest, testTransitionToStringExit) {
    Transition t;
    t.on = Handle("n", "out");
    // target_key left unset signals exit.
    ASSERT_NE(t.to_string().find("=> exit"), std::string::npos);
}

/// @brief it should format a Member wrapping a leaf node-key
TEST(IRTest, testMemberToStringLeafNode) {
    const auto m = node_member("A");
    ASSERT_EQ(m.to_string(), "A");
}

/// @brief it should format a Member wrapping a nested Scope
TEST(IRTest, testMemberToStringScopeBacked) {
    Scope s;
    s.key = "sub";
    s.mode = ScopeMode::Parallel;
    s.liveness = Liveness::Always;
    const auto m = scope_member(std::move(s));
    ASSERT_NE(m.to_string().find("sub"), std::string::npos);
}

/// @brief it should format an unset Member as the empty-member placeholder
TEST(IRTest, testMemberToStringEmpty) {
    const Member m{};
    ASSERT_EQ(m.to_string(), "(empty member)");
}

/// @brief it should derive Member::key from the set variant
TEST(IRTest, testMemberKey) {
    ASSERT_EQ(node_member("n1").key(), "n1");
    Scope s;
    s.key = "sub";
    ASSERT_EQ(scope_member(std::move(s)).key(), "sub");
    const Member empty{};
    ASSERT_TRUE(empty.key().empty());
}

/// @brief it should format a parallel Scope with strata
TEST(IRTest, testScopeToStringParallel) {
    Scope s;
    s.key = "stage_1";
    s.mode = ScopeMode::Parallel;
    s.liveness = Liveness::Gated;
    Members stratum;
    stratum.push_back(node_member("A"));
    s.strata.push_back(std::move(stratum));
    const auto str = s.to_string();
    ASSERT_NE(str.find("stage_1"), std::string::npos);
    ASSERT_NE(str.find("parallel"), std::string::npos);
    ASSERT_NE(str.find("gated"), std::string::npos);
    ASSERT_NE(str.find("stratum 0"), std::string::npos);
    ASSERT_NE(str.find("A"), std::string::npos);
}

/// @brief it should format a sequential Scope with steps and a transition
TEST(IRTest, testScopeToStringSequentialWithTransitions) {
    Scope s;
    s.key = "main";
    s.mode = ScopeMode::Sequential;
    s.liveness = Liveness::Gated;
    s.steps.push_back(node_member("first"));
    s.steps.push_back(node_member("second"));
    Transition t;
    t.on = Handle("first", "done");
    t.target_key = "second";
    s.transitions.push_back(t);
    const auto str = s.to_string();
    ASSERT_NE(str.find("main"), std::string::npos);
    ASSERT_NE(str.find("sequential"), std::string::npos);
    ASSERT_NE(str.find("first"), std::string::npos);
    ASSERT_NE(str.find("second"), std::string::npos);
    ASSERT_NE(str.find("=> second"), std::string::npos);
}

/// @brief it should use the (scope) placeholder when a Scope has no key
TEST(IRTest, testScopeToStringEmptyKeyPlaceholder) {
    Scope s;
    s.mode = ScopeMode::Parallel;
    s.liveness = Liveness::Always;
    Members stratum;
    stratum.push_back(node_member("x"));
    s.strata.push_back(std::move(stratum));
    ASSERT_NE(s.to_string().find("(scope)"), std::string::npos);
}

/// @brief it should recurse into nested Scopes via scope_member
TEST(IRTest, testScopeToStringNestedScopeMember) {
    Scope inner;
    inner.key = "inner";
    inner.mode = ScopeMode::Sequential;
    inner.liveness = Liveness::Gated;
    inner.steps.push_back(node_member("step1"));
    Scope outer;
    outer.key = "outer";
    outer.mode = ScopeMode::Parallel;
    outer.liveness = Liveness::Always;
    Members stratum;
    stratum.push_back(scope_member(std::move(inner)));
    outer.strata.push_back(std::move(stratum));
    const auto str = outer.to_string();
    ASSERT_NE(str.find("outer"), std::string::npos);
    ASSERT_NE(str.find("inner"), std::string::npos);
    ASSERT_NE(str.find("step1"), std::string::npos);
}

/// @brief it should access params by name using operator[]
TEST(IRTest, testParamsOperatorBracketByName) {
    types::Params params;
    types::Param p1;
    p1.name = "alpha";
    p1.value = 42;
    types::Param p2;
    p2.name = "beta";
    p2.value = 3.14;
    params.push_back(p1);
    params.push_back(p2);
    ASSERT_EQ(params["alpha"].value.get<int>(), 42);
    ASSERT_DOUBLE_EQ(params["beta"].value.get<double>(), 3.14);
}

/// @brief it should access params by index using operator[]
TEST(IRTest, testParamsOperatorBracketByIndex) {
    types::Params params;
    types::Param p1;
    p1.name = "first";
    p1.value = 100;
    types::Param p2;
    p2.name = "second";
    p2.value = 200;
    params.push_back(p1);
    params.push_back(p2);
    ASSERT_EQ(params[0].name, "first");
    ASSERT_EQ(params[0].value.get<int>(), 100);
    ASSERT_EQ(params[1].name, "second");
    ASSERT_EQ(params[1].value.get<int>(), 200);
}

/// @brief it should format a Param without a value
TEST(IRTest, testParamToString) {
    types::Param p;
    p.name = "threshold";
    p.type.kind = types::Kind::F64;
    ASSERT_EQ(p.to_string(), "threshold (f64)");
}

/// @brief it should format a Param with a value
TEST(IRTest, testParamToStringWithValue) {
    types::Param p;
    p.name = "threshold";
    p.type.kind = types::Kind::F64;
    p.value = 42.5;
    ASSERT_EQ(p.to_string(), "threshold (f64) = 42.5");
}

/// @brief it should format Params as comma-separated list
TEST(IRTest, testParamsToString) {
    types::Params params;
    types::Param p1;
    p1.name = "x";
    p1.type.kind = types::Kind::F32;
    types::Param p2;
    p2.name = "y";
    p2.type.kind = types::Kind::I32;
    params.push_back(p1);
    params.push_back(p2);
    ASSERT_EQ(params.to_string(), "x (f32), y (i32)");
}

/// @brief it should format empty Params as "(none)"
TEST(IRTest, testParamsToStringEmpty) {
    const types::Params params;
    ASSERT_EQ(params.to_string(), "(none)");
}

/// @brief it should format Channels with read and write
TEST(IRTest, testChannelsToString) {
    types::Channels ch;
    ch.read[1] = "sensor";
    ch.write[2] = "actuator";
    const auto str = ch.to_string();
    ASSERT_NE(str.find("read [1: sensor]"), std::string::npos);
    ASSERT_NE(str.find("write [2: actuator]"), std::string::npos);
}

/// @brief it should format empty Channels as "(none)"
TEST(IRTest, testChannelsToStringEmpty) {
    const types::Channels ch;
    ASSERT_EQ(ch.to_string(), "(none)");
}

/// @brief it should format a Node with type and sections
TEST(IRTest, testNodeToString) {
    Node n;
    n.key = "add_1";
    n.type = "add";
    types::Param inp;
    inp.name = "lhs";
    inp.type.kind = types::Kind::F64;
    n.inputs.push_back(inp);
    types::Param out;
    out.name = "result";
    out.type.kind = types::Kind::F64;
    n.outputs.push_back(out);
    const auto str = n.to_string();
    ASSERT_NE(str.find("add_1 (type: add)"), std::string::npos);
    ASSERT_NE(str.find("inputs: lhs (f64)"), std::string::npos);
    ASSERT_NE(str.find("outputs: result (f64)"), std::string::npos);
}

/// @brief it should format a Function with channels and params
TEST(IRTest, testFunctionToString) {
    Function f;
    f.key = "my_func";
    f.channels.read[1] = "sensor";
    types::Param out;
    out.name = "result";
    out.type.kind = types::Kind::F64;
    f.outputs.push_back(out);
    const auto str = f.to_string();
    ASSERT_NE(str.find("my_func"), std::string::npos);
    ASSERT_NE(str.find("channels:"), std::string::npos);
    ASSERT_NE(str.find("read [1: sensor]"), std::string::npos);
    ASSERT_NE(str.find("outputs: result (f64)"), std::string::npos);
}

/// @brief it should format a full IR tree rooted at a Scope
TEST(IRTest, testIRToString) {
    IR ir;

    Function fn;
    fn.key = "add";
    types::Param fn_out;
    fn_out.name = "result";
    fn_out.type.kind = types::Kind::F64;
    fn.outputs.push_back(fn_out);
    ir.functions.push_back(fn);

    Node n;
    n.key = "add_1";
    n.type = "add";
    ir.nodes.push_back(n);

    ir.edges.emplace_back(Handle("a", "out"), Handle("b", "in"), EdgeKind::Continuous);

    ir.root.key = "";
    ir.root.mode = ScopeMode::Parallel;
    ir.root.liveness = Liveness::Always;
    Members stratum;
    stratum.push_back(node_member("add_1"));
    ir.root.strata.push_back(std::move(stratum));

    const auto str = ir.to_string();
    ASSERT_NE(str.find("IR"), std::string::npos);
    ASSERT_NE(str.find("Functions"), std::string::npos);
    ASSERT_NE(str.find("add"), std::string::npos);
    ASSERT_NE(str.find("Nodes"), std::string::npos);
    ASSERT_NE(str.find("add_1 (type: add)"), std::string::npos);
    ASSERT_NE(str.find("Edges"), std::string::npos);
    ASSERT_NE(str.find("a.out -> b.in (continuous)"), std::string::npos);
    ASSERT_NE(str.find("Root"), std::string::npos);
    ASSERT_NE(str.find("parallel"), std::string::npos);
    ASSERT_NE(str.find("always"), std::string::npos);
}

/// @brief it should render an empty IR as the bare "IR" root label
TEST(IRTest, testIRToStringEmpty) {
    const IR ir{};
    ASSERT_EQ(ir.to_string(), "IR");
}

/// @brief it should omit sections that carry no entries
TEST(IRTest, testIRToStringOnlyNodes) {
    IR ir{};
    Node n;
    n.key = "only";
    n.type = "input";
    ir.nodes.push_back(n);
    const auto str = ir.to_string();
    ASSERT_NE(str.find("Nodes"), std::string::npos);
    ASSERT_EQ(str.find("Functions"), std::string::npos);
    ASSERT_EQ(str.find("Edges"), std::string::npos);
    ASSERT_EQ(str.find("Root"), std::string::npos);
}

/// @brief it should render multi-entry sections with both branching
/// ("├── ") and terminal ("└── ") tree prefixes
TEST(IRTest, testIRToStringMultipleEntriesTreeIndentation) {
    IR ir{};
    Function f1;
    f1.key = "f1";
    ir.functions.push_back(f1);
    Function f2;
    f2.key = "f2";
    ir.functions.push_back(f2);
    Node n1;
    n1.key = "n1";
    n1.type = "f1";
    ir.nodes.push_back(n1);
    Node n2;
    n2.key = "n2";
    n2.type = "f2";
    ir.nodes.push_back(n2);
    ir.edges.emplace_back(
        Handle("n1", "output"),
        Handle("n2", "input"),
        EdgeKind::Continuous
    );
    ir.edges.emplace_back(
        Handle("n2", "output"),
        Handle("n1", "input"),
        EdgeKind::Conditional
    );
    const auto str = ir.to_string();
    ASSERT_NE(str.find("├── "), std::string::npos);
    ASSERT_NE(str.find("└── "), std::string::npos);
    ASSERT_NE(str.find("n1.output -> n2.input"), std::string::npos);
    ASSERT_NE(str.find("n2.output => n1.input"), std::string::npos);
}

/// @brief it should stream Handle via operator<<
TEST(IRTest, testHandleStreamOperator) {
    const Handle h("node_a", "output");
    std::ostringstream ss;
    ss << h;
    ASSERT_EQ(ss.str(), "node_a.output");
}

/// @brief it should stream Edge via operator<<
TEST(IRTest, testEdgeStreamOperator) {
    const Edge e(Handle("a", "out"), Handle("b", "in"), EdgeKind::Continuous);
    std::ostringstream ss;
    ss << e;
    ASSERT_EQ(ss.str(), "a.out -> b.in (continuous)");
}

}
