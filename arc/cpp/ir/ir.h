// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cassert>
#include <map>
#include <optional>
#include <ostream>
#include <sstream>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/format.h"
#include "arc/cpp/types/types.h"
#include "arc/go/ir/arc/go/ir/ir.pb.h"
#include "arc/go/symbol/arc/go/symbol/symbol.pb.h"

namespace arc::ir {
inline const std::string default_output_param = "output";
inline const std::string default_input_param = "input";
inline const std::string lhs_input_param = "lhs_input";
inline const std::string rhs_input_param = "rhs_input";

enum class EdgeKind { Unspecified = 0, Continuous = 1, OneShot = 2 };

struct Handle {
    std::string node, param;

    Handle() = default;
    Handle(std::string node, std::string param):
        node(std::move(node)), param(std::move(param)) {}

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

    /// @brief Returns the string representation of the handle as "node.param".
    [[nodiscard]] std::string to_string() const { return node + "." + param; }

    friend std::ostream &operator<<(std::ostream &os, const Handle &h) {
        return os << h.to_string();
    }
};
}

template<>
struct std::hash<arc::ir::Handle> {
    size_t operator()(const arc::ir::Handle &h) const noexcept {
        return std::hash<std::string>{}(h.node + h.param);
    }
};

namespace arc::ir {
struct Edge {
    Handle source, target;
    EdgeKind kind = EdgeKind::Continuous;

    explicit Edge(const v1::ir::PBEdge &pb) {
        if (pb.has_source()) this->source = Handle(pb.source());
        if (pb.has_target()) this->target = Handle(pb.target());
        this->kind = static_cast<EdgeKind>(pb.kind());
    }

    void to_proto(v1::ir::PBEdge *pb) const {
        source.to_proto(pb->mutable_source());
        target.to_proto(pb->mutable_target());
        pb->set_kind(static_cast<v1::ir::PBEdgeKind>(kind));
    }

    Edge() = default;
    Edge(Handle src, Handle tgt, const EdgeKind k = EdgeKind::Continuous):
        source(std::move(src)), target(std::move(tgt)), kind(k) {}

    bool operator==(const Edge &other) const {
        return source == other.source && target == other.target && kind == other.kind;
    }

    [[nodiscard]] std::string to_string() const {
        const std::string arrow = kind == EdgeKind::OneShot ? " => " : " -> ";
        const std::string kind_str = kind == EdgeKind::OneShot ? "oneshot"
                                                               : "continuous";
        return source.to_string() + arrow + target.to_string() + " (" + kind_str + ")";
    }

    friend std::ostream &operator<<(std::ostream &os, const Edge &e) {
        return os << e.to_string();
    }
};
}

template<>
struct std::hash<arc::ir::Edge> {
    size_t operator()(const arc::ir::Edge &e) const noexcept {
        return std::hash<arc::ir::Handle>{}(e.source) ^
               std::hash<arc::ir::Handle>{}(e.target) << 1 ^
               std::hash<int>{}(static_cast<int>(e.kind)) << 2;
    }
};

namespace arc::ir {
struct Param {
    std::string name;
    types::Type type;
    std::optional<telem::SampleValue> value;

    explicit Param(const v1::types::PBParam &pb) {
        this->name = pb.name();
        if (pb.has_type()) this->type = types::Type(pb.type());
        if (pb.has_value()) this->value = telem::from_proto(pb.value());
    }

    Param() = default;

    void to_proto(v1::types::PBParam *pb) const {
        pb->set_name(name);
        type.to_proto(pb->mutable_type());
        if (value.has_value()) telem::to_proto(*value, pb->mutable_value());
    }

    /// @brief Returns the value cast to the requested type.
    /// @tparam T The type to cast to.
    /// @returns The value as type T.
    /// @note The compiler guarantees this param has a value.
    template<typename T>
    [[nodiscard]] T get() const {
        assert(value.has_value() && "Param has no value");
        return telem::cast<T>(*value);
    }

    [[nodiscard]] std::string to_string() const {
        std::string result = name + " (" + type.to_string() + ")";
        if (value.has_value()) result += " = " + telem::to_string(*value);
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

    /// @brief Returns the parameter with the given name.
    /// @param name The name of the parameter to find.
    /// @returns A reference to the parameter.
    /// @note The compiler guarantees this parameter exists.
    [[nodiscard]] const Param &operator[](const std::string &name) const {
        for (const auto &p: this->params)
            if (p.name == name) return p;
        assert(false && "Param not found");
        std::unreachable();
    }

    [[nodiscard]] std::vector<std::string> keys() const {
        std::vector<std::string> result;
        result.reserve(this->params.size());
        for (const auto &p: this->params)
            result.push_back(p.name);
        return result;
    }

    auto begin() { return this->params.begin(); }
    auto end() { return this->params.end(); }
    [[nodiscard]] auto begin() const { return this->params.begin(); }
    [[nodiscard]] auto end() const { return this->params.end(); }
    [[nodiscard]] size_t size() const { return this->params.size(); }
    [[nodiscard]] bool empty() const { return this->params.empty(); }

    /// @brief Returns the parameter at the given index.
    /// @param index The index of the parameter.
    /// @returns A reference to the parameter.
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

    explicit Channels(const v1::symbol::PBChannels &pb) {
        for (const auto &[key, value]: pb.read())
            read[key] = value;
        for (const auto &[key, value]: pb.write())
            write[key] = value;
    }

    void to_proto(v1::symbol::PBChannels *pb) const {
        auto *read_map = pb->mutable_read();
        for (const auto &[key, value]: read)
            (*read_map)[key] = value;
        auto *write_map = pb->mutable_write();
        for (const auto &[key, value]: write)
            (*write_map)[key] = value;
    }

    Channels() = default;

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

    explicit Node(const v1::ir::PBNode &pb) {
        this->key = pb.key();
        this->type = pb.type();
        if (pb.has_channels()) this->channels = Channels(pb.channels());
        this->config = Params(pb.config());
        this->inputs = Params(pb.inputs());
        this->outputs = Params(pb.outputs());
    }

    void to_proto(v1::ir::PBNode *pb) const {
        pb->set_key(key);
        pb->set_type(type);
        channels.to_proto(pb->mutable_channels());
        config.to_proto(pb->mutable_config());
        inputs.to_proto(pb->mutable_inputs());
        outputs.to_proto(pb->mutable_outputs());
    }

    Node() = default;
    explicit Node(std::string k): key(std::move(k)) {}

    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << " (type: " << type << ")\n";

        const bool has_config = !config.empty();
        const bool has_inputs = !inputs.empty();
        const bool has_outputs = !outputs.empty();

        bool is_last = !has_config && !has_inputs && !has_outputs;
        ss << prefix << tree_prefix(is_last) << "channels: " << channels.to_string()
           << "\n";

        if (has_config) {
            is_last = !has_inputs && !has_outputs;
            ss << prefix << tree_prefix(is_last) << "config: " << config.to_string()
               << "\n";
        }

        if (has_inputs) {
            is_last = !has_outputs;
            ss << prefix << tree_prefix(is_last) << "inputs: " << inputs.to_string()
               << "\n";
        }

        if (has_outputs)
            ss << prefix << tree_prefix(true) << "outputs: " << outputs.to_string()
               << "\n";

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

    explicit Function(const v1::ir::PBFunction &pb) {
        this->key = pb.key();
        if (pb.has_channels()) this->channels = Channels(pb.channels());
        this->config = Params(pb.config());
        this->inputs = Params(pb.inputs());
        this->outputs = Params(pb.outputs());
    }

    void to_proto(v1::ir::PBFunction *pb) const {
        pb->set_key(key);
        channels.to_proto(pb->mutable_channels());
        config.to_proto(pb->mutable_config());
        inputs.to_proto(pb->mutable_inputs());
        outputs.to_proto(pb->mutable_outputs());
    }

    Function() = default;
    explicit Function(std::string k): key(std::move(k)) {}

    /// @brief Returns the string representation of the function.
    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << "\n";

        const bool has_config = !config.empty();
        const bool has_inputs = !inputs.empty();
        const bool has_outputs = !outputs.empty();

        bool is_last = !has_config && !has_inputs && !has_outputs;
        ss << prefix << tree_prefix(is_last) << "channels: " << channels.to_string()
           << "\n";

        if (has_config) {
            is_last = !has_inputs && !has_outputs;
            ss << prefix << tree_prefix(is_last) << "config: " << config.to_string()
               << "\n";
        }

        if (has_inputs) {
            is_last = !has_outputs;
            ss << prefix << tree_prefix(is_last) << "inputs: " << inputs.to_string()
               << "\n";
        }

        if (has_outputs)
            ss << prefix << tree_prefix(true) << "outputs: " << outputs.to_string()
               << "\n";

        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Function &f) {
        return os << f.to_string();
    }
};

struct Strata {
    Strata() = default;

    explicit Strata(std::vector<std::vector<std::string>> layers):
        layers_(std::move(layers)) {}

    template<typename PBStrataContainer>
    explicit Strata(const PBStrataContainer &pb_strata) {
        layers_.reserve(pb_strata.size());
        for (const auto &pb_stratum: pb_strata) {
            std::vector<std::string> stratum;
            stratum.reserve(pb_stratum.nodes_size());
            for (const auto &node: pb_stratum.nodes())
                stratum.push_back(node);
            layers_.push_back(std::move(stratum));
        }
    }

    template<typename PBStrataRepeatedField>
    void to_proto(PBStrataRepeatedField *pb_strata) const {
        pb_strata->Reserve(static_cast<int>(layers_.size()));
        for (const auto &stratum: layers_) {
            auto *pb_stratum = pb_strata->Add();
            pb_stratum->mutable_nodes()->Reserve(static_cast<int>(stratum.size()));
            for (const auto &node: stratum)
                pb_stratum->add_nodes(node);
        }
    }

    /// @brief Returns the layer (stratum) at the given index.
    [[nodiscard]] const std::vector<std::string> &operator[](const size_t index) const {
        return layers_.at(index);
    }

    /// @brief Returns the layer (stratum) at the given index.
    [[nodiscard]] std::vector<std::string> &operator[](const size_t index) {
        return layers_.at(index);
    }

    [[nodiscard]] auto begin() const { return layers_.begin(); }
    [[nodiscard]] auto end() const { return layers_.end(); }
    auto begin() { return layers_.begin(); }
    auto end() { return layers_.end(); }
    [[nodiscard]] size_t size() const { return layers_.size(); }
    [[nodiscard]] bool empty() const { return layers_.empty(); }

    /// @brief Returns the string representation of the strata.
    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        if (layers_.empty()) return "";
        std::ostringstream ss;
        for (size_t i = 0; i < layers_.size(); ++i) {
            const bool is_last = i == layers_.size() - 1;
            ss << prefix << tree_prefix(is_last) << "[" << i << "]: ";
            for (size_t j = 0; j < layers_[i].size(); ++j) {
                if (j > 0) ss << ", ";
                ss << layers_[i][j];
            }
            ss << "\n";
        }
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Strata &s) {
        return os << s.to_string();
    }

private:
    std::vector<std::vector<std::string>> layers_;
};

struct Stage {
    std::string key;
    std::vector<std::string> nodes;
    Strata strata;

    Stage() = default;

    explicit Stage(const v1::ir::PBStage &pb) {
        this->key = pb.key();
        for (const auto &node: pb.nodes())
            this->nodes.push_back(node);
        this->strata = Strata(pb.strata());
    }

    void to_proto(v1::ir::PBStage *pb) const {
        pb->set_key(key);
        for (const auto &node: nodes)
            pb->add_nodes(node);
        strata.to_proto(pb->mutable_strata());
    }

    /// @brief Returns the string representation of the stage.
    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << ": [";
        for (size_t i = 0; i < nodes.size(); ++i) {
            if (i > 0) ss << ", ";
            ss << nodes[i];
        }
        ss << "]";
        if (!strata.empty()) ss << "\n" << strata.to_string_with_prefix(prefix);
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

    explicit Sequence(const v1::ir::PBSequence &pb) {
        this->key = pb.key();
        for (const auto &stage_pb: pb.stages())
            this->stages.emplace_back(stage_pb);
    }

    void to_proto(v1::ir::PBSequence *pb) const {
        pb->set_key(key);
        for (const auto &stage: stages)
            stage.to_proto(pb->add_stages());
    }

    /// @brief Returns the stage with the given key.
    /// @param stage_key The key of the stage to find.
    /// @returns A reference to the stage.
    /// @note The compiler guarantees this stage exists.
    [[nodiscard]] const Stage &operator[](const std::string &stage_key) const {
        for (const auto &stage: stages)
            if (stage.key == stage_key) return stage;
        assert(false && "Stage not found");
        std::unreachable();
    }

    /// @brief Returns the stage that follows the given stage in definition order.
    /// @param stage_key The key of the current stage.
    /// @returns The next stage, or std::nullopt if this is the last stage.
    [[nodiscard]] std::optional<Stage> next(const std::string &stage_key) const {
        for (size_t i = 0; i < stages.size(); ++i)
            if (stages[i].key == stage_key && i + 1 < stages.size())
                return stages[i + 1];
        return std::nullopt;
    }

    /// @brief Returns the string representation of the sequence.
    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;
        ss << key << "\n";
        for (size_t i = 0; i < stages.size(); ++i) {
            const bool is_last = i == stages.size() - 1;
            std::string stage_child_prefix = prefix + tree_indent(is_last);
            ss << prefix << tree_prefix(is_last)
               << stages[i].to_string_with_prefix(stage_child_prefix);
            if (stages[i].strata.empty()) ss << "\n";
        }
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Sequence &s) {
        return os << s.to_string();
    }
};

struct AuthorityConfig {
    std::optional<uint8_t> default_authority;
    std::map<std::string, uint8_t> channels;
    std::map<uint32_t, std::string> keys;

    AuthorityConfig() = default;

    explicit AuthorityConfig(const v1::ir::PBAuthorityConfig &pb) {
        if (pb.has_default_()) default_authority = static_cast<uint8_t>(pb.default_());
        for (const auto &[name, val]: pb.channels())
            channels[name] = static_cast<uint8_t>(val);
        for (const auto &[key, name]: pb.keys())
            keys[key] = name;
    }

    void to_proto(v1::ir::PBAuthorityConfig *pb) const {
        if (default_authority.has_value()) pb->set_default_(*default_authority);
        auto *ch_map = pb->mutable_channels();
        for (const auto &[name, val]: channels)
            (*ch_map)[name] = val;
        auto *keys_map = pb->mutable_keys();
        for (const auto &[key, name]: keys)
            (*keys_map)[key] = name;
    }
};

struct IR {
    std::vector<Function> functions;
    std::vector<Node> nodes;
    std::vector<Edge> edges;
    Strata strata;
    std::vector<Sequence> sequences;
    AuthorityConfig authority;

    IR() = default;

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
        if (pb.has_authority()) authority = AuthorityConfig(pb.authority());
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
        authority.to_proto(pb->mutable_authority());
    }

    /// @brief Returns the node with the given key.
    /// @param key The key of the node to find.
    /// @returns A reference to the node.
    /// @note The compiler guarantees this node exists.
    [[nodiscard]] const Node &node(const std::string &key) const {
        for (const auto &n: nodes)
            if (n.key == key) return n;
        assert(false && "Node not found");
        std::unreachable();
    }

    /// @brief Returns the function with the given key.
    /// @param key The key of the function to find.
    /// @returns A reference to the function.
    /// @note The compiler guarantees this function exists.
    [[nodiscard]] const Function &function(const std::string &key) const {
        for (const auto &fn: functions)
            if (fn.key == key) return fn;
        assert(false && "Function not found");
        std::unreachable();
    }

    /// @brief Returns the sequence with the given key.
    /// @param key The key of the sequence to find.
    /// @returns A reference to the sequence.
    /// @note The compiler guarantees this sequence exists.
    [[nodiscard]] const Sequence &sequence(const std::string &key) const {
        for (const auto &seq: sequences)
            if (seq.key == key) return seq;
        assert(false && "Sequence not found");
        std::unreachable();
    }

    /// @brief Returns the edge targeting the given handle, if one exists.
    /// @param target The target handle to search for.
    /// @returns The edge if found, or std::nullopt if no edge targets this handle.
    [[nodiscard]] std::optional<Edge> edge_to(const Handle &target) const {
        for (const auto &e: edges)
            if (e.target == target) return e;
        return std::nullopt;
    }

    /// @brief Returns all edges originating from the given node, grouped by output
    /// param.
    /// @param node_key The key of the source node.
    /// @returns A map from output parameter name to the edges from that parameter.
    [[nodiscard]] std::unordered_map<std::string, std::vector<Edge>>
    edges_from(const std::string &node_key) const {
        std::unordered_map<std::string, std::vector<Edge>> result;
        for (const auto &e: edges)
            if (e.source.node == node_key) result[e.source.param].push_back(e);
        return result;
    }

    /// @brief Returns all edges targeting the given node.
    /// @param node_key The key of the target node.
    /// @returns A vector of all edges into this node.
    [[nodiscard]] std::vector<Edge> edges_into(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.target.node == node_key) result.push_back(e);
        return result;
    }

    /// @brief Returns the string representation of the IR.
    [[nodiscard]] std::string to_string() const { return to_string_with_prefix(""); }

    /// @brief Returns the string representation with tree formatting.
    [[nodiscard]] std::string to_string_with_prefix(const std::string &prefix) const {
        std::ostringstream ss;

        const bool has_functions = !functions.empty();
        const bool has_nodes = !nodes.empty();
        const bool has_edges = !edges.empty();
        const bool has_strata = !strata.empty();
        const bool has_sequences = !sequences.empty();

        if (has_functions) {
            const bool is_last = !has_nodes && !has_edges && !has_strata &&
                                 !has_sequences;
            write_functions(ss, prefix, is_last);
        }

        if (has_nodes) {
            const bool is_last = !has_edges && !has_strata && !has_sequences;
            write_nodes(ss, prefix, is_last);
        }

        if (has_edges) {
            const bool is_last = !has_strata && !has_sequences;
            write_edges(ss, prefix, is_last);
        }

        if (has_strata) {
            const bool is_last = !has_sequences;
            write_strata(ss, prefix, is_last);
        }

        if (has_sequences) write_sequences(ss, prefix, true);

        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const IR &ir) {
        return os << ir.to_string();
    }

private:
    void write_functions(
        std::ostringstream &ss,
        const std::string &prefix,
        const bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Functions (" << functions.size() << ")\n";
        const std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < functions.size(); ++i) {
            const bool is_last = i == functions.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << functions[i].to_string_with_prefix(
                      child_prefix + tree_indent(is_last)
                  );
        }
    }

    void write_nodes(
        std::ostringstream &ss,
        const std::string &prefix,
        const bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Nodes (" << nodes.size() << ")\n";
        const std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < nodes.size(); ++i) {
            const bool is_last = i == nodes.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << nodes[i].to_string_with_prefix(child_prefix + tree_indent(is_last));
        }
    }

    void write_edges(
        std::ostringstream &ss,
        const std::string &prefix,
        const bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Edges (" << edges.size() << ")\n";
        const std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < edges.size(); ++i) {
            const bool is_last = i == edges.size() - 1;
            ss << child_prefix << tree_prefix(is_last) << edges[i].to_string() << "\n";
        }
    }

    void write_strata(
        std::ostringstream &ss,
        const std::string &prefix,
        const bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Strata (" << strata.size()
           << " layers)\n";
        const std::string child_prefix = prefix + tree_indent(last);
        ss << strata.to_string_with_prefix(child_prefix);
    }

    void write_sequences(
        std::ostringstream &ss,
        const std::string &prefix,
        const bool last
    ) const {
        ss << prefix << tree_prefix(last) << "Sequences (" << sequences.size() << ")\n";
        const std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < sequences.size(); ++i) {
            const bool is_last = i == sequences.size() - 1;
            ss << child_prefix << tree_prefix(is_last)
               << sequences[i].to_string_with_prefix(
                      child_prefix + tree_indent(is_last)
                  );
        }
    }
};
}
