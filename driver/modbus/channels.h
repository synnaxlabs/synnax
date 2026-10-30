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
#include <vector>

#include "client/cpp/modbus/types.gen.h"
#include "client/cpp/synnax.h"

#include "driver/modbus/device/device.h"

namespace driver::modbus::channel {
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

    /// @returns the number of bits the channel covers.
    [[nodiscard]] static std::size_t width() { return 1; }
};

/// @returns the number of 16-bit registers a value of the data type covers.
inline std::size_t register_count(const x::telem::DataType &dt) {
    return (dt.density() + 1) / 2;
}

/// @returns true if the Driver can encode and decode registers as the data type.
inline bool is_register_type(const x::telem::DataType &dt) {
    return dt.matches(
        {x::telem::INT8_T,
         x::telem::UINT8_T,
         x::telem::INT16_T,
         x::telem::UINT16_T,
         x::telem::INT32_T,
         x::telem::UINT32_T,
         x::telem::INT64_T,
         x::telem::UINT64_T,
         x::telem::FLOAT32_T,
         x::telem::FLOAT64_T}
    );
}

/// @brief configuration to read from a holding or input register.
struct InputRegister final : Input {
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief whether the byte order within each 16-bit word is swapped, from the
    /// channel's override or else the device.
    bool bytes_swapped;
    /// @brief whether the word order of multi-register values is swapped, from the
    /// channel's override or else the device.
    bool words_swapped;
    /// @brief String length for STRING data type
    int string_length;

    InputRegister(
        const ::synnax::modbus::BaseReadChannel &base,
        const ::synnax::modbus::RegisterValue &value,
        const std::int32_t string_length,
        const device::ConnectionConfig &conn
    ):
        Input(base),
        value_type(value.data_type),
        bytes_swapped(value.bytes_swapped.value_or(conn.swap_bytes)),
        words_swapped(value.words_swapped.value_or(conn.swap_words)),
        string_length(string_length) {}

    InputRegister(
        const ::synnax::modbus::HoldingRegisterReadChannel &cfg,
        const device::ConnectionConfig &conn
    ):
        InputRegister(cfg, cfg, cfg.string_length, conn) {}

    InputRegister(
        const ::synnax::modbus::InputRegisterReadChannel &cfg,
        const device::ConnectionConfig &conn
    ):
        InputRegister(cfg, cfg, cfg.string_length, conn) {}

    /// @returns the number of registers the channel covers.
    [[nodiscard]] std::size_t width() const { return register_count(value_type); }
};

/// @brief base class for output channels (writing to Modbus).
struct Output {
    /// @brief The Modbus register address
    uint16_t address;
    /// @brief The key of the channel to receive commands from
    synnax::channel::Key channel;
    /// @brief the name of the command channel, for errors. Bound at configure.
    std::string name;

    explicit Output(const ::synnax::modbus::BaseWriteChannel &cfg):
        address(cfg.address), channel(cfg.channel) {}
};

/// @brief Output channel for writing to coils
struct OutputCoil final : Output {
    using Output::Output;

    /// @returns the number of bits the channel covers.
    [[nodiscard]] static std::size_t width() { return 1; }
};

/// @brief Output channel for writing to holding registers
struct OutputHoldingRegister final : Output {
    /// @brief The data type to interpret the register(s) as
    x::telem::DataType value_type;
    /// @brief whether the byte order within each 16-bit word is swapped, from the
    /// channel's override or else the device.
    bool bytes_swapped;
    /// @brief whether the word order of multi-register values is swapped, from the
    /// channel's override or else the device.
    bool words_swapped;

    OutputHoldingRegister(
        const ::synnax::modbus::HoldingRegisterWriteChannel &cfg,
        const device::ConnectionConfig &conn
    ):
        Output(cfg),
        value_type(cfg.data_type),
        bytes_swapped(cfg.bytes_swapped.value_or(conn.swap_bytes)),
        words_swapped(cfg.words_swapped.value_or(conn.swap_words)) {}

    /// @returns the number of registers the channel covers.
    [[nodiscard]] std::size_t width() const { return register_count(value_type); }
};

/// @brief sorts a vector of channels in place by their address.
template<typename Channel>
void sort_by_address(std::vector<Channel> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        return a.address < b.address;
    });
}
}
