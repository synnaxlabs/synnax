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
#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <utility>
#include <vector>

#include "driver/mil1553/backend.h"

namespace driver::mil1553::simulated {
/// @brief the number of transfers a simulated monitor or remote terminal channel
/// holds before it drops the oldest.
constexpr std::size_t RECEIVE_CAPACITY = 4096;

/// @brief Backend simulates MIL-STD-1553 cards in process. Each card and channel pair
/// is one bus with terminals 0 to 30, and every transfer completes when a bus
/// controller makes it. A terminal that no remote terminal channel owns wraps data
/// around: it answers a transmit command on a subaddress with the words last received
/// on that subaddress, zero before any. An owned terminal answers with the words its
/// channel set through respond, or busy with no data when none are set. The
/// simulation ignores bus timing.
class Backend final : public mil1553::Backend {
public:
    /// @brief Bus is one simulated bus.
    struct Bus;

    /// @returns CONFIGURATION_ERROR when the properties are invalid or another
    /// remote terminal channel owns one of their terminals.
    std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::mil1553::Properties &props) override;

    /// @returns one simulated channel: card 0, channel 0.
    std::pair<std::vector<Info>, x::errors::Error> list() override;

private:
    /// @brief mu guards buses.
    std::mutex mu;
    /// @brief buses holds each bus by card and channel.
    std::map<std::pair<std::uint16_t, std::uint16_t>, std::shared_ptr<Bus>> buses;
};
}
