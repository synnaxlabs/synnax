// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <set>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

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

/// @brief parses an HTTP method from its string form.
/// @param str the method string (e.g. "GET").
/// @returns the parsed method paired with an error when the string is unknown.
inline std::pair<Method, x::errors::Error> parse_method(const std::string &str) {
    if (str == "GET") return {Method::GET, x::errors::NIL};
    if (str == "HEAD") return {Method::HEAD, x::errors::NIL};
    if (str == "POST") return {Method::POST, x::errors::NIL};
    if (str == "PUT") return {Method::PUT, x::errors::NIL};
    if (str == "DELETE") return {Method::DEL, x::errors::NIL};
    if (str == "PATCH") return {Method::PATCH, x::errors::NIL};
    if (str == "OPTIONS") return {Method::OPTIONS, x::errors::NIL};
    if (str == "TRACE") return {Method::TRACE, x::errors::NIL};
    if (str == "CONNECT") return {Method::CONNECT, x::errors::NIL};
    return {
        Method::GET,
        x::errors::Error(x::errors::VALIDATION, "unknown HTTP method '" + str + "'"),
    };
}

/// @brief parses an HTTP method from a JSON string field.
/// @param parser the JSON parser to read from.
/// @param path the field path.
/// @returns the parsed method.
inline Method parse_method(x::json::Parser &parser, const std::string &path) {
    const auto str = parser.field<std::string>(path);
    auto [method, err] = parse_method(str);
    if (err) parser.field_err(path, err);
    return method;
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

/// @brief static request configuration, set once at task setup time.
struct RequestConfig {
    /// @brief HTTP method.
    Method method;
    /// @brief URL path (appended to base_url).
    std::string path;
    /// @brief query parameters.
    std::map<std::string, std::string> query_params;
    /// @brief per-request headers.
    std::map<std::string, std::string> headers;
    /// @brief request body Content-Type; omitted when empty.
    std::string request_content_type;
};

/// @brief a fully resolved HTTP request.
struct Request {
    /// @brief fully resolved URL (base + path + query params).
    std::string url;
    /// @brief HTTP method.
    Method method;
    /// @brief request timeout.
    x::telem::TimeSpan timeout;
    /// @brief whether to verify SSL certificates.
    bool verify_ssl;
    /// @brief pre-formatted headers ("Key: Value"), merged from connection-level,
    /// per-request, and auth sources at build time.
    std::vector<std::string> headers;
    /// @brief request body.
    std::string body;
};

/// @brief an HTTP response.
struct Response {
    /// @brief HTTP status code.
    int status_code;
    /// @brief response body.
    std::string body;
    /// @brief time range spanning the request.
    x::telem::TimeRange time_range;
};

/// @brief validates that str is a well-formed JSON Pointer, accumulating an error
/// on parser at path when it is not.
inline void validate_pointer(
    const std::string &str,
    x::json::Parser &parser,
    const std::string &path
) {
    try {
        static_cast<void>(x::json::json::json_pointer(str));
    } catch (const x::json::json::parse_error &e) { parser.field_err(path, e.what()); }
}

/// @brief validates an endpoint's method, headers, and query parameters,
/// accumulating errors on parser. Header and query parameter names must be
/// non-empty and unique within the endpoint.
template<typename Endpoint>
void validate_request(const Endpoint &ep, x::json::Parser &parser) {
    if (auto [method, err] = parse_method(ep.method); err)
        parser.field_err("endpoints.method", err);
    std::set<std::string> names;
    for (const auto &h: ep.headers)
        if (h.name.empty())
            parser.field_err("endpoints.headers.name", "this field is required");
        else if (!names.insert(h.name).second)
            parser.field_err(
                "endpoints.headers.name",
                "duplicate header '" + h.name + "'"
            );
    if (ep.query_params.has_value()) {
        std::set<std::string> params;
        for (const auto &qp: *ep.query_params)
            if (qp.parameter.empty())
                parser.field_err(
                    "endpoints.query_params.parameter",
                    "this field is required"
                );
            else if (!params.insert(qp.parameter).second)
                parser.field_err(
                    "endpoints.query_params.parameter",
                    "duplicate query parameter '" + qp.parameter + "'"
                );
    }
}

/// @brief derives the static request configuration from a read or write endpoint's
/// method, path, headers, and query parameters. Assumes the endpoint has already
/// been validated.
template<typename Endpoint>
[[nodiscard]] RequestConfig request_config(const Endpoint &ep) {
    RequestConfig req;
    req.method = parse_method(ep.method).first;
    req.path = ep.path;
    for (const auto &h: ep.headers)
        if (!h.name.empty()) req.headers.emplace(h.name, h.value);
    if (ep.query_params.has_value())
        for (const auto &qp: *ep.query_params)
            if (!qp.parameter.empty()) req.query_params.emplace(qp.parameter, qp.value);
    return req;
}

}
