// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/connection/checker.h"
#include "freighter/cpp/mock/mock.h"

#include "core/pkg/api/grpc/connectivity/connectivity.pb.h"

namespace synnax::connection {
grpc::connectivity::CheckResponse make_response(
    const std::string &cluster_key = "test-cluster",
    const std::string &node_version = "0.54.0",
    const int64_t node_time = 0
) {
    grpc::connectivity::CheckResponse res;
    res.set_cluster_key(cluster_key);
    res.set_node_version(node_version);
    res.set_node_time(
        node_time != 0 ? node_time : x::telem::TimeStamp::now().nanoseconds()
    );
    return res;
}

TEST(TestChecker, ConnectedOnValidResponse) {
    auto res = make_response();
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res, res},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto checker = Checker(std::move(mock), 30 * x::telem::SECOND, "0.54.0");
    auto s = checker.state();
    EXPECT_EQ(s.status, Status::CONNECTED);
    EXPECT_EQ(s.cluster_key, "test-cluster");
    checker.stop();
}

TEST(TestChecker, FailedOnError) {
    auto res = make_response();
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res},
        std::vector<x::errors::Error>{freighter::UNREACHABLE}
    );
    auto checker = Checker(std::move(mock), 30 * x::telem::SECOND, "0.54.0");
    auto s = checker.state();
    EXPECT_EQ(s.status, Status::FAILED);
    checker.stop();
}

TEST(TestChecker, ClockSkewExceeded) {
    auto far_future = x::telem::TimeStamp::now() + x::telem::HOUR;
    auto res = make_response("test-cluster", "0.54.0", far_future.nanoseconds());
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res, res},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto checker = Checker(
        std::move(mock),
        30 * x::telem::SECOND,
        "0.54.0",
        "",
        x::telem::SECOND
    );
    auto s = checker.state();
    EXPECT_TRUE(s.clock_skew_exceeded);
    checker.stop();
}

TEST(TestChecker, StopHaltsThread) {
    auto res = make_response();
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res, res, res, res, res},
        std::vector<x::errors::Error>{
            x::errors::NIL,
            x::errors::NIL,
            x::errors::NIL,
            x::errors::NIL,
            x::errors::NIL
        }
    );
    auto checker = Checker(std::move(mock), 30 * x::telem::SECOND, "0.54.0");
    checker.stop();
}

TEST(TestChecker, VersionIncompatible) {
    auto res = make_response("test-cluster", "99.0.0");
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res, res},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto checker = Checker(std::move(mock), 30 * x::telem::SECOND, "0.54.0");
    auto s = checker.state();
    EXPECT_FALSE(s.client_server_compatible);
    checker.stop();
}

TEST(TestChecker, VersionCompatible) {
    auto res = make_response("test-cluster", "0.54.1");
    auto mock = std::make_unique<freighter::mock::UnaryClient<
        google::protobuf::Empty,
        grpc::connectivity::CheckResponse>>(
        std::vector<grpc::connectivity::CheckResponse>{res, res},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto checker = Checker(std::move(mock), 30 * x::telem::SECOND, "0.54.0");
    auto s = checker.state();
    EXPECT_TRUE(s.client_server_compatible);
    checker.stop();
}
}
