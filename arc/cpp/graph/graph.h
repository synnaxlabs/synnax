// Copyright 2026 Synnax Labs, Inc.
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

#include "nlohmann/json.hpp"

#include "x/cpp/spatial/spatial.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/proto/proto.h"
#include "arc/go/graph/arc/go/graph/graph.pb.h"

namespace arc::graph {
struct Viewport {
    spatial::XY position;
    float zoom = 1.0f;

    Viewport() = default;

    explicit Viewport(const v1::graph::PBViewport &pb):
        position(pb.position()), zoom(pb.zoom()) {}

    void to_proto(v1::graph::PBViewport *pb) const {
        this->position.to_proto(pb->mutable_position());
        pb->set_zoom(this->zoom);
    }
};

/// @brief Represents a visual node in the graph (includes position, unlike ir::Node)
struct Node {
    std::string key;
    std::string type;
    std::map<std::string, nlohmann::json> config;
    spatial::XY position;

    Node() = default;

    explicit Node(const v1::graph::PBNode &pb): key(pb.key()), type(pb.type()) {
        for (const auto &[config_key, config_value]: pb.config())
            this->config[config_key] = proto::pb_value_to_json(config_value);
        if (pb.has_position()) this->position = spatial::XY(pb.position());
    }

    void to_proto(v1::graph::PBNode *pb) const {
        pb->set_key(this->key);
        pb->set_type(this->type);
        auto *config_map = pb->mutable_config();
        for (const auto &[k, v]: this->config)
            proto::json_to_pb_value(v, &(*config_map)[k]);
        this->position.to_proto(pb->mutable_position());
    }
};

struct Graph {
    Viewport viewport;
    std::vector<ir::Function> functions;
    std::vector<ir::Edge> edges;
    std::vector<Node> nodes;

    Graph() = default;

    explicit Graph(const v1::graph::PBGraph &pb): viewport(pb.viewport()) {
        this->functions.reserve(pb.functions_size());
        for (const auto &fn_pb: pb.functions())
            this->functions.emplace_back(fn_pb);
        this->edges.reserve(pb.edges_size());
        for (const auto &edge_pb: pb.edges())
            this->edges.emplace_back(edge_pb);
        this->nodes.reserve(pb.nodes_size());
        for (const auto &node_pb: pb.nodes())
            this->nodes.emplace_back(node_pb);
    }

    void to_proto(v1::graph::PBGraph *pb) const {
        this->viewport.to_proto(pb->mutable_viewport());
        pb->mutable_functions()->Reserve(static_cast<int>(this->functions.size()));
        for (const auto &fn: this->functions)
            fn.to_proto(pb->add_functions());
        pb->mutable_edges()->Reserve(static_cast<int>(this->edges.size()));
        for (const auto &edge: this->edges)
            edge.to_proto(pb->add_edges());
        pb->mutable_nodes()->Reserve(static_cast<int>(this->nodes.size()));
        for (const auto &node: this->nodes)
            node.to_proto(pb->add_nodes());
    }
};
}
