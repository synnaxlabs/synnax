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
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/http/types/types.h"

namespace driver::http::device {

/// @brief authentication configuration for HTTP connections.
struct AuthConfig {
    std::string type = "none"; ///< "none", "bearer", "basic", or "api_key".
    std::string token; ///< Bearer token (when type == "bearer").
    std::string username; ///< Basic auth username (when type == "basic").
    std::string password; ///< Basic auth password (when type == "basic").
    std::string header; ///< API key header name (when type == "api_key").
    std::string key; ///< API key value (when type == "api_key").

    explicit AuthConfig(x::json::Parser parser):
        type(parser.field<std::string>("type", "none")) {
        if (type == "bearer") {
            token = parser.field<std::string>("token");
        } else if (type == "basic") {
            username = parser.field<std::string>("username");
            password = parser.field<std::string>("password");
        } else if (type == "api_key") {
            header = parser.field<std::string>("header");
            key = parser.field<std::string>("key");
        } else if (type != "none") {
            parser.field_err(
                "type",
                "unknown auth type '" + type +
                    "': must be 'none', 'bearer', 'basic', or 'api_key'"
            );
        }
    }

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j = {{"type", type}};
        if (type == "bearer") {
            j["token"] = token;
        } else if (type == "basic") {
            j["username"] = username;
            j["password"] = password;
        } else if (type == "api_key") {
            j["header"] = header;
            j["key"] = key;
        }
        return j;
    }
};

/// @brief connection configuration for an HTTP device.
struct ConnectionConfig {
    std::string base_url; ///< Base URL (e.g., "http://192.168.1.100:8080").
    uint32_t timeout_ms = 1000; ///< Request timeout in milliseconds.
    AuthConfig auth; ///< Authentication configuration.
    std::map<std::string, std::string> headers; ///< Custom headers.
    bool verify_ssl = true; ///< Whether to verify SSL certificates.

    /// @param parser the JSON parser to read configuration from.
    /// @param verify_ssl whether to verify SSL certificates (false only in tests).
    explicit ConnectionConfig(x::json::Parser parser, const bool verify_ssl = true):
        base_url(parser.field<std::string>("base_url")),
        timeout_ms(parser.field<uint32_t>("timeout_ms", 1000)),
        auth(AuthConfig(parser.optional_child("auth"))),
        headers(parser.field<std::map<std::string, std::string>>(
            "headers",
            std::map<std::string, std::string>{}
        )),
        verify_ssl(verify_ssl) {
        if (timeout_ms == 0)
            parser.field_err("timeout_ms", "must be greater than zero");
    }

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j = {
            {"base_url", base_url},
            {"timeout_ms", timeout_ms},
            {"auth", auth.to_json()},
        };
        if (!headers.empty()) {
            x::json::json h;
            for (const auto &[k, v]: headers)
                h[k] = v;
            j["headers"] = h;
        }
        return j;
    }
};

/// @brief static request configuration, set once at task setup time.
struct RequestConfig {
    Method method = Method::GET; ///< HTTP method.
    std::string path; ///< URL path (appended to base_url).
    std::map<std::string, std::string> query_params; ///< Query parameters.
    std::map<std::string, std::string> headers; ///< Per-request headers.
};

/// @brief an HTTP response.
struct Response {
    int status_code = 0; ///< HTTP status code.
    std::string body; ///< Response body.
    x::telem::TimeRange time_range; ///< Time range spanning the request.
};

/// @brief RAII wrapper around libcurl for making HTTP requests.
/// Curl handles are pre-built at construction time from the connection
/// and request configurations so the hot-path request() only needs to
/// set the body, perform I/O, and read results.
class Client {
    struct Handle;
    ConnectionConfig config_;
    void *multi_handle_;
    std::vector<Handle> handles_;

    Client(const Client &) = delete;
    Client &operator=(const Client &) = delete;

public:
    /// @brief constructs a client and pre-builds curl handles.
    /// @param config the connection configuration.
    /// @param requests the static request configurations.
    Client(ConnectionConfig config, const std::vector<RequestConfig> &requests);

    ~Client();

    /// @brief executes pre-configured requests with the given bodies.
    /// @param bodies one body per pre-configured request. For GET or DELETE
    /// requests, pass an empty string.
    /// @returns the responses and any connection-level error.
    std::pair<std::vector<Response>, x::errors::Error>
    request(const std::vector<std::string> &bodies);
};
}
