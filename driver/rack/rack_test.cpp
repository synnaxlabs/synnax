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
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "client/cpp/testutil/pipe.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/kv/kv.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/http/scan_task.h"
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

/// @brief returns arguments that run the Driver of rack through pipe.
x::args::Parser piped_args(
    const synnax::Synnax &client,
    const synnax::rack::Rack &rack,
    const CorePipe &pipe
) {
    const auto conn = test_client_config();
    auto args = fresh_args({
        "--host",
        "127.0.0.1",
        "--port",
        std::to_string(pipe.port),
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
    const auto cluster_key = client.connectivity->check().cluster_key;
    EXPECT_NIL(kv->set(
        "remote_info",
        x::json::json{{"rack_key", rack.key}, {"cluster_key", cluster_key}}.dump()
    ));
    return args;
}

/// @brief a rack config that reaches the Core again within milliseconds.
const x::breaker::Config FAST_RETRY{
    .name = "driver",
    .base_interval = x::telem::MILLISECOND,
    .max_retries = 2,
    .scale = 1,
    .max_interval = x::telem::MILLISECOND,
};

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

/// @brief it should keep waiting for an unreachable Core past its retry limit after its
/// first run.
TEST(RackTest, WaitsForAnUnreachableCorePastTheRetryLimit) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("outage rack"));
    CorePipe pipe(test_client_config().port);
    auto args = piped_args(client, rack, pipe);
    std::atomic shut_down = false;
    Rack r(FAST_RETRY);
    r.start(args, [&] { shut_down = true; });
    const auto running = [&] {
        const synnax::rack::RetrieveOptions opts{.include_status = true};
        const auto [remote, err] = client.racks.retrieve(rack.key, opts);
        return !err && remote.status.has_value() && remote.status->details.running;
    };
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        running(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    pipe.pause();
    std::this_thread::sleep_for(std::chrono::seconds(1));
    ASSERT_FALSE(shut_down);
    pipe.resume();
    ASSERT_NIL(r.stop());
}

/// @brief it should leave a scanner disabled during a Core outage stopped once the Core
/// is back, with a status that names the state.
TEST(RackTest, ScannerDisabledDuringAnOutageStaysStopped) {
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("scanner outage rack"));
    CorePipe pipe(test_client_config().port);
    auto args = piped_args(client, rack, pipe);
    Rack r(FAST_RETRY);
    r.start(args, [] {});
    const auto scanner = [&] {
        auto tasks = ASSERT_NIL_P(rack.tasks.retrieve(
            synnax::task::Where{.type = http::SCAN_TASK_TYPE},
            synnax::task::RetrieveOptions{.include_status = true}
        ));
        EXPECT_LE(tasks.size(), 1);
        return tasks.empty() ? synnax::task::Task{} : tasks[0];
    };
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        [&] {
            const auto t = scanner();
            return t.status.has_value() && t.status->details.running;
        }(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    pipe.pause();
    ASSERT_NIL(client.tasks.dispatch(
        scanner().key,
        {synnax::task::set({.path = {"config", "disabled"}, .value = true})}
    ));
    pipe.resume();
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        [&] {
            const auto t = scanner();
            return t.status.has_value() && t.status->message == "Scanner disabled";
        }(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    const auto status = scanner().status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_DISABLED);
    EXPECT_FALSE(status.details.running);
    ASSERT_NIL(r.stop());
}
}
