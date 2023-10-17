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


namespace Auth {
typedef Freighter::UnaryClient<
        api::v1::LoginResponse,
        api::v1::LoginRequest> LoginClient;


class Middleware : public Freighter::PassthroughMiddleware {
private:
    std::string token = "";
    bool authenticated = false;
    Freighter::Error err = Freighter::NIL;
    Auth::LoginClient *login_client;
    std::string username;
    std::string password;

public:
    Middleware(Auth::LoginClient *login_client, const std::string &username, const std::string &password) :
            login_client(login_client), username(username), password(password) {
    }

    std::pair<Freighter::Context, Freighter::Error> operator()(Freighter::Context context) override {
        if (!authenticated) {
            api::v1::LoginRequest req;
            req.set_username(username);
            req.set_password(password);
            auto [res, exc] = login_client->send("/auth_login/login", req);
            if (exc) {
                err = exc;
                return {context, err};
            }
            token = res.token();
            authenticated = true;
        }
        if (err) return {context, err};
        context.set("authorization", "Bearer " + token);
        return Freighter::PassthroughMiddleware::operator()(context);
    }
};
}

