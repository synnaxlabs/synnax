// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <filesystem>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/env/env.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should keep the state of a user in the folder of that user while the
/// Driver service is not installed.
TEST(StateTest, UserStateWithoutService) {
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
