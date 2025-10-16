// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>
#include <sstream>
#include <algorithm>
#include <cctype>

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/visa/channels.h"

namespace visa::util {

/// @brief trims whitespace from both ends of a string.
inline std::string trim(const std::string &str) {
    const auto start = std::find_if_not(str.begin(), str.end(), [](unsigned char ch) {
        return std::isspace(ch);
    });
    const auto end = std::find_if_not(str.rbegin(), str.rend(), [](unsigned char ch) {
        return std::isspace(ch);
    }).base();
    return start < end ? std::string(start, end) : "";
}

/// @brief parses a single float from a SCPI response.
inline std::pair<double, xerrors::Error> parse_float(const std::string &response) {
    const std::string trimmed = trim(response);
    if (trimmed.empty())
        return {0.0, xerrors::Error("empty response")};

    char *end;
    errno = 0;
    const double value = strtod(trimmed.c_str(), &end);

    if (errno != 0 || end == trimmed.c_str())
        return {0.0, xerrors::Error("failed to parse float from: " + response)};

    return {value, xerrors::NIL};
}

/// @brief parses a single integer from a SCPI response.
inline std::pair<int64_t, xerrors::Error> parse_int(const std::string &response) {
    const std::string trimmed = trim(response);
    if (trimmed.empty())
        return {0, xerrors::Error("empty response")};

    char *end;
    errno = 0;
    const int64_t value = strtoll(trimmed.c_str(), &end, 10);

    if (errno != 0 || end == trimmed.c_str() || *end != '\0')
        return {0, xerrors::Error("failed to parse integer from: " + response)};

    return {value, xerrors::NIL};
}

/// @brief parses a boolean from a SCPI response.
inline std::pair<bool, xerrors::Error> parse_bool(const std::string &response) {
    const std::string trimmed = trim(response);
    if (trimmed.empty())
        return {false, xerrors::Error("empty response")};

    // Try numeric
    if (trimmed == "1" || trimmed == "0")
        return {trimmed == "1", xerrors::NIL};

    // Try text
    std::string upper = trimmed;
    std::transform(upper.begin(), upper.end(), upper.begin(), ::toupper);

    if (upper == "ON" || upper == "TRUE" || upper == "YES")
        return {true, xerrors::NIL};
    if (upper == "OFF" || upper == "FALSE" || upper == "NO")
        return {false, xerrors::NIL};

    return {false, xerrors::Error("failed to parse boolean from: " + response)};
}

/// @brief parses a comma-separated float array.
inline std::pair<std::vector<double>, xerrors::Error>
parse_float_array(const std::string &response, const std::string &delimiter = ",") {
    std::vector<double> values;
    std::stringstream ss(response);
    std::string token;

    while (std::getline(ss, token, delimiter[0])) {
        auto [val, err] = parse_float(token);
        if (err) return {std::vector<double>{}, err};
        values.push_back(val);
    }

    if (values.empty())
        return {std::vector<double>{}, xerrors::Error("no values parsed from array")};

    return {values, xerrors::NIL};
}

/// @brief parses an IEEE 488.2 binary block header (#<digit><length>).
/// @returns the length of the binary data and any error.
inline std::pair<size_t, xerrors::Error>
parse_binary_header(const std::string &response) {
    if (response.empty() || response[0] != '#')
        return {0, xerrors::Error("binary block must start with #")};

    if (response.length() < 2)
        return {0, xerrors::Error("binary block header too short")};

    const int num_digits = response[1] - '0';
    if (num_digits < 1 || num_digits > 9)
        return {0, xerrors::Error("invalid binary block digit count")};

    if (response.length() < static_cast<size_t>(2 + num_digits))
        return {0, xerrors::Error("binary block header truncated")};

    const std::string length_str = response.substr(2, num_digits);
    try {
        const size_t length = std::stoul(length_str);
        return {length, xerrors::NIL};
    } catch (const std::exception &e) {
        return {0, xerrors::Error("failed to parse binary block length: " +
                                  std::string(e.what()))};
    }
}

/// @brief parses a SCPI response based on the channel configuration.
inline std::pair<telem::Series, xerrors::Error>
parse_response(const std::string &response, const channel::InputChannel &ch) {
    switch (ch.format) {
        case channel::ResponseFormat::FLOAT: {
            auto [value, err] = parse_float(response);
            if (err) return {telem::Series(telem::UNKNOWN_T, 0), err};
            auto series = telem::Series(ch.data_type, 1);
            series.write(value);
            return {std::move(series), xerrors::NIL};
        }

        case channel::ResponseFormat::INTEGER: {
            auto [value, err] = parse_int(response);
            if (err) return {telem::Series(telem::UNKNOWN_T, 0), err};
            auto series = telem::Series(ch.data_type, 1);
            series.write(value);
            return {std::move(series), xerrors::NIL};
        }

        case channel::ResponseFormat::STRING: {
            const std::string trimmed = trim(response);
            auto series = telem::Series(trimmed, telem::STRING_T);
            return {std::move(series), xerrors::NIL};
        }

        case channel::ResponseFormat::FLOAT_ARRAY: {
            auto [values, err] = parse_float_array(response, ch.delimiter);
            if (err) return {telem::Series(telem::UNKNOWN_T, 0), err};

            const size_t expected_len = ch.array_length;
            if (expected_len > 0 && values.size() != expected_len)
                return {
                    telem::Series(telem::UNKNOWN_T, 0),
                    xerrors::Error("array length mismatch: expected " +
                                   std::to_string(expected_len) + ", got " +
                                   std::to_string(values.size()))
                };

            auto series = telem::Series(ch.data_type, values.size());
            for (const double v: values)
                series.write(v);
            return {std::move(series), xerrors::NIL};
        }

        case channel::ResponseFormat::BINARY_BLOCK: {
            auto [length, err] = parse_binary_header(response);
            if (err) return {telem::Series(telem::UNKNOWN_T, 0), err};

            // Binary data starts after header: #<digit><length>
            const size_t num_digits = response[1] - '0';
            const size_t header_len = 2 + num_digits;

            if (response.length() < header_len + length)
                return {telem::Series(telem::UNKNOWN_T, 0), xerrors::Error("binary block data truncated")};

            const auto *data = reinterpret_cast<const uint8_t *>(
                response.data() + header_len
            );
            const size_t num_samples = length / ch.data_type.density();

            auto series = telem::Series(ch.data_type, num_samples);
            series.write(data, length);
            return {std::move(series), xerrors::NIL};
        }

        case channel::ResponseFormat::BOOLEAN: {
            auto [value, err] = parse_bool(response);
            if (err) return {telem::Series(telem::UNKNOWN_T, 0), err};
            auto series = telem::Series(telem::UINT8_T, 1);
            series.write(static_cast<uint8_t>(value));
            return {std::move(series), xerrors::NIL};
        }

        default:
            return {telem::Series(telem::UNKNOWN_T, 0), xerrors::Error("unsupported response format")};
    }
}

}