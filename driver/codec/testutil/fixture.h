// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <charconv>
#include <cstdint>
#include <filesystem>
#include <optional>
#include <string>
#include <system_error>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

namespace driver::codec::testutil {
/// @brief returns the text of the value at pointer in doc, or nothing when there is
/// none. The fixtures hold numbers such as 1e999 that nlohmann rejects, so it reads
/// them as text.
std::optional<std::string> raw_at(const std::string &doc, const std::string &pointer);

/// @brief returns the text of the value at pointer in doc, failing when there is none.
std::string required_at(const std::string &doc, const std::string &pointer);

/// @brief returns the string at pointer in doc, or an empty string when there is none.
std::string string_at(const std::string &doc, const std::string &pointer);

/// @brief returns the bytes of the lowercase hex text at pointer in doc, failing when
/// there is none.
std::vector<std::uint8_t> bytes_at(const std::string &doc, const std::string &pointer);

/// @brief returns the integer that raw, the text of a JSON integer, holds exactly.
template<typename T>
T integer_of(const std::string &raw) {
    T v{};
    const auto [end, ec] = std::from_chars(raw.data(), raw.data() + raw.size(), v);
    EXPECT_TRUE(ec == std::errc{} && end == raw.data() + raw.size())
        << raw << " is not an integer in the range of its type";
    return v;
}

/// @brief returns the error text of a Go validation error without its suffix.
std::string error_data(const std::string &go);

/// @brief returns the message m, in the form that the fixtures write it. Message is
/// JSONMessage, TextMessage, or BinaryMessage.
template<typename Message>
Message parse_message(const nlohmann::json &m);

/// @brief returns the text of the fixture at path.
std::string read_fixture(const std::filesystem::path &path);

/// @brief returns the JSON fixtures in dir, in path order.
std::vector<std::filesystem::path> fixture_paths(const std::filesystem::path &dir);
}
