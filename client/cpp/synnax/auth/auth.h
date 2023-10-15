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
        api::v1::LoginRequest,
        grpc::Status> LoginClient;


class Client {
private:
    std::string token;
    LoginClient *login_client;
public:
    Client(LoginClient *login_client);

    void login(const std::string &username, const std::string &password);

    Freighter::Middleware *tokenMiddleware();
};
}