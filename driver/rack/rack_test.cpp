// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "client/cpp/testutil/pipe.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/kv/kv.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/rack.h"

namespace driver::rack {
const std::string RACK_STATE_PATH = "/tmp/rack-test/state.json";

/// @brief returns arguments that point the rack at a fresh state file.
x::args::Parser fresh_args(std::vector<std::string> extra) {
    std::filesystem::remove(RACK_STATE_PATH);
    std::vector<std::string> argv{"program", "--state-file", RACK_STATE_PATH};
    argv.insert(argv.end(), extra.begin(), extra.end());
    return x::args::Parser(argv);
}

/// @brief it should stop without an error while it waits to reach the Core again.
TEST(RackTest, StopDuringReconnectReportsNoError) {
    auto args = fresh_args({"--port", "1"});
    std::atomic shut_down = false;
    Rack r;
    r.start(args, [&] { shut_down = true; });
    // The rack creates its state file just before it first tries to reach the Core.
    ASSERT_EVENTUALLY_TRUE(std::filesystem::exists(RACK_STATE_PATH));
    ASSERT_NIL(r.stop());
    ASSERT_FALSE(shut_down);
}

/// @brief it should call on_shutdown and return the error that stopped it.
TEST(RackTest, FatalErrorStopsRack) {
    const auto conn = test_client_config();
    auto args = fresh_args({
        "--host",
        conn.host,
        "--port",
        std::to_string(conn.port),
        "--username",
        "rack-test-nobody",
        "--password",
        "wrong",
    });
    std::atomic shut_down = false;
    Rack r;
    r.start(args, [&] { shut_down = true; });
    ASSERT_EVENTUALLY_TRUE(shut_down.load());
    ASSERT_OCCURRED_AS(r.stop(), synnax::auth::ERR_INVALID_CREDENTIALS);
}

/// @brief a rack that reaches the Core through a pipe.
struct PipedRack {
    synnax::Synnax client = new_test_client();
    synnax::rack::Rack remote;
    CorePipe pipe;
    std::atomic<bool> shut_down = false;
    Rack rack;

    explicit PipedRack(const x::breaker::Config &breaker_config):
        pipe(test_client_config().port), rack(breaker_config) {
        const auto conn = test_client_config();
        this->remote = ASSERT_NIL_P(this->client.racks.create("outage rack"));
        const auto cluster_key = this->client.connectivity->check().cluster_key;
        auto args = fresh_args({
            "--host",
            "127.0.0.1",
            "--port",
            std::to_string(this->pipe.port),
            "--username",
            conn.username,
            "--password",
            conn.password,
        });
        const auto kv = ASSERT_NIL_P(
            x::kv::JSONFile::open_file({
                .path = RACK_STATE_PATH,
                .dir_mode = std::filesystem::perms::owner_all,
                .file_mode = std::filesystem::perms::owner_read |
                             std::filesystem::perms::owner_write,
            })
        );
        EXPECT_NIL(kv->set(
            "remote_info",
            x::json::json{{"rack_key", this->remote.key}, {"cluster_key", cluster_key}}
                .dump()
        ));
        this->rack.start(args, [this] { this->shut_down = true; });
    }

    /// @returns whether the Core holds a running status for the rack.
    bool running() const {
        const synnax::rack::RetrieveOptions opts{.include_status = true};
        const auto [r, err] = this->client.racks.retrieve(this->remote.key, opts);
        return !err && r.status.has_value() && r.status->details.running;
    }
};

const x::breaker::Config FAST_BREAKER{
    .name = "driver",
    .base_interval = x::telem::MILLISECOND,
    .max_retries = 2,
    .scale = 1,
    .max_interval = x::telem::MILLISECOND,
};

/// @brief it should stop after the startup retries of its config file when it cannot
/// reach the Core before its first run. The default of 200 retries would take 20 s.
TEST(RackTest, StopsAfterStartupRetries) {
    const std::string config_path = "/tmp/rack-test/config.json";
    std::filesystem::create_directories("/tmp/rack-test");
    std::ofstream(config_path) << R"({"retry": {"startup_retries": 2}})";
    auto args = fresh_args({"--port", "1", "--config", config_path});
    std::atomic shut_down = false;
    Rack r({
        .name = "driver",
        .base_interval = x::telem::MILLISECOND * 100,
        .scale = 1,
        .max_interval = x::telem::MILLISECOND * 100,
    });
    r.start(args, [&] { shut_down = true; });
    ASSERT_EVENTUALLY_TRUE(shut_down.load());
    ASSERT_OCCURRED_AS(r.stop(), freighter::UNREACHABLE);
}

/// @brief it should keep waiting for an unreachable Core past its retry limit after its
/// first run.
TEST(RackTest, WaitsForAnUnreachableCorePastTheRetryLimit) {
    PipedRack r(FAST_BREAKER);
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        r.running(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    r.pipe.pause();
    std::this_thread::sleep_for(std::chrono::seconds(1));
    ASSERT_FALSE(r.shut_down);
    r.pipe.resume();
    ASSERT_NIL(r.rack.stop());
}

/// @brief it should keep running when the Core resets its streams.
TEST(RackTest, KeepsRunningThroughAStreamReset) {
    PipedRack r(FAST_BREAKER);
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        r.running(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    r.pipe.reset();
    std::this_thread::sleep_for(std::chrono::seconds(1));
    ASSERT_FALSE(r.shut_down);
    ASSERT_NIL(r.rack.stop());
}
}
