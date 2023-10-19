// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>

/// protos
#include "v1/auth.pb.h"
#include "freighter/freighter.h"
#include <grpcpp/grpcpp.h>

/// Auth meta data key. NOTE: This must be lowercase, GRPC will panic on capitalized
/// or uppercase keys.
const std::string HEADER_KEY = "authorization";
/// Auth value prefix.
const std::string HEADER_VALUE_PREFIX = "Bearer ";

/// @brief type alias for the auth login transport.
typedef freighter::UnaryClient<
        api::v1::LoginResponse,
        api::v1::LoginRequest
> AuthLoginClient;


/// @brief AuthMiddleware for authenticating requests using a bearer token. AuthMiddleware has
/// no preference on order when provided to use.
class AuthMiddleware : public freighter::PassthroughMiddleware {
private:
    /// Token to be used for authentication. Empty when auth_attempted is false or error
    /// is not nil.
    std::string token;
    /// Whether or not an authentication attempt was made with the server. If set to true
    /// and err is not nil, authentication has failed and the middleware will not attempt
    /// to authenticate again.
    bool auth_attempted = false;
    /// Accumulated error from authentication attempts.
    freighter::Error err = freighter::NIL;
    /// Transport for authentication requests.
    AuthLoginClient *login_client;
    /// Username to be used for authentication.
    std::string username;
    /// Password to be used for authentication.
    std::string password;

public:
    AuthMiddleware(
            AuthLoginClient *login_client,
            const std::string &username,
            const std::string &password
    ) :
            login_client(login_client), username(username), password(password) {
    }

    /// Implements freighter::AuthMiddleware::operator().
    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context context) override {
        if (!auth_attempted) {
            api::v1::LoginRequest req;
            req.set_username(username);
            req.set_password(password);
            auto [res, exc] = login_client->send("/auth_login/login", req);
            if (exc) {
                err = exc;
                return {context, err};
            }
            token = res.token();
            auth_attempted = true;
        }
        if (err) return {context, err};
        context.set(HEADER_KEY, HEADER_VALUE_PREFIX + token);
        return freighter::PassthroughMiddleware::operator()(context);
    }
};

