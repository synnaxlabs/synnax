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
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::master {
/// @brief the DC registers of one slave that runs in a DC mode.
struct DCSetup {
    uint16_t position = 0;
    dc::Registers registers;
};

/// @brief returns the DC registers of each slave that plan enables in a DC mode, the
/// chosen mode or the one dc::select picks, at the period of plan.
/// @returns errors::SLAVE_CONFIG_ERROR when a slave cannot use its chosen mode, when a
/// mode gives values the registers cannot hold, or when the SYNC0 cycles that mark
/// process data share no dc::MIN_EVENT_CYCLE grid or longer with the period.
std::pair<std::vector<DCSetup>, x::errors::Error>
resolve_dc(const std::vector<slave::DiscoveryResult> &slaves, const Plan &plan);
}
