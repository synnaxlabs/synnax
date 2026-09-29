// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <map>
#include <memory>
#include <mutex>
#include <utility>

#include "driver/arinc429/backend.h"

namespace driver::arinc429::simulated {
/// @brief the number of words a simulated receive channel holds before it drops the
/// oldest.
constexpr std::size_t RECEIVE_CAPACITY = 16384;

/// @brief Backend simulates ARINC 429 cards in process. Each card and channel pair is
/// one wire: the words its transmitter writes arrive, in order, at every receiver
/// open on the same pair when they are written. A wire has at most one transmitter,
/// as on a real bus. The simulation ignores the bit rate.
class Backend final : public arinc429::Backend {
public:
    /// @brief Line is one simulated wire.
    struct Line;

    std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::arinc429::Properties &props, Direction direction) override;

    /// @returns one simulated channel: card 0, channel 0.
    std::pair<std::vector<Info>, x::errors::Error> list() override;

private:
    /// @brief mu guards lines.
    std::mutex mu;
    /// @brief lines holds each wire by card and channel.
    std::map<std::pair<std::uint16_t, std::uint16_t>, std::shared_ptr<Line>> lines;
};
}
