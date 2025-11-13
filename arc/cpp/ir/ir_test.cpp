// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include <nlohmann/json.hpp>

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/ir/types.h"

using namespace arc::ir;
using json = nlohmann::json;

// ══════════════════════════════════════════════════════════════════════════════
// UNIT TESTS - Individual Components
// ══════════════════════════════════════════════════════════════════════════════

TEST(TypeTest, ParsePrimitiveType) {
    const json j = {{"kind", 7}}; // I32
    xjson::Parser parser(j);

    Type type(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(type.kind, TypeKind::I32);
    EXPECT_TRUE(type.is_valid());
    EXPECT_EQ(type.elem, nullptr);
}

TEST(TypeTest, ParseCompoundType) {
    const std::string json_str = R"({"kind": 15, "elem": {"kind": 10}})"; // Series<F64>
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Type type(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(type.kind, TypeKind::Series);
    ASSERT_NE(type.elem, nullptr);
    EXPECT_EQ(type.elem->kind, TypeKind::F64);
}

TEST(TypeTest, RoundTrip) {
    // Create a compound type
    Type original(TypeKind::Chan, Type(TypeKind::TimeStamp));

    // Serialize to JSON
    auto json = original.to_json();

    // Parse back
    auto parser = xjson::Parser(json);
    Type parsed(parser);
    EXPECT_TRUE(parser.ok());

    // Verify equality
    EXPECT_EQ(parsed.kind, original.kind);
    ASSERT_NE(parsed.elem, nullptr);
    EXPECT_EQ(parsed.elem->kind, original.elem->kind);
}

TEST(HandleTest, ParseHandle) {
    const std::string json_str = R"({"node": "n1", "param": "output"})";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Handle handle(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(handle.node, "n1");
    EXPECT_EQ(handle.param, "output");
}

TEST(HandleTest, RoundTrip) {
    Handle original("node_key", "param_name");

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Handle parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed, original);
}

TEST(EdgeTest, ParseEdge) {
    const std::string json_str = R"({
        "source": {"node": "n1", "param": "output"},
        "target": {"node": "n2", "param": "input"}
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Edge edge(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(edge.source.node, "n1");
    EXPECT_EQ(edge.source.param, "output");
    EXPECT_EQ(edge.target.node, "n2");
    EXPECT_EQ(edge.target.param, "input");
}

TEST(EdgeTest, RoundTrip) {
    Edge original(Handle("src", "out"), Handle("tgt", "in"));

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Edge parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed, original);
}

TEST(ParamTest, ParseParamWithValue) {
    const std::string json_str = R"({
        "name": "threshold",
        "type": {"kind": 10},
        "value": 42.5
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Param param(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(param.name, "threshold");
    EXPECT_EQ(param.type.kind, TypeKind::F64);
    EXPECT_DOUBLE_EQ(param.value.get<double>(), 42.5);
}

TEST(ParamTest, ParseParamWithoutValue) {
    const std::string json_str = R"({
        "name": "input",
        "type": {"kind": 7}
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Param param(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(param.name, "input");
    EXPECT_EQ(param.type.kind, TypeKind::I32);
    EXPECT_TRUE(param.value.is_null());
}

TEST(ParamTest, RoundTrip) {
    Param original;
    original.name = "config_param";
    original.type = Type(TypeKind::U32);
    original.value = 100;

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Param parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed.name, original.name);
    EXPECT_EQ(parsed.type.kind, original.type.kind);
    EXPECT_EQ(parsed.value.get<int>(), original.value.get<int>());
}

TEST(ParamsTest, GetMethod) {
    Params params;
    params.params = {Param(), Param(), Param()};
    params.params[0].name = "a";
    params.params[0].type = Type(TypeKind::I32);
    params.params[1].name = "b";
    params.params[1].type = Type(TypeKind::F64);
    params.params[2].name = "c";
    params.params[2].type = Type(TypeKind::String);

    const auto *type_a = params.get("a");
    ASSERT_NE(type_a, nullptr);
    EXPECT_EQ(type_a->kind, TypeKind::I32);

    const auto *type_b = params.get("b");
    ASSERT_NE(type_b, nullptr);
    EXPECT_EQ(type_b->kind, TypeKind::F64);

    const auto *type_not_found = params.get("nonexistent");
    EXPECT_EQ(type_not_found, nullptr);
}

TEST(ParamsTest, KeysMethod) {
    Params params;
    params.params = {Param(), Param(), Param()};
    params.params[0].name = "first";
    params.params[1].name = "second";
    params.params[2].name = "third";

    auto keys = params.keys();
    ASSERT_EQ(keys.size(), 3);
    EXPECT_EQ(keys[0], "first");
    EXPECT_EQ(keys[1], "second");
    EXPECT_EQ(keys[2], "third");
}

TEST(ParamsTest, RoundTrip) {
    Params original;
    original.params = {Param(), Param()};
    original.params[0].name = "input";
    original.params[0].type = Type(TypeKind::I32);
    original.params[1].name = "output";
    original.params[1].type = Type(TypeKind::F64);
    original.params[1].value = 3.14;

    auto json = original.to_json();
    EXPECT_TRUE(json.is_array());
    EXPECT_EQ(json.size(), 2);

    // Parse back - need to parse as vector first, then wrap in Params
    xjson::Parser parser(json);
    auto params_vec = parser.field<std::vector<Param>>("");
    EXPECT_TRUE(parser.ok());
    Params parsed(params_vec);

    EXPECT_EQ(parsed.size(), original.size());
    auto keys = parsed.keys();
    EXPECT_EQ(keys[0], "input");
    EXPECT_EQ(keys[1], "output");
}

TEST(ChannelsTest, ParseChannels) {
    const std::string json_str = R"({
        "read": {"1": "param_a", "2": "param_b"},
        "write": {"3": "param_out"}
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Channels channels(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(channels.read.size(), 2);
    EXPECT_EQ(channels.read[1], "param_a");
    EXPECT_EQ(channels.read[2], "param_b");
    EXPECT_EQ(channels.write.size(), 1);
    EXPECT_EQ(channels.write[3], "param_out");
}

TEST(ChannelsTest, ParseEmptyChannels) {
    const std::string json_str = R"({"read": {}, "write": {}})";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Channels channels(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_TRUE(channels.read.empty());
    EXPECT_TRUE(channels.write.empty());
}

TEST(ChannelsTest, RoundTrip) {
    Channels original;
    original.read[10] = "sensor_input";
    original.read[20] = "timestamp";
    original.write[30] = "actuator_output";

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Channels parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed.read, original.read);
    EXPECT_EQ(parsed.write, original.write);
}

TEST(StrataTest, ParseStrata) {
    const std::string json_str = R"([
        ["input_a", "input_b"],
        ["compute_1", "compute_2"],
        ["output"]
    ])";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Strata strata(parser);
    EXPECT_TRUE(parser.ok());
    ASSERT_EQ(strata.strata.size(), 3);
    EXPECT_EQ(strata.strata[0].size(), 2);
    EXPECT_EQ(strata.strata[0][0], "input_a");
    EXPECT_EQ(strata.strata[0][1], "input_b");
    EXPECT_EQ(strata.strata[1].size(), 2);
    EXPECT_EQ(strata.strata[2].size(), 1);
    EXPECT_EQ(strata.strata[2][0], "output");
}

TEST(StrataTest, RoundTrip) {
    Strata original;
    original.strata = {{"n1", "n2"}, {"n3"}, {"n4", "n5", "n6"}};

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Strata parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed.strata, original.strata);
}

TEST(NodeTest, ParseNode) {
    const std::string json_str = R"({
        "key": "add_node",
        "type": "add",
        "channels": {"read": {}, "write": {}},
        "config": [],
        "inputs": [
            {"name": "a", "type": {"kind": 7}},
            {"name": "b", "type": {"kind": 7}}
        ],
        "outputs": [
            {"name": "output", "type": {"kind": 7}}
        ]
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Node node(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(node.key, "add_node");
    EXPECT_EQ(node.type, "add");
    EXPECT_EQ(node.inputs.size(), 2);
    EXPECT_EQ(node.outputs.size(), 1);
}

TEST(NodeTest, RoundTrip) {
    Node original("test_node");
    original.type = "custom";
    original.channels.read[1] = "input_channel";

    Param input_param;
    input_param.name = "data";
    input_param.type = Type(TypeKind::F64);
    original.inputs.params.push_back(input_param);

    Param output_param;
    output_param.name = "result";
    output_param.type = Type(TypeKind::F64);
    original.outputs.params.push_back(output_param);

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Node parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed.key, original.key);
    EXPECT_EQ(parsed.type, original.type);
    EXPECT_EQ(parsed.inputs.size(), original.inputs.size());
    EXPECT_EQ(parsed.outputs.size(), original.outputs.size());
}

TEST(FunctionTest, ParseFunction) {
    const std::string json_str = R"({
        "key": "multiply",
        "channels": {"read": {}, "write": {}},
        "config": [
            {"name": "scalar", "type": {"kind": 10}, "value": 2.0}
        ],
        "inputs": [
            {"name": "input", "type": {"kind": 10}}
        ],
        "outputs": [
            {"name": "output", "type": {"kind": 10}}
        ]
    })";
    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    Function func(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(func.key, "multiply");
    EXPECT_EQ(func.config.size(), 1);
    EXPECT_EQ(func.inputs.size(), 1);
    EXPECT_EQ(func.outputs.size(), 1);
}

TEST(FunctionTest, RoundTrip) {
    Function original("test_func");

    Param config_param;
    config_param.name = "threshold";
    config_param.type = Type(TypeKind::I32);
    config_param.value = 100;
    original.config.params.push_back(config_param);

    auto json = original.to_json();
    auto parser = xjson::Parser(json);
    Function parsed(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(parsed.key, original.key);
    EXPECT_EQ(parsed.config.size(), 1);
}

// ══════════════════════════════════════════════════════════════════════════════
// INTEGRATION TESTS - Full IR
// ══════════════════════════════════════════════════════════════════════════════

TEST(IRTest, ParseCompleteIR) {
    const std::string json_str = R"({
        "functions": [
            {
                "key": "add",
                "channels": {"read": {}, "write": {}},
                "config": [],
                "inputs": [
                    {"name": "a", "type": {"kind": 7}},
                    {"name": "b", "type": {"kind": 7}}
                ],
                "outputs": [
                    {"name": "output", "type": {"kind": 7}}
                ]
            }
        ],
        "nodes": [
            {
                "key": "n1",
                "type": "add",
                "channels": {"read": {}, "write": {}},
                "config": [],
                "inputs": [
                    {"name": "a", "type": {"kind": 7}},
                    {"name": "b", "type": {"kind": 7}}
                ],
                "outputs": [
                    {"name": "output", "type": {"kind": 7}}
                ]
            }
        ],
        "edges": [
            {
                "source": {"node": "input_a", "param": "value"},
                "target": {"node": "n1", "param": "a"}
            },
            {
                "source": {"node": "input_b", "param": "value"},
                "target": {"node": "n1", "param": "b"}
            }
        ],
        "strata": [
            ["input_a", "input_b"],
            ["n1"]
        ]
    })";

    json j = json::parse(json_str);
    xjson::Parser parser(j);
    EXPECT_TRUE(parser.ok());

    IR ir(parser);
    EXPECT_TRUE(parser.ok());

    EXPECT_EQ(ir.functions.size(), 1);
    EXPECT_EQ(ir.nodes.size(), 1);
    EXPECT_EQ(ir.edges.size(), 2);
    EXPECT_EQ(ir.strata.strata.size(), 2);

    // Verify function
    EXPECT_EQ(ir.functions[0].key, "add");

    // Verify node
    EXPECT_EQ(ir.nodes[0].key, "n1");
    EXPECT_EQ(ir.nodes[0].type, "add");

    // Verify edges
    EXPECT_EQ(ir.edges[0].source.node, "input_a");
    EXPECT_EQ(ir.edges[0].target.node, "n1");

    // Verify helper methods
    const auto *func = ir.find_function("add");
    ASSERT_NE(func, nullptr);
    EXPECT_EQ(func->key, "add");

    const auto *node = ir.find_node("n1");
    ASSERT_NE(node, nullptr);
    EXPECT_EQ(node->key, "n1");
}

TEST(IRTest, RoundTripCompleteIR) {
    // Build IR programmatically
    IR original;

    // Add function
    Function func("my_func");
    Param input;
    input.name = "x";
    input.type = Type(TypeKind::F32);
    func.inputs.params.push_back(input);
    original.functions.push_back(func);

    // Add node
    Node node("node_1");
    node.type = "my_func";
    node.inputs = func.inputs;
    original.nodes.push_back(node);

    // Add edge
    original.edges.push_back(Edge(Handle("src", "out"), Handle("node_1", "x")));

    // Add strata
    original.strata.strata = {{"src"}, {"node_1"}};

    // Serialize
    auto json = original.to_json();

    // Parse back
    auto parser = xjson::Parser(json);
    IR parsed(parser);
    EXPECT_TRUE(parser.ok());

    // Verify structure
    EXPECT_EQ(parsed.functions.size(), original.functions.size());
    EXPECT_EQ(parsed.nodes.size(), original.nodes.size());
    EXPECT_EQ(parsed.edges.size(), original.edges.size());
    EXPECT_EQ(parsed.strata.strata.size(), original.strata.strata.size());
}

// ══════════════════════════════════════════════════════════════════════════════
// ERROR TESTS - Malformed JSON
// ══════════════════════════════════════════════════════════════════════════════

TEST(ErrorTest, MissingRequiredField) {
    const json j = {{"param", "value"}}; // Missing "node"
    xjson::Parser parser(j);

    Handle handle(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GT(parser.errors->size(), 0);
}

TEST(ErrorTest, WrongTypeForField) {
    const json j = {{"kind", "not_a_number"}};
    xjson::Parser parser(j);

    Type type(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GT(parser.errors->size(), 0);
}

TEST(ErrorTest, MissingNestedField) {
    const json j = {
        {"source", {{"node", "n1"}}}, // Missing "param"
        {"target", {{"node", "n2"}, {"param", "in"}}}
    };

    xjson::Parser parser(j);
    Edge edge(parser);

    EXPECT_FALSE(parser.ok());
    EXPECT_GT(parser.errors->size(), 0);
}
