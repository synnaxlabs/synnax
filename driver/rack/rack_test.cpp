// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstdlib>
#include <filesystem>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/env/env.h"
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

/// @brief it should keep the state of a user in the folder of that user while the
/// Driver service is not installed.
TEST(RackTest, UserStateWithoutService) {
#ifdef __linux__
    GTEST_SKIP() << "Linux keeps the state of every process in the folder of the "
                    "computer";
#endif
    const auto home = std::filesystem::temp_directory_path() / "rack-test-home";
    std::filesystem::remove_all(home);
#ifdef _WIN32
    const std::string var = "LOCALAPPDATA";
    const auto expected = home / "synnax-driver";
#else
    const std::string var = "HOME";
    const auto expected = home / "Library" / "Application Support" / "synnax-driver";
#endif
    const char *previous = std::getenv(var.c_str());
    const char *sudo_user = std::getenv("SUDO_USER");
    x::defer::defer restore([&,
                             previous = std::string(previous ? previous : ""),
                             sudo_user = std::string(sudo_user ? sudo_user : "")] {
        x::env::set(var, previous);
        if (!sudo_user.empty()) x::env::set("SUDO_USER", sudo_user);
        std::filesystem::remove_all(home);
    });
    x::env::set(var, home.string());
    x::env::unset("SUDO_USER");
    auto args = x::args::Parser(std::vector<std::string>{"program"});
    Config cfg;
    ASSERT_NIL(cfg.load_persisted_state(args));
    ASSERT_EQ(cfg.state_dir, expected);
}
}
