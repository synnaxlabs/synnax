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
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "client/cpp/modbus/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/plan.h"
#include "driver/modbus/registers/registers.h"

namespace driver::modbus::channel {
/// @brief returns the field of a coil or discrete input in a block that holds one byte
/// per bit.
/// @param offset the bit's offset from the first address of the block.
inline synnax::library::BinaryField bit_field(const std::size_t offset) {
    synnax::library::BinaryField f;
    f.start_bit = static_cast<std::uint16_t>(offset * 8);
    f.bit_length = 1;
    return f;
}

/// @brief base class for input channels (reading from Modbus).
struct Input {
    /// @brief The Modbus register address
    uint16_t address;
    /// @brief The key of the synnax channel to write data to
    synnax::channel::Key synnax_key;
    /// @brief The synnax channel object
    synnax::channel::Channel ch;

    explicit Input(const ::synnax::modbus::BaseReadChannel &cfg):
        address(cfg.address), synnax_key(cfg.channel) {}

    /// @brief Binds remote channel information
    void bind_remote_info(const synnax::channel::Channel &remote_ch) {
        this->ch = remote_ch;
    }
};

/// @brief configuration to read from a coil or discrete input.
struct InputDiscrete final : Input {
    using Input::Input;

    /// @brief returns the codec field of the channel in a block that starts at start.
    [[nodiscard]] std::pair<synnax::library::BinaryField, x::errors::Error>
    field(const std::uint16_t start) const {
        return {bit_field(this->address - start), x::errors::NIL};
    }
};

/// @brief configuration to read from a holding or input register.
struct InputRegister final : Input {
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool bytes_swapped;
    /// @brief The word order for multi-register values
    bool words_swapped;
    /// @brief String length for STRING data type
    int string_length;

    InputRegister(
        const ::synnax::modbus::BaseReadChannel &base,
        const ::synnax::modbus::RegisterValue &value,
        const std::int32_t string_length
    ):
        Input(base),
        value_type(value.data_type),
        bytes_swapped(value.bytes_swapped),
        words_swapped(value.words_swapped),
        string_length(string_length) {}

    explicit InputRegister(const ::synnax::modbus::HoldingRegisterReadChannel &cfg):
        InputRegister(cfg, cfg, cfg.string_length) {}

    explicit InputRegister(const ::synnax::modbus::InputRegisterReadChannel &cfg):
        InputRegister(cfg, cfg, cfg.string_length) {}

    /// @brief returns the codec field of the channel in a block that starts at start.
    [[nodiscard]] std::pair<synnax::library::BinaryField, x::errors::Error>
    field(const std::uint16_t start) const {
        return registers::field(
            this->value_type,
            this->bytes_swapped,
            this->words_swapped,
            this->address - start
        );
    }
};

/// @brief base class for output channels (writing to Modbus).
struct Output {
    /// @brief The Modbus register address
    uint16_t address;
    /// @brief The key of the channel to receive commands from
    synnax::channel::Key channel;

    explicit Output(const ::synnax::modbus::BaseWriteChannel &cfg):
        address(cfg.address), channel(cfg.channel) {}
};

/// @brief Output channel for writing to coils
struct OutputCoil final : Output {
    /// @brief the type a command converts to. Any non-zero value turns the coil on.
    x::telem::DataType value_type = x::telem::UINT8_T;

    using Output::Output;

    /// @brief returns the codec field of the channel in a block that starts at start.
    [[nodiscard]] std::pair<synnax::library::BinaryField, x::errors::Error>
    field(const std::uint16_t start) const {
        return {bit_field(this->address - start), x::errors::NIL};
    }
};

/// @brief Output channel for writing to holding registers
struct OutputHoldingRegister final : Output {
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief The byte order for multi-register values
    bool bytes_swapped;
    /// @brief The word order for multi-register values
    bool words_swapped;

    explicit OutputHoldingRegister(
        const ::synnax::modbus::HoldingRegisterWriteChannel &cfg
    ):
        Output(cfg),
        value_type(cfg.data_type),
        bytes_swapped(cfg.bytes_swapped),
        words_swapped(cfg.words_swapped) {}

    /// @brief returns the codec field of the channel in a block that starts at start.
    [[nodiscard]] std::pair<synnax::library::BinaryField, x::errors::Error>
    field(const std::uint16_t start) const {
        return registers::whole_field(
            this->value_type,
            this->bytes_swapped,
            this->words_swapped,
            this->address - start
        );
    }
};

/// @brief sorts a vector of channels in place by their address.
template<typename Channel>
void sort_by_address(std::vector<Channel> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        return a.address < b.address;
    });
}

/// @brief compiles a codec plan whose slots are the channels' values, in order, in a
/// block that starts at the first channel's address.
/// @param channels non-empty and sorted by address.
/// @returns VALIDATION for an unsupported data type, or LAYOUT_ERROR when the block
/// is too large for the codec.
template<typename Channel>
std::pair<codec::Plan, x::errors::Error> compile(const std::vector<Channel> &channels) {
    synnax::library::MessageEntry message;
    const auto start = channels.front().address;
    for (const auto &ch: channels) {
        auto [f, err] = ch.field(start);
        if (err) return {{}, err};
        f.key = x::uuid::create();
        f.name = "address " + std::to_string(ch.address);
        message.fields.emplace_back(std::move(f));
    }
    return codec::Plan::compile(message);
}
}
