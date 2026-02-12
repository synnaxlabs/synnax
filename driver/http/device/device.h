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
#include <memory>
#include <mutex>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/errors/errors.h"

namespace driver::http::device {
/// @brief HTTP client error (4xx responses or configuration issues).
const x::errors::Error CLIENT_ERR =
    errors::CRITICAL_HARDWARE_ERROR.sub("http.client");
/// @brief HTTP server error (5xx responses).
const x::errors::Error SERVER_ERR =
    errors::TEMPORARY_HARDWARE_ERROR.sub("http.server");
/// @brief HTTP request timeout.
const x::errors::Error TIMEOUT_ERR =
    errors::TEMPORARY_HARDWARE_ERROR.sub("http.timeout");
/// @brief HTTP server unreachable.
const x::errors::Error UNREACHABLE_ERR =
    errors::TEMPORARY_HARDWARE_ERROR.sub("http.unreachable");
/// @brief HTTP response parse error.
const x::errors::Error PARSE_ERR =
    errors::CRITICAL_HARDWARE_ERROR.sub("http.parse");

/// @brief authentication configuration for HTTP connections.
struct AuthConfig {
    std::string type = "none"; ///< "none", "bearer", "basic", or "api_key".
    std::string token; ///< Bearer token (when type == "bearer").
    std::string username; ///< Basic auth username (when type == "basic").
    std::string password; ///< Basic auth password (when type == "basic").
    std::string header; ///< API key header name (when type == "api_key").
    std::string key; ///< API key value (when type == "api_key").

    AuthConfig() = default;

    explicit AuthConfig(x::json::Parser parser):
        type(parser.field<std::string>("type", "none")),
        token(parser.field<std::string>("token", "")),
        username(parser.field<std::string>("username", "")),
        password(parser.field<std::string>("password", "")),
        header(parser.field<std::string>("header", "")),
        key(parser.field<std::string>("key", "")) {}

    [[nodiscard]] x::json::json to_json() const {
        return {
            {"type", type},
            {"token", token},
            {"username", username},
            {"password", password},
            {"header", header},
            {"key", key},
        };
    }
};

/// @brief connection configuration for an HTTP device.
struct ConnectionConfig {
    std::string base_url; ///< Base URL (e.g., "http://192.168.1.100:8080").
    uint32_t timeout_ms = 30000; ///< Request timeout in milliseconds.
    AuthConfig auth; ///< Authentication configuration.
    std::map<std::string, std::string> headers; ///< Custom headers.

    ConnectionConfig() = default;

    explicit ConnectionConfig(x::json::Parser parser):
        base_url(parser.field<std::string>("base_url")),
        timeout_ms(parser.field<uint32_t>("timeout_ms", 30000)),
        auth(AuthConfig(parser.optional_child("auth"))),
        headers(parser.field<std::map<std::string, std::string>>(
            "headers", std::map<std::string, std::string>{}
        )) {}

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j = {
            {"base_url", base_url},
            {"timeout_ms", timeout_ms},
            {"auth", auth.to_json()},
        };
        if (!headers.empty()) {
            x::json::json h;
            for (const auto &[k, v]: headers) h[k] = v;
            j["headers"] = h;
        }
        return j;
    }
};

/// @brief an outgoing HTTP request.
struct Request {
    std::string method = "GET"; ///< HTTP method.
    std::string path; ///< URL path (appended to base_url).
    std::string body; ///< Request body.
    std::map<std::string, std::string> query_params; ///< Query parameters.
    std::map<std::string, std::string> headers; ///< Per-request headers.
    std::string content_type = "application/json"; ///< Content-Type header.
};

/// @brief an HTTP response.
struct Response {
    int status_code = 0; ///< HTTP status code.
    std::string body; ///< Response body.
    x::telem::TimeStamp request_start; ///< Timestamp before the request.
    x::telem::TimeStamp request_end; ///< Timestamp after the response.
};

/// @brief RAII wrapper around libcurl for making HTTP requests.
class Client {
    ConnectionConfig config_;
    void *multi_handle_;
    mutable std::mutex mu_;

    Client(const Client &) = delete;
    Client &operator=(const Client &) = delete;

public:
    /// @brief constructs a client with the given connection configuration.
    explicit Client(ConnectionConfig config);

    ~Client();

    /// @brief executes a single HTTP request.
    /// @param req the request to execute.
    /// @returns the response and any connection-level error.
    std::pair<Response, x::errors::Error> request(const Request &req);

    /// @brief executes multiple HTTP requests in parallel.
    /// @param requests the requests to execute.
    /// @returns the responses and any error.
    std::pair<std::vector<Response>, x::errors::Error>
    request_parallel(const std::vector<Request> &requests);
};

/// @brief manages HTTP client connections, pooling by base URL.
class Manager {
    mutable std::mutex mu_;
    std::map<std::string, std::weak_ptr<Client>> clients_;

public:
    Manager() = default;

    /// @brief acquires a client for the given connection configuration. Reuses
    /// existing clients when possible (keyed by base_url).
    /// @param config the connection configuration.
    /// @returns a shared pointer to the client and any error.
    std::pair<std::shared_ptr<Client>, x::errors::Error>
    acquire(const ConnectionConfig &config);
};
}
