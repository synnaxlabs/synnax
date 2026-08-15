// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <optional>
#include <string>
#include <type_traits>
#include <variant>
#include <vector>

#include "client/cpp/ethercat/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::channel {
/// @brief base class for EtherCAT PDO channel configurations.
/// Inherits PDO addressing from pdo::Entry.
struct Channel : pdo::Entry {
    /// @brief whether this channel is enabled for data exchange.
    bool enabled;
    /// @brief the key of the slave device in Synnax.
    std::string device_key;

    virtual ~Channel() = default;

protected:
    Channel(
        const slave::Properties &slave,
        const bool is_input,
        const bool disabled,
        std::string device
    ):
        pdo::Entry{slave.position, 0, 0, 0, is_input, x::telem::UNKNOWN_T},
        enabled(!disabled),
        device_key(std::move(device)) {}

    /// @brief copies an inline PDO address into the entry.
    void bind_address(const ::synnax::ethercat::PDOAddress &addr) {
        this->index = addr.index;
        this->sub_index = addr.sub_index;
        this->bit_length = addr.bit_length;
        this->data_type = addr.data_type;
    }

    /// @brief copies a discovered PDO's address into the entry, accumulating a
    /// field error under the given path prefix when the PDO was not found.
    void resolve_pdo(
        const std::optional<pdo::Properties> &pdo,
        const std::string &name,
        const x::json::Parser &parser,
        const std::string &path
    ) {
        if (!pdo)
            return parser.field_err(
                path + "pdo",
                "PDO '" + name + "' not found in slave"
            );
        this->index = pdo->index;
        this->sub_index = pdo->sub_index;
        this->bit_length = pdo->bit_length;
        this->data_type = pdo->data_type;
    }
};

/// @brief input channel (TxPDO, slave->master).
struct Input final : Channel {
    /// @brief the key of the Synnax channel to write data to.
    synnax::channel::Key synnax_key;
    /// @brief the Synnax channel object (populated after remote lookup).
    synnax::channel::Channel ch;

    Input(
        const ::synnax::ethercat::InputChannelAutomatic &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, true, cfg.disabled, cfg.device), synnax_key(cfg.channel) {
        this->resolve_pdo(slave.find_input_pdo(cfg.pdo), cfg.pdo, parser, path);
    }

    Input(
        const ::synnax::ethercat::InputChannelManual &cfg,
        const slave::Properties &slave
    ):
        Channel(slave, true, cfg.disabled, cfg.device), synnax_key(cfg.channel) {
        this->bind_address(cfg);
    }

    /// @brief binds remote channel information retrieved from Synnax.
    void bind_remote_info(const synnax::channel::Channel &remote_ch) {
        this->ch = remote_ch;
    }
};

/// @brief output channel (RxPDO, master->slave).
struct Output final : Channel {
    /// @brief the key of the Synnax channel to receive commands from.
    synnax::channel::Key command_key;
    /// @brief the key of the Synnax channel to write state feedback to.
    synnax::channel::Key state_key;
    /// @brief the Synnax state channel object (populated after remote lookup).
    synnax::channel::Channel state_ch;

    Output(
        const ::synnax::ethercat::OutputChannelAutomatic &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, false, cfg.disabled, cfg.device),
        command_key(cfg.cmd_channel),
        state_key(cfg.state_channel) {
        this->resolve_pdo(slave.find_output_pdo(cfg.pdo), cfg.pdo, parser, path);
    }

    Output(
        const ::synnax::ethercat::OutputChannelManual &cfg,
        const slave::Properties &slave
    ):
        Channel(slave, false, cfg.disabled, cfg.device),
        command_key(cfg.cmd_channel),
        state_key(cfg.state_channel) {
        this->bind_address(cfg);
    }

    void bind_remote_info(const synnax::channel::Channel &state_channel) {
        this->state_ch = state_channel;
    }
};

/// @brief the shared base fields of a parsed input channel configuration.
inline const ::synnax::ethercat::BaseInputChannel &
base(const ::synnax::ethercat::InputChannel &cfg) {
    return std::visit(
        [](const auto &c) -> const ::synnax::ethercat::BaseInputChannel & { return c; },
        cfg
    );
}

/// @brief the shared base fields of a parsed output channel configuration.
inline const ::synnax::ethercat::BaseOutputChannel &
base(const ::synnax::ethercat::OutputChannel &cfg) {
    return std::visit(
        [](const auto &c) -> const ::synnax::ethercat::BaseOutputChannel & {
            return c;
        },
        cfg
    );
}

/// @brief constructs a runtime input channel from a parsed configuration,
/// accumulating errors on the parser under the given path prefix.
inline std::unique_ptr<Input> make_input(
    const ::synnax::ethercat::InputChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path = ""
) {
    return std::visit(
        [&](const auto &c) -> std::unique_ptr<Input> {
            using T = std::decay_t<decltype(c)>;
            if constexpr (std::is_same_v<T, ::synnax::ethercat::InputChannelAutomatic>)
                return std::make_unique<Input>(c, slave, parser, path);
            else
                return std::make_unique<Input>(c, slave);
        },
        cfg
    );
}

/// @brief constructs a runtime output channel from a parsed configuration,
/// accumulating errors on the parser under the given path prefix.
inline std::unique_ptr<Output> make_output(
    const ::synnax::ethercat::OutputChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path = ""
) {
    return std::visit(
        [&](const auto &c) -> std::unique_ptr<Output> {
            using T = std::decay_t<decltype(c)>;
            if constexpr (std::is_same_v<T, ::synnax::ethercat::OutputChannelAutomatic>)
                return std::make_unique<Output>(c, slave, parser, path);
            else
                return std::make_unique<Output>(c, slave);
        },
        cfg
    );
}

/// @brief sorts a vector of channel pointers by slave position, then by index.
template<typename ChannelPtr>
void sort_by_position(std::vector<ChannelPtr> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        if (a->slave_position != b->slave_position)
            return a->slave_position < b->slave_position;
        return a->index < b->index;
    });
}
}
