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

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

namespace modbus::channel {
/// @brief Base class for all Modbus channels
struct Channel {
    /// @brief Whether the channel is enabled
    bool enabled;
    /// @brief The Modbus register address
    uint16_t address;

    explicit Channel(xjson::Parser &parser):
        enabled(parser.optional<bool>("enabled", true)),
        address(parser.required<uint16_t>("address")) {}

    /// @brief Virtual destructor
    virtual ~Channel() = default;
};

/// @brief base class for input channels (reading from Modbus)
struct Input : virtual Channel {
    /// @brief The key of the synnax channel to write data to
    synnax::ChannelKey synnax_key;
    /// @brief The synnax channel object
    synnax::Channel ch;

    explicit Input(xjson::Parser &parser):
        Channel(parser), synnax_key(parser.required<synnax::ChannelKey>("channel")) {}

    /// @brief Binds remote channel information
    void bind_remote_info(const synnax::Channel &remote_ch) { this->ch = remote_ch; }
};

/// @brief configuration to read from a discrete input.
struct InputDiscrete final : Input {
    explicit InputDiscrete(xjson::Parser &parser): Channel(parser), Input(parser) {}
};

/// @brief configuration to read from an input register.
struct InputRegister final : Input {
    /// @brief The data type to interpret the register(s) as
    telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;
    /// @brief String length for STRING data type
    int string_length;

    explicit InputRegister(xjson::Parser &parser):
        Channel(parser),
        Input(parser),
        value_type(telem::DataType(parser.required<std::string>("data_type"))),
        swap_bytes(parser.optional<bool>("swap_bytes", false)),
        swap_words(parser.optional<bool>("swap_words", false)),
        string_length(parser.optional<int>("string_length", 0)) {}
};

/// @brief Output channel for writing to coils
struct OutputCoil final : Channel {
    /// @brief The key of the channel to write to the coil
    synnax::ChannelKey channel;

    explicit OutputCoil(xjson::Parser &parser):
        Channel(parser), channel(parser.required<synnax::ChannelKey>("channel")) {}
};

/// @brief Output channel for writing to holding registers
struct OutputHoldingRegister final : Channel {
    /// @brief The key of the channel to write to the register
    synnax::ChannelKey channel;
    /// @brief The data type to interpret the register(s) as
    telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;

    explicit OutputHoldingRegister(xjson::Parser &parser):
        Channel(parser),
        channel(parser.required<synnax::ChannelKey>("channel")),
        value_type(telem::DataType(parser.required<std::string>("data_type"))),
        swap_bytes(parser.optional<bool>("swap_bytes", false)),
        swap_words(parser.optional<bool>("swap_words", false)) {}
};

/// @brief sorts a vector of channels in place by their address.
template<typename Channel>
void sort_by_address(std::vector<Channel> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        return a.address < b.address;
    });
}
}
