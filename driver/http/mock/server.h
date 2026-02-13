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
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/http/types/types.h"
#include "httplib.h"

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
    httplib::Headers headers; ///< Request headers.
    httplib::Params query_params; ///< Decoded query parameters.
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
    std::unique_ptr<httplib::Server> svr_;
    std::thread thread_;
    std::atomic<bool> running_{false};
    std::string host_;
    bool secure_;
    int port_ = 0;
    mutable std::mutex mu_;
    std::vector<ReceivedRequest> requests_;

    static Method parse_httplib_method(const std::string &m) {
        if (m == "POST") return Method::POST;
        if (m == "PUT") return Method::PUT;
        if (m == "DELETE") return Method::DELETE;
        if (m == "PATCH") return Method::PATCH;
        if (m == "GET") return Method::GET;
        throw std::runtime_error("unsupported HTTP method: " + m);
    }

public:
    explicit Server(const ServerConfig &config):
        host_(config.host), secure_(config.secure) {
        if (secure_) {
            svr_ = std::make_unique<httplib::SSLServer>(
                config.cert_path.c_str(),
                config.key_path.c_str()
            );
        } else {
            svr_ = std::make_unique<httplib::Server>();
        }
        for (const auto &route: config.routes)
            register_route(route);
    }

    ~Server() { stop(); }

    Server(const Server &) = delete;
    Server &operator=(const Server &) = delete;

    /// @brief starts the server in a background thread.
    x::errors::Error start() {
        if (running_) return x::errors::NIL;
        if (!svr_->is_valid())
            return x::errors::Error("mock server is not valid (bad TLS cert?)");
        port_ = svr_->bind_to_any_port(host_);
        if (port_ < 0) return x::errors::Error("failed to bind mock HTTP server");
        running_ = true;
        thread_ = std::thread([this] { svr_->listen_after_bind(); });
        svr_->wait_until_ready();
        return x::errors::NIL;
    }

    /// @brief stops the server and joins the background thread.
    void stop() {
        if (!running_) return;
        running_ = false;
        svr_->stop();
        if (thread_.joinable()) thread_.join();
    }

    /// @brief returns the base URL of the running server.
    [[nodiscard]] std::string base_url() const {
        const auto scheme = secure_ ? "https" : "http";
        return std::string(scheme) + "://" + host_ + ":" + std::to_string(port_);
    }

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
            .method = parse_httplib_method(req.method),
            .path = req.path,
            .body = req.body,
            .headers = req.headers,
            .query_params = req.params,
        });
    }

    void register_route(const Route &route) {
        auto handler = [this,
                        route](const httplib::Request &req, httplib::Response &res) {
            log_request(req);
            if (route.delay.count() > 0) std::this_thread::sleep_for(route.delay);
            res.status = route.status_code;
            res.set_content(route.response_body, route.content_type);
        };

        switch (route.method) {
            case Method::GET:
                svr_->Get(route.path, handler);
                break;
            case Method::POST:
                svr_->Post(route.path, handler);
                break;
            case Method::PUT:
                svr_->Put(route.path, handler);
                break;
            case Method::DELETE:
                svr_->Delete(route.path, handler);
                break;
            case Method::PATCH:
                svr_->Patch(route.path, handler);
                break;
        }
    }
};
}
