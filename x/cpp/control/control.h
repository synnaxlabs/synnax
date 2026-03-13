// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "x/cpp/json/json.h"

#include "x/go/control/x/go/control/control.pb.h"

namespace x::control {
using Authority = std::uint8_t;
constexpr Authority AUTHORITY_ABSOLUTE = 255;

struct Subject {
    std::string name;
    std::string key;

    void to_proto(::control::ControlSubject *s) const {
        s->set_name(name);
        s->set_key(key);
    }

    bool operator==(const Subject &other) const { return this->key == other.key; }

    bool operator!=(const Subject &other) const { return !(*this == other); }
};

/// @brief per-channel authority state as reported by the server.
struct State {
    /// @brief the channel key this state applies to.
    std::uint32_t resource;
    /// @brief the subject holding authority.
    Subject subject;
    /// @brief the authority level held.
    Authority authority;

    static State parse(x::json::Parser &parser) {
        auto sub = parser.child("subject");
        return {
            .resource = parser.field<std::uint32_t>("resource"),
            .subject =
                {
                    .name = sub.field<std::string>("name"),
                    .key = sub.field<std::string>("key"),
                },
            .authority = parser.field<Authority>("authority"),
        };
    }
};

/// @brief a transfer of authority from one subject to another on a channel.
struct Transfer {
    /// @brief the previous authority holder. Null on initial acquire.
    std::optional<State> from;
    /// @brief the new authority holder. Null on release.
    std::optional<State> to;

    static Transfer parse(x::json::Parser &parser) {
        Transfer t;
        auto j = parser.get_json();
        if (j.contains("from") && !j["from"].is_null()) {
            auto from_parser = parser.child("from");
            t.from = State::parse(from_parser);
        }
        if (j.contains("to") && !j["to"].is_null()) {
            auto to_parser = parser.child("to");
            t.to = State::parse(to_parser);
        }
        return t;
    }
};

/// @brief a batch of authority transfers, matching the server's wire format.
struct Update {
    std::vector<Transfer> transfers;

    static Update parse(x::json::Parser &parser) {
        Update u;
        parser.iter("transfers", [&u](x::json::Parser &tp) {
            u.transfers.push_back(Transfer::parse(tp));
        });
        return u;
    }
};
}
