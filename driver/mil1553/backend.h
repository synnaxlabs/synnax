// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/mil1553/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/queue.h"
#include "driver/codec/mil1553.h"

namespace driver::mil1553 {
/// @brief Transfer is one transfer made over or seen on the bus.
struct Transfer {
    /// @brief command is the command word the bus controller sent.
    codec::mil1553::Command command;
    /// @brief answered is true when the remote terminal sent a status word.
    bool answered = false;
    /// @brief status is the status word the remote terminal sent, when answered.
    codec::mil1553::Status status;
    /// @brief words holds the data words that moved. The first command.count are
    /// valid, except for a transmit command the terminal did not answer or answered
    /// busy, which moves none.
    std::array<std::uint16_t, codec::mil1553::MAX_WORDS> words{};
    /// @brief time is when the transfer ended, on the host clock.
    x::telem::TimeStamp time;

    /// @returns the number of valid data words.
    [[nodiscard]] std::size_t count() const {
        if (this->command.transmit && (!this->answered || this->status.busy)) return 0;
        return this->command.count;
    }
};

/// @brief Info describes one channel a backend can open.
struct Info {
    /// @brief card is the index of the card within the backend.
    std::uint16_t card = 0;
    /// @brief channel is the index of the channel on the card.
    std::uint16_t channel = 0;
    /// @brief name is a human-readable name for the channel.
    std::string name;
};

/// @brief Channel is one open MIL-STD-1553 channel in the role its properties name.
/// Safe to use from one thread at a time.
class Channel {
public:
    virtual ~Channel() = default;

    /// @brief makes one transfer as the bus controller. For a receive command, words
    /// holds the command.count words to send; for a transmit command it is ignored.
    /// @returns CRITICAL_HARDWARE_ERROR when the channel is not a bus controller or
    /// the command is a mode code, broadcast, or has the wrong number of words, or
    /// the error of the card.
    virtual std::pair<Transfer, x::errors::Error> transact(
        const codec::mil1553::Command &command,
        std::span<const std::uint16_t> words
    ) = 0;

    /// @brief reads the transfers seen since the last read, waiting up to timeout for
    /// the first. A monitor sees every transfer; a remote terminal sees the transfers
    /// to its terminals.
    /// @returns CRITICAL_HARDWARE_ERROR when the channel is a bus controller, or the
    /// error of the card.
    virtual std::pair<bus::Batch, x::errors::Error>
    read(std::span<Transfer> out, x::telem::TimeSpan timeout) = 0;

    /// @brief sets the words a terminal of this channel answers a transmit command
    /// on subaddress with. Words past the end of the command's count are cut, and a
    /// count past the end of words is filled with zeros.
    /// @returns CRITICAL_HARDWARE_ERROR when the channel is not a remote terminal or
    /// does not own rt, subaddress is not from 1 to 30, or words is empty or longer
    /// than 32.
    virtual x::errors::Error respond(
        std::uint8_t rt,
        std::uint8_t subaddress,
        std::span<const std::uint16_t> words
    ) = 0;
};

/// @brief Backend is one driver library for MIL-STD-1553 cards. Safe for concurrent
/// use.
class Backend {
public:
    virtual ~Backend() = default;

    /// @brief opens the channel the properties name, in their role.
    /// @returns the load error of the vendor library when it is missing, or
    /// CONFIGURATION_ERROR when the role or terminals are invalid or the channel
    /// cannot be opened in the role.
    virtual std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::mil1553::Properties &props) = 0;

    /// @brief lists the channels the backend can open.
    /// @returns the load error of the vendor library when it is missing.
    virtual std::pair<std::vector<Info>, x::errors::Error> list() = 0;
};

/// @brief checks the role and terminals of props: a remote terminal owns one or
/// more distinct terminals from 0 to 30, and other roles own none.
/// @returns CONFIGURATION_ERROR naming the problem.
x::errors::Error validate(const synnax::mil1553::Properties &props);
}
