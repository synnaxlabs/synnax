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
#include "x/cpp/xtest/xtest.h"

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
        std::cout << args.field<std::string>("--state-file") << std::endl;
        ASSERT_NIL(rack::Config::clear_persisted_state(args));
    }
};

/// @brief it should load default configuration values.
TEST_F(RackConfigTest, testDefault) {
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.username, "synnax");
    ASSERT_EQ(cfg.connection.password, "seldon");
    ASSERT_NE(cfg.rack.key, 0);
    ASSERT_NE(cfg.rack.name, "");
}

/// @brief it should load rack key from persisted state file.
TEST_F(RackConfigTest, loadRackFromPersistedState) {
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    const auto rack_key = cfg.rack.key;
    const auto cfg2 = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_NE(cfg2.rack.key, 0);
    ASSERT_EQ(cfg2.rack.key, rack_key);
}

/// @brief it should create a new rack after clearing persisted state.
TEST_F(RackConfigTest, clearRackFromPersistedState) {
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_NE(cfg.rack.key, 0);
    ASSERT_NIL(rack::Config::clear_persisted_state(args));
    const auto cfg2 = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_NE(cfg2.rack.key, cfg.rack.key);
}

/// @brief it should save and load connection parameters from persisted state.
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
    ASSERT_OCCURRED_AS(err, freighter::UNREACHABLE);
    ASSERT_EQ(cfg.connection.host, "dog");
    ASSERT_EQ(cfg.connection.port, 450);
    ASSERT_EQ(cfg.connection.username, "cat");
    ASSERT_EQ(cfg.connection.password, "nip");
}

/// @brief it should load rack configuration from remote info in persisted state.
TEST_F(RackConfigTest, parseRackFromConfigArg) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("abc rack"));
    rack::RemoteInfo remote_info{
        .rack_key = rack.key,
        .cluster_key = client.auth->cluster_info.cluster_key,
    };
    rack::Config::save_remote_info(args, remote_info);
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_EQ(cfg.rack.key, rack.key);
    ASSERT_EQ(cfg.rack.name, "abc rack");
    ASSERT_EQ(cfg.remote_info.cluster_key, client.auth->cluster_info.cluster_key);
}

/// @brief it should recreate rack when cluster key does not match.
TEST_F(RackConfigTest, recreateOnClusterKeyMismatch) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("abc rack"));
    rack::Config::save_remote_info(
        args,
        {
            .rack_key = rack.key,
            .cluster_key = "abc",
        }
    );
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    ASSERT_NE(cfg.rack.key, rack.key);
    ASSERT_NE(cfg.remote_info.cluster_key, "abc");
}

/// @brief it should load default timing configuration.
TEST_F(RackConfigTest, testDefaultTimingConfig) {
    const auto cfg = ASSERT_NIL_P(rack::Config::load(args, brk));
    // Assert default timing values
    ASSERT_TRUE(cfg.timing.correct_skew); // Assuming the default is true
}

/// @brief it should load timing configuration from config file.
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
    const auto cfg = ASSERT_NIL_P(rack::Config::load(config_args, brk));
    ASSERT_FALSE(cfg.timing.correct_skew); // Verify the loaded value

    // Clean up
    std::remove(config_path.c_str());
}

/// @brief it should load connection parameters from command line arguments.
TEST_F(RackConfigTest, loadFromCommandLineArgs) {
    xargs::Parser args_with_config(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--host",
            "localhost",
            "--port",
            "9090",
            "--username",
            "arguser",
            "--password",
            "argpass"
        }
    );

    const auto [cfg, err] = rack::Config::load(args_with_config, brk);
    ASSERT_OCCURRED_AS(err, AUTH_ERROR);
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

/// @brief it should load connection parameters from environment variables.
TEST_F(RackConfigTest, loadFromEnvironmentVariables) {
    // Set environment variables
    xenv::set("SYNNAX_DRIVER_HOST", "localhost");
    xenv::set("SYNNAX_DRIVER_PORT", "9090");
    xenv::set("SYNNAX_DRIVER_USERNAME", "envuser");
    xenv::set("SYNNAX_DRIVER_PASSWORD", "envpass");

    const auto [cfg, err] = rack::Config::load(args, brk);
    ASSERT_OCCURRED_AS(err, AUTH_ERROR);
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "envuser");
    ASSERT_EQ(cfg.connection.password, "envpass");

    // Clean up environment variables
    xenv::unset("SYNNAX_DRIVER_HOST");
    xenv::unset("SYNNAX_DRIVER_PORT");
    xenv::unset("SYNNAX_DRIVER_USERNAME");
    xenv::unset("SYNNAX_DRIVER_PASSWORD");
}

/// @brief it should respect configuration precedence: args > env > file.
TEST_F(RackConfigTest, configurationPrecedence) {
    // Create config file
    const std::string config_path = "/tmp/rack-config-test/config.json";
    std::ofstream config_file(config_path);
    config_file << R"({
        "connection": {
            "host": "localhost",
            "port": 6060,
            "username": "fileuser",
            "password": "filepass"
        }
    })";
    config_file.close();

    // Set environment variables (should override file)
    xenv::set("SYNNAX_DRIVER_PORT", "9090");
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

    const auto [cfg, err] = rack::Config::load(args_with_config, brk);
    ASSERT_OCCURRED_AS(err, AUTH_ERROR);

    // Command line args should take precedence
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

// We need to explicitly define a main function here instead of using gtest_main
// because otherwise the lua interpreters main function will get executed instead.
int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
