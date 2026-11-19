// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <format>
#include <string>

#include "driver/ethercat/master/dc.h"

namespace driver::ethercat::master {
std::pair<std::vector<DCSetup>, x::errors::Error>
resolve_dc(const std::vector<slave::DiscoveryResult> &slaves, const Plan &plan) {
    const auto period = plan.period;
    std::vector<DCSetup> setups;
    std::vector<x::telem::TimeSpan> cycles;
    std::string events;
    for (const auto &slave: slaves) {
        const auto &props = slave.properties;
        const auto it = plan.slaves.find(props.position);
        const auto config = it == plan.slaves.end() ? slave::Config{} : it->second;
        if (!config.enabled) continue;
        const auto choice = config.op_mode ? config.op_mode : props.catalog_op_mode();
        const auto [mode, select_err] = dc::select(props.op_modes, choice);
        if (select_err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "module {} ({}) cannot use its chosen mode: {}",
                        props.position,
                        props.name,
                        select_err.data
                    )
                )
            };
        if (!mode) continue;
        auto [regs, err] = dc::resolve(
            *mode,
            period,
            choice ? choice->cycles : dc::Cycles{}
        );
        if (err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "module {} ({}) cannot run its DC mode at a bus cycle of "
                        "{}: {}",
                        props.position,
                        props.name,
                        period.to_string(),
                        err.data
                    )
                )
            };
        setups.push_back({.position = props.position, .registers = regs});
        if (!regs.marks_data() || (regs.assign_activate & dc::PDI_SYNC_UNIT) != 0)
            continue;
        cycles.emplace_back(regs.sync0_cycle);
        if (!events.empty()) events += ", ";
        events += std::format(
            "module {} ({}) every {}",
            props.position,
            props.name,
            cycles.back().to_string()
        );
    }
    if (!cycles.empty() && dc::grid(period, cycles) <
                               static_cast<double>(dc::MIN_EVENT_CYCLE.nanoseconds()))
        return {
            {},
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                std::format(
                    "SYNC0 events do not line up with the bus cycle of {}: {}. Change "
                    "the bus rate or the SYNC0 cycles so that a span of {} or more "
                    "divides the bus cycle and each SYNC0 cycle.",
                    period.to_string(),
                    events,
                    dc::MIN_EVENT_CYCLE.to_string()
                )
            )
        };
    return {setups, x::errors::NIL};
}
}
