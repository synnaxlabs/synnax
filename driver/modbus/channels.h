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

#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

namespace driver::modbus::channel {
/// @brief Base class for all Modbus channels
struct Channel {
    /// @brief Whether the channel is enabled
    bool enabled;
    /// @brief The Modbus register address
    uint16_t address;

    explicit Channel(x::json::Parser &parser):
        enabled(parser.field<bool>("enabled", true)),
        address(parser.field<uint16_t>("address")) {}

    /// @brief Virtual destructor
    virtual ~Channel() = default;
};

/// @brief base class for input channels (reading from Modbus)
struct Input : virtual Channel {
    /// @brief The key of the synnax channel to write data to
    synnax::channel::Key synnax_key;
    /// @brief The synnax channel object
    synnax::channel::Channel ch;

    explicit Input(x::json::Parser &parser):
        Channel(parser), synnax_key(parser.field<synnax::channel::Key>("channel")) {}

    /// @brief Binds remote channel information
    void bind_remote_info(const synnax::channel::Channel &remote_ch) { this->ch = remote_ch; }
};

/// @brief configuration to read from a discrete input.
struct InputDiscrete final : Input {
    explicit InputDiscrete(x::json::Parser &parser): Channel(parser), Input(parser) {}
};

/// @brief configuration to read from an input register.
struct InputRegister final : Input {
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;
    /// @brief String length for STRING data type
    int string_length;

    explicit InputRegister(x::json::Parser &parser):
        Channel(parser),
        Input(parser),
        value_type(x::telem::DataType(parser.field<std::string>("data_type"))),
        swap_bytes(parser.field<bool>("swap_bytes", false)),
        swap_words(parser.field<bool>("swap_words", false)),
        string_length(parser.field<int>("string_length", 0)) {}
};

/// @brief Output channel for writing to coils
struct OutputCoil final : Channel {
    /// @brief The key of the channel to write to the coil
    synnax::channel::Key channel;

    explicit OutputCoil(x::json::Parser &parser):
        Channel(parser), channel(parser.field<synnax::channel::Key>("channel")) {}
};

/// @brief Output channel for writing to holding registers
struct OutputHoldingRegister final : Channel {
    /// @brief The key of the channel to write to the register
    synnax::channel::Key channel;
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;

    explicit OutputHoldingRegister(x::json::Parser &parser):
        Channel(parser),
        channel(parser.field<synnax::channel::Key>("channel")),
        value_type(x::telem::DataType(parser.field<std::string>("data_type"))),
        swap_bytes(parser.field<bool>("swap_bytes", false)),
        swap_words(parser.field<bool>("swap_words", false)) {}
};

/// @brief sorts a vector of channels in place by their address.
template<typename Channel>
void sort_by_address(std::vector<Channel> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        return a.address < b.address;
    });
}
}
