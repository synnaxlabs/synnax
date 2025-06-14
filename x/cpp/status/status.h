// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>

/// internal
#include "nlohmann/json.hpp"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

/// @brief utility packages for managing status messages.
namespace status {
const std::string VARIANT_SUCCESS = "success";
const std::string VARIANT_ERROR = "error";
const std::string VARIANT_WARNING = "warning";
const std::string VARIANT_INFO = "info";
const std::string VARIANT_DISABLED = "disabled";
const std::string VARIANT_LOADING = "loading";

template<typename T>
struct Status {
    std::string key;
    std::string variant;
    std::string message;
    std::string description;
    telem::TimeStamp time = telem::TimeStamp::now();
    T details;

    static Status parse(xjson::Parser &parser) {
        return Status{
            .key = parser.required<std::string>("key"),
            .variant = parser.required<std::string>("variant"),
            .message = parser.required<std::string>("message"),
            .description = parser.required<std::string>("description"),
            .time = telem::TimeStamp(parser.required<std::int64_t>("time")),
            .details = T::parse(parser.required<json>("details")),
        };
    }

    [[nodiscard]] json to_json() const {
        json j;
        j["key"] = key;
        j["variant"] = variant;
        j["message"] = message;
        j["description"] = description;
        j["time"] = time.nanoseconds();
        j["details"] = details.to_json();
        return j;
    }
};
}
