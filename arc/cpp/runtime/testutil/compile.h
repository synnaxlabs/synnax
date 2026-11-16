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
#include <variant>
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
    /// @brief the declared name of the index channel; empty for a virtual channel.
    std::string index;
    /// @brief true when this spec declares an index channel.
    bool is_index = false;
};

/// @brief the channels a test program runs against, created on the cluster under
/// generated names so repeated runs never collide. An index is declared before the
/// channels it indexes.
class Channels {
    std::unordered_map<std::string, synnax::channel::Channel> created;

    [[nodiscard]] std::pair<synnax::channel::Channel, x::errors::Error>
    create(const synnax::Synnax &client, const ChannelSpec &spec) const {
        const auto name = make_unique_channel_name(spec.name);
        if (spec.is_index) return client.channels.create(name, spec.data_type, 0, true);
        if (spec.index.empty()) return client.channels.create(name, spec.data_type);
        return client.channels.create(name, spec.data_type, this->key(spec.index));
    }

public:
    Channels(const synnax::Synnax &client, const std::vector<ChannelSpec> &specs) {
        for (const auto &spec: specs) {
            auto [ch, err] = this->create(client, spec);
            if (err)
                throw std::runtime_error(
                    "Failed to create channel " + spec.name + ": " + err.message()
                );
            this->created.emplace(spec.name, std::move(ch));
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
    synnax::arc::AutomationArc automation{
        {.name = make_unique_channel_name("test_arc")}
    };
    automation.source = synnax::arc::TextSource{.text = {.raw = source}};
    synnax::arc::Arc arc = automation;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::arc::RetrieveOptions opts;
    opts.include_program = true;
    auto [compiled, err] = client.arcs.retrieve(synnax::arc::base(arc).key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    const auto &program = std::get<synnax::arc::AutomationArc>(compiled).program;
    if (!program.has_value()) throw std::runtime_error("Compiled arc has no program");
    return *program;
}
}
