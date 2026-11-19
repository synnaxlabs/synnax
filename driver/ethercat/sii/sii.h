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

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/pdo/pdo.h"

/// @brief parsers for the categories of a slave's SII (EEPROM).
namespace driver::ethercat::sii {
/// @brief the SII word where the categories start.
constexpr uint16_t CATEGORIES = 0x40;
/// @brief the category type that ends the categories.
constexpr uint16_t END = 0xFFFF;
/// @brief the type of the strings category.
constexpr uint16_t STRINGS = 10;
/// @brief the type of the DC category.
constexpr uint16_t DC = 60;

/// @brief returns the SII string at index, or an empty string when index is 0.
using Strings = std::function<std::string(uint8_t index)>;

/// @brief parses the strings category. data holds the category after its length word.
/// @returns a lookup that gives an empty string for an index the category does not
/// hold.
Strings parse_strings(std::span<const uint8_t> data);

/// @brief parses a TxPDO or RxPDO category. data holds the category after its length
/// word. Keeps only the PDOs on a sync manager that keep accepts, and leaves out a PDO
/// that runs past the end of data.
std::vector<pdo::PDO> parse_pdos(
    std::span<const uint8_t> data,
    const std::function<bool(uint8_t sync)> &keep,
    const Strings &strings
);

/// @brief parses the DC category, the synchronization modes of a slave, in SII order.
/// data holds the category after its length word.
std::vector<dc::OpMode>
parse_op_modes(std::span<const uint8_t> data, const Strings &strings);
}
