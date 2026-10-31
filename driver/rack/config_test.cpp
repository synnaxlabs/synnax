// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/env/env.h"
#include "x/cpp/kv/kv.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/rack.h"

namespace driver::rack {
const std::string STATE_PATH = "/tmp/rack-config-test/state.json";

synnax::Config key_conn(const std::string &key) {
    auto conn = test_client_config();
    conn.api_key = key;
    return conn;
}

class RackConfigTest : public ::testing::Test {
protected:
    x::args::Parser args;
    x::breaker::Breaker brk;

    void SetUp() override {
        args = x::args::Parser(
            std::vector<std::string>{"program", "--state-file", STATE_PATH}
        );
        ASSERT_NIL(Config::clear_persisted_state(args));
    }

    /// @brief saves remote info with the key of a rack that needs enrollment.
    static void save_unenrolled_remote_info(
        const synnax::rack::Key key,
        const std::string &cluster
    ) {
        const auto kv = ASSERT_NIL_P(
            x::kv::JSONFile::open_file({
                .path = STATE_PATH,
                .dir_mode = std::filesystem::perms::owner_all,
                .file_mode = std::filesystem::perms::owner_read |
                             std::filesystem::perms::owner_write,
            })
        );
        ASSERT_NIL(kv->set(
            "remote_info",
            x::json::json{{"rack_key", key}, {"cluster_key", cluster}}.dump()
        ));
    }

    /// @brief returns the persisted state with each saved value parsed.
    static x::json::json read_state() {
        std::ifstream file(STATE_PATH);
        auto state = x::json::json::parse(file);
        for (auto &v: state)
            v = x::json::json::parse(v.get<std::string>());
        return state;
    }
};

/// @brief it should enroll a new rack with the default credentials.
TEST_F(RackConfigTest, Default) {
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_FALSE(cfg.connection.api_key.empty());
    ASSERT_TRUE(cfg.connection.username.empty());
    ASSERT_TRUE(cfg.connection.password.empty());
    ASSERT_NE(cfg.rack.key, 0);
    ASSERT_NE(cfg.rack.name, "");
}

/// @brief it should load rack key from persisted state file.
TEST_F(RackConfigTest, LoadRackFromPersistedState) {
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    const auto rack_key = cfg.rack.key;
    const auto cfg2 = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_NE(cfg2.rack.key, 0);
    ASSERT_EQ(cfg2.rack.key, rack_key);
}

/// @brief it should create a new rack after clearing persisted state.
TEST_F(RackConfigTest, ClearRackFromPersistedState) {
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_NE(cfg.rack.key, 0);
    ASSERT_NIL(Config::clear_persisted_state(args));
    const auto cfg2 = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_NE(cfg2.rack.key, cfg.rack.key);
}

/// @brief it should save and load connection parameters from persisted state.
TEST_F(RackConfigTest, SaveConnParamsToPersistedState) {
    ASSERT_NIL(
        Config::save_conn_params(
            args,
            {
                .host = "dog",
                .port = 450,
                .username = "cat",
                .password = "nip",
            }
        )
    );
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_EQ(cfg.connection.host, "dog");
    ASSERT_EQ(cfg.connection.port, 450);
    ASSERT_EQ(cfg.connection.username, "cat");
    ASSERT_EQ(cfg.connection.password, "nip");
}

/// @brief it should enroll the saved rack that needs enrollment.
TEST_F(RackConfigTest, EnrollSavedRack) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("abc rack"));
    const auto cluster_key = ASSERT_NIL_P(
        x::uuid::UUID::parse(client.connectivity->check().cluster_key)
    );
    save_unenrolled_remote_info(rack.key, cluster_key.to_string());
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_EQ(cfg.rack.key, rack.key);
    ASSERT_EQ(cfg.rack.name, "abc rack");
    ASSERT_EQ(cfg.remote_info.cluster_key, cluster_key);
    const auto state = read_state();
    ASSERT_FALSE(state["remote_info"].contains("rack_key"));
    ASSERT_FALSE(state["conn_params"].contains("username"));
    ASSERT_FALSE(state["conn_params"].contains("password"));
    ASSERT_EQ(state["conn_params"]["api_key"], cfg.connection.api_key);
}

/// @brief it should keep the saved rack and the password when the caller cannot create
/// an API key for the rack.
TEST_F(RackConfigTest, EnrollKeepsSavedRackWhenKeyDenied) {
    const auto client = new_test_client();
    const auto caller = ASSERT_NIL_P(client.racks.create("caller rack"));
    const auto caller_key = ASSERT_NIL_P(
        client.api_keys.create("test", {"rack", std::to_string(caller.key)})
    );
    const auto saved = ASSERT_NIL_P(client.racks.create("saved rack"));
    const auto cluster_key = ASSERT_NIL_P(
        x::uuid::UUID::parse(client.connectivity->check().cluster_key)
    );
    auto conn = test_client_config();
    conn.api_key = caller_key.token;
    Config cfg{
        .remote_info = {.rack_key = saved.key, .cluster_key = cluster_key},
        .connection = conn,
    };
    ASSERT_OCCURRED_AS(cfg.enroll(), synnax::auth::ERR_ACCESS_DENIED);
    ASSERT_EQ(cfg.rack.key, saved.key);
    ASSERT_EQ(cfg.connection.password, conn.password);
    ASSERT_NIL_P(client.racks.retrieve(saved.key));
}

/// @brief it should keep a saved rack key until the rack is enrolled.
TEST_F(RackConfigTest, RemoteInfoKeepsUnenrolledRackKey) {
    const RemoteInfo info{.rack_key = 12};
    ASSERT_EQ(info.to_json()["rack_key"], 12);
    ASSERT_FALSE(RemoteInfo{}.to_json().contains("rack_key"));
}

/// @brief it should enroll a new rack when the saved cluster key does not match.
TEST_F(RackConfigTest, RecreateOnClusterKeyMismatch) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("abc rack"));
    save_unenrolled_remote_info(rack.key, "00000000-0000-0000-0000-000000000001");
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_NE(cfg.rack.key, rack.key);
    ASSERT_NE(
        cfg.remote_info.cluster_key,
        ASSERT_NIL_P(x::uuid::UUID::parse("00000000-0000-0000-0000-000000000001"))
    );
}

/// @brief it should save the address used for enrollment with its API key.
TEST_F(RackConfigTest, APIKeyKeepsEnrollmentAddress) {
    ASSERT_NIL(
        Config::save_conn_params(
            args,
            {.host = "dog", .port = 450, .username = "cat", .password = "nip"}
        )
    );
    auto conn = key_conn("syk_saved");
    conn.host = "fish";
    conn.port = 451;
    ASSERT_NIL(Config::save_api_key(args, conn));
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_EQ(cfg.connection.host, "fish");
    ASSERT_EQ(cfg.connection.port, 451);
    ASSERT_EQ(cfg.connection.api_key, "syk_saved");
}

/// @brief it should run as the rack its API key logs in as.
TEST_F(RackConfigTest, LoadRackFromAPIKey) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("key rack"));
    const auto key = ASSERT_NIL_P(
        client.api_keys.create("test", {"rack", std::to_string(rack.key)})
    );
    ASSERT_NIL(Config::save_api_key(args, key_conn(key.token)));
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    ASSERT_EQ(cfg.rack.key, rack.key);
    ASSERT_EQ(cfg.connection.api_key, key.token);
}

/// @brief it should refuse an API key that logs in as a user.
TEST_F(RackConfigTest, RejectUserAPIKey) {
    const auto client = new_test_client();
    const auto key = ASSERT_NIL_P(client.api_keys.create("test"));
    ASSERT_NIL(Config::save_api_key(args, key_conn(key.token)));
    ASSERT_OCCURRED_AS_P(Config::load(args, brk), x::errors::VALIDATION);
}

/// @brief it should default a missing username when only a password is set.
TEST_F(RackConfigTest, PasswordOnly) {
    x::env::set("SYNNAX_DRIVER_PASSWORD", "envpass");
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_DRIVER_PASSWORD"); });
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_EQ(cfg.connection.username, "synnax");
    ASSERT_EQ(cfg.connection.password, "envpass");
}

/// @brief it should default a missing password when only a username is set.
TEST_F(RackConfigTest, UsernameOnly) {
    x::env::set("SYNNAX_DRIVER_USERNAME", "envuser");
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_DRIVER_USERNAME"); });
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_EQ(cfg.connection.username, "envuser");
    ASSERT_EQ(cfg.connection.password, "seldon");
}

/// @brief it should prefer a saved API key over a username and password.
TEST_F(RackConfigTest, APIKeyWinsOverPassword) {
    ASSERT_NIL(Config::save_api_key(args, key_conn("syk_saved")));
    x::args::Parser password_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--username",
            "arguser",
            "--password",
            "argpass"
        }
    );
    const auto cfg = ASSERT_NIL_P(Config::resolve(password_args, brk));
    ASSERT_EQ(cfg.connection.api_key, "syk_saved");
    ASSERT_TRUE(cfg.connection.username.empty());
    ASSERT_TRUE(cfg.connection.password.empty());
}

/// @brief it should refuse a state file that is a symbolic link.
TEST_F(RackConfigTest, RefuseSymlinkStateFile) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows has no O_NOFOLLOW";
#endif
    const std::string target = "/tmp/rack-config-test/target.json";
    std::ofstream(target) << "{}";
    std::filesystem::remove(STATE_PATH);
    std::filesystem::create_symlink(target, STATE_PATH);
    x::defer::defer cleanup([&] {
        std::filesystem::remove(STATE_PATH);
        std::filesystem::remove(target);
    });
    ASSERT_OCCURRED_AS(Config::save_api_key(args, key_conn("syk_saved")), STATE_ERROR);
    ASSERT_OCCURRED_AS(Config::save_conn_params(args, {}), STATE_ERROR);
    ASSERT_OCCURRED_AS(Config::save_remote_info(args, {}), STATE_ERROR);
}

/// @brief it should keep the persisted state file readable only by its owner.
TEST_F(RackConfigTest, StateFileOwnerOnly) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows files have no POSIX mode";
#endif
    ASSERT_NIL(Config::save_api_key(args, key_conn("syk_saved")));
    ASSERT_EQ(
        std::filesystem::status(STATE_PATH).permissions() & std::filesystem::perms::all,
        std::filesystem::perms::owner_read | std::filesystem::perms::owner_write
    );
}

/// @brief it should load default timing configuration.
TEST_F(RackConfigTest, DefaultTimingConfig) {
    const auto cfg = ASSERT_NIL_P(Config::load(args, brk));
    // Assert default timing values
    ASSERT_TRUE(cfg.timing.correct_skew); // Assuming the default is true
}

/// @brief it should load timing configuration from config file.
TEST_F(RackConfigTest, LoadTimingConfigFromFile) {
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
    x::args::Parser config_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path
        }
    );

    // Load config and verify timing settings
    const auto cfg = ASSERT_NIL_P(Config::load(config_args, brk));
    ASSERT_FALSE(cfg.timing.correct_skew); // Verify the loaded value

    // Clean up
    std::remove(config_path.c_str());
}

/// @brief it should load task manager configuration from config file.
TEST_F(RackConfigTest, LoadManagerConfigFromFile) {
    const std::string config_path = "/tmp/rack-config-test/manager-config.json";
    std::ofstream config_file(config_path);
    config_file << R"({"manager": {"worker_count": 8, "op_timeout": 15}})";
    config_file.close();
    x::defer::defer cleanup([&] { std::remove(config_path.c_str()); });
    x::args::Parser config_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path
        }
    );
    const auto cfg = ASSERT_NIL_P(Config::load(config_args, brk));
    ASSERT_EQ(cfg.manager.worker_count, 8);
    ASSERT_EQ(cfg.manager.op_timeout, 15 * x::telem::SECOND);
}

/// @brief it should load connection parameters from command line arguments.
TEST_F(RackConfigTest, LoadFromCommandLineArgs) {
    x::args::Parser args_with_config(
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

    const auto cfg = ASSERT_NIL_P(Config::resolve(args_with_config, brk));
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

/// @brief it should load connection parameters from environment variables.
TEST_F(RackConfigTest, LoadFromEnvironmentVariables) {
    x::env::set("SYNNAX_DRIVER_HOST", "localhost");
    x::env::set("SYNNAX_DRIVER_PORT", "9090");
    x::env::set("SYNNAX_DRIVER_USERNAME", "envuser");
    x::env::set("SYNNAX_DRIVER_PASSWORD", "envpass");
    x::defer::defer unset_env([] {
        x::env::unset("SYNNAX_DRIVER_HOST");
        x::env::unset("SYNNAX_DRIVER_PORT");
        x::env::unset("SYNNAX_DRIVER_USERNAME");
        x::env::unset("SYNNAX_DRIVER_PASSWORD");
    });
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "envuser");
    ASSERT_EQ(cfg.connection.password, "envpass");
}

/// @brief it should load the secure flag from environment variables.
TEST_F(RackConfigTest, LoadSecureFromEnvironmentVariables) {
    x::env::set("SYNNAX_DRIVER_SECURE", "true");
    x::defer::defer unset_env([] { x::env::unset("SYNNAX_DRIVER_SECURE"); });
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_TRUE(cfg.connection.secure);
}

/// @brief it should load the secure flag from the config file.
TEST_F(RackConfigTest, LoadSecureFromConfigFile) {
    const std::string config_path = "/tmp/rack-config-test/secure-config.json";
    std::ofstream config_file(config_path);
    config_file << R"({"connection": {"secure": true}})";
    config_file.close();
    x::defer::defer cleanup([&] { std::remove(config_path.c_str()); });
    x::args::Parser config_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path
        }
    );
    const auto cfg = ASSERT_NIL_P(Config::resolve(config_args, brk));
    ASSERT_TRUE(cfg.connection.secure);
}

/// @brief it should load the secure flag from command line arguments.
TEST_F(RackConfigTest, LoadSecureFromCommandLineArgs) {
    x::args::Parser secure_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--secure",
            "true"
        }
    );
    const auto cfg = ASSERT_NIL_P(Config::resolve(secure_args, brk));
    ASSERT_TRUE(cfg.connection.secure);
}

/// @brief it should load the secure flag saved by the login command.
TEST_F(RackConfigTest, LoadSecureFromPersistedState) {
    ASSERT_NIL(Config::save_conn_params(args, {.secure = true}));
    const auto cfg = ASSERT_NIL_P(Config::resolve(args, brk));
    ASSERT_TRUE(cfg.connection.secure);
}

/// @brief it should let command line arguments turn secure off.
TEST_F(RackConfigTest, ArgsOverrideSecureFromFile) {
    const std::string config_path = "/tmp/rack-config-test/secure-config.json";
    std::ofstream config_file(config_path);
    config_file << R"({"connection": {"secure": true}})";
    config_file.close();
    x::defer::defer cleanup([&] { std::remove(config_path.c_str()); });
    x::args::Parser insecure_args(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json",
            "--config",
            config_path,
            "--secure",
            "false"
        }
    );
    const auto cfg = ASSERT_NIL_P(Config::resolve(insecure_args, brk));
    ASSERT_FALSE(cfg.connection.secure);
}

/// @brief it should respect configuration precedence: args > env > file.
TEST_F(RackConfigTest, ConfigurationPrecedence) {
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
    x::env::set("SYNNAX_DRIVER_PORT", "9090");
    x::env::set("SYNNAX_DRIVER_USERNAME", "envuser");
    x::env::set("SYNNAX_DRIVER_PASSWORD", "envpass");
    x::defer::defer unset_env([&] {
        x::env::unset("SYNNAX_DRIVER_PORT");
        x::env::unset("SYNNAX_DRIVER_USERNAME");
        x::env::unset("SYNNAX_DRIVER_PASSWORD");
        std::remove(config_path.c_str());
    });

    // Set command line args (should override environment)
    x::args::Parser args_with_config(
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

    const auto cfg = ASSERT_NIL_P(Config::resolve(args_with_config, brk));

    // Command line args should take precedence
    ASSERT_EQ(cfg.connection.host, "localhost");
    ASSERT_EQ(cfg.connection.port, 9090);
    ASSERT_EQ(cfg.connection.username, "arguser");
    ASSERT_EQ(cfg.connection.password, "argpass");
}

}
