// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <string>
#include <vector>

#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/proto/proto.h"
#include "arc/go/graph/arc/go/graph/graph.pb.h"

namespace arc::graph {

/// @brief Represents an XY coordinate (position or viewport location)
struct XY {
    float x = 0.0f;
    float y = 0.0f;

    XY() = default;
    XY(const float x, const float y): x(x), y(y) {}

    explicit XY(xjson::Parser p) {
        x = p.field<float>("x");
        y = p.field<float>("y");
    }

    explicit XY(const v1::graph::XY &pb) {
        x = pb.x();
        y = pb.y();
    }

    [[nodiscard]] nlohmann::json to_json() const { return {{"x", x}, {"y", y}}; }

    void to_proto(v1::graph::XY *pb) const {
        pb->set_x(x);
        pb->set_y(y);
    }
};

/// @brief Represents the viewport state of the graph editor
struct Viewport {
    XY position;
    float zoom = 1.0f;

    Viewport() = default;

    explicit Viewport(xjson::Parser p) {
        position = p.field<XY>("position");
        zoom = p.field<float>("zoom");
    }

    explicit Viewport(const arc::v1::graph::PBViewport &pb) {
        if (pb.has_position()) position = XY(pb.position());
        zoom = pb.zoom();
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {{"position", position.to_json()}, {"zoom", zoom}};
    }

    void to_proto(arc::v1::graph::PBViewport *pb) const {
        position.to_proto(pb->mutable_position());
        pb->set_zoom(zoom);
    }
};

/// @brief Represents a visual node in the graph (includes position, unlike ir::Node)
struct Node {
    std::string key;
    std::string type;
    std::map<std::string, nlohmann::json> config;
    XY position;

    Node() = default;

    explicit Node(xjson::Parser p) {
        key = p.field<std::string>("key");
        type = p.field<std::string>("type");
        config = p.field<std::map<std::string, nlohmann::json>>("config");
        position = p.field<XY>("position");
    }

    explicit Node(const v1::graph::PBNode &pb) {
        key = pb.key();
        type = pb.type();
        for (const auto &[config_key, config_value]: pb.config())
            config[config_key] = proto::pb_value_to_json(config_value);
        if (pb.has_position()) position = XY(pb.position());
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"key", key},
            {"type", type},
            {"config", config},
            {"position", position.to_json()}
        };
    }

    void to_proto(v1::graph::PBNode *pb) const {
        pb->set_key(key);
        pb->set_type(type);
        auto *config_map = pb->mutable_config();
        for (const auto &[k, v]: config)
            proto::json_to_pb_value(v, &(*config_map)[k]);
        position.to_proto(pb->mutable_position());
    }
};

struct Graph {
    Viewport viewport;
    std::vector<ir::Function> functions;
    std::vector<ir::Edge> edges;
    std::vector<Node> nodes;

    Graph() = default;

    explicit Graph(xjson::Parser p) {
        viewport = p.field<Viewport>("viewport");
        this->functions = p.field<std::vector<ir::Function>>("functions");
        this->edges = p.field<std::vector<ir::Edge>>("edges");
        this->nodes = p.field<std::vector<Node>>("nodes");
    }

    explicit Graph(const v1::graph::PBGraph &pb) {
        if (pb.has_viewport()) viewport = Viewport(pb.viewport());
        functions.reserve(pb.functions_size());
        for (const auto &fn_pb: pb.functions())
            functions.emplace_back(fn_pb);
        edges.reserve(pb.edges_size());
        for (const auto &edge_pb: pb.edges())
            edges.emplace_back(edge_pb);
        nodes.reserve(pb.nodes_size());
        for (const auto &node_pb: pb.nodes())
            nodes.emplace_back(node_pb);
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json functions_json = nlohmann::json::array();
        for (const auto &fn: functions)
            functions_json.push_back(fn.to_json());
        nlohmann::json edges_json = nlohmann::json::array();
        for (const auto &edge: edges)
            edges_json.push_back(edge.to_json());
        nlohmann::json nodes_json = nlohmann::json::array();
        for (const auto &node: nodes)
            nodes_json.push_back(node.to_json());
        return {
            {"viewport", viewport.to_json()},
            {"functions", functions_json},
            {"edges", edges_json},
            {"nodes", nodes_json}
        };
    }

    void to_proto(v1::graph::PBGraph *pb) const {
        viewport.to_proto(pb->mutable_viewport());
        pb->mutable_functions()->Reserve(static_cast<int>(functions.size()));
        for (const auto &fn: functions)
            fn.to_proto(pb->add_functions());
        pb->mutable_edges()->Reserve(static_cast<int>(edges.size()));
        for (const auto &edge: edges)
            edge.to_proto(pb->add_edges());
        pb->mutable_nodes()->Reserve(static_cast<int>(nodes.size()));
        for (const auto &node: nodes)
            node.to_proto(pb->add_nodes());
    }
};
}
