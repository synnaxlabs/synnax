// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <map>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/http/types/types.h"

namespace driver::http::mock {
/// @brief a single route to register on the mock server.
struct Route {
    Method method = Method::GET; ///< HTTP method.
    std::string path; ///< URL path pattern (e.g., "/api/data").
    int status_code = 200; ///< HTTP status code to respond with.
    std::string response_body; ///< Response body content.
    std::string content_type = "application/json"; ///< Content-Type header.
    std::chrono::milliseconds delay{0}; ///< Delay before responding.
};

/// @brief a received request logged by the mock server.
struct ReceivedRequest {
    Method method; ///< HTTP method.
    std::string path; ///< Request path.
    std::string body; ///< Request body.
    std::multimap<std::string, std::string> headers; ///< Request headers.
    std::multimap<std::string, std::string> query_params; ///< Decoded query params.
};

/// @brief configuration for the mock HTTP server.
struct ServerConfig {
    std::string host = "127.0.0.1"; ///< Bind address.
    std::vector<Route> routes; ///< Routes to register.
    bool secure = false; ///< Use HTTPS with self-signed certificate.
    std::string cert_path; ///< Path to TLS certificate (when secure).
    std::string key_path; ///< Path to TLS private key (when secure).
};

/// @brief a mock HTTP server for testing, backed by cpp-httplib.
class Server {
public:
    explicit Server(const ServerConfig &config);
    ~Server();

    Server(const Server &) = delete;
    Server &operator=(const Server &) = delete;

    /// @brief starts the server in a background thread.
    x::errors::Error start();

    /// @brief stops the server and joins the background thread.
    void stop();

    /// @brief returns the base URL of the running server.
    [[nodiscard]] std::string base_url() const;

    /// @brief returns all requests received by the server.
    [[nodiscard]] std::vector<ReceivedRequest> received_requests() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};
}
