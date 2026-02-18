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

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

namespace x::json {
/// @brief Converts a google::protobuf::Struct to json.
/// @param pb The protobuf Struct to convert.
/// @returns A pair containing the JSON and an error if one occurred.
inline std::pair<json, errors::Error> from_struct(const google::protobuf::Struct &pb) {
    std::string json_str;
    const auto status = google::protobuf::util::MessageToJsonString(pb, &json_str);
    if (!status.ok())
        return {{}, errors::Error(errors::VALIDATION, std::string(status.message()))};
    try {
        return {nlohmann::json::parse(json_str), errors::NIL};
    } catch (const nlohmann::json::parse_error &e) {
        return {{}, errors::Error(errors::VALIDATION, e.what())};
    }
}

/// @brief Converts json to a google::protobuf::Struct.
/// @param j The JSON to convert.
/// @returns A pair containing the Struct and an error if one occurred.
inline std::pair<google::protobuf::Struct, errors::Error>
to_struct(const nlohmann::json &j) {
    google::protobuf::Struct pb;
    const auto status = google::protobuf::util::JsonStringToMessage(j.dump(), &pb);
    if (!status.ok())
        return {{}, errors::Error(errors::VALIDATION, std::string(status.message()))};
    return {pb, errors::NIL};
}

/// @brief Converts json to a google::protobuf::Struct, setting it on
/// the provided pointer. Returns an error if conversion fails.
/// @param j The JSON to convert.
/// @param pb Pointer to the Struct to populate.
/// @returns An error if the conversion failed.
inline errors::Error to_struct(const nlohmann::json &j, google::protobuf::Struct *pb) {
    const auto status = google::protobuf::util::JsonStringToMessage(j.dump(), pb);
    if (!status.ok())
        return errors::Error(errors::VALIDATION, std::string(status.message()));
    return errors::NIL;
}
}
