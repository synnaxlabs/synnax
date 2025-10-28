// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/ir/ir.h"

#include <nlohmann/json.hpp>

namespace arc {
namespace ir {

size_t Type::density() const {
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
        return 0;  // Variable size
    default:
        return 0;
    }
}

// ============================================================================
// JSON Parsing
// ============================================================================

// Helper: Parse TypeKind from string
TypeKind parse_type_kind(const std::string &kind_str) {
    if (kind_str == "u8") return TypeKind::U8;
    if (kind_str == "u16") return TypeKind::U16;
    if (kind_str == "u32") return TypeKind::U32;
    if (kind_str == "u64") return TypeKind::U64;
    if (kind_str == "i8") return TypeKind::I8;
    if (kind_str == "i16") return TypeKind::I16;
    if (kind_str == "i32") return TypeKind::I32;
    if (kind_str == "i64") return TypeKind::I64;
    if (kind_str == "f32") return TypeKind::F32;
    if (kind_str == "f64") return TypeKind::F64;
    if (kind_str == "string") return TypeKind::String;
    if (kind_str == "timestamp") return TypeKind::TimeStamp;
    if (kind_str == "timespan") return TypeKind::TimeSpan;
    if (kind_str == "chan") return TypeKind::Chan;
    if (kind_str == "series") return TypeKind::Series;
    return TypeKind::Invalid;
}

// Parse Type from JSON
Type parse_type(const nlohmann::json &j) {
    if (j.is_string()) {
        return Type(parse_type_kind(j.get<std::string>()));
    }
    // TODO: Handle complex types (series[f64], chan[i32], etc.)
    return Type{};
}

// Parse Params from JSON
Params parse_params(const nlohmann::json &j) {
    Params params;
    if (j.is_null() || !j.is_object()) return params;

    for (auto &[key, type_json] : j.items()) {
        params.keys.push_back(key);
        params.values[key] = parse_type(type_json);
    }
    return params;
}

// Parse Channels from JSON
Channels parse_channels(const nlohmann::json &j) {
    Channels channels;
    if (j.is_null() || !j.is_object()) return channels;

    if (j.contains("read") && j["read"].is_object()) {
        for (auto &[key_str, param] : j["read"].items()) {
            uint32_t chan_key = std::stoul(key_str);
            channels.read[chan_key] = param.get<std::string>();
        }
    }

    if (j.contains("write") && j["write"].is_object()) {
        for (auto &[param, key_val] : j["write"].items()) {
            channels.write[param] = key_val.get<uint32_t>();
        }
    }

    return channels;
}

}  // namespace ir
}  // namespace arc

// ============================================================================
// JSON Serialization (nlohmann::json integration)
// ============================================================================

namespace nlohmann {

// Type serialization
template <>
struct adl_serializer<arc::ir::Type> {
    static void from_json(const json &j, arc::ir::Type &t) {
        t = arc::ir::parse_type(j);
    }
};

// Handle serialization
template <>
struct adl_serializer<arc::ir::Handle> {
    static void from_json(const json &j, arc::ir::Handle &h) {
        if (j.contains("node")) h.node = j["node"].get<std::string>();
        if (j.contains("param")) h.param = j["param"].get<std::string>();
    }
};

// Edge serialization
template <>
struct adl_serializer<arc::ir::Edge> {
    static void from_json(const json &j, arc::ir::Edge &e) {
        if (j.contains("source")) e.source = j["source"].get<arc::ir::Handle>();
        if (j.contains("target")) e.target = j["target"].get<arc::ir::Handle>();
    }
};

// Node serialization
template <>
struct adl_serializer<arc::ir::Node> {
    static void from_json(const json &j, arc::ir::Node &n) {
        if (j.contains("key")) n.key = j["key"].get<std::string>();
        if (j.contains("type")) n.type = j["type"].get<std::string>();

        if (j.contains("config_values") && j["config_values"].is_object()) {
            for (auto &[k, v] : j["config_values"].items()) {
                n.config_values[k] = v;
            }
        }

        if (j.contains("channels")) {
            n.channels = arc::ir::parse_channels(j["channels"]);
        }

        if (j.contains("config")) {
            n.config = arc::ir::parse_params(j["config"]);
        }

        if (j.contains("inputs")) {
            n.inputs = arc::ir::parse_params(j["inputs"]);
        }

        if (j.contains("outputs")) {
            n.outputs = arc::ir::parse_params(j["outputs"]);
        }
    }
};

// Function serialization
template <>
struct adl_serializer<arc::ir::Function> {
    static void from_json(const json &j, arc::ir::Function &fn) {
        if (j.contains("key")) fn.key = j["key"].get<std::string>();

        if (j.contains("body") && j["body"].is_object()) {
            if (j["body"].contains("raw")) {
                fn.raw_body = j["body"]["raw"].get<std::string>();
            }
        }

        if (j.contains("channels")) {
            fn.channels = arc::ir::parse_channels(j["channels"]);
        }

        if (j.contains("config")) {
            fn.config = arc::ir::parse_params(j["config"]);
        }

        if (j.contains("inputs")) {
            fn.inputs = arc::ir::parse_params(j["inputs"]);
        }

        if (j.contains("outputs")) {
            fn.outputs = arc::ir::parse_params(j["outputs"]);
        }
    }
};

// IR serialization
template <>
struct adl_serializer<arc::ir::IR> {
    static void from_json(const json &j, arc::ir::IR &ir) {
        if (j.contains("functions") && j["functions"].is_array()) {
            ir.functions = j["functions"].get<std::vector<arc::ir::Function>>();
        }

        if (j.contains("nodes") && j["nodes"].is_array()) {
            ir.nodes = j["nodes"].get<std::vector<arc::ir::Node>>();
        }

        if (j.contains("edges") && j["edges"].is_array()) {
            ir.edges = j["edges"].get<std::vector<arc::ir::Edge>>();
        }

        if (j.contains("strata") && j["strata"].is_array()) {
            ir.strata = j["strata"].get<arc::ir::Strata>();
        }
    }
};

}  // namespace nlohmann
