// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/test/test.h"

#include "core/pkg/api/grpc/v1/auth.pb.h"

namespace synnax::auth {
const std::string MOCK_CLUSTER_KEY = "748d31e2-5732-4cb5-8bc9-64d4ad51efe8";

api::v1::LoginResponse mock_login_response() {
    api::v1::LoginResponse res;
    res.set_token("abc");
    res.mutable_cluster_info()->set_cluster_key(MOCK_CLUSTER_KEY);
    return res;
}

/// @brief it should correctly authenticate with a Synnax cluster.
TEST(TestAuth, testLoginHappyPath) {
    auto res = mock_login_response();
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        res,
        x::errors::NIL
    );
    const auto mw = std::make_shared<Middleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * x::telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{1, x::errors::NIL};
    mock_client.use(mw);
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should return an error if credentials are invalid.
TEST(TestAuth, testLoginInvalidCredentials) {
    auto res = mock_login_response();
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        res,
        INVALID_CREDENTIALS
    );
    const auto mw = std::make_shared<Middleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * x::telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{1, x::errors::NIL};
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    ASSERT_OCCURRED_AS(err, INVALID_CREDENTIALS);
}

/// @brief it should retry authentication if the authentication token is invalid.
TEST(TestAuth, testLoginRetry) {
    auto res = mock_login_response();
    auto mock_login_client = std::make_unique<
        MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
        std::vector<api::v1::LoginResponse>{res, res},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    const auto mw = std::make_shared<Middleware>(
        std::move(mock_login_client),
        "synnax",
        "seldon",
        5 * x::telem::SECOND
    );
    auto mock_client = MockUnaryClient<int, int>{
        {1, 1},
        {x::errors::Error(INVALID_TOKEN, ""), x::errors::NIL}
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
    std::shared_ptr<Middleware> mw;
    MockUnaryClient<int, int> mock_client;

    void SetUp() override { res = mock_login_response(); }

    void setupTest(x::errors::Error first_error) {
        mock_login_client = std::make_unique<
            MockUnaryClient<api::v1::LoginRequest, api::v1::LoginResponse>>(
            std::vector<api::v1::LoginResponse>{res, res},
            std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
        );
        mw = std::make_shared<Middleware>(
            std::move(mock_login_client),
            "synnax",
            "seldon",
            5 * x::telem::SECOND
        );
        mock_client = MockUnaryClient<int, int>{{1, 1}, {first_error, x::errors::NIL}};
        mock_client.use(mw);
    }
};

/// @brief it should retry authentication if the authentication token is invalid.
TEST_F(TestAuthRetry, RetryOnInvalidToken) {
    setupTest(x::errors::Error(INVALID_TOKEN, ""));
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should retry authentication if the authentication token is expired.
TEST_F(TestAuthRetry, RetryOnExpiredToken) {
    setupTest(x::errors::Error(EXPIRED_TOKEN, ""));
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should correctly parse all fields from a valid ClusterInfo proto.
TEST(TestClusterInfo, testFromProto) {
    api::v1::ClusterInfo pb;
    pb.set_cluster_key(MOCK_CLUSTER_KEY);
    pb.set_node_version("1.2.3");
    pb.set_node_key(42);
    pb.set_node_time(5000000000);
    const auto info = ASSERT_NIL_P(ClusterInfo::from_proto(pb));
    ASSERT_EQ(info.cluster_key.to_string(), MOCK_CLUSTER_KEY);
    ASSERT_EQ(info.node_version, "1.2.3");
    ASSERT_EQ(info.node_key, 42);
    ASSERT_EQ(info.node_time, x::telem::TimeStamp(5000000000));
}

/// @brief it should return an error when the cluster key UUID is invalid.
TEST(TestClusterInfo, testFromProtoInvalidUUID) {
    api::v1::ClusterInfo pb;
    pb.set_cluster_key("not-a-valid-uuid");
    pb.set_node_version("1.0.0");
    pb.set_node_key(1);
    pb.set_node_time(0);
    ASSERT_OCCURRED_AS_P(ClusterInfo::from_proto(pb), x::uuid::INVALID);
}

/// @brief it should return an error when the cluster key is empty.
TEST(TestClusterInfo, testFromProtoEmptyKey) {
    api::v1::ClusterInfo pb;
    pb.set_node_version("1.0.0");
    ASSERT_OCCURRED_AS_P(ClusterInfo::from_proto(pb), x::uuid::INVALID);
}

/// @brief it should correctly handle zero values for node_key and node_time.
TEST(TestClusterInfo, testFromProtoZeroValues) {
    api::v1::ClusterInfo pb;
    pb.set_cluster_key(MOCK_CLUSTER_KEY);
    pb.set_node_key(0);
    pb.set_node_time(0);
    const auto info = ASSERT_NIL_P(ClusterInfo::from_proto(pb));
    ASSERT_EQ(info.node_key, 0);
    ASSERT_EQ(info.node_time, x::telem::TimeStamp(0));
    ASSERT_TRUE(info.node_version.empty());
}

/// @brief it should roundtrip ClusterInfo through proto -> C++ -> proto -> C++.
TEST(TestClusterInfo, testFromProtoRoundtrip) {
    api::v1::ClusterInfo pb;
    pb.set_cluster_key(MOCK_CLUSTER_KEY);
    pb.set_node_version("2.5.1");
    pb.set_node_key(99);
    pb.set_node_time(123456789);
    const auto first = ASSERT_NIL_P(ClusterInfo::from_proto(pb));
    api::v1::ClusterInfo pb2;
    pb2.set_cluster_key(first.cluster_key.to_string());
    pb2.set_node_version(first.node_version);
    pb2.set_node_key(first.node_key);
    pb2.set_node_time(first.node_time.nanoseconds());
    const auto second = ASSERT_NIL_P(ClusterInfo::from_proto(pb2));
    ASSERT_EQ(first.cluster_key, second.cluster_key);
    ASSERT_EQ(first.node_version, second.node_version);
    ASSERT_EQ(first.node_key, second.node_key);
    ASSERT_EQ(first.node_time, second.node_time);
}
}
