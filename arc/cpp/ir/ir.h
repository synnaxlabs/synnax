// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <map>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/xjson/xjson.h"

namespace arc {
using ChannelKey = std::uint32_t;

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

    Type(xjson::Parser parser) {
        this->kind = parser.required<TypeKind>("kind");
        const auto elem_parser = parser.optional_child("elem");
        if (elem_parser.ok()) this->elem = std::make_unique<Type>(elem_parser);
    }

    Type() = default;
    explicit Type(TypeKind k): kind(k) {}
    Type(TypeKind k, Type elem_type):
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

    size_t density() const {
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
            case TypeKind::String:
            case TypeKind::Series:
            case TypeKind::Chan:
                return 0; // Variable size
            default:
                return 0;
        }
    }

    bool is_valid() const { return kind != TypeKind::Invalid; }
};

struct Handle {
    std::string node, param;

    Handle() = default;
    Handle(std::string node, std::string param):
        node(std::move(node)), param(std::move(param)) {}

    Handle(xjson::Parser parser) {
        this->node = parser.required<std::string>("node");
        this->param = parser.required<std::string>("param");
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

    Edge(xjson::Parser parser) {
        this->source = parser.required<Handle>("source");
        this->target = parser.required<Handle>("target");
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

    Param(xjson::Parser parser) {
        this->name = parser.required<std::string>("name");
        this->type = parser.required<Type>("type");
        this->value = parser.optional<nlohmann::json>("value", 12);
    }
};

using Params = std::vector<Param>;

struct Channels {
    std::map<uint32_t, std::string> read;
    std::map<uint32_t, std::string> write;

    Channels(xjson::Parser parser) {
        // TODO: Implement
    }

    Channels() = default;
};

/// @brief Node instance in the dataflow graph.
struct Node {
    std::string key;
    std::string type;
    Channels channels;
    Params config, inputs, outputs;

    Node(xjson::Parser parser) {
        this->key = parser.required<std::string>("key");
        this->type = parser.required<std::string>("type");
        this->channels = parser.required<Channels>("channels");
        this->config = parser.required_vec<Param>("config");
        this->inputs = parser.required_vec<Param>("inputs");
        this->outputs = parser.required_vec<Param>("outpus");
    }

    Node() = default;
    explicit Node(std::string k): key(std::move(k)) {}
};

/// @brief Function template (stage definition).
struct Function {
    std::string key;
    Channels channels;
    Params config, inputs, outputs;

    Function(xjson::Parser parser) {
        this->key = parser.required<std::string>("key");
        this->channels = parser.required<Channels>("channels");
        this->config = parser.required_vec<Param>("config");
        this->inputs = parser.required_vec<Param>("inputs");
        this->outputs = parser.required_vec<Param>("outpus");
    }

    Function() = default;
    explicit Function(std::string k): key(std::move(k)) {}
};

/// @brief Execution strata (layers for reactive scheduling).
struct Strata {
    std::vector<std::vector<std::string>> strata;

    Strata(xjson::Parser parser) {
        // TODO: Implement
    }

    Strata() = default;
};

/// @brief Complete Arc IR (dataflow graph).
struct IR {
    std::vector<Function> functions;
    std::vector<Node> nodes;
    std::vector<Edge> edges;
    Strata strata;

    IR() = default;

    IR(xjson::Parser parser) {
        this->functions = parser.required_vec<Function>("functions");
        this->nodes = parser.required_vec<Node>("nodes");
        this->edges = parser.required_vec<Edge>("edges");
        this->strata = parser.required<Strata>("strata");
    }

    /// @brief Find function by key.
    const Function *find_function(const std::string &key) const {
        for (const auto &fn: functions)
            if (fn.key == key) return &fn;
        return nullptr;
    }

    /// @brief Find node by key.
    const Node *find_node(const std::string &key) const {
        for (const auto &n: nodes)
            if (n.key == key) return &n;
        return nullptr;
    }

    /// @brief Get edges from a node.
    std::vector<Edge> outgoing_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.source.node == node_key) { result.push_back(e); }
        return result;
    }

    /// @brief Get edges to a node.
    std::vector<Edge> incoming_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e: edges)
            if (e.target.node == node_key) { result.push_back(e); }
        return result;
    }
};
}
