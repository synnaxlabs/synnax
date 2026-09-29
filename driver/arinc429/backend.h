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
#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/arinc429/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/queue.h"
#include "driver/codec/arinc429.h"

namespace driver::arinc429 {
/// @brief Direction is the way words travel on an open channel.
enum class Direction : std::uint8_t { RECEIVE, TRANSMIT };

/// @brief Received is one word a receive channel got.
struct Received {
    /// @brief word is the word as it arrived, parity included.
    codec::arinc429::Word word;
    /// @brief time is when the word arrived, on the host clock.
    x::telem::TimeStamp time;
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

/// @brief Channel is one open ARINC 429 channel. Safe to use from one thread at a
/// time.
class Channel {
public:
    virtual ~Channel() = default;

    /// @brief reads the words that arrived, waiting up to timeout for the first.
    /// @returns CRITICAL_HARDWARE_ERROR when the channel transmits, or the error of
    /// the card.
    virtual std::pair<bus::Batch, x::errors::Error>
    read(std::span<Received> out, x::telem::TimeSpan timeout) = 0;

    /// @brief transmits words in order.
    /// @returns CRITICAL_HARDWARE_ERROR when the channel receives, or the error of
    /// the card.
    virtual x::errors::Error write(std::span<const codec::arinc429::Word> words) = 0;
};

/// @brief Backend is one driver library for ARINC 429 cards. Safe for concurrent
/// use.
class Backend {
public:
    virtual ~Backend() = default;

    /// @brief opens the channel the properties name.
    /// @returns the load error of the vendor library when it is missing, or
    /// CONFIGURATION_ERROR when the channel cannot be opened in the direction.
    virtual std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::arinc429::Properties &props, Direction direction) = 0;

    /// @brief lists the channels the backend can open.
    /// @returns the load error of the vendor library when it is missing.
    virtual std::pair<std::vector<Info>, x::errors::Error> list() = 0;
};
}
