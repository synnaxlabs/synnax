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

#include "nlohmann/json.hpp"

#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "x/go/status/x/go/status/status.pb.h"

/// @brief utility packages for managing status messages.
namespace x::status {
/// @brief a successful operation.
const std::string VARIANT_SUCCESS = "success";
/// @brief an operation that encountered an error.
const std::string VARIANT_ERROR = "error";
/// @brief an operation that encountered a warning.
const std::string VARIANT_WARNING = "warning";
/// @brief general informational message.
const std::string VARIANT_INFO = "info";
/// @brief an operation that is disabled.
const std::string VARIANT_DISABLED = "disabled";
/// @brief an operation that is in the process of loading or starting up.
const std::string VARIANT_LOADING = "loading";

struct DefaultDetails {
    static json::json to_json() { return json::json::object(); }

    static DefaultDetails parse(json::Parser &) { return DefaultDetails{}; }
};

/// @brief a standardized type for communicating status information across a Synnax
/// cluster.
/// @tparam Details - a custom details field that can be used to provide custom
/// information from a specific status provider. This type must implement the following
/// methods:
///
///     json to_json() - returns a nlohmann::json representation of the details.
///     static Details parse(json::Parser &parser) - parses a Details object from
///     its JSON representation.
template<typename Details = DefaultDetails>
struct Status {
    /// @brief a unique key for the status message.
    std::string key;
    /// @brief a human-readable name for the status.
    std::string name;
    /// @brief the variant of the status message. This should be one of the
    /// status::VARIANT_* constants.
    std::string variant;
    /// @brief a short, descriptive message about the status.
    std::string message;
    /// @brief optional longer description of the status.
    std::string description;
    /// @brief the time at which the status was created.
    telem::TimeStamp time = telem::TimeStamp(0);
    /// @brief custom details about the status.
    Details details = Details{};

    /// @brief parses a Status object from a JSON representation.
    static Status parse(json::Parser &parser) {
        auto details = parser.child("details");
        return Status{
            .key = parser.field<std::string>("key"),
            .name = parser.field<std::string>("name", ""),
            .variant = parser.field<std::string>("variant"),
            .message = parser.field<std::string>("message"),
            .description = parser.field<std::string>("description"),
            .time = telem::TimeStamp(parser.field<std::int64_t>("time")),
            .details = Details::parse(details),
        };
    }

    /// @brief converts the Status object to its JSON representation.
    [[nodiscard]] json::json to_json() const {
        json::json j;
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
    static std::pair<Status, errors::Error> from_proto(const ::status::PBStatus &pb) {
        Status status{
            .key = pb.key(),
            .name = pb.name(),
            .variant = pb.variant(),
            .message = pb.message(),
            .description = pb.description(),
            .time = telem::TimeStamp(pb.time()),
            .details = Details{},
        };
        if (!pb.details().empty()) {
            json::Parser parser(pb.details());
            status.details = Details::parse(parser);
            if (!parser.ok()) return {Status(), parser.error()};
        }
        return {status, errors::NIL};
    }

    /// @brief converts the Status to its protobuf representation.
    /// @param pb the protobuf message to encode the fields into.
    void to_proto(::status::PBStatus *pb) const {
        pb->set_key(key);
        pb->set_name(name);
        pb->set_variant(variant);
        pb->set_message(message);
        pb->set_description(description);
        pb->set_time(time.nanoseconds());
        pb->set_details(details.to_json().dump());
    }

    /// @brief returns true if the status is at its zero/default value.
    [[nodiscard]] bool is_zero() const {
        return key.empty() && name.empty() && variant.empty() && message.empty() &&
               description.empty() && time.nanoseconds() == 0;
    }
};
}
