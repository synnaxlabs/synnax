// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <string>

namespace url {
/// @brief A simple URL builder.
struct URL {
    /// @brief The IP address of the target.
    std::string ip;
    /// @brief The port of the target.
    std::uint16_t port = 0;
    /// @brief Supplementary path information.
    std::string path;

    URL() = default;

    /// @brief Creates a URL with the given IP, port, and path.
    URL(std::string ip, std::uint16_t port, const std::string &path);

    /// @brief Parses the given address into a URL.
    /// @throws std::invalid_argument if the address is not a valid URL.
    explicit URL(const std::string &address);

    /// @brief Creates a child URL by appending the given path to the current path.
    /// @returns the child URL. It is guaranteed to have a single slash between the
    /// current path and child path, and have a trailing slash.
    [[nodiscard]] URL child(const std::string &child_path) const;

    /// @brief Converts the URL to a string.
    /// @returns the URL as a string.
    [[nodiscard]] std::string to_string() const;

    [[nodiscard]] std::string host_address() const;
};
}
