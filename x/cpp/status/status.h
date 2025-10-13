// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "nlohmann/json.hpp"

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

/// @brief utility packages for managing status messages.
namespace status {
namespace variant {
/// @brief a successful operation.
const std::string SUCCESS = "success";
/// @brief an operation that encountered an error.
/// Named ERR instead of ERROR to avoid confusion with the C++ `error` keyword.
const std::string ERR = "error";
/// @brief an operation that encountered a warning.
const std::string WARNING = "warning";
/// @brief general informational message.
const std::string INFO = "info";
/// @brief an operation that is disabled.
const std::string DISABLED = "disabled";
/// @brief an operation that is in the process of loading or starting up.
const std::string LOADING = "loading";
}

struct DefaultDetails {
    static json to_json() { return json::object(); }

    static DefaultDetails parse(xjson::Parser &) { return DefaultDetails{}; }
};

/// @brief a standardized type for communicating status information across a Synnax
/// cluster.
/// @tparam Details - a custom details field that can be used to provide custom
/// information from a specific status provider. This type must implement the following
/// methods:
///
///     json to_json() - returns a nlohmann::json representation of the details.
///     static Details parse(xjson::Parser &parser) - parses a Details object from
///     its JSON representation.
template<typename Details = DefaultDetails>
struct Status {
    /// @brief a unique key for the status message.
    std::string key;
    /// @brief the variant of the status message. This should be one of the
    /// status::variant::* constants.
    std::string variant;
    /// @brief a short, descriptive message about the status.
    std::string message;
    /// @brief optional longer description of the status.
    std::string description;
    /// @brief the time at which the status was created.
    telem::TimeStamp time = telem::TimeStamp::now();
    /// @brief custom details about the status.
    Details details;

    /// @brief parses a Status object from a JSON representation.
    static Status parse(xjson::Parser &parser) {
        auto details = parser.child("details");
        return Status{
            .key = parser.required<std::string>("key"),
            .variant = parser.required<std::string>("variant"),
            .message = parser.required<std::string>("message"),
            .description = parser.required<std::string>("description"),
            .time = telem::TimeStamp(parser.required<std::int64_t>("time")),
            .details = Details::parse(details),
        };
    }

    /// @brief converts the Status object to its JSON representation.
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
