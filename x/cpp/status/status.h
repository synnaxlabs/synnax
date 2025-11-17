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

#include "x/go/status/x/go/status/status.pb.h"

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
    std::string key = "";
    /// @brief a human-readable name for the status.
    std::string name = "";
    /// @brief the variant of the status message. This should be one of the
    /// status::variant::* constants.
    std::string variant = "";
    /// @brief a short, descriptive message about the status.
    std::string message = "";
    /// @brief optional longer description of the status.
    std::string description = "";
    /// @brief the time at which the status was created.
    telem::TimeStamp time = telem::TimeStamp(0);
    /// @brief custom details about the status.
    Details details = Details{};

    /// @brief default constructor.
    Status() = default;

    /// @brief parses a Status object from a JSON representation.
    static Status parse(xjson::Parser &parser) {
        auto details = parser.child("details");
        return Status{
            .key = parser.required<std::string>("key"),
            .name = parser.optional<std::string>("name", ""),
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
        j["name"] = name;
        j["variant"] = variant;
        j["message"] = message;
        j["description"] = description;
        j["time"] = time.nanoseconds();
        j["details"] = details.to_json();
        return j;
    }

    /// @brief constructs a Status from its protobuf representation.
    explicit Status(const ::status::PBStatus &pb):
        key(pb.key()),
        name(pb.name()),
        variant(pb.variant()),
        message(pb.message()),
        description(pb.description()),
        time(telem::TimeStamp(pb.time())),
        details(Details{}) {}

    /// @brief converts the Status to its protobuf representation.
    /// @param pb the protobuf message to encode the fields into.
    void to_proto(::status::PBStatus *pb) const {
        pb->set_key(key);
        pb->set_name(name);
        pb->set_variant(variant);
        pb->set_message(message);
        pb->set_description(description);
        pb->set_time(time.nanoseconds());
    }
};
}
