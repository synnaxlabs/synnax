// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <stdexcept>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/program/program.h"

namespace arc::runtime::testutil {
/// @brief replaces every occurrence of from in s with to.
inline std::string
replace_all(std::string s, const std::string &from, const std::string &to) {
    for (size_t at = s.find(from); at != std::string::npos; at = s.find(from, at))
        s.replace(at, from.size(), to);
    return s;
}

/// @brief a channel a test program reads or writes, named as its source refers to it.
struct ChannelSpec {
    std::string name;
    x::telem::DataType data_type;
};

/// @brief the channels a test program runs against, created on the cluster under
/// generated names so repeated runs never collide.
class Channels {
    std::unordered_map<std::string, synnax::channel::Channel> created;

public:
    Channels(const synnax::Synnax &client, const std::vector<ChannelSpec> &specs) {
        for (const auto &[name, data_type]: specs) {
            auto [ch, err] = client.channels.create(
                make_unique_channel_name(name),
                data_type
            );
            if (err)
                throw std::runtime_error(
                    "Failed to create channel " + name + ": " + err.message()
                );
            this->created.emplace(name, std::move(ch));
        }
    }

    /// @brief rewrites every %name% placeholder in source to the generated name of the
    /// channel declared as name.
    [[nodiscard]] std::string substitute(std::string source) const {
        for (const auto &[name, ch]: this->created)
            source = replace_all(std::move(source), "%" + name + "%", ch.name);
        return source;
    }

    /// @brief returns the key of the channel declared as name.
    [[nodiscard]] synnax::channel::Key key(const std::string &name) const {
        const auto it = this->created.find(name);
        if (it == this->created.end())
            throw std::runtime_error("channel not declared: " + name);
        return it->second.key;
    }
};

/// @brief Compiles an Arc program via the Synnax client.
inline arc::program::Program
compile_text(const synnax::Synnax &client, const std::string &source) {
    auto arc = synnax::arc::Arc{
        .name = make_unique_channel_name("test_arc"),
        .mode = synnax::arc::MODE_TEXT
    };
    arc.text.raw = source;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::arc::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    if (!compiled.program.has_value())
        throw std::runtime_error("Compiled arc has no program");
    return *compiled.program;
}
}
