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
    /// @brief HTTP method.
    Method method = Method::GET;
    /// @brief URL path pattern (e.g., "/api/data").
    std::string path;
    /// @brief HTTP status code to respond with.
    int status_code = 200;
    /// @brief response body content.
    std::string response_body;
    /// @brief Content-Type header.
    std::string content_type = "application/json";
    /// @brief delay before responding.
    std::chrono::milliseconds delay{0};
};

/// @brief a received request logged by the mock server.
struct ReceivedRequest {
    /// @brief HTTP method.
    Method method;
    /// @brief request path.
    std::string path;
    /// @brief request body.
    std::string body;
    /// @brief request headers.
    std::multimap<std::string, std::string> headers;
    /// @brief decoded query params.
    std::multimap<std::string, std::string> query_params;
};

/// @brief configuration for the mock HTTP server.
struct ServerConfig {
    /// @brief bind address.
    std::string host = "127.0.0.1";
    /// @brief routes to register.
    std::vector<Route> routes;
    /// @brief use HTTPS with self-signed certificate.
    bool secure = false;
    /// @brief path to TLS certificate (when secure).
    std::string cert_path;
    /// @brief path to TLS private key (when secure).
    std::string key_path;
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
