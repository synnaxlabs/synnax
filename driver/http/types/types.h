// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "x/cpp/json/json.h"

namespace driver::http {
/// @brief supported HTTP methods.
enum class Method { GET, HEAD, POST, PUT, DEL, PATCH, OPTIONS, TRACE, CONNECT };

/// @brief returns the HTTP method string (e.g., Method::DEL returns "DELETE").
/// @param m the method.
/// @returns the method string.
constexpr const char *to_string(const Method m) {
    switch (m) {
        case Method::GET:
            return "GET";
        case Method::HEAD:
            return "HEAD";
        case Method::POST:
            return "POST";
        case Method::PUT:
            return "PUT";
        case Method::DEL:
            return "DELETE";
        case Method::PATCH:
            return "PATCH";
        case Method::OPTIONS:
            return "OPTIONS";
        case Method::TRACE:
            return "TRACE";
        case Method::CONNECT:
            return "CONNECT";
    }
    throw std::invalid_argument("unknown HTTP method");
}

/// @brief parses an HTTP method from a JSON string field.
/// @param parser the JSON parser to read from.
/// @param path the field path.
/// @returns the parsed method.
inline Method parse_method(x::json::Parser &parser, const std::string &path) {
    const auto str = parser.field<std::string>(path);
    if (str == "GET") return Method::GET;
    if (str == "HEAD") return Method::HEAD;
    if (str == "POST") return Method::POST;
    if (str == "PUT") return Method::PUT;
    if (str == "DELETE") return Method::DEL;
    if (str == "PATCH") return Method::PATCH;
    if (str == "OPTIONS") return Method::OPTIONS;
    if (str == "TRACE") return Method::TRACE;
    if (str == "CONNECT") return Method::CONNECT;
    parser.field_err(path, "unknown HTTP method '" + str + "'");
    return Method::GET;
}

/// @brief returns true if the method allows a request body.
constexpr bool has_request_body(const Method m) {
    switch (m) {
        case Method::GET:
        case Method::HEAD:
        case Method::DEL:
        case Method::CONNECT:
        case Method::TRACE:
            return false;
        default:
            return true;
    }
}

/// @brief returns true if the method produces a response body.
constexpr bool has_response_body(const Method m) {
    switch (m) {
        case Method::HEAD:
        case Method::CONNECT:
            return false;
        default:
            return true;
    }
}

}
