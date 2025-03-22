// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>

/// external
#include "modbus/modbus.h"

/// module
#include "x/cpp/xjson/xjson.h"
#include "client/cpp/synnax.h"

namespace modbus::channel {
/// @brief Base class for all Modbus channels
struct Channel {
    /// @brief Whether the channel is enabled
    bool enabled;
    /// @brief The Modbus register address
    uint16_t address;
    
    explicit Channel(xjson::Parser &parser) :
        enabled(parser.optional<bool>("enabled", true)),
        address(parser.required<uint16_t>("address")) {
    }
    
    /// @brief Virtual destructor
    virtual ~Channel() = default;
};

/// @brief Base class for input channels (reading from Modbus)
struct Input : virtual Channel {
    /// @brief The key of the synnax channel to write data to
    synnax::ChannelKey synnax_key;
    /// @brief The synnax channel object
    synnax::Channel ch;

    explicit Input(xjson::Parser &parser) :
        Channel(parser),
        synnax_key(parser.required<synnax::ChannelKey>("channel")) {
    }
    
    /// @brief Binds remote channel information
    void bind_remote_info(const synnax::Channel &remote_ch) {
        this->ch = remote_ch;
    }
};

/// @brief Base class for output channels (writing to Modbus)
struct Output : virtual Channel {
    /// @brief The key of the command channel
    synnax::ChannelKey cmd_ch_key;
    
    /// @brief The key of the state channel
    synnax::ChannelKey state_ch_key;
    
    /// @brief The state channel object
    synnax::Channel state_ch;
    
    explicit Output(xjson::Parser &parser) :
        Channel(parser),
        cmd_ch_key(parser.required<synnax::ChannelKey>("cmd_key", "cmd_channel")),
        state_ch_key(parser.required<synnax::ChannelKey>("state_key", "state_channel")) {
    }
    
    /// @brief Binds remote state channel information
    void bind_state_info(const synnax::Channel &remote_state_ch) {
        this->state_ch = remote_state_ch;
    }
};

struct InputBit final : Input {
    explicit InputBit(xjson::Parser &parser) :
        Channel(parser),
        Input(parser) {
    }
};

/// @brief Input channel for reading holding or input registers
struct InputRegister final : Input {
    /// @brief The data type to interpret the register(s) as
    telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;
    /// @brief String length for STRING data type
    int string_length;
    
    explicit InputRegister(xjson::Parser &parser) :
        Channel(parser),
        Input(parser),
        value_type(telem::DataType(parser.required<std::string>("data_type"))),
        swap_bytes(parser.optional<bool>("swap_bytes", false)),
        swap_words(parser.optional<bool>("swap_words", false)),
        string_length(parser.optional<int>("string_length", 0)) {
        }
};

/// @brief Output channel for writing to coils
struct OutputCoilChannel final : Output {
    explicit OutputCoilChannel(xjson::Parser &parser) : 
        Channel(parser),
        Output(parser) {
    }
};

/// @brief Output channel for writing to holding registers
struct OutputHoldingRegisterChannel final : Output {
    /// @brief The data type to interpret the register(s) as
    telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool swap_bytes;
    /// @brief The word order for multi-register values
    bool swap_words;
    /// @brief String length for STRING data type
    int string_length;
    
    explicit OutputHoldingRegisterChannel(xjson::Parser &parser) :
        Channel(parser),
        Output(parser),
        value_type(telem::DataType(parser.required<std::string>("data_type"))),
        swap_bytes(parser.optional<bool>("swap_bytes", false)),
        swap_words(parser.optional<bool>("swap_words", false)),
        string_length(parser.optional<int>("string_length", 0)) {
    }
};

const std::string INPUT_BIT_TYPE = "input_bit";
const std::string INPUT_REGISTER_TYPE = "input_register";
const std::string OUTPUT_COIL_TYPE = "output_coil";
const std::string OUTPUT_HOLDING_REGISTER_TYPE = "output_holding_register";
}
