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
#include <functional>
#include <map>
#include <memory>
#include <string>

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
    explicit Channel(x::json::Parser &parser, const slave::Properties &slave):
        pdo::Entry{slave.position, 0, 0, 0, true, x::telem::UNKNOWN_T},
        enabled(parser.field<bool>("enabled", true)),
        device_key(parser.field<std::string>("device")) {}
};

/// @brief base input channel (TxPDO, slave->master).
struct Input : virtual Channel {
    /// @brief the key of the Synnax channel to write data to.
    synnax::channel::Key synnax_key;
    /// @brief the Synnax channel object (populated after remote lookup).
    synnax::channel::Channel ch;

    /// @brief binds remote channel information retrieved from Synnax.
    void bind_remote_info(const synnax::channel::Channel &remote_ch) {
        this->ch = remote_ch;
    }

protected:
    explicit Input(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave),
        synnax_key(parser.field<synnax::channel::Key>("channel")) {
        this->is_input = true;
    }
};

/// @brief automatic input channel that resolves PDO address from slave device
/// properties.
struct AutomaticInput final : Input {
    /// @brief the name of the PDO to look up in slave device properties.
    std::string pdo_name;

    explicit AutomaticInput(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave),
        Input(parser, slave),
        pdo_name(parser.field<std::string>("pdo")) {
        auto pdo = slave.find_input_pdo(this->pdo_name);
        if (!pdo) {
            parser.field_err("pdo", "PDO '" + this->pdo_name + "' not found in slave");
            return;
        }
        this->index = pdo->index;
        this->sub_index = pdo->sub_index;
        this->bit_length = pdo->bit_length;
        this->data_type = x::telem::DataType(pdo->data_type);
    }
};

/// @brief manual input channel where user specifies PDO address inline.
struct ManualInput final : Input {
    explicit ManualInput(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave), Input(parser, slave) {
        this->index = static_cast<uint16_t>(parser.field<int>("index"));
        this->sub_index = static_cast<uint8_t>(parser.field<int>("sub_index"));
        this->bit_length = static_cast<uint8_t>(parser.field<int>("bit_length"));
        this->data_type = x::telem::DataType(parser.field<std::string>("data_type"));
    }
};

/// @brief factory function type for creating input channels.
using InputFactory = std::function<
    std::unique_ptr<Input>(x::json::Parser &, const slave::Properties &)>;

/// @brief parses an input channel from JSON configuration.
inline std::unique_ptr<Input>
parse_input(x::json::Parser &parser, const slave::Properties &slave) {
    static const std::map<std::string, InputFactory> INPUT_FACTORIES = {
        {"automatic",
         [](x::json::Parser &cfg, const slave::Properties &s) {
             return std::make_unique<AutomaticInput>(cfg, s);
         }},
        {"manual", [](x::json::Parser &cfg, const slave::Properties &s) {
             return std::make_unique<ManualInput>(cfg, s);
         }}
    };
    const auto type = parser.field<std::string>("type");
    const auto it = INPUT_FACTORIES.find(type);
    if (it != INPUT_FACTORIES.end()) return it->second(parser, slave);
    parser.field_err("type", "unknown channel type: " + type);
    return nullptr;
}

/// @brief base output channel (RxPDO, master->slave).
struct Output : virtual Channel {
    /// @brief the key of the Synnax channel to receive commands from.
    synnax::channel::Key command_key;
    /// @brief the key of the Synnax channel to write state feedback to.
    synnax::channel::Key state_key;
    /// @brief the Synnax state channel object (populated after remote lookup).
    synnax::channel::Channel state_ch;

    void bind_remote_info(const synnax::channel::Channel &state_channel) {
        this->state_ch = state_channel;
    }

protected:
    explicit Output(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave),
        command_key(parser.field<synnax::channel::Key>("cmd_channel")),
        state_key(parser.field<synnax::channel::Key>("state_channel", 0)) {
        this->is_input = false;
    }
};

/// @brief automatic output channel that resolves PDO address from slave device
/// properties.
struct AutomaticOutput final : Output {
    /// @brief the name of the PDO to look up in slave device properties.
    std::string pdo_name;

    explicit AutomaticOutput(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave),
        Output(parser, slave),
        pdo_name(parser.field<std::string>("pdo")) {
        auto pdo = slave.find_output_pdo(this->pdo_name);
        if (!pdo) {
            parser.field_err("pdo", "PDO '" + this->pdo_name + "' not found in slave");
            return;
        }
        this->index = pdo->index;
        this->sub_index = pdo->sub_index;
        this->bit_length = pdo->bit_length;
        this->data_type = x::telem::DataType(pdo->data_type);
    }
};

/// @brief manual output channel where user specifies PDO address inline.
struct ManualOutput final : Output {
    explicit ManualOutput(x::json::Parser &parser, const slave::Properties &slave):
        Channel(parser, slave), Output(parser, slave) {
        this->index = static_cast<uint16_t>(parser.field<int>("index"));
        this->sub_index = static_cast<uint8_t>(parser.field<int>("sub_index"));
        this->bit_length = static_cast<uint8_t>(parser.field<int>("bit_length"));
        this->data_type = x::telem::DataType(parser.field<std::string>("data_type"));
    }
};

/// @brief factory function type for creating output channels.
using OutputFactory = std::function<
    std::unique_ptr<Output>(x::json::Parser &, const slave::Properties &)>;

/// @brief parses an output channel from JSON configuration.
inline std::unique_ptr<Output>
parse_output(x::json::Parser &parser, const slave::Properties &slave) {
    static const std::map<std::string, OutputFactory> OUTPUT_FACTORIES = {
        {"automatic",
         [](x::json::Parser &cfg, const slave::Properties &s) {
             return std::make_unique<AutomaticOutput>(cfg, s);
         }},
        {"manual", [](x::json::Parser &cfg, const slave::Properties &s) {
             return std::make_unique<ManualOutput>(cfg, s);
         }}
    };
    const auto type = parser.field<std::string>("type");
    const auto it = OUTPUT_FACTORIES.find(type);
    if (it != OUTPUT_FACTORIES.end()) return it->second(parser, slave);
    parser.field_err("type", "unknown channel type: " + type);
    return nullptr;
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
