// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>

#include "google/protobuf/struct.pb.h"
#include "google/protobuf/util/json_util.h"
#include "nlohmann/json.hpp"

#include "x/cpp/xerrors/errors.h"

using json = nlohmann::json;

namespace xjson {

/// @brief Converts a google::protobuf::Struct to nlohmann::json.
/// @param pb The protobuf Struct to convert.
/// @returns A pair containing the JSON and an error if one occurred.
inline std::pair<json, xerrors::Error>
from_struct(const google::protobuf::Struct &pb) {
    std::string json_str;
    const auto status = google::protobuf::util::MessageToJsonString(pb, &json_str);
    if (!status.ok())
        return {{}, xerrors::Error(xerrors::VALIDATION, std::string(status.message()))};
    try {
        return {json::parse(json_str), xerrors::NIL};
    } catch (const json::parse_error &e) {
        return {{}, xerrors::Error(xerrors::VALIDATION, e.what())};
    }
}

/// @brief Converts nlohmann::json to a google::protobuf::Struct.
/// @param j The JSON to convert.
/// @returns A pair containing the Struct and an error if one occurred.
inline std::pair<google::protobuf::Struct, xerrors::Error> to_struct(const json &j) {
    google::protobuf::Struct pb;
    const auto status =
        google::protobuf::util::JsonStringToMessage(j.dump(), &pb);
    if (!status.ok())
        return {{}, xerrors::Error(xerrors::VALIDATION, std::string(status.message()))};
    return {pb, xerrors::NIL};
}

/// @brief Converts nlohmann::json to a google::protobuf::Struct, setting it on
/// the provided pointer. Returns an error if conversion fails.
/// @param j The JSON to convert.
/// @param pb Pointer to the Struct to populate.
/// @returns An error if the conversion failed.
inline xerrors::Error to_struct(const json &j, google::protobuf::Struct *pb) {
    const auto status =
        google::protobuf::util::JsonStringToMessage(j.dump(), pb);
    if (!status.ok())
        return xerrors::Error(xerrors::VALIDATION, std::string(status.message()));
    return xerrors::NIL;
}

}  // namespace xjson
