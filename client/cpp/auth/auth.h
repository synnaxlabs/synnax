// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/auth.pb.h"

/// Auth metadata key. NOTE: This must be lowercase, GRPC will panic on capitalized or
/// uppercase keys.
const std::string HEADER_KEY = "authorization";
/// Auth value prefix.
const std::string HEADER_VALUE_PREFIX = "Bearer ";

/// @brief type alias for the auth login transport.
typedef freighter::UnaryClient<
    api::v1::LoginRequest,
    api::v1::LoginResponse
> AuthLoginClient;


/// @brief AuthMiddleware for authenticating requests using a bearer token. AuthMiddleware has
/// no preference on order when provided to use.
class AuthMiddleware final : public freighter::PassthroughMiddleware {
    /// Token to be used for authentication. Empty when auth_attempted is false or error
    /// is not nil.
    std::string token;
    /// Whether the middleware has successfully authenticated with the server.
    bool authenticated = false;
    /// Transport for authentication requests.
    std::unique_ptr<AuthLoginClient> login_client;
    /// Username to be used for authentication.
    std::string username;
    /// Password to be used for authentication.
    std::string password;
    /// The maximum number of times to retry authentication.
    std::uint32_t max_retries;
    /// Number of times authentication has been retried.
    std::uint32_t retry_count = 0;

public:
    AuthMiddleware(
        std::unique_ptr<AuthLoginClient> login_client,
        std::string username,
        std::string password,
        const std::uint32_t max_retries
    ) : login_client(std::move(login_client)),
        username(std::move(username)),
        password(std::move(password)),
        max_retries(max_retries) {
    }

    /// @brief authenticates with the credentials provided when construction the 
    /// Synnax client.
    freighter::Error authenticate() {
        api::v1::LoginRequest req;
        req.set_username(this->username);
        req.set_password(this->password);
        auto [res, err] = login_client->send("/auth/login", req);
        if (err) return err;
        this->token = res.token();
        this->authenticated = true;
        this->retry_count = 0;
        return freighter::NIL;
    }

    /// @brief implements freighter::Middleware, ensuring that all requests to the
    /// Synnax cluster are appropriately authenticated.
    std::pair<freighter::Context, freighter::Error> operator()(
        freighter::Context context,
        freighter::Next *next
    ) override {
        if (!this->authenticated)
            if (auto err = this->authenticate(); err)
                return {context, err};
        context.set(HEADER_KEY, HEADER_VALUE_PREFIX + token);
        auto [res_ctx, err] = next->operator()(context);
        if (err.matches(synnax::INVALID_TOKEN) && retry_count < max_retries) {
            this->authenticated = false;
            this->retry_count++;
            return this->operator()(context, next);
        }
        return {res_ctx, err};
    }
};
