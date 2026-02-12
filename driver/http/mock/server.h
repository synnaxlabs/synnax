// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "httplib.h"

#include "x/cpp/errors/errors.h"

namespace driver::http::mock {
/// @brief a single route to register on the mock server.
struct Route {
    std::string method; ///< HTTP method (GET, POST, PUT, DELETE, PATCH).
    std::string path; ///< URL path pattern (e.g., "/api/data").
    int status_code = 200; ///< HTTP status code to respond with.
    std::string response_body; ///< Response body content.
    std::string content_type = "application/json"; ///< Content-Type header.
    std::chrono::milliseconds delay{0}; ///< Delay before responding.
};

/// @brief a received request logged by the mock server.
struct ReceivedRequest {
    std::string method; ///< HTTP method.
    std::string path; ///< Request path.
    std::string body; ///< Request body.
    httplib::Headers headers; ///< Request headers.
};

/// @brief configuration for the mock HTTP server.
struct ServerConfig {
    std::string host = "127.0.0.1"; ///< Bind address.
    int port = 0; ///< Port (0 = auto-select).
    std::vector<Route> routes; ///< Routes to register.
};

/// @brief a mock HTTP server for testing, backed by cpp-httplib.
class Server {
    httplib::Server svr_;
    std::thread thread_;
    std::atomic<bool> running_{false};
    std::string host_;
    int port_;
    mutable std::mutex mu_;
    std::vector<ReceivedRequest> requests_;

public:
    explicit Server(const ServerConfig &config): host_(config.host), port_(config.port) {
        for (const auto &route: config.routes) register_route(route);
    }

    ~Server() { stop(); }

    Server(const Server &) = delete;
    Server &operator=(const Server &) = delete;

    /// @brief starts the server in a background thread.
    x::errors::Error start() {
        if (running_) return x::errors::NIL;
        if (port_ == 0) {
            port_ = svr_.bind_to_any_port(host_);
            if (port_ < 0)
                return x::errors::Error("failed to bind mock HTTP server");
        } else {
            if (!svr_.bind_to_port(host_, port_))
                return x::errors::Error(
                    "failed to bind mock HTTP server to port " +
                    std::to_string(port_)
                );
        }
        running_ = true;
        thread_ = std::thread([this] { svr_.listen_after_bind(); });
        return x::errors::NIL;
    }

    /// @brief stops the server and joins the background thread.
    void stop() {
        if (!running_) return;
        running_ = false;
        svr_.stop();
        if (thread_.joinable()) thread_.join();
    }

    /// @brief returns the base URL of the running server (e.g.,
    /// "http://127.0.0.1:8080").
    [[nodiscard]] std::string base_url() const {
        return "http://" + host_ + ":" + std::to_string(port_);
    }

    /// @brief returns the port the server is listening on.
    [[nodiscard]] int port() const { return port_; }

    /// @brief returns all requests received by the server.
    [[nodiscard]] std::vector<ReceivedRequest> received_requests() const {
        std::lock_guard lock(mu_);
        return requests_;
    }

    /// @brief clears the list of received requests.
    void clear_requests() {
        std::lock_guard lock(mu_);
        requests_.clear();
    }

private:
    void log_request(const httplib::Request &req) {
        std::lock_guard lock(mu_);
        requests_.push_back({
            .method = req.method,
            .path = req.path,
            .body = req.body,
            .headers = req.headers,
        });
    }

    void register_route(const Route &route) {
        auto handler = [this, route](
                           const httplib::Request &req, httplib::Response &res
                       ) {
            log_request(req);
            if (route.delay.count() > 0)
                std::this_thread::sleep_for(route.delay);
            res.status = route.status_code;
            res.set_content(route.response_body, route.content_type);
        };

        if (route.method == "GET")
            svr_.Get(route.path, handler);
        else if (route.method == "POST")
            svr_.Post(route.path, handler);
        else if (route.method == "PUT")
            svr_.Put(route.path, handler);
        else if (route.method == "DELETE")
            svr_.Delete(route.path, handler);
        else if (route.method == "PATCH")
            svr_.Patch(route.path, handler);
    }
};
}
