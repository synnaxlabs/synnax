// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <span>
#include <string_view>

/// @brief the zone database that x/go/tz/gen.go writes into zoneinfo.inc.
namespace x::tz::zoneinfo {

/// @brief the name and place of one zone file in data().
struct Entry {
    /// @brief the zone name, such as "America/Denver".
    std::string_view name;
    /// @brief where the zone file starts in data().
    std::size_t offset;
    /// @brief the size of the zone file in bytes.
    std::size_t size;
};

/// @brief returns every zone file's bytes, back to back.
std::string_view data();

/// @brief returns the zone files in data(), sorted by name.
std::span<const Entry> entries();

}
