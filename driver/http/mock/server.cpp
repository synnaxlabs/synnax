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
    int port = 0;
    mutable std::mutex mu;
    std::vector<ReceivedRequest> requests;

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
        std::lock_guard lock(mu);
        std::multimap<std::string, std::string> headers(
            req.headers.begin(),
            req.headers.end()
        );
        std::multimap<std::string, std::string> params(
            req.params.begin(),
            req.params.end()
        );
        requests.push_back({
            .method = parse_httplib_method(req.method),
            .path = req.path,
            .body = req.body,
            .headers = std::move(headers),
            .query_params = std::move(params),
        });
    }

    void register_route(const Route &route) {
        auto handler = [this,
                        route](const httplib::Request &req, httplib::Response &res) {
            log_request(req);
            if (route.delay > x::telem::TimeSpan::ZERO())
                std::this_thread::sleep_for(route.delay.chrono());
            res.status = route.status_code;
            res.set_content(route.response_body, route.content_type);
        };

        switch (route.method) {
            case Method::GET:
                svr->Get(route.path, handler);
                break;
            case Method::POST:
                svr->Post(route.path, handler);
                break;
            case Method::PUT:
                svr->Put(route.path, handler);
                break;
            case Method::DEL:
                svr->Delete(route.path, handler);
                break;
            case Method::PATCH:
                svr->Patch(route.path, handler);
                break;
            case Method::OPTIONS:
                svr->Options(route.path, handler);
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

Server::Server(const ServerConfig &config): impl_(std::make_unique<Impl>()) {
    impl_->host = config.host;
    impl_->secure = config.secure;
    if (config.secure) {
        impl_->svr = std::make_unique<httplib::SSLServer>(
            config.cert_path.c_str(),
            config.key_path.c_str()
        );
    } else {
        impl_->svr = std::make_unique<httplib::Server>();
    }
    for (const auto &route: config.routes)
        impl_->register_route(route);
}

Server::~Server() {
    stop();
}

x::errors::Error Server::start() {
    if (impl_->running) return x::errors::NIL;
    if (!impl_->svr->is_valid())
        return x::errors::Error("mock server is not valid (bad TLS cert?)");
    impl_->port = impl_->svr->bind_to_any_port(impl_->host);
    if (impl_->port < 0) return x::errors::Error("failed to bind mock HTTP server");
    impl_->running = true;
    impl_->thread = std::thread([this] { impl_->svr->listen_after_bind(); });
    impl_->svr->wait_until_ready();
    return x::errors::NIL;
}

void Server::stop() {
    if (!impl_->running) return;
    impl_->running = false;
    impl_->svr->stop();
    if (impl_->thread.joinable()) impl_->thread.join();
}

std::string Server::base_url() const {
    const std::string scheme = impl_->secure ? "https" : "http";
    return scheme + "://" + impl_->host + ":" + std::to_string(impl_->port);
}

std::vector<ReceivedRequest> Server::received_requests() const {
    std::lock_guard lock(impl_->mu);
    return impl_->requests;
}
}
