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
#include <utility>

#include "client/cpp/device/device.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/http/types/types.h"

namespace driver::http::device {

/// @brief authentication configuration for HTTP connections.
struct AuthConfig {
    /// @brief authentication type: "none", "bearer", "basic", or "api_key".
    std::string type = "none";
    /// @brief bearer token (when type == "bearer").
    std::string token;
    /// @brief basic auth username (when type == "basic").
    std::string username;
    /// @brief basic auth password (when type == "basic").
    std::string password;
    /// @brief header name (when type == "api_key" and send_as == "header").
    std::string header;
    /// @brief query parameter name (when type == "api_key" and send_as ==
    /// "query_param").
    std::string parameter;
    /// @brief API key value (when type == "api_key").
    std::string key;
    /// @brief how to send the API key: "header" or "query_param"
    /// (when type == "api_key").
    std::string send_as = "header";

    [[nodiscard]] explicit AuthConfig(x::json::Parser parser):
        type(parser.field<std::string>("type", "none")) {
        if (type == "bearer") {
            token = parser.field<std::string>("token");
        } else if (type == "basic") {
            username = parser.field<std::string>("username");
            password = parser.field<std::string>("password");
        } else if (type == "api_key") {
            key = parser.field<std::string>("key");
            send_as = parser.field<std::string>("send_as", "header");
            if (send_as == "header")
                header = parser.field<std::string>("header");
            else if (send_as == "query_param")
                parameter = parser.field<std::string>("parameter");
            else
                parser.field_err(
                    "send_as",
                    "must be 'header' or 'query_param', got '" + send_as + "'"
                );
        } else if (type != "none") {
            parser.field_err(
                "type",
                "unknown auth type '" + type +
                    "': must be 'none', 'bearer', 'basic', or 'api_key'"
            );
        }
    }
};

/// @brief connection configuration for an HTTP device.
struct ConnectionConfig {
    /// @brief base URL (e.g., "http://192.168.1.100:8080").
    std::string base_url;
    /// @brief request timeout.
    x::telem::TimeSpan timeout;
    /// @brief authentication configuration.
    AuthConfig auth;
    /// @brief whether to verify SSL certificates.
    bool verify_ssl;

    /// @param parser the JSON parser to read configuration from.
    [[nodiscard]] explicit ConnectionConfig(x::json::Parser parser):
        base_url(parser.field<std::string>("base_url")),
        timeout(parser.field<uint32_t>("timeout_ms", 100) * x::telem::MILLISECOND),
        auth(AuthConfig(parser.optional_child("auth"))),
        verify_ssl(parser.field<bool>("verify_ssl", true)) {
        if (!base_url.starts_with("http://") && !base_url.starts_with("https://"))
            parser.field_err(
                "base_url",
                "must start with 'http://' or 'https://' followed by a host, got '" +
                    base_url + "'"
            );
        else if (base_url == "http://" || base_url == "https://")
            parser.field_err(
                "base_url",
                "must include a host after the scheme, got '" + base_url + "'"
            );
        if (timeout <= x::telem::TimeSpan::ZERO())
            parser.field_err("timeout_ms", "must be positive");
    }
};

/// @brief retrieves a device by key and constructs a ConnectionConfig from its
/// properties and location.
/// @param devices the Synnax device client.
/// @param device_key the key of the device to retrieve.
/// @returns the connection config paired with an error (nil on success).
[[nodiscard]] std::pair<ConnectionConfig, x::errors::Error> retrieve_connection(
    const synnax::device::Client &devices,
    const std::string &device_key
);

/// @brief merges a ConnectionConfig and RequestConfig into a fully resolved Request.
/// @param conn the device-level connection configuration.
/// @param req the per-endpoint request configuration.
/// @returns the resolved request.
[[nodiscard]] Request
build_request(const ConnectionConfig &conn, const RequestConfig &req);

}
