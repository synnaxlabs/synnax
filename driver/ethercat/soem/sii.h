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
#include <functional>
#include <span>
#include <string>
#include <vector>

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/dc/dc.h"

/// @brief parsers for the categories of a slave's SII (EEPROM).
namespace driver::ethercat::soem::sii {
/// @brief returns the SII string at index, or an empty string when index is 0.
using Strings = std::function<std::string(uint8_t index)>;

/// @brief parses a TxPDO or RxPDO category. data holds the category after its length
/// word. Leaves out a PDO on a sync manager at or above max_syncs, and a PDO that runs
/// past the end of data.
std::vector<coe::PDO>
parse_pdos(std::span<const uint8_t> data, uint8_t max_syncs, const Strings &strings);

/// @brief parses the DC category, the synchronization modes of a slave, in SII order.
/// data holds the category after its length word.
std::vector<dc::OpMode>
parse_op_modes(std::span<const uint8_t> data, const Strings &strings);
}
