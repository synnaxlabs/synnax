// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>

#include <gtest/gtest.h>

#include "client/cpp/auth/auth.h"
#include "freighter/cpp/mock/mock.h"
#include "x/cpp/xtest/xtest.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/auth.pb.h"

/// @brief it should correctly authenticate with a Synnax cluster.
TEST(TestAuth, testLoginHappyPath) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        res,
        xerrors::NIL
    );
    const auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{1, xerrors::NIL};
    mock_client.use(mw);
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should return an error if credentials are invalid.
TEST(TestAuth, testLoginInvalidCredentials) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        res,
        INVALID_CREDENTIALS
    );
    const auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{1, xerrors::NIL};
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    ASSERT_OCCURRED_AS(err, INVALID_CREDENTIALS);
}

/// @brief it should retry authentication if the authentication token is invalid.
TEST(TestAuth, testLoginRetry) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        std::vector<api::v1::LoginResponse>{res, res},
        std::vector<xerrors::Error>{xerrors::NIL, xerrors::NIL}
    );
    const auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{
        {1, 1},
        {xerrors::Error(INVALID_TOKEN, ""), xerrors::NIL}
    };
    mock_client.use(mw);
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

class TestAuthRetry : public ::testing::Test {
protected:
    api::v1::LoginResponse res;
    std::unique_ptr<MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>
        mock_login_client;
    std::shared_ptr<AuthMiddleware> mw;
    MockUnaryClient<int, int> mock_client;

    void SetUp() override { res.set_token("abc"); }

    void setup_test(const xerrors::Error &first_error) {
        mock_login_client = std::make_unique<
            MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
            std::vector<api::v1::LoginResponse>{res, res},
            std::vector<xerrors::Error>{xerrors::NIL, xerrors::NIL}
        );
        mw = std::make_shared<AuthMiddleware>(
            std::move(mock_login_client),
            "synnax",
            "seldon",
            5 * telem::SECOND
        );
        mock_client = MockUnaryClient<int, int>{{1, 1}, {first_error, xerrors::NIL}};
        mock_client.use(mw);
    }
};

/// @brief it should retry authentication if the authentication token is invalid.
TEST_F(TestAuthRetry, RetryOnInvalidToken) {
    setup_test(xerrors::Error(INVALID_TOKEN, ""));
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should retry authentication if the authentication token is expired.
TEST_F(TestAuthRetry, RetryOnExpiredToken) {
    setup_test(xerrors::Error(EXPIRED_TOKEN, ""));
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}
