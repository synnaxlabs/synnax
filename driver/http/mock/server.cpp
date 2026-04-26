// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <mutex>
#include <stdexcept>
#include <thread>

#include "driver/http/mock/server.h"
// Disable GCC 13 false positive warning in <regex> header (included by httplib.h)
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include "httplib.h"
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif

namespace driver::http::mock {
struct Server::Impl {
    std::unique_ptr<httplib::Server> svr;
    std::thread thread;
    std::atomic<bool> running{false};
    std::string host;
    bool secure;
    std::string cert_path;
    std::string key_path;
    int port = 0;
    mutable std::mutex mu;
    std::vector<ReceivedRequest> requests;
    std::vector<Route> routes;
    std::vector<std::shared_ptr<std::atomic<int>>> failure_counters;

    static Method parse_httplib_method(const std::string &m) {
        if (m == "GET") return Method::GET;
        if (m == "HEAD") return Method::HEAD;
        if (m == "POST") return Method::POST;
        if (m == "PUT") return Method::PUT;
        if (m == "DELETE") return Method::DEL;
        if (m == "PATCH") return Method::PATCH;
        if (m == "OPTIONS") return Method::OPTIONS;
        if (m == "TRACE") return Method::TRACE;
        if (m == "CONNECT") return Method::CONNECT;
        throw std::runtime_error("unsupported HTTP method: " + m);
    }

    void log_request(const httplib::Request &req) {
        std::lock_guard lock(this->mu);
        std::multimap<std::string, std::string> headers(
            req.headers.begin(),
            req.headers.end()
        );
        std::multimap<std::string, std::string> params(
            req.params.begin(),
            req.params.end()
        );
        this->requests.push_back({
            .method = parse_httplib_method(req.method),
            .path = req.path,
            .body = req.body,
            .headers = std::move(headers),
            .query_params = std::move(params),
        });
    }

    void create_server() {
        if (this->secure) {
            this->svr = std::make_unique<httplib::SSLServer>(
                this->cert_path.c_str(),
                this->key_path.c_str()
            );
        } else {
            this->svr = std::make_unique<httplib::Server>();
        }
        this->failure_counters.clear();
        for (const auto &route: this->routes)
            this->failure_counters.push_back(
                std::make_shared<std::atomic<int>>(route.remaining_failures)
            );
        for (size_t i = 0; i < this->routes.size(); i++)
            this->register_route(this->routes[i], this->failure_counters[i]);
    }

    void register_route(
        const Route &route,
        const std::shared_ptr<std::atomic<int>> &counter
    ) {
        auto handler = [this,
                        route,
                        counter](const httplib::Request &req, httplib::Response &res) {
            log_request(req);
            if (route.delay > x::telem::TimeSpan::ZERO())
                std::this_thread::sleep_for(route.delay.chrono());
            if (!route.redirect_to.empty()) {
                res.status = route.status_code;
                res.set_redirect(route.redirect_to, route.status_code);
                return;
            }
            if (route.remaining_failures > 0 && counter->fetch_sub(1) > 0) {
                res.status = route.status_code;
                res.set_content(route.response_body, route.content_type);
            } else if (route.remaining_failures > 0) {
                res.status = 200;
                res.set_content(R"({"status":"ok"})", route.content_type);
            } else {
                res.status = route.status_code;
                res.set_content(route.response_body, route.content_type);
            }
        };

        switch (route.method) {
            case Method::GET:
                this->svr->Get(route.path, handler);
                break;
            case Method::POST:
                this->svr->Post(route.path, handler);
                break;
            case Method::PUT:
                this->svr->Put(route.path, handler);
                break;
            case Method::DEL:
                this->svr->Delete(route.path, handler);
                break;
            case Method::PATCH:
                this->svr->Patch(route.path, handler);
                break;
            case Method::OPTIONS:
                this->svr->Options(route.path, handler);
                break;
            case Method::HEAD:
                throw std::runtime_error("httplib does not support HEAD methods");
            case Method::TRACE:
                throw std::runtime_error("httplib does not support TRACE methods");
            case Method::CONNECT:
                throw std::runtime_error("httplib does not support CONNECT methods");
        }
    }
};

Server::Server(const ServerConfig &config): impl(std::make_unique<Impl>()) {
    this->impl->host = config.host;
    this->impl->secure = config.secure;
    this->impl->cert_path = config.cert_path;
    this->impl->key_path = config.key_path;
    this->impl->routes = config.routes;
    this->impl->create_server();
}

Server::~Server() {
    this->stop();
}

x::errors::Error Server::start() {
    if (this->impl->running) return x::errors::NIL;
    // Recreate the httplib server so that start() works after a stop().
    this->impl->create_server();
    if (!this->impl->svr->is_valid())
        return x::errors::Error("mock server is not valid (bad TLS cert?)");
    // If we have a previous port (restart), reuse it so existing clients keep working.
    if (this->impl->port > 0) {
        if (!this->impl->svr->bind_to_port(this->impl->host, this->impl->port))
            return x::errors::Error(
                "failed to re-bind mock HTTP server to port " +
                std::to_string(this->impl->port)
            );
    } else {
        this->impl->port = this->impl->svr->bind_to_any_port(this->impl->host);
        if (this->impl->port < 0)
            return x::errors::Error("failed to bind mock HTTP server");
    }
    this->impl->running = true;
    this->impl->thread = std::thread([this] { this->impl->svr->listen_after_bind(); });
    this->impl->svr->wait_until_ready();
    return x::errors::NIL;
}

void Server::stop() {
    if (!this->impl->running) return;
    this->impl->running = false;
    this->impl->svr->stop();
    if (this->impl->thread.joinable()) this->impl->thread.join();
}

std::string Server::base_url() const {
    const std::string scheme = this->impl->secure ? "https" : "http";
    return scheme + "://" + this->impl->host + ":" + std::to_string(this->impl->port);
}

std::vector<ReceivedRequest> Server::received_requests() const {
    std::lock_guard lock(this->impl->mu);
    return this->impl->requests;
}

void Server::clear_requests() {
    std::lock_guard lock(this->impl->mu);
    this->impl->requests.clear();
}
}
