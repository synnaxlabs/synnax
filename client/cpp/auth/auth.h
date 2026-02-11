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
#include <mutex>
#include <string>

#include "glog/logging.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/os/os.h"
#include "x/cpp/telem/clock_skew.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/v1/auth.pb.h"

namespace synnax::auth {
/// @brief auth metadata key. NOTE: This must be lowercase, GRPC will panic on
/// capitalized or uppercase keys.
const std::string HEADER_KEY = "authorization";
/// @brief auth token prefix that will be parsed by the cluster.
const std::string HEADER_VALUE_PREFIX = "Bearer ";

/// @brief type alias for the auth login transport.
using LoginClient = freighter::
    UnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>;

const x::errors::Error AUTH_ERROR = x::errors::SY.sub("auth");
const x::errors::Error INVALID_TOKEN = AUTH_ERROR.sub("invalid_token");
const x::errors::Error EXPIRED_TOKEN = AUTH_ERROR.sub("expired_token");
const x::errors::Error INVALID_CREDENTIALS = AUTH_ERROR.sub("invalid-credentials");
const std::vector RETRY_ON_ERRORS = {INVALID_TOKEN, EXPIRED_TOKEN};

/// @brief diagnostic information about the Synnax cluster.
struct ClusterInfo {
    /// @brief a unique UUID key for the cluster.
    std::string cluster_key;
    /// @brief the version string of the Synnax node. Follows the semver format.
    std::string node_version;
    /// @brief the key of the node within the cluster.
    std::uint32_t node_key = 0;
    /// @brief the time of the node at the midpoint of the server processing the
    /// request.
    x::telem::TimeStamp node_time = x::telem::TimeStamp(0);

    ClusterInfo() = default;

    explicit ClusterInfo(const api::v1::ClusterInfo &info):
        cluster_key(info.cluster_key()),
        node_version(info.node_version()),
        node_key(info.node_key()),
        node_time(info.node_time()) {}
};

/// @brief AuthMiddleware for authenticating requests using a bearer token.
/// AuthMiddleware has no preference on order when provided to use. Middleware is safe
/// to use concurrently.
class Middleware final : public freighter::PassthroughMiddleware {
    /// Token to be used for authentication. Empty when auth_attempted is false or error
    /// is not nil.
    std::string token;
    /// Whether the middleware has successfully authenticated with the server.
    std::atomic<bool> authenticated = false;
    /// Transport for authentication requests.
    std::unique_ptr<LoginClient> login_client;
    /// Username to be used for authentication.
    std::string username;
    /// Password to be used for authentication.
    std::string password;
    /// @brief
    std::mutex mu;
    /// @brief the maximum clock skew between the client and server before logging a
    /// warning.
    x::telem::TimeSpan clock_skew_threshold;

public:
    /// Cluster information.
    ClusterInfo cluster_info = ClusterInfo();

    Middleware(
        std::unique_ptr<LoginClient> login_client,
        std::string username,
        std::string password,
        const x::telem::TimeSpan clock_skew_threshold
    ):
        login_client(std::move(login_client)),
        username(std::move(username)),
        password(std::move(password)),
        clock_skew_threshold(clock_skew_threshold) {}

    /// @brief authenticates with the credentials provided when constructing the
    /// Synnax client.
    x::errors::Error authenticate() {
        std::lock_guard lock(mu);
        api::v1::LoginRequest req;
        req.set_username(this->username);
        req.set_password(this->password);
        auto skew_calc = x::telem::ClockSkewCalculator();
        skew_calc.start();
        auto [res, err] = login_client->send("/auth/login", req);
        if (err) return err;
        this->token = res.token();
        this->cluster_info = ClusterInfo(res.cluster_info());
        skew_calc.end(this->cluster_info.node_time);

        if (skew_calc.exceeds(this->clock_skew_threshold)) {
            auto [host, _] = x::os::get_hostname();
            auto direction = "ahead";
            if (skew_calc.skew() > x::telem::TimeSpan::ZERO()) direction = "behind";
            LOG(WARNING) << "measured excessive clock skew between this host and the "
                            "Synnax cluster.";
            LOG(WARNING) << "this host (" << host << ") is " << direction
                         << "by approximately " << skew_calc.skew().abs();
            LOG(
                WARNING
            ) << "this may cause problems with time-series data consistency. We highly "
                 "recommend synchronizing your clock with the Synnax cluster.";
        }

        this->authenticated = true;
        return x::errors::NIL;
    }

    /// @brief implements freighter::Middleware, ensuring that all requests to the
    /// Synnax cluster are appropriately authenticated.
    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        if (!this->authenticated)
            if (const auto err = this->authenticate()) return {context, err};
        context.set(HEADER_KEY, HEADER_VALUE_PREFIX + this->token);
        auto [res_ctx, err] = next(context);
        if (err.matches(RETRY_ON_ERRORS)) {
            this->authenticated = false;
            return this->operator()(context, next);
        }
        return {res_ctx, err};
    }
};
}
