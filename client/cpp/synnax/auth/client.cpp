// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/auth/auth.h"
#include "v1/auth.pb.h"

using namespace Auth;

Client::Client(Auth::AuthClient *auth_client, const std::string &username, const std::string &password) {
    this->username = username;
    this->password = password;
    api::v1::LoginRequest req;
    req.set_username(username);
    req.set_password(password);
    auto [res, exc] = auth_client->send("/auth_login/login", req);
    if (!exc.ok()) throw exc;
    token = res.token();
}


class TokenMiddleware : public Freighter::PassthroughMiddleware {
private:
    std::string token;
public:
    explicit TokenMiddleware(const std::string &token) {
        this->token = token;
    }

    std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context context) override {
        context.set("Authorization", "Bearer " + token);
        return Freighter::PassthroughMiddleware::operator()(context);
    }
};

Freighter::Middleware *Client::middleware() { return new TokenMiddleware(token); }

