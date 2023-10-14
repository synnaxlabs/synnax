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
#include "synnax/channel/channel.h"
#include <grpcpp/grpcpp.h>


namespace Auth {
    typedef Freighter::UnaryClient<
            api::v1::LoginResponse,
            api::v1::LoginRequest,
            grpc::Status> AuthClient;


    class Client {
    private:
        std::string token;
    public:
        std::string username;
        std::string password;

        Client(AuthClient *auth_client, const std::string &username, const std::string &password);

        Freighter::Middleware* tokenMiddleware();
    };
}