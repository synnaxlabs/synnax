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
#include <optional>
#include <span>
#include <string>
#include <system_error>
#include <utility>

#include "x/cpp/errors/errors.h"

/// @brief port operations that Linux and macOS share.
namespace driver::serial::native {
/// @returns the translated errno for a setting that the port rejected.
x::errors::Error failed(const std::string &setting, const std::string &port);

/// @brief removes the parity, framing, and break marks that PARMRK puts in the input,
/// and turns each doubled 0xFF back into one. data may end inside a mark.
/// @param marked how many bytes of a mark ended the last call. strip updates it.
/// @returns the number of bytes left at the front of data, and the number of marks.
std::pair<std::size_t, std::size_t>
strip(std::uint8_t &marked, std::span<std::uint8_t> data);

/// @returns the port's receive overrun count, or nothing when the port does not count
/// overruns.
std::pair<std::optional<std::uint32_t>, std::error_code> overrun_count(int fd);

/// @returns the overruns between last and count, which the port's 32-bit counter may
/// have wrapped. The first count sets last and gives zero.
std::size_t
overruns_since(std::optional<std::uint32_t> &last, std::optional<std::uint32_t> count);
}
