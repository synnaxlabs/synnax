// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/auth/auth.h"

#include <memory>
#include <gtest/gtest.h>

#include "freighter/cpp/mock/mock.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/auth.pb.h"

/// @brief it should correctly authenticate with a Synnax cluster.
TEST(TestAuth, testLoginHappyPath) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<MockUnaryClient<
        api::v1::LoginRequest,
        api::v1::LoginResponse
    > >(res, freighter::NIL);
    auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        3
    );
    auto mock_client = MockUnaryClient<int, int>{1, freighter::NIL};
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    EXPECT_TRUE(err.matches(freighter::NIL));
}

/// @brief it should return an error if credentials are invalid.
TEST(TestAuth, testLoginInvalidCredentials) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<MockUnaryClient<
        api::v1::LoginRequest,
        api::v1::LoginResponse
    > >(res, freighter::Error(synnax::INVALID_CREDENTIALS, ""));
    auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        3
    );
    auto mock_client = MockUnaryClient<int, int>{1, freighter::NIL};
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    EXPECT_TRUE(err) << err.message();
    EXPECT_TRUE(err.matches(synnax::INVALID_CREDENTIALS));
}
