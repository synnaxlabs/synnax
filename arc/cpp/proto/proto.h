// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <google/protobuf/struct.pb.h>
#include <nlohmann/json.hpp>

namespace arc::proto {

/// @brief Converts a google::protobuf::Value to nlohmann::json
/// @param v The protobuf Value to convert
/// @return The JSON representation
inline nlohmann::json pb_value_to_json(const google::protobuf::Value &v) {
    switch (v.kind_case()) {
        case google::protobuf::Value::kNullValue:
            return nullptr;
        case google::protobuf::Value::kNumberValue:
            return v.number_value();
        case google::protobuf::Value::kStringValue:
            return v.string_value();
        case google::protobuf::Value::kBoolValue:
            return v.bool_value();
        case google::protobuf::Value::kStructValue: {
            nlohmann::json obj = nlohmann::json::object();
            for (const auto &[key, value]: v.struct_value().fields()) {
                obj[key] = pb_value_to_json(value);
            }
            return obj;
        }
        case google::protobuf::Value::kListValue: {
            nlohmann::json arr = nlohmann::json::array();
            for (const auto &elem: v.list_value().values()) {
                arr.push_back(pb_value_to_json(elem));
            }
            return arr;
        }
        default:
            return nullptr;
    }
}

/// @brief Converts nlohmann::json to google::protobuf::Value
/// @param j The JSON value to convert
/// @param v Pointer to the protobuf Value to populate
inline void json_to_pb_value(const nlohmann::json &j, google::protobuf::Value *v) {
    if (j.is_null()) {
        v->set_null_value(google::protobuf::NULL_VALUE);
    } else if (j.is_boolean()) {
        v->set_bool_value(j.get<bool>());
    } else if (j.is_number_integer()) {
        v->set_number_value(static_cast<double>(j.get<std::int64_t>()));
    } else if (j.is_number_unsigned()) {
        v->set_number_value(static_cast<double>(j.get<std::uint64_t>()));
    } else if (j.is_number_float()) {
        v->set_number_value(j.get<double>());
    } else if (j.is_string()) {
        v->set_string_value(j.get<std::string>());
    } else if (j.is_object()) {
        auto *struct_value = v->mutable_struct_value();
        for (auto it = j.begin(); it != j.end(); ++it) {
            json_to_pb_value(it.value(), &(*struct_value->mutable_fields())[it.key()]);
        }
    } else if (j.is_array()) {
        auto *list_value = v->mutable_list_value();
        for (const auto &elem: j) {
            json_to_pb_value(elem, list_value->add_values());
        }
    }
}

}
