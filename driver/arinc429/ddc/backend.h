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
#include <mutex>
#include <utility>
#include <vector>

#include "driver/arinc429/backend.h"
#include "driver/arinc429/ddc/api.h"

namespace driver::arinc429::ddc {
// TODO: confirm the range of logical device numbers the DDC Card Manager assigns.
/// @brief the number of logical device numbers list probes.
constexpr std::uint16_t MAX_CARDS = 16;

// TODO: confirm on a card whether bits 0 to 7 of an SDK word hold the label in wire
// order, as Word does, or in octal digit order, which would need codec::reverse.
/// @brief converts a word read in the original bit format, whose bit 31 is 1 when
/// the word arrived with even parity, to the word as it arrived.
codec::arinc429::Word from_card(unsigned long raw);

/// @brief converts a word to the original bit format for a transmitter that
/// generates odd parity.
unsigned long to_card(codec::arinc429::Word word);

/// @brief Backend reaches DDC ARINC 429 cards through the DD-42992 SDK. Card is the
/// logical device number from the DDC Card Manager, and channel is the receiver or
/// transmitter number less one. The backend loads the SDK on first use and keeps
/// its load error until the Driver restarts.
class Backend final : public arinc429::Backend {
public:
    /// @brief Cards is the state channels share with the backend.
    struct Cards;

    Backend();

    std::pair<std::unique_ptr<Channel>, x::errors::Error>
    open(const synnax::arinc429::Properties &props, Direction direction) override;

    /// @brief lists every channel of every card the SDK can open, skipping cards
    /// that are absent or in use by another process.
    std::pair<std::vector<Info>, x::errors::Error> list() override;

private:
    std::shared_ptr<Cards> cards;
};
}
