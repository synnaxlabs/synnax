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
#include <cstdint>
#include <map>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

namespace arc {
using ChannelKey = std::uint32_t;

namespace ir {

enum class TypeKind : uint8_t {
    Invalid = 0,
    U8 = 1,
    U16 = 2,
    U32 = 3,
    U64 = 4,
    I8 = 5,
    I16 = 6,
    I32 = 7,
    I64 = 8,
    F32 = 9,
    F64 = 10,
    String = 11,
    TimeStamp = 12,
    TimeSpan = 13,
    Chan = 14,
    Series = 15,
};

// @brief a discrimated union representing a type in the arc programming language.
struct Type {
    /// @brief the kind of the type.
    TypeKind kind = TypeKind::Invalid;
    /// @brief the element type for channels or series.
    std::unique_ptr<Type> elem;

    explicit Type(xjson::Parser parser) {
        this->kind = parser.field<TypeKind>("kind");
        const auto elem_parser = parser.optional_child("elem");
        if (elem_parser.ok()) this->elem = std::make_unique<Type>(elem_parser);
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json j;
        j["kind"] = static_cast<uint8_t>(kind);
        if (elem) j["elem"] = elem->to_json();
        return j;
    }

    Type() = default;
    explicit Type(const TypeKind k): kind(k) {}
    Type(const TypeKind k, Type elem_type):
        kind(k), elem(std::make_unique<Type>(std::move(elem_type))) {}

    Type(const Type &other): kind(other.kind) {
        if (other.elem) { elem = std::make_unique<Type>(*other.elem); }
    }

    Type &operator=(const Type &other) {
        if (this != &other) {
            kind = other.kind;
            if (other.elem)
                elem = std::make_unique<Type>(*other.elem);
            else
                elem.reset();
        }
        return *this;
    }

    Type(Type &&) = default;
    Type &operator=(Type &&) = default;

    [[nodiscard]] size_t density() const {
        switch (kind) {
            case TypeKind::U8:
            case TypeKind::I8:
                return 1;
            case TypeKind::U16:
            case TypeKind::I16:
                return 2;
            case TypeKind::U32:
            case TypeKind::I32:
            case TypeKind::F32:
                return 4;
            case TypeKind::U64:
            case TypeKind::I64:
            case TypeKind::F64:
            case TypeKind::TimeStamp:
            case TypeKind::TimeSpan:
                return 8;
            default:
                return 0;
        }
    }

    [[nodiscard]] bool is_valid() const { return kind != TypeKind::Invalid; }

    /// @brief Convert arc IR type to telem data type.
    /// @return Corresponding telem data type.
    [[nodiscard]] telem::DataType telem() const;
};

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

/// @brief Edge connecting two handles in the dataflow graph.
struct Edge {
    Handle source, target; ///< Output parameter

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

    Edge() = default;
    Edge(Handle src, Handle tgt): source(std::move(src)), target(std::move(tgt)) {}

    bool operator==(const Edge &other) const {
        return source == other.source && target == other.target;
    }
};

struct Param {
    std::string name;
    Type type;
    nlohmann::json value;

    explicit Param(xjson::Parser parser) {
        this->name = parser.field<std::string>("name");
        this->type = parser.field<Type>("type");
        this->value = parser.field<nlohmann::json>("value", nlohmann::json(nullptr));
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json j;
        j["name"] = name;
        j["type"] = type.to_json();
        if (!value.is_null()) j["value"] = value;
        return j;
    }

    Param() = default;
};

/// @brief Named, ordered parameters for a function or node.
struct Params {
    std::vector<Param> params;

    Params() = default;
    explicit Params(std::vector<Param> p): params(std::move(p)) {}

    /// @brief Get parameter type by name, returns nullptr if not found
    [[nodiscard]] const Type *get(const std::string &name) const {
        for (const auto &p: this->params) {
            if (p.name == name) return &p.type;
        }
        return nullptr;
    }

    /// @brief Extract all parameter names
    [[nodiscard]] std::vector<std::string> keys() const {
        std::vector<std::string> result;
        result.reserve(this->params.size());
        for (const auto &p: this->params) {
            result.push_back(p.name);
        }
        return result;
    }

    /// @brief Convert params to JSON array
    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json arr = nlohmann::json::array();
        for (const auto &p: this->params) {
            arr.push_back(p.to_json());
        }
        return arr;
    }

    /// @brief Vector interface for iteration
    auto begin() { return this->params.begin(); }
    auto end() { return this->params.end(); }
    [[nodiscard]] auto begin() const { return this->params.begin(); }
    [[nodiscard]] auto end() const { return this->params.end(); }
    [[nodiscard]] size_t size() const { return this->params.size(); }
    [[nodiscard]] bool empty() const { return this->params.empty(); }

    Param &operator[](size_t index) { return params.at(index); }
    [[nodiscard]] const Param &operator[](size_t index) const {
        return params.at(index);
    }
};

struct Channels {
    std::map<uint32_t, std::string> read;
    std::map<uint32_t, std::string> write;

    explicit Channels(xjson::Parser parser) {
        this->read = parser.field<std::map<uint32_t, std::string>>("read", {});
        this->write = parser.field<std::map<uint32_t, std::string>>("write", {});
    }

    [[nodiscard]] nlohmann::json to_json() const {
        // nlohmann::json requires string keys in objects, so convert uint32_t to string
        nlohmann::json read_obj = nlohmann::json::object();
        for (const auto &[key, value]: read) {
            read_obj[std::to_string(key)] = value;
        }

        nlohmann::json write_obj = nlohmann::json::object();
        for (const auto &[key, value]: write) {
            write_obj[std::to_string(key)] = value;
        }

        return {{"read", read_obj}, {"write", write_obj}};
    }

    Channels() = default;
};

/// @brief Node instance in the dataflow graph.
struct Node {
    std::string key;
    std::string type;
    Channels channels;
    Params config, inputs, outputs;

    explicit Node(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->type = parser.field<std::string>("type");
        this->channels = parser.field<Channels>("channels");
        this->config = Params(parser.field<std::vector<Param>>("config"));
        this->inputs = Params(parser.field<std::vector<Param>>("inputs"));
        this->outputs = Params(parser.field<std::vector<Param>>("outputs"));
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

    Node() = default;
    explicit Node(std::string k): key(std::move(k)) {}
};

/// @brief Function template (stage definition).
struct Function {
    std::string key;
    Channels channels;
    Params config, inputs, outputs;

    explicit Function(xjson::Parser parser) {
        this->key = parser.field<std::string>("key");
        this->channels = parser.field<Channels>("channels");
        this->config = Params(parser.field<std::vector<Param>>("config"));
        this->inputs = Params(parser.field<std::vector<Param>>("inputs"));
        this->outputs = Params(parser.field<std::vector<Param>>("outputs"));
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

    Function() = default;
    explicit Function(std::string k): key(std::move(k)) {}
};

/// @brief Execution strata (layers for reactive scheduling).
struct Strata {
    std::vector<std::vector<std::string>> strata;

    explicit Strata(xjson::Parser parser) {
        this->strata = parser.field<std::vector<std::vector<std::string>>>("");
    }

    [[nodiscard]] nlohmann::json to_json() const { return strata; }

    Strata() = default;
};

/// @brief Complete Arc IR (dataflow graph).
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
        for (const auto &fn: functions) {
            functions_arr.push_back(fn.to_json());
        }

        nlohmann::json nodes_arr = nlohmann::json::array();
        for (const auto &node: nodes) {
            nodes_arr.push_back(node.to_json());
        }

        nlohmann::json edges_arr = nlohmann::json::array();
        for (const auto &edge: edges) {
            edges_arr.push_back(edge.to_json());
        }

        return {
            {"functions", functions_arr},
            {"nodes", nodes_arr},
            {"edges", edges_arr},
            {"strata", strata.to_json()}
        };
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
            if (e.source.node == node_key) { result.push_back(e); }
        return result;
    }

    [[nodiscard]] std::vector<Edge> incoming_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.target.node == node_key) { result.push_back(e); }
        return result;
    }
};

} // namespace ir
} // namespace arc
