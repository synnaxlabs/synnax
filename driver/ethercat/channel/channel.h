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
#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/device/device.h"
#include "driver/ethercat/master/slave_info.h"

namespace ethercat::channel {
/// @brief base class for EtherCAT PDO channel configurations.
struct Channel {
    /// @brief whether this channel is enabled for data exchange.
    bool enabled;
    /// @brief the key of the slave device in Synnax.
    std::string device_key;
    /// @brief position of the slave on the EtherCAT bus.
    uint16_t slave_position;
    /// @brief index of the PDO object in the CoE object dictionary (e.g., 0x6000).
    uint16_t index;
    /// @brief subindex of the PDO object.
    uint8_t subindex;
    /// @brief size of the data in bits.
    uint8_t bit_length;
    /// @brief data type of the PDO.
    telem::DataType data_type;

    virtual ~Channel() = default;

    /// @brief returns the byte length rounded up from bit_length.
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }

    /// @brief converts this channel configuration to a PDOEntry.
    [[nodiscard]] PDOEntry to_pdo_entry(const bool is_input) const {
        return {
            this->slave_position,
            this->index,
            this->subindex,
            this->bit_length,
            is_input,
            this->data_type
        };
    }

protected:
    explicit Channel(xjson::Parser &parser, const device::SlaveProperties &slave):
        enabled(parser.field<bool>("enabled", true)),
        device_key(parser.field<std::string>("device")),
        slave_position(slave.position),
        index(0),
        subindex(0),
        bit_length(0),
        data_type(telem::UNKNOWN_T) {}
};

/// @brief base input channel (TxPDO, slave->master).
struct Input : virtual Channel {
    /// @brief the key of the Synnax channel to write data to.
    synnax::ChannelKey synnax_key;
    /// @brief the Synnax channel object (populated after remote lookup).
    synnax::Channel ch;

    /// @brief binds remote channel information retrieved from Synnax.
    void bind_remote_info(const synnax::Channel &remote_ch) { this->ch = remote_ch; }

protected:
    explicit Input(xjson::Parser &parser, const device::SlaveProperties &slave):
        Channel(parser, slave),
        synnax_key(parser.field<synnax::ChannelKey>("channel")) {}
};

/// @brief automatic input channel that resolves PDO address from slave device
/// properties.
struct AutomaticInput final : Input {
    /// @brief the name of the PDO to look up in slave device properties.
    std::string pdo_name;

    explicit AutomaticInput(
        xjson::Parser &parser,
        const device::SlaveProperties &slave
    ):
        Channel(parser, slave),
        Input(parser, slave),
        pdo_name(parser.field<std::string>("pdo")) {
        auto pdo = slave.find_input_pdo(pdo_name);
        if (!pdo) {
            parser.field_err("pdo", "PDO '" + pdo_name + "' not found in slave");
            return;
        }
        index = pdo->index;
        subindex = pdo->subindex;
        bit_length = pdo->bit_length;
        data_type = telem::DataType(pdo->data_type);
    }
};

/// @brief manual input channel where user specifies PDO address inline.
struct ManualInput final : Input {
    explicit ManualInput(xjson::Parser &parser, const device::SlaveProperties &slave):
        Channel(parser, slave), Input(parser, slave) {
        index = static_cast<uint16_t>(parser.field<int>("index"));
        subindex = static_cast<uint8_t>(parser.field<int>("subindex"));
        bit_length = static_cast<uint8_t>(parser.field<int>("bit_length"));
        data_type = telem::DataType(parser.field<std::string>("data_type"));
    }
};

/// @brief factory function type for creating input channels.
using InputFactory = std::function<
    std::unique_ptr<Input>(xjson::Parser &, const device::SlaveProperties &)>;

/// @brief parses an input channel from JSON configuration.
inline std::unique_ptr<Input>
parse_input(xjson::Parser &parser, const device::SlaveProperties &slave) {
    static const std::map<std::string, InputFactory> INPUT_FACTORIES = {
        {"automatic",
         [](xjson::Parser &cfg, const device::SlaveProperties &s) {
             return std::make_unique<AutomaticInput>(cfg, s);
         }},
        {"manual", [](xjson::Parser &cfg, const device::SlaveProperties &s) {
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
    synnax::ChannelKey command_key;
    /// @brief the key of the Synnax channel to write state feedback to.
    synnax::ChannelKey state_key;
    /// @brief the Synnax state channel object (populated after remote lookup).
    synnax::Channel state_ch;

    void bind_remote_info(const synnax::Channel &state_channel) {
        this->state_ch = state_channel;
    }

protected:
    explicit Output(xjson::Parser &parser, const device::SlaveProperties &slave):
        Channel(parser, slave),
        command_key(parser.field<synnax::ChannelKey>("cmd_channel")),
        state_key(parser.field<synnax::ChannelKey>("state_channel", 0)) {}
};

/// @brief automatic output channel that resolves PDO address from slave device
/// properties.
struct AutomaticOutput final : Output {
    /// @brief the name of the PDO to look up in slave device properties.
    std::string pdo_name;

    explicit AutomaticOutput(
        xjson::Parser &parser,
        const device::SlaveProperties &slave
    ):
        Channel(parser, slave),
        Output(parser, slave),
        pdo_name(parser.field<std::string>("pdo")) {
        auto pdo = slave.find_output_pdo(pdo_name);
        if (!pdo) {
            parser.field_err("pdo", "PDO '" + pdo_name + "' not found in slave");
            return;
        }
        index = pdo->index;
        subindex = pdo->subindex;
        bit_length = pdo->bit_length;
        data_type = telem::DataType(pdo->data_type);
    }
};

/// @brief manual output channel where user specifies PDO address inline.
struct ManualOutput final : Output {
    explicit ManualOutput(xjson::Parser &parser, const device::SlaveProperties &slave):
        Channel(parser, slave), Output(parser, slave) {
        index = static_cast<uint16_t>(parser.field<int>("index"));
        subindex = static_cast<uint8_t>(parser.field<int>("subindex"));
        bit_length = static_cast<uint8_t>(parser.field<int>("bit_length"));
        data_type = telem::DataType(parser.field<std::string>("data_type"));
    }
};

/// @brief factory function type for creating output channels.
using OutputFactory = std::function<
    std::unique_ptr<Output>(xjson::Parser &, const device::SlaveProperties &)>;

/// @brief parses an output channel from JSON configuration.
inline std::unique_ptr<Output>
parse_output(xjson::Parser &parser, const device::SlaveProperties &slave) {
    static const std::map<std::string, OutputFactory> OUTPUT_FACTORIES = {
        {"automatic",
         [](xjson::Parser &cfg, const device::SlaveProperties &s) {
             return std::make_unique<AutomaticOutput>(cfg, s);
         }},
        {"manual", [](xjson::Parser &cfg, const device::SlaveProperties &s) {
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
