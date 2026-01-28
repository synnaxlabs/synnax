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
#include <string>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/master/slave_info.h"

namespace ethercat::channel {
/// Base class for EtherCAT PDO channel configurations.
struct Channel {
    /// Whether this channel is enabled for data exchange.
    bool enabled;

    /// Position of the slave on the EtherCAT bus (0-based).
    uint16_t slave_position;

    /// Index of the PDO object in the CoE object dictionary (e.g., 0x6000).
    uint16_t index;

    /// Subindex of the PDO object.
    uint8_t subindex;

    /// Size of the data in bits.
    uint8_t bit_length;

    explicit Channel(xjson::Parser &parser):
        enabled(parser.field<bool>("enabled", true)),
        slave_position(parser.field<uint16_t>("slave_position")),
        index(static_cast<uint16_t>(parser.field<int>("index"))),
        subindex(static_cast<uint8_t>(parser.field<int>("subindex"))),
        bit_length(static_cast<uint8_t>(parser.field<int>("bit_length"))) {}

    virtual ~Channel() = default;

    /// Returns the byte length rounded up from bit_length.
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }

    /// Converts this channel configuration to a PDOEntry.
    [[nodiscard]] PDOEntry to_pdo_entry(const bool is_input) const {
        return PDOEntry(slave_position, index, subindex, bit_length, is_input);
    }
};

/// Input channel configuration (TxPDO, slave→master).
struct Input final : Channel {
    /// The key of the Synnax channel to write data to.
    synnax::ChannelKey synnax_key;

    /// The Synnax channel object (populated after remote lookup).
    synnax::Channel ch;

    /// Offset into the input buffer where this channel's data resides.
    size_t buffer_offset;

    explicit Input(xjson::Parser &parser):
        Channel(parser),
        synnax_key(parser.field<synnax::ChannelKey>("channel")),
        buffer_offset(0) {}

    /// Binds remote channel information retrieved from Synnax.
    void bind_remote_info(const synnax::Channel &remote_ch) { this->ch = remote_ch; }
};

/// Output channel configuration (RxPDO, master→slave).
struct Output final : Channel {
    /// The key of the Synnax channel to receive commands from.
    synnax::ChannelKey command_key;

    /// The key of the Synnax channel to write state feedback to.
    synnax::ChannelKey state_key;

    /// The Synnax state channel object (populated after remote lookup).
    synnax::Channel state_ch;

    /// Offset into the output buffer where this channel's data resides.
    size_t buffer_offset;

    explicit Output(xjson::Parser &parser):
        Channel(parser),
        command_key(parser.field<synnax::ChannelKey>("channel")),
        state_key(parser.field<synnax::ChannelKey>("state_channel", 0)),
        buffer_offset(0) {}
};

/// Sorts a vector of channels in place by slave position, then by index.
template<typename ChannelType>
void sort_by_position(std::vector<ChannelType> &channels) {
    std::sort(channels.begin(), channels.end(), [](const auto &a, const auto &b) {
        if (a.slave_position != b.slave_position)
            return a.slave_position < b.slave_position;
        return a.index < b.index;
    });
}
}
