// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <map>
#include <string>
#include <vector>

#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/proto/proto.h"
#include "arc/cpp/types/types.h"
#include "arc/go/ir/arc/go/ir/ir.pb.h"
#include "arc/go/symbol/arc/go/symbol/symbol.pb.h"

namespace arc::ir {
constexpr std::string default_output_param = "output";
constexpr std::string default_input_param = "input";
constexpr std::string lhs_input_param = "lhs_input";
constexpr std::string rhs_input_param = "rhs_input";

struct Handle {
    std::string node, param;

    Handle() = default;
    Handle(std::string node, std::string param):
        node(std::move(node)), param(std::move(param)) {}

    explicit Handle(xjson::Parser parser) {
        this->node = parser.field<std::string>("node");
        this->param = parser.field<std::string>("param");
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {{"node", node}, {"param", param}};
    }

    explicit Handle(const v1::ir::PBHandle &pb) {
        this->node = pb.node();
        this->param = pb.param();
    }

    void to_proto(v1::ir::PBHandle *pb) const {
        pb->set_node(node);
        pb->set_param(param);
    }

    bool operator==(const Handle &other) const {
        return node == other.node && param == other.param;
    }

    bool operator!=(const Handle &other) const { return !(*this == other); }

    struct Hasher {
        size_t operator()(const Handle &handle) const {
            return std::hash<std::string>()(handle.node + handle.param);
        }
    };
};

struct Edge {
    Handle source, target;

    explicit Edge(xjson::Parser parser) {
        this->source = parser.field<Handle>("source");
        this->target = parser.field<Handle>("target");
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"source", source.to_json()},
            {"target", target.to_json()},
        };
    }

    explicit Edge(const arc::v1::ir::PBEdge &pb) {
        if (pb.has_source()) this->source = Handle(pb.source());
        if (pb.has_target()) this->target = Handle(pb.target());
    }

    void to_proto(arc::v1::ir::PBEdge *pb) const {
        source.to_proto(pb->mutable_source());
        target.to_proto(pb->mutable_target());
    }

    Edge() = default;
    Edge(Handle src, Handle tgt): source(std::move(src)), target(std::move(tgt)) {}

    bool operator==(const Edge &other) const {
        return source == other.source && target == other.target;
    }
};

struct Param {
    std::string name;
    types::Type type;
    nlohmann::json value;

    explicit Param(xjson::Parser parser) {
        this->name = parser.field<std::string>("name");
        this->type = parser.field<types::Type>("type");
        this->value = parser.field<nlohmann::json>("value", nlohmann::json(nullptr));
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json j;
        j["name"] = name;
        j["type"] = type.to_json();
        if (!value.is_null()) j["value"] = value;
        return j;
    }

    explicit Param(const arc::v1::types::PBParam &pb) {
        this->name = pb.name();
        if (pb.has_type()) this->type = types::Type(pb.type());
        if (pb.has_value()) this->value = arc::proto::pb_value_to_json(pb.value());
    }

    void to_proto(arc::v1::types::PBParam *pb) const {
        pb->set_name(name);
        type.to_proto(pb->mutable_type());
        if (!value.is_null()) arc::proto::json_to_pb_value(value, pb->mutable_value());
    }

    Param() = default;
};

struct Params {
    std::vector<Param> params;

    Params() = default;
    explicit Params(std::vector<Param> p): params(std::move(p)) {}

    template<typename PBParamContainer>
    explicit Params(const PBParamContainer &pb_params) {
        params.reserve(pb_params.size());
        for (const auto &pb_param: pb_params)
            params.emplace_back(pb_param);
    }

    template<typename PBParamRepeatedField>
    void to_proto(PBParamRepeatedField *pb_params) const {
        pb_params->Reserve(static_cast<int>(params.size()));
        for (const auto &param: params)
            param.to_proto(pb_params->Add());
    }

    [[nodiscard]] const Param *get(const std::string &name) const {
        for (const auto &p: this->params) {
            if (p.name == name) return &p;
        }
        return nullptr;
    }

    [[nodiscard]] std::vector<std::string> keys() const {
        std::vector<std::string> result;
        result.reserve(this->params.size());
        for (const auto &p: this->params)
            result.push_back(p.name);
        return result;
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json arr = nlohmann::json::array();
        for (const auto &p: this->params)
            arr.push_back(p.to_json());
        return arr;
    }

    auto begin() { return this->params.begin(); }
    auto end() { return this->params.end(); }
    [[nodiscard]] auto begin() const { return this->params.begin(); }
    [[nodiscard]] auto end() const { return this->params.end(); }
    [[nodiscard]] size_t size() const { return this->params.size(); }
    [[nodiscard]] bool empty() const { return this->params.empty(); }

    Param &operator[](const size_t index) { return params.at(index); }
    [[nodiscard]] const Param &operator[](const size_t index) const {
        return params.at(index);
    }
};

struct Channels {
    std::map<types::ChannelKey, std::string> read;
    std::map<types::ChannelKey, std::string> write;

    explicit Channels(xjson::Parser parser) {
        this->read = parser.field<std::map<types::ChannelKey, std::string>>("read", {});
        this->write = parser.field<std::map<types::ChannelKey, std::string>>(
            "write",
            {}
        );
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json read_obj = nlohmann::json::object();
        for (const auto &[key, value]: read)
            read_obj[std::to_string(key)] = value;
        nlohmann::json write_obj = nlohmann::json::object();
        for (const auto &[key, value]: write)
            write_obj[std::to_string(key)] = value;
        return {{"read", read_obj}, {"write", write_obj}};
    }

    explicit Channels(const arc::v1::symbol::PBChannels &pb) {
        for (const auto &[key, value]: pb.read())
            read[key] = value;
        for (const auto &[key, value]: pb.write())
            write[key] = value;
    }

    void to_proto(arc::v1::symbol::PBChannels *pb) const {
        auto *read_map = pb->mutable_read();
        for (const auto &[key, value]: read)
            (*read_map)[key] = value;
        auto *write_map = pb->mutable_write();
        for (const auto &[key, value]: write)
            (*write_map)[key] = value;
    }

    Channels() = default;
};

struct Node {
    std::string key;
    std::string type;
    Channels channels;
    Params config, inputs, outputs;

    explicit Node(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->type = parser.field<std::string>("type");
        this->channels = parser.field<Channels>("channels");
        this->config = Params(parser.field<std::vector<Param>>("config", {}));
        this->inputs = Params(parser.field<std::vector<Param>>("inputs", {}));
        this->outputs = Params(parser.field<std::vector<Param>>("outputs", {}));
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"key", key},
            {"type", type},
            {"channels", channels.to_json()},
            {"config", config.to_json()},
            {"inputs", inputs.to_json()},
            {"outputs", outputs.to_json()}
        };
    }

    explicit Node(const arc::v1::ir::PBNode &pb) {
        this->key = pb.key();
        this->type = pb.type();
        if (pb.has_channels()) this->channels = Channels(pb.channels());
        this->config = Params(pb.config());
        this->inputs = Params(pb.inputs());
        this->outputs = Params(pb.outputs());
    }

    void to_proto(arc::v1::ir::PBNode *pb) const {
        pb->set_key(key);
        pb->set_type(type);
        channels.to_proto(pb->mutable_channels());
        config.to_proto(pb->mutable_config());
        inputs.to_proto(pb->mutable_inputs());
        outputs.to_proto(pb->mutable_outputs());
    }

    Node() = default;
    explicit Node(std::string k): key(std::move(k)) {}
};

struct Function {
    std::string key;
    Channels channels;
    Params config, inputs, outputs;

    explicit Function(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->channels = parser.field<Channels>("channels");
        this->config = Params(parser.field<std::vector<Param>>("config", {}));
        this->inputs = Params(parser.field<std::vector<Param>>("inputs", {}));
        this->outputs = Params(parser.field<std::vector<Param>>("outputs", {}));
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"key", key},
            {"channels", channels.to_json()},
            {"config", config.to_json()},
            {"inputs", inputs.to_json()},
            {"outputs", outputs.to_json()}
        };
    }

    explicit Function(const arc::v1::ir::PBFunction &pb) {
        this->key = pb.key();
        if (pb.has_channels()) this->channels = Channels(pb.channels());
        this->config = Params(pb.config());
        this->inputs = Params(pb.inputs());
        this->outputs = Params(pb.outputs());
    }

    void to_proto(arc::v1::ir::PBFunction *pb) const {
        pb->set_key(key);
        channels.to_proto(pb->mutable_channels());
        config.to_proto(pb->mutable_config());
        inputs.to_proto(pb->mutable_inputs());
        outputs.to_proto(pb->mutable_outputs());
        // Note: body field is not in C++ Function struct
    }

    Function() = default;
    explicit Function(std::string k): key(std::move(k)) {}
};

struct Strata {
    std::vector<std::vector<std::string>> strata;

    explicit Strata(xjson::Parser parser) {
        this->strata = parser.field<std::vector<std::vector<std::string>>>("");
    }

    [[nodiscard]] nlohmann::json to_json() const { return strata; }

    template<typename PBStrataContainer>
    explicit Strata(const PBStrataContainer &pb_strata) {
        strata.reserve(pb_strata.size());
        for (const auto &pb_stratum: pb_strata) {
            std::vector<std::string> stratum;
            stratum.reserve(pb_stratum.nodes_size());
            for (const auto &node: pb_stratum.nodes())
                stratum.push_back(node);
            strata.push_back(std::move(stratum));
        }
    }

    template<typename PBStrataRepeatedField>
    void to_proto(PBStrataRepeatedField *pb_strata) const {
        pb_strata->Reserve(static_cast<int>(strata.size()));
        for (const auto &stratum: strata) {
            auto *pb_stratum = pb_strata->Add();
            pb_stratum->mutable_nodes()->Reserve(static_cast<int>(stratum.size()));
            for (const auto &node: stratum)
                pb_stratum->add_nodes(node);
        }
    }

    Strata() = default;
};

struct IR {
    std::vector<Function> functions;
    std::vector<Node> nodes;
    std::vector<Edge> edges;
    Strata strata;

    IR() = default;

    explicit IR(xjson::Parser parser) {
        this->functions = parser.field<std::vector<Function>>("functions");
        this->nodes = parser.field<std::vector<Node>>("nodes");
        this->edges = parser.field<std::vector<Edge>>("edges");
        this->strata = parser.field<Strata>("strata");
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json functions_arr = nlohmann::json::array();
        for (const auto &fn: functions)
            functions_arr.push_back(fn.to_json());
        nlohmann::json nodes_arr = nlohmann::json::array();
        for (const auto &node: nodes)
            nodes_arr.push_back(node.to_json());
        nlohmann::json edges_arr = nlohmann::json::array();
        for (const auto &edge: edges)
            edges_arr.push_back(edge.to_json());
        return {
            {"functions", functions_arr},
            {"nodes", nodes_arr},
            {"edges", edges_arr},
            {"strata", strata.to_json()}
        };
    }

    explicit IR(const v1::ir::PBIR &pb) {
        functions.reserve(pb.functions_size());
        for (const auto &fn_pb: pb.functions())
            functions.emplace_back(fn_pb);
        nodes.reserve(pb.nodes_size());
        for (const auto &node_pb: pb.nodes())
            nodes.emplace_back(node_pb);
        edges.reserve(pb.edges_size());
        for (const auto &edge_pb: pb.edges())
            edges.emplace_back(edge_pb);
        strata = Strata(pb.strata());
    }

    void to_proto(arc::v1::ir::PBIR *pb) const {
        pb->mutable_functions()->Reserve(static_cast<int>(functions.size()));
        for (const auto &fn: functions)
            fn.to_proto(pb->add_functions());
        pb->mutable_nodes()->Reserve(static_cast<int>(nodes.size()));
        for (const auto &node: nodes)
            node.to_proto(pb->add_nodes());
        pb->mutable_edges()->Reserve(static_cast<int>(edges.size()));
        for (const auto &edge: edges)
            edge.to_proto(pb->add_edges());
        strata.to_proto(pb->mutable_strata());
    }

    using function_iterator = std::vector<Function>::iterator;
    using const_function_iterator = std::vector<Function>::const_iterator;
    using node_iterator = std::vector<Node>::iterator;
    using const_node_iterator = std::vector<Node>::const_iterator;
    using edge_iterator = std::vector<Edge>::iterator;
    using const_edge_iterator = std::vector<Edge>::const_iterator;

    [[nodiscard]] function_iterator find_function(const std::string &key) {
        return std::ranges::find_if(functions, [&](const auto &fn) {
            return fn.key == key;
        });
    }
    [[nodiscard]] const_function_iterator find_function(const std::string &key) const {
        return std::ranges::find_if(functions, [&](const auto &fn) {
            return fn.key == key;
        });
    }

    [[nodiscard]] node_iterator find_node(const std::string &key) {
        return std::ranges::find_if(nodes, [&](const auto &node) {
            return node.key == key;
        });
    }

    [[nodiscard]] const_node_iterator find_node(const std::string &key) const {
        return std::ranges::find_if(nodes, [&](const auto &node) {
            return node.key == key;
        });
    }

    [[nodiscard]] const_edge_iterator find_edge_by_target(const Handle &handle) {
        return std::ranges::find_if(edges, [&](const auto &edge) {
            return edge.target == handle;
        });
    }

    [[nodiscard]] std::vector<Edge> outgoing_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.source.node == node_key) result.push_back(e);
        return result;
    }

    [[nodiscard]] std::vector<Edge> incoming_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.target.node == node_key) result.push_back(e);
        return result;
    }
};
}
