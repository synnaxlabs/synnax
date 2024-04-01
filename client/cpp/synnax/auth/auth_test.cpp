#include <gtest/gtest.h>
#include <memory>
#include "client/cpp/synnax/auth/auth.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/auth.pb.h"
#include "freighter/cpp/freighter/mock/mock.h"

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

/// @brief it should retry authentication if the authentication token is invalid.
TEST(TestAuth, testLoginRetry) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<MockUnaryClient<
        api::v1::LoginRequest,
        api::v1::LoginResponse
    >>(
        std::vector<api::v1::LoginResponse>{res, res},
        std::vector<freighter::Error>{freighter::NIL, freighter::NIL}
    );
    auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        3
    );
    auto mock_client = MockUnaryClient<int, int>{
        {1, 1},
        {freighter::Error(synnax::INVALID_TOKEN, ""), freighter::NIL}
    };
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    EXPECT_FALSE(err) << err.message();
    EXPECT_TRUE(err.matches(freighter::NIL));
}

/// @brief it should return an invalid token error if the maximum number of retries is exceeded.
TEST(TestAuth, testExceedMaxRetries) {
    auto res = api::v1::LoginResponse();
    res.set_token("abc");
    auto mock_login_client = std::make_unique<MockUnaryClient<
        api::v1::LoginRequest,
        api::v1::LoginResponse
    >>(
        std::vector<api::v1::LoginResponse>{res, res, res},
        std::vector<freighter::Error>{freighter::NIL, freighter::NIL, freighter::NIL}
    );
    auto mw = std::make_shared<AuthMiddleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        2
    );
    auto invalid_token = freighter::Error(synnax::INVALID_TOKEN, "");
    auto mock_client = MockUnaryClient<int, int>{
        {1, 1, 1},
        {invalid_token, invalid_token, invalid_token}
    };
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    EXPECT_TRUE(err) << err.message();
    EXPECT_TRUE(err.matches(synnax::INVALID_TOKEN));
}