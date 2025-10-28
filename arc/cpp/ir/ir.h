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
#include <nlohmann/json_fwd.hpp>
#include <string>
#include <vector>

namespace arc {
namespace ir {

// Forward declaration
using ChannelKey = std::uint32_t;

/// @brief Arc type kinds (matches Go's types.Kind).
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

/// @brief Arc type with kind and optional element type.
struct Type {
    TypeKind kind = TypeKind::Invalid;
    std::unique_ptr<Type> elem;  ///< For series/chan element type

    Type() = default;
    explicit Type(TypeKind k) : kind(k) {}
    Type(TypeKind k, Type elem_type)
        : kind(k), elem(std::make_unique<Type>(std::move(elem_type))) {}

    // Custom copy constructor for deep copy
    Type(const Type &other) : kind(other.kind) {
        if (other.elem) {
            elem = std::make_unique<Type>(*other.elem);
        }
    }

    // Custom copy assignment
    Type &operator=(const Type &other) {
        if (this != &other) {
            kind = other.kind;
            if (other.elem) {
                elem = std::make_unique<Type>(*other.elem);
            } else {
                elem.reset();
            }
        }
        return *this;
    }

    // Default move operations
    Type(Type &&) = default;
    Type &operator=(Type &&) = default;

    /// @brief Get byte size for fixed-size types.
    size_t density() const;

    /// @brief Check if type is valid.
    bool is_valid() const { return kind != TypeKind::Invalid; }
};

/// @brief Named parameters (inputs/outputs/config).
struct Params {
    std::vector<std::string> keys;   ///< Parameter names in order
    std::map<std::string, Type> values;  ///< Name → Type mapping

    /// @brief Get number of parameters.
    size_t count() const { return keys.size(); }

    /// @brief Check if parameter exists.
    bool contains(const std::string &name) const {
        return values.find(name) != values.end();
    }

    /// @brief Get type for parameter.
    const Type *get(const std::string &name) const {
        auto it = values.find(name);
        if (it == values.end()) return nullptr;
        return &it->second;
    }
};

/// @brief Handle referencing a node's parameter.
struct Handle {
    std::string node;   ///< Node key
    std::string param;  ///< Parameter name

    Handle() = default;
    Handle(std::string n, std::string p) : node(std::move(n)), param(std::move(p)) {}
};

/// @brief Edge connecting two handles in the dataflow graph.
struct Edge {
    Handle source;  ///< Output parameter
    Handle target;  ///< Input parameter

    Edge() = default;
    Edge(Handle src, Handle tgt) : source(std::move(src)), target(std::move(tgt)) {}
};

/// @brief Channel references in a node.
struct Channels {
    std::map<uint32_t, std::string> read;   ///< ChannelKey → param name
    std::map<std::string, uint32_t> write;  ///< param name → ChannelKey

    /// @brief Check if node reads any channels.
    bool has_reads() const { return !read.empty(); }

    /// @brief Check if node writes any channels.
    bool has_writes() const { return !write.empty(); }
};

/// @brief Node instance in the dataflow graph.
struct Node {
    std::string key;                      ///< Unique node identifier
    std::string type;                     ///< Function type name
    std::map<std::string, nlohmann::json> config_values;  ///< Runtime configuration
    Channels channels;                    ///< Channel references
    Params config;                        ///< Config parameter types
    Params inputs;                        ///< Input parameter types
    Params outputs;                       ///< Output parameter types

    Node() = default;
    explicit Node(std::string k) : key(std::move(k)) {}
};

/// @brief Function template (stage definition).
struct Function {
    std::string key;      ///< Function name
    std::string raw_body; ///< Original source code
    Params config;        ///< Config parameters
    Params inputs;        ///< Input parameters
    Params outputs;       ///< Output parameters
    Channels channels;    ///< Channel references

    Function() = default;
    explicit Function(std::string k) : key(std::move(k)) {}
};

/// @brief Execution strata (layers for reactive scheduling).
using Strata = std::vector<std::vector<std::string>>;

/// @brief Complete Arc IR (dataflow graph).
struct IR {
    std::vector<Function> functions;  ///< Function templates
    std::vector<Node> nodes;          ///< Node instances
    std::vector<Edge> edges;          ///< Dataflow connections
    Strata strata;                    ///< Execution layers

    /// @brief Find function by key.
    const Function *find_function(const std::string &key) const {
        for (const auto &fn : functions) {
            if (fn.key == key) return &fn;
        }
        return nullptr;
    }

    /// @brief Find node by key.
    const Node *find_node(const std::string &key) const {
        for (const auto &n : nodes) {
            if (n.key == key) return &n;
        }
        return nullptr;
    }

    /// @brief Get edges from a node.
    std::vector<Edge> outgoing_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e : edges) {
            if (e.source.node == node_key) {
                result.push_back(e);
            }
        }
        return result;
    }

    /// @brief Get edges to a node.
    std::vector<Edge> incoming_edges(const std::string &node_key) const {
        std::vector<Edge> result;
        for (const auto &e : edges) {
            if (e.target.node == node_key) {
                result.push_back(e);
            }
        }
        return result;
    }
};

// ============================================================================
// Parsing Helpers (used by JSON serializers)
// ============================================================================

/// @brief Parse Type from JSON value.
Type parse_type(const nlohmann::json &j);

/// @brief Parse Params from JSON object.
Params parse_params(const nlohmann::json &j);

/// @brief Parse Channels from JSON object.
Channels parse_channels(const nlohmann::json &j);

}  // namespace ir
}  // namespace arc
