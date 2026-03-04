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
#include <string>
#include <utility>
#include <vector>

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
    /// @brief custom headers applied to every request (legacy compat).
    std::map<std::string, std::string> headers;
    /// @brief query parameters applied to every request (legacy compat).
    std::map<std::string, std::string> query_params;
    /// @brief whether to verify SSL certificates.
    bool verify_ssl;

    /// @param parser the JSON parser to read configuration from.
    [[nodiscard]] explicit ConnectionConfig(x::json::Parser parser):
        base_url(parser.field<std::string>("base_url")),
        timeout(parser.field<uint32_t>("timeout_ms", 100) * x::telem::MILLISECOND),
        auth(AuthConfig(parser.optional_child("auth"))),
        headers(parser.field<std::map<std::string, std::string>>(
            "headers",
            std::map<std::string, std::string>{}
        )),
        query_params(parser.field<std::map<std::string, std::string>>(
            "query_params",
            std::map<std::string, std::string>{}
        )),
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
std::pair<ConnectionConfig, x::errors::Error> retrieve_connection(
    const synnax::device::Client &devices,
    const std::string &device_key
);

/// @brief classifies an HTTP status code into an error.
/// @param status_code the HTTP response status code.
/// @returns nil for 2xx, CLIENT_ERROR for 4xx, SERVER_ERROR for 5xx.
x::errors::Error classify_status(int status_code);

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
    /// @brief expected response Content-Type; also sent as Accept.
    std::string response_content_type;
    /// @brief request body Content-Type; omitted when empty.
    std::string request_content_type;
};

/// @brief an HTTP response.
struct Response {
    /// @brief HTTP status code.
    int status_code = 0;
    /// @brief response body.
    std::string body;
    /// @brief time range spanning the request.
    x::telem::TimeRange time_range;
};

/// @brief a handle to a curl request. Should not be constructed or used directly.
struct Handle;
/// @brief RAII wrapper around a curl multi handle.
struct MultiHandle;

/// @brief RAII wrapper around libcurl for making HTTP requests. Curl handles are
/// pre-built at construction time from the connection and request configurations so the
/// hot-path request() only needs to set the body, perform I/O, and read results.
class Client {
    std::unique_ptr<MultiHandle> multi_handle;
    std::vector<Handle> handles;

    Client(const Client &) = delete;
    Client &operator=(const Client &) = delete;

    Client(ConnectionConfig config, const std::vector<RequestConfig> &requests);

    Client();

public:
    Client(Client &&other) noexcept;

    /// @brief constructs and validates a client with pre-built curl handles.
    /// @param config the connection configuration.
    /// @param requests the static request configurations.
    /// @returns the client paired with a validation error (nil on success).
    static std::pair<Client, x::errors::Error>
    create(ConnectionConfig config, const std::vector<RequestConfig> &requests);

    ~Client();

    /// @brief executes pre-configured requests with the given bodies.
    /// @param bodies one body per pre-configured request. For TRACE requests or
    /// requests without a body, pass an empty string.
    /// @param poll_timeout maximum time to wait for socket activity between polls.
    /// @returns the per-request responses paired with a transfer-level error
    /// (non-nil when the entire batch fails, e.g. curl_multi_perform error).
    std::pair<std::vector<std::pair<Response, x::errors::Error>>, x::errors::Error>
    execute_requests(
        const std::vector<std::string> &bodies,
        x::telem::TimeSpan poll_timeout = 1 * x::telem::SECOND
    );
};
}
