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
#include <string>
#include <vector>

#include "nlohmann/json.hpp"

#include "x/cpp/map/insertion.h"

namespace arc::ir {
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

struct Type {
    TypeKind kind = TypeKind::Invalid;
    std::unique_ptr<Type> elem; ///< For series/chan element type

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

    size_t density() const;

    bool is_valid() const { return kind != TypeKind::Invalid; }
};

using Params = map::Insertion<Type>;

struct Handle {
    std::string node, param;

    Handle() = default;
    Handle(std::string n, std::string p): node(std::move(n)), param(std::move(p)) {}

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

    Edge() = default;
    Edge(Handle src, Handle tgt): source(std::move(src)), target(std::move(tgt)) {}

    bool operator==(const Edge &other) const {
        return source == other.source && target == other.target;
    }
};

/// @brief Channel references in a node.
struct Channels {
    std::map<uint32_t, std::string> read; ///< ChannelKey → param name
    std::map<std::string, uint32_t> write; ///< param name → ChannelKey

    /// @brief Check if node reads any channels.
    bool has_reads() const { return !read.empty(); }

    /// @brief Check if node writes any channels.
    bool has_writes() const { return !write.empty(); }
};

/// @brief Node instance in the dataflow graph.
struct Node {
    std::string key; ///< Unique node identifier
    std::string type; ///< Function type name
    std::map<std::string, nlohmann::json> config_values; ///< Runtime configuration
    Channels channels; ///< Channel references
    Params config, inputs, outputs; ///< Config parameter types

    Node() = default;
    explicit Node(std::string k): key(std::move(k)) {}
};

/// @brief Function template (stage definition).
struct Function {
    std::string key;
    std::string raw_body;
    Params config, inputs, outputs;
    Channels channels;

    Function() = default;
    explicit Function(std::string k): key(std::move(k)) {}
};

/// @brief Execution strata (layers for reactive scheduling).
using Strata = std::vector<std::vector<std::string>>;

/// @brief Complete Arc IR (dataflow graph).
struct IR {
    std::vector<Function> functions; ///< Function templates
    std::vector<Node> nodes; ///< Node instances
    std::vector<Edge> edges; ///< Dataflow connections
    Strata strata; ///< Execution layers

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
