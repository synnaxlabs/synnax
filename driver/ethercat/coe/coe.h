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
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/pdo/pdo.h"

namespace driver::ethercat::coe {
/// @brief an SDO read of an object that the slave does not have.
const x::errors::Error OBJECT_MISSING = errors::SDO_READ_ERROR.sub("object_missing");
/// @brief an SDO read of a sub-index that an object of the slave does not have.
const x::errors::Error SUBINDEX_MISSING = errors::SDO_READ_ERROR.sub(
    "subindex_missing"
);
/// @brief an SDO read that got no reply.
const x::errors::Error NO_REPLY = errors::SDO_READ_ERROR.sub("no_reply");

/// @brief returns the error for an SDO read of index:sub_index that failed. Matches
/// NO_REPLY when no reply came, OBJECT_MISSING when the slave does not have the
/// object, and SUBINDEX_MISSING when the object does not have the sub-index.
x::errors::Error
read_error(uint16_t index, uint8_t sub_index, const errors::SDOFailure &failure);

/// @brief returns read_error() for a read that the slave aborted with abort_code. For
/// an abort_code of 0, it returns errors::SDO_READ_ERROR, as a master that reports only
/// abort codes cannot tell a read with no reply from one with a bad reply.
x::errors::Error read_error(uint16_t index, uint8_t sub_index, uint32_t abort_code);

/// @brief reads an unsigned value of at most four bytes from the object dictionary of
/// one slave. Returns read_error() when the read fails.
using Read = std::function<
    std::pair<uint32_t, x::errors::Error>(uint16_t index, uint8_t sub_index)>;

/// @brief reads a whole object from the object dictionary of one slave with complete
/// access. Returns its bytes from sub-index 0, or nullopt when the read fails.
using ReadAll = std::function<std::optional<std::vector<uint8_t>>(uint16_t index)>;

/// @brief returns a Read for read_layout that reads the sync manager type, PDO assign,
/// and PDO mapping objects with one read_all call each, as SOEM reads them. It uses
/// read for any other object, and for an object that read_all fails or that is too
/// short for its count.
Read with_complete_access(Read read, ReadAll read_all);

/// @brief what read_layout does after a failed read.
enum class OnError {
    /// @brief returns the error and no layout.
    FAIL,
    /// @brief leaves out what the read describes and keeps the rest, as SOEM does when
    /// it maps a slave from CoE. A count above the maximum reads as the maximum.
    SKIP,
};

/// @brief reads the PDOs that a slave assigns to each process data sync manager and the
/// objects each PDO maps. Returns no sync managers when the slave has no sync manager
/// type object (0x1C00) or no process data sync manager.
/// @param max_syncs the most sync managers the master configures on a slave. The
/// layout holds no sync manager past it.
/// @returns errors::PDO_DISCOVERY_ERROR when a PDO assign or mapping object counts more
/// sub-indices than it can hold, or the error of a failed read. With OnError::SKIP, the
/// first such error comes with the layout, except a failed read of 0x1C00:00.
std::pair<std::vector<pdo::Sync>, x::errors::Error>
read_layout(const Read &read, uint32_t max_syncs, OnError on_error);
}
