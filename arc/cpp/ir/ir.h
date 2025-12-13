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
#include <ostream>
#include <sstream>
#include <string>
#include <vector>

#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/ir/format.h"
#include "arc/cpp/proto/proto.h"
#include "arc/cpp/types/types.h"
#include "arc/go/ir/arc/go/ir/ir.pb.h"
#include "arc/go/symbol/arc/go/symbol/symbol.pb.h"

namespace arc::ir {
constexpr std::string default_output_param = "output";
constexpr std::string default_input_param = "input";
constexpr std::string lhs_input_param = "lhs_input";
constexpr std::string rhs_input_param = "rhs_input";

enum class EdgeKind { Continuous = 0, OneShot = 1 };

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

    /// @brief Returns the string representation of the handle as "node.param".
    [[nodiscard]] std::string to_string() const { return node + "." + param; }

    friend std::ostream &operator<<(std::ostream &os, const Handle &h) {
        return os << h.to_string();
    }
};

struct Edge {
    Handle source, target;
    EdgeKind kind = EdgeKind::Continuous;

    explicit Edge(xjson::Parser parser) {
        this->source = parser.field<Handle>("source");
        this->target = parser.field<Handle>("target");
        auto kind_val = parser.field<int>("kind", 0);
        this->kind = static_cast<EdgeKind>(kind_val);
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"source", source.to_json()},
            {"target", target.to_json()},
            {"kind", static_cast<int>(kind)},
        };
    }

    explicit Edge(const arc::v1::ir::PBEdge &pb) {
        if (pb.has_source()) this->source = Handle(pb.source());
        if (pb.has_target()) this->target = Handle(pb.target());
        this->kind = static_cast<EdgeKind>(pb.kind());
    }

    void to_proto(arc::v1::ir::PBEdge *pb) const {
        source.to_proto(pb->mutable_source());
        target.to_proto(pb->mutable_target());
        pb->set_kind(static_cast<arc::v1::ir::PBEdgeKind>(kind));
    }

    Edge() = default;
    Edge(Handle src, Handle tgt, EdgeKind k = EdgeKind::Continuous)
        : source(std::move(src)), target(std::move(tgt)), kind(k) {}

    bool operator==(const Edge &other) const {
        return source == other.source && target == other.target && kind == other.kind;
    }

    /// @brief Returns the string representation of the edge.
    [[nodiscard]] std::string to_string() const {
        std::string arrow = (kind == EdgeKind::OneShot) ? " => " : " -> ";
        std::string kind_str = (kind == EdgeKind::OneShot) ? "oneshot" : "continuous";
        return source.to_string() + arrow + target.to_string() + " (" + kind_str + ")";
    }

    friend std::ostream &operator<<(std::ostream &os, const Edge &e) {
        return os << e.to_string();
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

    /// @brief Returns the string representation of the param.
    [[nodiscard]] std::string to_string() const {
        std::string result = name + " (" + type.to_string() + ")";
        if (!value.is_null()) result += " = " + value.dump();
        return result;
    }

    friend std::ostream &operator<<(std::ostream &os, const Param &p) {
        return os << p.to_string();
    }
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

    /// @brief Returns the string representation of the params.
    [[nodiscard]] std::string to_string() const { return format_params(*this); }

    friend std::ostream &operator<<(std::ostream &os, const Params &p) {
        return os << p.to_string();
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

    /// @brief Returns the string representation of the channels.
    [[nodiscard]] std::string to_string() const { return format_channels(*this); }

    friend std::ostream &operator<<(std::ostream &os, const Channels &c) {
        return os << c.to_string();
    }
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

    /// @brief Returns the string representation of the node.
    [[nodiscard]] std::string to_string() const {
        return to_string_with_prefix("");
    }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << " (type: " << type << ")\n";

        const bool has_config = !config.empty();
        const bool has_inputs = !inputs.empty();
        const bool has_outputs = !outputs.empty();

        // Channels
        bool is_last = !has_config && !has_inputs && !has_outputs;
        ss << prefix << tree_prefix(is_last) << "channels: " << channels.to_string()
           << "\n";

        // Config (if any)
        if (has_config) {
            is_last = !has_inputs && !has_outputs;
            ss << prefix << tree_prefix(is_last) << "config: " << config.to_string()
               << "\n";
        }

        // Inputs
        if (has_inputs) {
            is_last = !has_outputs;
            ss << prefix << tree_prefix(is_last) << "inputs: " << inputs.to_string()
               << "\n";
        }

        // Outputs
        if (has_outputs) {
            ss << prefix << tree_prefix(true) << "outputs: " << outputs.to_string()
               << "\n";
        }

        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Node &n) {
        return os << n.to_string();
    }
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

    /// @brief Returns the string representation of the function.
    [[nodiscard]] std::string to_string() const {
        return to_string_with_prefix("");
    }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << "\n";

        const bool has_config = !config.empty();
        const bool has_inputs = !inputs.empty();
        const bool has_outputs = !outputs.empty();

        // Channels
        bool is_last = !has_config && !has_inputs && !has_outputs;
        ss << prefix << tree_prefix(is_last) << "channels: " << channels.to_string()
           << "\n";

        // Config (if any)
        if (has_config) {
            is_last = !has_inputs && !has_outputs;
            ss << prefix << tree_prefix(is_last) << "config: " << config.to_string()
               << "\n";
        }

        // Inputs
        if (has_inputs) {
            is_last = !has_outputs;
            ss << prefix << tree_prefix(is_last) << "inputs: " << inputs.to_string()
               << "\n";
        }

        // Outputs
        if (has_outputs) {
            ss << prefix << tree_prefix(true) << "outputs: " << outputs.to_string()
               << "\n";
        }

        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Function &f) {
        return os << f.to_string();
    }
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

    /// @brief Returns the string representation of the strata.
    [[nodiscard]] std::string to_string() const {
        return to_string_with_prefix("");
    }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        if (strata.empty()) return "";
        std::ostringstream ss;
        for (size_t i = 0; i < strata.size(); ++i) {
            bool is_last = i == strata.size() - 1;
            ss << prefix << tree_prefix(is_last) << "[" << i << "]: ";
            for (size_t j = 0; j < strata[i].size(); ++j) {
                if (j > 0) ss << ", ";
                ss << strata[i][j];
            }
            ss << "\n";
        }
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Strata &s) {
        return os << s.to_string();
    }
};

struct Stage {
    std::string key;
    std::vector<std::string> nodes;

    Stage() = default;

    explicit Stage(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->nodes = parser.field<std::vector<std::string>>("nodes", {});
    }

    [[nodiscard]] nlohmann::json to_json() const {
        return {{"key", key}, {"nodes", nodes}};
    }

    explicit Stage(const arc::v1::ir::PBStage &pb) {
        this->key = pb.key();
        for (const auto &node : pb.nodes())
            this->nodes.push_back(node);
    }

    void to_proto(arc::v1::ir::PBStage *pb) const {
        pb->set_key(key);
        for (const auto &node : nodes)
            pb->add_nodes(node);
    }

    /// @brief Returns the string representation of the stage.
    [[nodiscard]] std::string to_string() const {
        std::ostringstream ss;
        ss << key << ": [";
        for (size_t i = 0; i < nodes.size(); ++i) {
            if (i > 0) ss << ", ";
            ss << nodes[i];
        }
        ss << "]";
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Stage &s) {
        return os << s.to_string();
    }
};

struct Sequence {
    std::string key;
    std::vector<Stage> stages;

    Sequence() = default;

    explicit Sequence(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->stages = parser.field<std::vector<Stage>>("stages", {});
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json stages_arr = nlohmann::json::array();
        for (const auto &stage : stages)
            stages_arr.push_back(stage.to_json());
        return {{"key", key}, {"stages", stages_arr}};
    }

    explicit Sequence(const arc::v1::ir::PBSequence &pb) {
        this->key = pb.key();
        for (const auto &stage_pb : pb.stages())
            this->stages.emplace_back(stage_pb);
    }

    void to_proto(arc::v1::ir::PBSequence *pb) const {
        pb->set_key(key);
        for (const auto &stage : stages)
            stage.to_proto(pb->add_stages());
    }

    [[nodiscard]] const Stage* find_stage(const std::string &stage_key) const {
        for (const auto &stage : stages)
            if (stage.key == stage_key) return &stage;
        return nullptr;
    }

    /// @brief Returns the string representation of the sequence.
    [[nodiscard]] std::string to_string() const {
        return to_string_with_prefix("");
    }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << "\n";
        for (size_t i = 0; i < stages.size(); ++i) {
            bool is_last = i == stages.size() - 1;
            ss << prefix << tree_prefix(is_last) << stages[i].to_string() << "\n";
        }
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Sequence &s) {
        return os << s.to_string();
    }
};

struct IR {
    std::vector<Function> functions;
    std::vector<Node> nodes;
    std::vector<Edge> edges;
    Strata strata;
    std::vector<Sequence> sequences;

    IR() = default;

    explicit IR(xjson::Parser parser) {
        this->functions = parser.field<std::vector<Function>>("functions");
        this->nodes = parser.field<std::vector<Node>>("nodes");
        this->edges = parser.field<std::vector<Edge>>("edges");
        this->strata = parser.field<Strata>("strata");
        this->sequences = parser.field<std::vector<Sequence>>("sequences", {});
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
        nlohmann::json sequences_arr = nlohmann::json::array();
        for (const auto &seq: sequences)
            sequences_arr.push_back(seq.to_json());
        return {
            {"functions", functions_arr},
            {"nodes", nodes_arr},
            {"edges", edges_arr},
            {"strata", strata.to_json()},
            {"sequences", sequences_arr}
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
        sequences.reserve(pb.sequences_size());
        for (const auto &seq_pb: pb.sequences())
            sequences.emplace_back(seq_pb);
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
        pb->mutable_sequences()->Reserve(static_cast<int>(sequences.size()));
        for (const auto &seq: sequences)
            seq.to_proto(pb->add_sequences());
    }

    [[nodiscard]] const Sequence* find_sequence(const std::string &key) const {
        for (const auto &seq : sequences)
            if (seq.key == key) return &seq;
        return nullptr;
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

    /// @brief Returns the string representation of the IR.
    [[nodiscard]] std::string to_string() const {
        return to_string_with_prefix("");
    }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;

        const bool has_functions = !functions.empty();
        const bool has_nodes = !nodes.empty();
        const bool has_edges = !edges.empty();
        const bool has_strata = !strata.strata.empty();
        const bool has_sequences = !sequences.empty();

        // Functions
        if (has_functions) {
            bool is_last = !has_nodes && !has_edges && !has_strata && !has_sequences;
            write_functions(ss, prefix, is_last);
        }

        // Nodes
        if (has_nodes) {
            bool is_last = !has_edges && !has_strata && !has_sequences;
            write_nodes(ss, prefix, is_last);
        }

        // Edges
        if (has_edges) {
            bool is_last = !has_strata && !has_sequences;
            write_edges(ss, prefix, is_last);
        }

        // Strata
        if (has_strata) {
            bool is_last = !has_sequences;
            write_strata(ss, prefix, is_last);
        }

        // Sequences
        if (has_sequences) { write_sequences(ss, prefix, true); }

        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const IR &ir) {
        return os << ir.to_string();
    }

private:
    void write_functions(
        std::ostringstream &ss,
        const std::string &prefix,
        bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Functions (" << functions.size()
           << ")\n";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < functions.size(); ++i) {
            bool is_last = i == functions.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << functions[i].to_string_with_prefix(child_prefix + tree_indent(is_last)
               );
        }
    }

    void
    write_nodes(std::ostringstream &ss, const std::string &prefix, bool last) const {
        ss << prefix << tree_prefix(last) << "Nodes (" << nodes.size() << ")\n";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < nodes.size(); ++i) {
            bool is_last = i == nodes.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << nodes[i].to_string_with_prefix(child_prefix + tree_indent(is_last));
        }
    }

    void
    write_edges(std::ostringstream &ss, const std::string &prefix, bool last) const {
        ss << prefix << tree_prefix(last) << "Edges (" << edges.size() << ")\n";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < edges.size(); ++i) {
            bool is_last = i == edges.size() - 1;
            ss << child_prefix << tree_prefix(is_last) << edges[i].to_string() << "\n";
        }
    }

    void
    write_strata(std::ostringstream &ss, const std::string &prefix, bool last) const {
        ss << prefix << tree_prefix(last) << "Strata (" << strata.strata.size()
           << " layers)\n";
        std::string child_prefix = prefix + tree_indent(last);
        ss << strata.to_string_with_prefix(child_prefix);
    }

    void write_sequences(
        std::ostringstream &ss,
        const std::string &prefix,
        bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Sequences (" << sequences.size()
           << ")\n";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < sequences.size(); ++i) {
            bool is_last = i == sequences.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << sequences[i].to_string_with_prefix(
                      child_prefix + tree_indent(is_last)
               );
        }
    }
};
}
