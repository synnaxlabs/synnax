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
#include <cmath>
#include <format>
#include <limits>
#include <memory>
#include <optional>
#include <string>
#include <unordered_set>
#include <variant>
#include <vector>

#include "client/cpp/ethercat/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/telem/telem.h"

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

    /// @brief binds the entry at an address on the slave, with the slave's bit length
    /// and the configured data type. Accumulates a field error under path and binds
    /// nothing when the slave lacks the entry or the type does not fit it. A disabled
    /// channel binds nothing.
    void bind_entry(
        const uint16_t index,
        const uint8_t sub_index,
        const x::telem::DataType &data_type,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path
    ) {
        if (!this->enabled) return;
        const auto pdo = slave.find_pdo(index, sub_index, this->is_input);
        if (!pdo) {
            parser.field_err(
                path + "index",
                "slave " + slave.name + " has no " + this->direction() + " entry at " +
                    telem::format_index_sub_index(index, sub_index)
            );
            return;
        }
        static const std::vector NUMBER_TYPES = {
            x::telem::FLOAT64_T,
            x::telem::FLOAT32_T,
            x::telem::INT64_T,
            x::telem::INT32_T,
            x::telem::INT16_T,
            x::telem::INT8_T,
            x::telem::UINT64_T,
            x::telem::UINT32_T,
            x::telem::UINT16_T,
            x::telem::UINT8_T,
        };
        const auto entry = "entry " + telem::format_index_sub_index(index, sub_index) +
                           " on slave " + slave.name;
        const auto width = telem::infer_type_from_bit_length(pdo->bit_length).density();
        const auto pick = "Pick a " + std::to_string(width) + " byte ";
        if (!data_type.matches(NUMBER_TYPES)) {
            parser.field_err(
                path + "data_type",
                "data type " + data_type.name() + " cannot be exchanged with " + entry +
                    ". " + pick + "number type."
            );
            return;
        }
        if (data_type.density() != width) {
            parser.field_err(
                path + "data_type",
                "data type " + data_type.name() + " is " + bytes(data_type.density()) +
                    " but " + entry + " holds " + bytes(pdo->byte_length()) + ". " +
                    pick + "type."
            );
            return;
        }
        this->index = index;
        this->sub_index = sub_index;
        this->bit_length = pdo->bit_length;
        this->data_type = data_type;
    }

    /// @brief accumulates a field error under path for an enabled channel whose entry
    /// name matched no single entry, listing the addresses that carry the name.
    void reject_unresolved(
        const std::string &name,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path
    ) const {
        if (!this->enabled) return;
        std::vector<std::string> addresses;
        std::vector<std::string> labels;
        for (const auto &pdo: this->is_input ? slave.input_pdos : slave.output_pdos) {
            if (pdo.is_padding() || pdo.name != name) continue;
            auto addr = telem::format_index_sub_index(pdo.index, pdo.sub_index);
            if (std::ranges::find(addresses, addr) != addresses.end()) continue;
            std::string label = addr;
            if (pdo.pdo_name && !pdo.pdo_name->empty())
                label += " (" + *pdo.pdo_name + ")";
            addresses.push_back(std::move(addr));
            labels.push_back(std::move(label));
        }
        const auto on_slave = " on slave " + slave.name;
        std::string msg = "'" + name + "' matches ";
        if (labels.empty())
            msg += "no " + this->direction() + " entry" + on_slave;
        else if (labels.size() == 1)
            msg += this->direction() + " entry " + labels.front() + on_slave;
        else {
            msg += "more than one " + this->direction() + " entry" + on_slave + ": ";
            for (size_t i = 0; i < labels.size(); i++)
                msg += (i == 0 ? "" : ", ") + labels[i];
        }
        parser.field_err(path + "pdo", msg + ". Pick an entry.");
    }

private:
    [[nodiscard]] std::string direction() const {
        return this->is_input ? "input" : "output";
    }

    static std::string bytes(const size_t n) {
        return std::to_string(n) + (n == 1 ? " byte" : " bytes");
    }
};

/// @brief input channel (TxPDO, slave->master).
struct Input final : Channel {
    /// @brief the key of the Synnax channel to write data to.
    synnax::channel::Key synnax_key;
    /// @brief the Synnax channel object (populated after remote lookup).
    synnax::channel::Channel ch;

    Input(
        const ::synnax::ethercat::EntryReadChannel &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, true, cfg.disabled, cfg.device), synnax_key(cfg.channel) {
        this->bind_entry(cfg.index, cfg.sub_index, cfg.data_type, slave, parser, path);
    }

    Input(
        const ::synnax::ethercat::UnresolvedReadChannel &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, true, cfg.disabled, cfg.device), synnax_key(cfg.channel) {
        this->reject_unresolved(cfg.pdo, slave, parser, path);
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
    /// @brief the state channel, bound by the owning task when state_key is set.
    std::optional<synnax::channel::Channel> state;
    /// @brief the value the entry takes when the task stops.
    double stop_value = 0;

    Output(
        const ::synnax::ethercat::EntryWriteChannel &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, false, cfg.disabled, cfg.device),
        command_key(cfg.cmd_channel),
        state_key(cfg.state_channel),
        stop_value(cfg.stop_value) {
        this->bind_entry(cfg.index, cfg.sub_index, cfg.data_type, slave, parser, path);
        this->check_stop_value(slave, parser, path);
    }

    Output(
        const ::synnax::ethercat::UnresolvedWriteChannel &cfg,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ):
        Channel(slave, false, cfg.disabled, cfg.device),
        command_key(cfg.cmd_channel),
        state_key(cfg.state_channel) {
        this->reject_unresolved(cfg.pdo, slave, parser, path);
    }

private:
    /// @brief accumulates a field error under path when the stop value does not fit the
    /// bound entry.
    void check_stop_value(
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path
    ) const {
        if (!this->enabled) return;
        const double v = this->stop_value;
        const auto reject = [&](const std::string &range) {
            parser.field_err(
                path + "stop_value",
                std::format(
                    "stop value {} does not fit entry {} on slave {}. Pick {}.",
                    v,
                    telem::format_index_sub_index(this->index, this->sub_index),
                    slave.name,
                    range
                )
            );
        };
        if (this->data_type == x::telem::FLOAT32_T) {
            constexpr float MAX = std::numeric_limits<float>::max();
            if (std::abs(v) > MAX)
                reject(std::format("a value from {} to {}", -MAX, MAX));
            return;
        }
        static const std::vector SIGNED_TYPES = {
            x::telem::INT64_T,
            x::telem::INT32_T,
            x::telem::INT16_T,
            x::telem::INT8_T,
        };
        static const std::vector UNSIGNED_TYPES = {
            x::telem::UINT64_T,
            x::telem::UINT32_T,
            x::telem::UINT16_T,
            x::telem::UINT8_T,
        };
        const bool is_signed = this->data_type.matches(SIGNED_TYPES);
        if (!is_signed && !this->data_type.matches(UNSIGNED_TYPES)) return;
        // A double rounds whole numbers of magnitude 2^53 and above, so a larger stop
        // value may not be the one the user entered.
        constexpr int DIGITS = std::numeric_limits<double>::digits;
        constexpr int64_t EXACT = (int64_t{1} << DIGITS) - 1;
        const int bits = is_signed ? this->bit_length - 1 : this->bit_length;
        const int64_t max = bits >= DIGITS ? EXACT : (int64_t{1} << bits) - 1;
        const int64_t min = is_signed ? std::max(-EXACT, -max - 1) : 0;
        if (v < static_cast<double>(min) || v > static_cast<double>(max) ||
            v != std::trunc(v))
            reject(std::format("a whole number from {} to {}", min, max));
    }
};

/// @brief the shared base fields of a parsed input channel configuration.
inline const ::synnax::ethercat::BaseReadChannel &
base(const ::synnax::ethercat::ReadChannel &cfg) {
    return std::visit(
        [](const auto &c) -> const ::synnax::ethercat::BaseReadChannel & { return c; },
        cfg
    );
}

/// @brief the shared base fields of a parsed output channel configuration.
inline const ::synnax::ethercat::BaseWriteChannel &
base(const ::synnax::ethercat::WriteChannel &cfg) {
    return std::visit(
        [](const auto &c) -> const ::synnax::ethercat::BaseWriteChannel & { return c; },
        cfg
    );
}

/// @brief constructs a runtime input channel from a parsed configuration,
/// accumulating errors on the parser under the given path prefix.
inline std::unique_ptr<Input> make_input(
    const ::synnax::ethercat::ReadChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path = ""
) {
    return std::visit(
        [&](const auto &c) { return std::make_unique<Input>(c, slave, parser, path); },
        cfg
    );
}

/// @brief constructs a runtime output channel from a parsed configuration,
/// accumulating errors on the parser under the given path prefix.
inline std::unique_ptr<Output> make_output(
    const ::synnax::ethercat::WriteChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path = ""
) {
    return std::visit(
        [&](const auto &c) { return std::make_unique<Output>(c, slave, parser, path); },
        cfg
    );
}

/// @brief tracks the entries a task's enabled channels use, so that no two channels
/// share one.
class Claims {
    std::unordered_set<pdo::Key, pdo::KeyHash> keys;

public:
    /// @brief claims the channel's entry, accumulating a field error under path when
    /// another channel holds it. A channel that bound no entry claims nothing.
    void claim(
        const Channel &ch,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    ) {
        if (ch.index == 0) return;
        const pdo::Key key{ch.slave_position, ch.index, ch.sub_index, ch.is_input};
        if (this->keys.insert(key).second) return;
        parser.field_err(
            path + "index",
            "another channel already uses entry " +
                telem::format_index_sub_index(ch.index, ch.sub_index) + " on slave " +
                slave.name
        );
    }
};

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
