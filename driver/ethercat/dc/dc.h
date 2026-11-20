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
#include <format>
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
/// @brief the AssignActivate bit that gives the sync unit to the PDI, so the master
/// writes no cycle registers.
constexpr uint16_t PDI_SYNC_UNIT = 0x0001;
/// @brief the largest time a 32-bit DC register holds.
const x::telem::TimeSpan REGISTER_LIMIT(std::numeric_limits<uint32_t>::max());
/// @brief the shortest SYNC0 cycle whose events mark when a slave takes or applies
/// process data, the period of the fastest bus rate. A faster SYNC0 clocks hardware in
/// the slave, such as an ADC.
const x::telem::TimeSpan MIN_EVENT_CYCLE = x::telem::MICROSECOND * 100;

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

    /// @brief true when the SYNC0 events mark when the slave takes or applies process
    /// data: SYNC0 runs with a cycle of MIN_EVENT_CYCLE or more.
    [[nodiscard]] bool marks_data() const {
        return x::telem::TimeSpan(this->sync0_cycle) >= MIN_EVENT_CYCLE;
    }
};

/// @brief returns the largest span in nanoseconds that divides each cycle a whole
/// number of times and divides the period within a nanosecond. SYNC0 events with these
/// cycles fall at the same phases in each span. Requires at least one cycle, and each
/// cycle must be positive.
double grid(x::telem::TimeSpan period, const std::vector<x::telem::TimeSpan> &cycles);

/// @brief SYNC0 and SYNC1 cycles that replace the cycles a mode gives.
struct Cycles {
    /// @brief the SYNC0 cycle, or zero for the cycle of the mode.
    x::telem::TimeSpan sync0{0};
    /// @brief the time between SYNC1 events, or zero for the cycle of the mode. A
    /// nonzero cycle turns SYNC1 on in a mode that leaves it off.
    x::telem::TimeSpan sync1{0};

    bool operator==(const Cycles &) const = default;
};

/// @brief returns errors::SLAVE_CONFIG_ERROR when cycles gives a cycle to a mode that
/// cannot use it: one that does not run SYNC0, or one that gives the sync unit to the
/// PDI.
inline x::errors::Error check_cycles(const OpMode &mode, const Cycles &cycles) {
    const bool sync0 = (mode.assign_activate & SYNC0_ACTIVE) != 0;
    const bool pdi = (mode.assign_activate & PDI_SYNC_UNIT) != 0;
    if (sync0 && !pdi) return x::errors::NIL;
    const auto *const reason = sync0 ? "gives the sync unit to the PDI"
                                     : "does not run SYNC0";
    for (const auto &[name, cycle]:
         {std::pair{"SYNC0", cycles.sync0}, std::pair{"SYNC1", cycles.sync1}})
        if (cycle != 0)
            return x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                std::format(
                    "mode {}(AssignActivate 0x{:04X}) {}, so it cannot use a {} cycle "
                    "of {}",
                    mode.name.empty() ? "" : "\"" + mode.name + "\" ",
                    mode.assign_activate,
                    reason,
                    name,
                    cycle.to_string()
                )
            );
    return x::errors::NIL;
}

/// @brief the mode that a user chose for a slave.
struct Choice {
    /// @brief the index of the mode in the modes of the slave.
    uint16_t index = 0;
    /// @brief the AssignActivate of the mode, which shows that the index is stale when
    /// the slave lists other modes.
    uint16_t assign_activate = 0;
    /// @brief the cycles that replace the cycles of the mode.
    Cycles cycles;

    bool operator==(const Choice &) const = default;

    /// @brief parses a choice from JSON, adding a field error for a negative cycle.
    static Choice parse(x::json::Parser parser) {
        Choice choice{
            .index = parser.field<uint16_t>("index"),
            .assign_activate = parser.field<uint16_t>("assign_activate"),
            .cycles = {
                .sync0 = x::telem::TimeSpan(parser.field<int64_t>("sync0_cycle", 0)),
                .sync1 = x::telem::TimeSpan(parser.field<int64_t>("sync1_cycle", 0)),
            },
        };
        if (choice.cycles.sync0 < 0)
            parser.field_err("sync0_cycle", "must be 0 or more");
        if (choice.cycles.sync1 < 0)
            parser.field_err("sync1_cycle", "must be 0 or more");
        return choice;
    }

    /// @brief serializes the choice to JSON.
    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"index", this->index},
            {"assign_activate", this->assign_activate},
            {"sync0_cycle", this->cycles.sync0.nanoseconds()},
            {"sync1_cycle", this->cycles.sync1.nanoseconds()},
        };
    }
};

/// @brief returns the DC mode a slave runs in, or nullopt for SM sync. With no choice,
/// it is the first DC mode of a slave that lists no SM mode.
/// @returns errors::SLAVE_CONFIG_ERROR, with the modes of the slave, when modes has no
/// mode at the index of choice with its AssignActivate, or when choice gives a cycle to
/// a mode that cannot use it.
inline std::pair<std::optional<OpMode>, x::errors::Error>
select(const std::vector<OpMode> &modes, const std::optional<Choice> &choice = {}) {
    if (!choice) {
        const auto sm = [](const OpMode &m) { return m.assign_activate == 0; };
        if (modes.empty() || std::ranges::any_of(modes, sm))
            return {std::nullopt, x::errors::NIL};
        return {modes.front(), x::errors::NIL};
    }
    if (choice->index < modes.size() &&
        modes[choice->index].assign_activate == choice->assign_activate) {
        const auto &mode = modes[choice->index];
        if (const auto err = check_cycles(mode, choice->cycles))
            return {std::nullopt, err};
        if (mode.assign_activate == 0) return {std::nullopt, x::errors::NIL};
        return {mode, x::errors::NIL};
    }
    std::string listed;
    for (size_t i = 0; i < modes.size(); ++i) {
        listed += std::format("{}{} ", listed.empty() ? "" : ", ", i);
        if (!modes[i].name.empty()) listed += std::format("\"{}\" ", modes[i].name);
        listed += std::format("(AssignActivate 0x{:04X})", modes[i].assign_activate);
    }
    return {
        std::nullopt,
        x::errors::Error(
            errors::SLAVE_CONFIG_ERROR,
            std::format(
                "the module does not list mode {} with AssignActivate 0x{:04X}. {}",
                choice->index,
                choice->assign_activate,
                listed.empty() ? "It lists no modes." : "It lists " + listed + "."
            )
        )
    };
}

/// @brief returns the DC register values of the mode at the bus period. A nonzero cycle
/// in cycles replaces the cycle that the mode gives.
/// @param cycles must pass the check of select.
/// @returns errors::SLAVE_CONFIG_ERROR, with the value and its limit, when the mode
/// gives no SYNC0 cycle, a SYNC0 cycle that is not a whole number of nanoseconds, a
/// SYNC1 cycle that is not a multiple of the SYNC0 cycle, a SYNC1 shift that puts SYNC1
/// before SYNC0, or a cycle that a register cannot hold.
inline std::pair<Registers, x::errors::Error> resolve(
    const OpMode &mode,
    const x::telem::TimeSpan period,
    const Cycles &cycles = {}
) {
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
    if (cycles.sync0 != 0) sync0 = cycles.sync0;
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
    if (cycles.sync1 != 0) regs.assign_activate |= SYNC1_ACTIVE;
    if ((regs.assign_activate & SYNC1_ACTIVE) == 0) return {regs, x::errors::NIL};
    auto sync1 = sync0;
    if (mode.sync1_factor > 0) sync1 = sync0 * mode.sync1_factor;
    if (mode.sync1_factor < 0) sync1 = period * -mode.sync1_factor;
    if (cycles.sync1 != 0) sync1 = cycles.sync1;
    if (sync1 % sync0 != 0)
        return fail(
            "SYNC1 cycle of " + sync1.to_string() +
            " is not a multiple of the SYNC0 cycle of " + sync0.to_string()
        );
    const auto shift = x::telem::TimeSpan(mode.sync1_shift);
    // Checked before the register value, which a chosen cycle could overflow.
    const auto sync1_limit = REGISTER_LIMIT + sync0 - shift;
    if (sync1 > sync1_limit)
        return fail(
            "SYNC1 cycle of " + sync1.to_string() + " is over the limit of " +
            sync1_limit.to_string()
        );
    const auto sync1_register = sync1 - sync0 + shift;
    if (sync1_register < 0)
        return fail(
            "SYNC1 shift of " + shift.to_string() + " is under the limit of " +
            (sync0 - sync1).to_string()
        );
    regs.sync1_cycle = static_cast<uint32_t>(sync1_register.nanoseconds());
    return {regs, x::errors::NIL};
}

}
