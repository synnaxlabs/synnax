// Copyright 2024 Synnax Labs, Inc.
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
private:
    /// Token to be used for authentication. Empty when auth_attempted is false or error
    /// is not nil.
    std::string token;
    /// Whether or not an authentication attempt was made with the server. If set to true
    /// and err is not nil, authentication has failed and the middleware will not attempt
    /// to authenticate again.
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
        std::uint32_t max_retries
    ) : login_client(std::move(login_client)),
        username(std::move(username)),
        password(std::move(password)),
        max_retries(max_retries) {
    }

    /// Implements freighter::AuthMiddleware::operator().
    std::pair<freighter::Context, freighter::Error> operator()(
        freighter::Context context,
        freighter::Next *next
    ) override {
        if (!authenticated) {
            api::v1::LoginRequest req;
            req.set_username(username);
            req.set_password(password);
            auto [res, err] = login_client->send("/auth/login", req);
            if (err) {
                return {context, err};
            }
            token = res.token();
            authenticated = true;
            retry_count = 0;
        }
        context.set(HEADER_KEY, HEADER_VALUE_PREFIX + token);
        auto [res_ctx, err] = next->operator()(context);
        if (err.matches(synnax::INVALID_TOKEN) && retry_count < max_retries) {
            authenticated = false;
            retry_count++;
            return this->operator()(context, next);
        }
        return {res_ctx, err};
    }
};
