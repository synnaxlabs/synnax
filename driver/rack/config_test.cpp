// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/rack/rack.h"

/// module
#include "client/cpp/testutil/testutil.h"

class RackConfigTest: public ::testing::Test {
protected:
    xargs::Parser args;
    breaker::Breaker brk;

    void SetUp() override {
        args = xargs::Parser(std::vector<std::string>{"program", "--state-file", "/tmp/rack-config-test/state.json"});
        std::cout << args.required<std::string>("--state-file") << std::endl;
        const auto c_err = rack::Config::clear_persisted_state(args);
        ASSERT_FALSE(c_err) << c_err;
    }

};

TEST_F(RackConfigTest, testDefault) {
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.username, "synnax");
    ASSERT_EQ(cfg.connection.password, "seldon");
    ASSERT_NE(cfg.rack.key, 0);
    ASSERT_NE(cfg.rack.name, "");
}

TEST_F(RackConfigTest, loadRackFromPersistedState) {
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    auto rack_key = cfg.rack.key;
    auto [cfg2, err2] = rack::Config::load(args, brk);
    ASSERT_FALSE(err2) << err2;
    ASSERT_NE(cfg2.rack.key, 0);
    ASSERT_EQ(cfg2.rack.key, rack_key);
}

TEST_F(RackConfigTest, clearRackFromPersistedState) {
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_NE(cfg.rack.key, 0);
    auto c_err2 = rack::Config::clear_persisted_state(args);
    ASSERT_FALSE(c_err2) << c_err2;
    auto [cfg2, err2] = rack::Config::load(args, brk);
    ASSERT_FALSE(err2) << err2;
    ASSERT_NE(cfg2.rack.key, cfg.rack.key);
}

TEST_F(RackConfigTest, saveConnParamsToPersistedState) {
    rack::Config::save_conn_params(args, {
        .host = "dog",
        .port = 450,
        .username = "cat",
        .password = "nip",
    });
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_TRUE(err) << err;
    ASSERT_TRUE(err.matches(freighter::UNREACHABLE)) << err;
    ASSERT_EQ(cfg.connection.host, "dog");
    ASSERT_EQ(cfg.connection.port, 450);
    ASSERT_EQ(cfg.connection.username, "cat");
    ASSERT_EQ(cfg.connection.password, "nip");
}

TEST_F(RackConfigTest, parseRackFromConfigArg) {
    const auto client = new_test_client();
    auto [rack, r_err] = client.hardware.create_rack("abc rack");
    ASSERT_FALSE(r_err) << r_err;
    rack::RemoteInfo remote_info{
        .rack_key = rack.key,
        .cluster_key = client.auth->cluster_info.cluster_key,
    };
    rack::Config::save_remote_info(args, remote_info);
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_EQ(cfg.rack.key, rack.key);
    ASSERT_EQ(cfg.rack.name, "abc rack");
    ASSERT_EQ(cfg.remote.cluster_key, client.auth->cluster_info.cluster_key);
}

TEST_F(RackConfigTest, recreateOnClusterKeyMismatch) {
    const auto client = new_test_client();
    auto [rack, r_err] = client.hardware.create_rack("abc rack");
    ASSERT_FALSE(r_err) << r_err;
    rack::Config::save_remote_info(args, {
        .rack_key = rack.key,
        .cluster_key = "abc",
    });
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_NE(cfg.rack.key, rack.key);
    ASSERT_NE(cfg.remote.cluster_key, "abc");
}

// We need to explicitly define a main function here instead of using gtest_main
// because otherwise the lua interpreters main function will get executed instead.
int main(int argc, char** argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}

