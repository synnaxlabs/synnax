// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/xenv/xenv.h"

#include "driver/rack/rack.h"

class RackConfigTest : public ::testing::Test {
protected:
    xargs::Parser args;
    breaker::Breaker brk;

    void SetUp() override {
        args = xargs::Parser(
            std::vector<std::string>{
                "program",
                "--state-file",
                "/tmp/rack-config-test/state.json"
            }
        );
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
    const auto rack_key = cfg.rack.key;
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
    rack::Config::save_conn_params(
        args,
        {
            .host = "dog",
            .port = 450,
            .username = "cat",
            .password = "nip",
        }
    );
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
    auto [rack, r_err] = client.racks.create("abc rack");
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
    ASSERT_EQ(cfg.remote_info.cluster_key, client.auth->cluster_info.cluster_key);
}

TEST_F(RackConfigTest, recreateOnClusterKeyMismatch) {
    const auto client = new_test_client();
    auto [rack, r_err] = client.racks.create("abc rack");
    ASSERT_FALSE(r_err) << r_err;
    rack::Config::save_remote_info(
        args,
        {
            .rack_key = rack.key,
            .cluster_key = "abc",
        }
    );
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_NE(cfg.rack.key, rack.key);
    ASSERT_NE(cfg.remote_info.cluster_key, "abc");
}

TEST_F(RackConfigTest, testDefaultTimingConfig) {
    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_FALSE(err) << err;
    // Assert default timing values
    ASSERT_TRUE(cfg.timing.correct_skew); // Assuming the default is true
}

TEST_F(RackConfigTest, loadTimingConfigFromFile) {
    // Create a temporary config file with timing settings
    const std::string config_path = "/tmp/rack-config-test/config.json";
    std::ofstream config_file(config_path);
    config_file << R"({
        "timing": {
            "correct_skew": false
        }
    })";
    config_file.close();

    // Set up args with config file
    xargs::Parser config_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path
        }
    );

    // Load config and verify timing settings
    auto [cfg, err] = rack::Config::load(config_args, brk);
    ASSERT_FALSE(err) << err;
    ASSERT_FALSE(cfg.timing.correct_skew); // Verify the loaded value

    // Clean up
    std::remove(config_path.c_str());
}

TEST_F(RackConfigTest, loadFromCommandLineArgs) {
    xargs::Parser args_with_config(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--host",
            "arghost",
            "--port",
            "8080",
            "--username",
            "arguser",
            "--password",
            "argpass"
        }
    );

    auto [cfg, err] = rack::Config::load(args_with_config, brk);
    ASSERT_EQ(cfg.connection.host, "arghost");
    ASSERT_EQ(cfg.connection.port, 8080);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

TEST_F(RackConfigTest, loadFromEnvironmentVariables) {
    // Set environment variables
    xenv::set("SYNNAX_DRIVER_HOST", "envhost");
    xenv::set("SYNNAX_DRIVER_PORT", "7070");
    xenv::set("SYNNAX_DRIVER_USERNAME", "envuser");
    xenv::set("SYNNAX_DRIVER_PASSWORD", "envpass");

    auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_EQ(cfg.connection.host, "envhost");
    ASSERT_EQ(cfg.connection.port, 7070);
    ASSERT_EQ(cfg.connection.username, "envuser");
    ASSERT_EQ(cfg.connection.password, "envpass");

    // Clean up environment variables
    xenv::unset("SYNNAX_DRIVER_HOST");
    xenv::unset("SYNNAX_DRIVER_PORT");
    xenv::unset("SYNNAX_DRIVER_USERNAME");
    xenv::unset("SYNNAX_DRIVER_PASSWORD");
}

TEST_F(RackConfigTest, configurationPrecedence) {
    // Create config file
    const std::string config_path = "/tmp/rack-config-test/config.json";
    std::ofstream config_file(config_path);
    config_file << R"({
        "connection": {
            "host": "filehost",
            "port": 6060,
            "username": "fileuser",
            "password": "filepass"
        }
    })";
    config_file.close();

    // Set environment variables (should override file)
    xenv::set("SYNNAX_DRIVER_PORT", "7070");
    xenv::set("SYNNAX_DRIVER_USERNAME", "envuser");
    xenv::set("SYNNAX_DRIVER_PASSWORD", "envpass");
    x::defer unset_env([&] {
        xenv::unset("SYNNAX_DRIVER_PORT");
        xenv::unset("SYNNAX_DRIVER_USERNAME");
        xenv::unset("SYNNAX_DRIVER_PASSWORD");
        std::remove(config_path.c_str());
    });

    // Set command line args (should override environment)
    xargs::Parser args_with_config(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path,
            "--username",
            "arguser",
            "--password",
            "argpass"
        }
    );

    auto [cfg, err] = rack::Config::load(args_with_config, brk);

    // Command line args should take precedence
    ASSERT_EQ(cfg.connection.host, "filehost");
    ASSERT_EQ(cfg.connection.port, 7070);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

// We need to explicitly define a main function here instead of using gtest_main
// because otherwise the lua interpreters main function will get executed instead.
int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
