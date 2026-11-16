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
#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

/// @brief resolves IANA time zone names against a zone database embedded in the
/// binary, the same one x/go/tz reads.
namespace x::tz {

/// @brief the UTC offset and abbreviation in effect in a zone at one instant.
struct Info {
    /// @brief seconds east of UTC.
    int32_t offset = 0;
    /// @brief the zone abbreviation, such as "MST" or "+03".
    std::string abbreviation;
};

/// @brief the rules of one zone, read from its TZif data.
class Zone {
public:
    /// @brief reads TZif data for the zone called name.
    /// @returns a VALIDATION error when data is not valid TZif.
    static std::pair<std::shared_ptr<const Zone>, x::errors::Error>
    parse(std::string name, std::string_view data);

    /// @brief the zone's name, such as "America/Denver".
    [[nodiscard]] const std::string &name() const { return this->zone_name; }

    /// @brief returns the offset and abbreviation in effect at unix_seconds.
    [[nodiscard]] Info at(int64_t unix_seconds) const;

private:
    struct LocalType {
        int32_t offset;
        bool dst;
        std::string abbreviation;
    };

    struct Transition {
        int64_t at;
        uint8_t type;
    };

    std::string zone_name;
    std::vector<LocalType> types;
    std::vector<Transition> transitions;
    /// @brief the POSIX rule string for instants after the last transition.
    std::string rule;

    [[nodiscard]] size_t first_type() const;
};

/// @brief resolves zone names to zones. Names match without regard to case. Not safe
/// for concurrent use.
class Database {
public:
    /// @brief indexes the zone database embedded in the binary.
    Database();

    /// @brief returns the zone called name, matched without regard to case.
    /// @returns a NOT_FOUND error when the database has no zone with that name.
    std::pair<std::shared_ptr<const Zone>, x::errors::Error>
    load(std::string_view name);

    /// @brief returns every zone name in the database, sorted.
    [[nodiscard]] std::vector<std::string> names() const;

private:
    struct File {
        std::string_view name;
        std::string_view data;
    };

    /// @brief each zone file, by lowercase name.
    std::unordered_map<std::string, File> files;
    /// @brief each zone already loaded, by lowercase name.
    std::unordered_map<std::string, std::shared_ptr<const Zone>> zones;
};

}
