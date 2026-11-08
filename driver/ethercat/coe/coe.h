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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::coe {
/// @brief an SDO read of an object that the slave does not have.
const x::errors::Error OBJECT_MISSING = errors::SDO_READ_ERROR.sub("object_missing");

/// @brief returns the error for an SDO read of index:sub_index that the slave aborted
/// with abort_code, or that got no answer when abort_code is 0. Matches OBJECT_MISSING
/// when the slave does not have the object.
x::errors::Error read_error(uint16_t index, uint8_t sub_index, uint32_t abort_code);

/// @brief reads an unsigned value of at most four bytes from the object dictionary of
/// one slave. Returns read_error() when the read fails.
using Read = std::function<
    std::pair<uint32_t, x::errors::Error>(uint16_t index, uint8_t sub_index)>;

/// @brief an object that a PDO maps.
struct Entry {
    /// @brief the index of the object.
    uint16_t index;
    /// @brief the sub-index of the object.
    uint8_t sub_index;
    /// @brief the number of bits the PDO maps.
    uint8_t bit_length;
    /// @brief the name of the object, or empty when the slave gives none.
    std::string name;
    /// @brief the CoE data type of the object.
    telem::DataType type = telem::DataType::EC_UNKNOWN;
};

/// @brief a PDO and the objects it maps, in order.
struct PDO {
    /// @brief the index of the PDO, such as 0x1A00.
    uint16_t index;
    /// @brief the objects the PDO maps.
    std::vector<Entry> entries;
    /// @brief the name of the PDO, when the slave gives one.
    std::optional<std::string> name;
};

/// @brief a process data sync manager and the PDOs it carries.
struct Sync {
    /// @brief the number of the sync manager.
    uint8_t index;
    /// @brief whether the sync manager carries inputs.
    bool is_input;
    /// @brief the PDOs the sync manager carries, in order.
    std::vector<PDO> pdos;
    /// @brief the slave has no PDO assign object for the sync manager, so its PDOs are
    /// the ones its EEPROM lists.
    bool fixed = false;
};

/// @brief reads the PDOs that a slave assigns to each process data sync manager and the
/// objects each PDO maps. Returns no sync managers when the slave has no sync manager
/// type object (0x1C00) or no process data sync manager.
/// @param max_syncs the most sync managers the master configures on a slave.
/// @returns errors::PDO_DISCOVERY_ERROR when an object counts more sub-indices than it
/// can hold, or the error of a failed read.
std::pair<std::vector<Sync>, x::errors::Error>
read_layout(const Read &read, uint32_t max_syncs);

/// @brief appends the objects that pdo maps to pdos, as padding where the PDO maps no
/// object. Infers a data type from the bit length when the CoE type is unknown.
void append(std::vector<pdo::Properties> &pdos, bool is_input, const PDO &pdo);
}
