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

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/telem/telem.h"

namespace visa::channel {

/// @brief Response format types for SCPI command responses.
enum class ResponseFormat {
    FLOAT,         // Single float value
    INTEGER,       // Single integer value
    STRING,        // String value
    FLOAT_ARRAY,   // Comma-separated float array
    BINARY_BLOCK,  // IEEE 488.2 binary block
    BOOLEAN,       // Boolean value (0/1, ON/OFF, TRUE/FALSE)
};

/// @brief parses a ResponseFormat from a string.
/// Returns FLOAT as default for invalid values.
inline ResponseFormat parse_response_format(const std::string &str) {
    if (str == "float") return ResponseFormat::FLOAT;
    if (str == "integer") return ResponseFormat::INTEGER;
    if (str == "string") return ResponseFormat::STRING;
    if (str == "float_array") return ResponseFormat::FLOAT_ARRAY;
    if (str == "binary_block") return ResponseFormat::BINARY_BLOCK;
    if (str == "boolean") return ResponseFormat::BOOLEAN;
    return ResponseFormat::FLOAT; // Default to float for invalid values
}

/// @brief converts a ResponseFormat to a string.
inline std::string to_string(const ResponseFormat format) {
    switch (format) {
        case ResponseFormat::FLOAT: return "float";
        case ResponseFormat::INTEGER: return "integer";
        case ResponseFormat::STRING: return "string";
        case ResponseFormat::FLOAT_ARRAY: return "float_array";
        case ResponseFormat::BINARY_BLOCK: return "binary_block";
        case ResponseFormat::BOOLEAN: return "boolean";
        default: return "unknown";
    }
}

/// @brief Base channel configuration.
struct BaseChannel {
    synnax::ChannelKey synnax_key;
    synnax::Channel ch;
    std::string scpi_command;

    explicit BaseChannel(xjson::Parser &parser):
        synnax_key(parser.required<synnax::ChannelKey>("channel")),
        scpi_command(parser.required<std::string>("scpi_command")) {}

    BaseChannel(
        const synnax::ChannelKey synnax_key,
        std::string scpi_command
    ):
        synnax_key(synnax_key),
        scpi_command(std::move(scpi_command)) {}
};

/// @brief Input channel configuration (for reading from instrument).
struct InputChannel : BaseChannel {
    ResponseFormat format;
    telem::DataType data_type;
    std::string delimiter;
    size_t array_length;
    bool enabled;

    explicit InputChannel(xjson::Parser &parser):
        BaseChannel(parser),
        format(parse_response_format(parser.required<std::string>("format"))),
        data_type(telem::DataType(parser.optional<std::string>("data_type", "float64"))),
        delimiter(parser.optional<std::string>("delimiter", ",")),
        array_length(parser.optional<size_t>("array_length", 0)),
        enabled(parser.optional<bool>("enabled", true)) {}

    InputChannel(
        const synnax::ChannelKey synnax_key,
        std::string scpi_command,
        const ResponseFormat format,
        const telem::DataType data_type = telem::FLOAT64_T,
        std::string delimiter = ",",
        const size_t array_length = 0,
        const bool enabled = true
    ):
        BaseChannel(synnax_key, std::move(scpi_command)),
        format(format),
        data_type(data_type),
        delimiter(std::move(delimiter)),
        array_length(array_length),
        enabled(enabled) {}
};

/// @brief Output channel configuration (for writing to instrument).
struct OutputChannel : BaseChannel {
    std::string command_template;

    explicit OutputChannel(xjson::Parser &parser):
        BaseChannel(parser),
        command_template(parser.required<std::string>("command_template")) {}

    OutputChannel(
        const synnax::ChannelKey synnax_key,
        std::string scpi_command,
        std::string command_template
    ):
        BaseChannel(synnax_key, std::move(scpi_command)),
        command_template(std::move(command_template)) {}
};

}