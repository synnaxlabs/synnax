// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <functional>
#include <ostream>
#include <string>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "boost/uuid/uuid.hpp"
#include "boost/uuid/uuid_generators.hpp"
#include "boost/uuid/uuid_hash.hpp"
#include "boost/uuid/uuid_io.hpp"

namespace x::uuid {

/// @brief Error type for invalid UUID parsing.
const errors::Error INVALID = errors::SY.sub("uuid.invalid");

/// @brief Generate a new random UUID (v4).
/// @returns A newly generated random UUID.

/// @brief A wrapper class around boost::uuid::uuid providing value semantics
/// and integration with Synnax's serialization infrastructure.
class UUID {
    boost::uuids::uuid value;

public:
    /// @brief Default constructor - creates a nil UUID (all zeros).
    UUID();

    /// @brief Construct from boost::uuid::uuid.
    explicit UUID(const boost::uuids::uuid &u);

    /// @brief Construct from raw bytes (16 bytes).
    explicit UUID(const std::array<std::uint8_t, 16> &bytes);

    /// @brief Parse a UUID from a string representation.
    /// @param str String in format "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    /// @returns A pair of (UUID, Error) - Error is nil on success.
    static std::pair<UUID, errors::Error> parse(const std::string &str);

    /// @brief Parse from JSON parser (for Oracle integration).
    /// Reads the current value as a string and parses it as a UUID.
    /// @param parser The JSON parser positioned at a string value.
    /// @returns The parsed UUID, or a nil UUID if parsing fails.
    static UUID parse(json::Parser parser);

    /// @brief Check if this is a nil (all-zeros) UUID.
    /// @returns true if this UUID is nil, false otherwise.
    [[nodiscard]] bool is_nil() const;

    /// @brief Convert to string representation.
    /// @returns UUID in format "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
    [[nodiscard]] std::string to_string() const;

    /// @brief Convert to JSON (as a string).
    /// @returns JSON string representation of the UUID.
    [[nodiscard]] json::json to_json() const;

    /// @brief Access the underlying boost::uuid.
    /// @returns Reference to the underlying boost::uuids::uuid.
    [[nodiscard]] const boost::uuids::uuid &underlying() const;

    /// @brief Access raw byte data.
    /// @returns Pointer to the 16-byte UUID data.
    [[nodiscard]] const std::uint8_t *data() const;

    /// @brief Size of UUID in bytes (always 16).
    [[nodiscard]] static constexpr std::size_t size() { return 16; }

    // Comparison operators
    bool operator==(const UUID &other) const;
    bool operator!=(const UUID &other) const;
    bool operator<(const UUID &other) const;
    bool operator>(const UUID &other) const;
    bool operator<=(const UUID &other) const;
    bool operator>=(const UUID &other) const;

    // Stream output
    friend std::ostream &operator<<(std::ostream &os, const UUID &uuid);
};

/// @brief A nil (all-zeros) UUID constant.
inline const UUID NIL;

inline UUID generate() {
    static thread_local boost::uuids::random_generator gen;
    return UUID{gen()};
}

}

/// @brief Hash support for use in std::unordered_map/set.
template<>
struct std::hash<x::uuid::UUID> {
    size_t operator()(const x::uuid::UUID &uuid) const noexcept;
};
