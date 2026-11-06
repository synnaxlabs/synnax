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

#include "x/cpp/json/json.h"

namespace driver::ethercat::dc {

/// @brief a synchronization mode that a slave lists in its SII, with the raw values of
/// the SII. A slave lists SM synchronization as a mode with an AssignActivate of zero.
struct OpMode {
    /// @brief name of the mode, such as "DC-Synchron". Empty when the slave gives none.
    std::string name;
    /// @brief value for the DC activation registers 0x0980 and 0x0981.
    uint16_t assign_activate = 0;
    /// @brief SYNC0 cycle in nanoseconds. Zero means that sync0_factor sets the cycle.
    uint32_t sync0_cycle = 0;
    /// @brief SYNC0 cycle as a multiple of the bus cycle when positive, or as a
    /// fraction of it (bus cycle / |factor|) when negative.
    int16_t sync0_factor = 0;
    /// @brief SYNC0 shift in nanoseconds.
    int32_t sync0_shift = 0;
    /// @brief SYNC1 cycle as a multiple of the SYNC0 cycle when positive, or of the
    /// bus cycle when negative.
    int16_t sync1_factor = 0;
    /// @brief SYNC1 shift in nanoseconds after SYNC0.
    int32_t sync1_shift = 0;

    bool operator==(const OpMode &) const = default;

    /// @brief parses a mode from JSON.
    static OpMode parse(x::json::Parser parser) {
        return {
            .name = parser.field<std::string>("name"),
            .assign_activate = parser.field<uint16_t>("assign_activate"),
            .sync0_cycle = parser.field<uint32_t>("sync0_cycle"),
            .sync0_factor = parser.field<int16_t>("sync0_factor"),
            .sync0_shift = parser.field<int32_t>("sync0_shift"),
            .sync1_factor = parser.field<int16_t>("sync1_factor"),
            .sync1_shift = parser.field<int32_t>("sync1_shift"),
        };
    }

    /// @brief serializes the mode to JSON.
    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"name", this->name},
            {"assign_activate", this->assign_activate},
            {"sync0_cycle", this->sync0_cycle},
            {"sync0_factor", this->sync0_factor},
            {"sync0_shift", this->sync0_shift},
            {"sync1_factor", this->sync1_factor},
            {"sync1_shift", this->sync1_shift},
        };
    }
};

}
