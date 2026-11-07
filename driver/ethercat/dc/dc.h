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
#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/errors/errors.h"

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

/// @brief the AssignActivate bit that turns on SYNC0.
constexpr uint16_t SYNC0_ACTIVE = 0x0200;
/// @brief the AssignActivate bit that turns on SYNC1.
constexpr uint16_t SYNC1_ACTIVE = 0x0400;
/// @brief the largest time a 32-bit DC register holds.
const x::telem::TimeSpan REGISTER_LIMIT(std::numeric_limits<uint32_t>::max());

/// @brief values for the DC registers of a slave at one bus cycle.
struct Registers {
    /// @brief value for the DC activation registers 0x0980 and 0x0981.
    uint16_t assign_activate = 0;
    /// @brief SYNC0 cycle in nanoseconds, for register 0x09A0. Zero when SYNC0 is off.
    uint32_t sync0_cycle = 0;
    /// @brief time in nanoseconds from a SYNC0 event to the next SYNC1 event, for
    /// register 0x09A4. Zero when SYNC1 is off.
    uint32_t sync1_cycle = 0;
    /// @brief SYNC0 shift in nanoseconds, added to the start time.
    int32_t sync0_shift = 0;

    bool operator==(const Registers &) const = default;
};

/// @brief returns the mode a slave runs in when no mode is chosen for it: its first DC
/// mode, or nullopt when it lists an SM mode or no mode.
inline std::optional<OpMode> select(const std::vector<OpMode> &modes) {
    const auto sm = [](const OpMode &m) { return m.assign_activate == 0; };
    if (modes.empty() || std::ranges::any_of(modes, sm)) return std::nullopt;
    return modes.front();
}

/// @brief returns the DC register values of the mode at the bus period.
/// @returns errors::SLAVE_CONFIG_ERROR, with the value and its limit, when the mode
/// gives no SYNC0 cycle, a SYNC0 cycle that is not a whole number of nanoseconds, a
/// SYNC1 cycle that is not a multiple of the SYNC0 cycle, a SYNC1 shift that puts
/// SYNC1 before SYNC0, or a cycle that a register cannot hold.
inline std::pair<Registers, x::errors::Error>
resolve(const OpMode &mode, const x::telem::TimeSpan period) {
    const auto fail = [](const std::string &msg) {
        return std::pair{
            Registers{},
            x::errors::Error(errors::SLAVE_CONFIG_ERROR, msg)
        };
    };
    Registers regs{
        .assign_activate = mode.assign_activate,
        .sync0_shift = mode.sync0_shift,
    };
    if ((mode.assign_activate & SYNC0_ACTIVE) == 0) return {regs, x::errors::NIL};
    x::telem::TimeSpan sync0(mode.sync0_cycle);
    if (sync0 == 0 && mode.sync0_factor > 0) sync0 = period * mode.sync0_factor;
    if (sync0 == 0 && mode.sync0_factor < 0) {
        const int64_t divisor = -mode.sync0_factor;
        if (period.nanoseconds() % divisor != 0)
            return fail(
                "SYNC0 cycle of " + period.to_string() + " / " +
                std::to_string(divisor) + " is not a whole number of nanoseconds"
            );
        sync0 = period / divisor;
    }
    if (sync0 == 0) return fail("mode " + mode.name + " gives no SYNC0 cycle");
    if (sync0 > REGISTER_LIMIT)
        return fail(
            "SYNC0 cycle of " + sync0.to_string() + " is over the limit of " +
            REGISTER_LIMIT.to_string()
        );
    regs.sync0_cycle = static_cast<uint32_t>(sync0.nanoseconds());
    if ((mode.assign_activate & SYNC1_ACTIVE) == 0) return {regs, x::errors::NIL};
    auto sync1 = sync0;
    if (mode.sync1_factor > 0) sync1 = sync0 * mode.sync1_factor;
    if (mode.sync1_factor < 0) sync1 = period * -mode.sync1_factor;
    if (sync1 % sync0 != 0)
        return fail(
            "SYNC1 cycle of " + sync1.to_string() +
            " is not a multiple of the SYNC0 cycle of " + sync0.to_string()
        );
    const auto shift = x::telem::TimeSpan(mode.sync1_shift);
    const auto sync1_register = sync1 - sync0 + shift;
    if (sync1_register < 0)
        return fail(
            "SYNC1 shift of " + shift.to_string() + " is under the limit of " +
            (sync0 - sync1).to_string()
        );
    if (sync1_register > REGISTER_LIMIT)
        return fail(
            "SYNC1 cycle of " + sync1.to_string() + " is over the limit of " +
            (REGISTER_LIMIT + sync0 - shift).to_string()
        );
    regs.sync1_cycle = static_cast<uint32_t>(sync1_register.nanoseconds());
    return {regs, x::errors::NIL};
}

}
