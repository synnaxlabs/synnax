// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/args/args.h"
#include "x/cpp/env/env.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/rack.h"

namespace driver::rack {
namespace {
const std::vector<std::string> DEFAULTS = {"http", "modbus", "opc"};

Config load(const std::vector<std::string> &flags, x::errors::Error &err) {
    std::vector<std::string> argv = {"program"};
    argv.insert(argv.end(), flags.begin(), flags.end());
    x::args::Parser args(argv);
    Config cfg;
    cfg.integrations = DEFAULTS;
    err = cfg.load_args(args);
    return cfg;
}
}

/// @brief it should keep the starting integrations when no integration flag is given.
TEST(ArgsTest, KeepIntegrationsWithoutFlags) {
    x::errors::Error err;
    const auto cfg = load({}, err);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.integrations, DEFAULTS);
}

/// @brief it should run only the integrations that --enable-integrations names.
TEST(ArgsTest, EnableIntegrationsRunsOnlyThose) {
    x::errors::Error err;
    const auto cfg = load({"--enable-integrations", "http,ni"}, err);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.integrations, (std::vector<std::string>{"http", "ni"}));
}

/// @brief it should run each integration once when --enable-integrations repeats it.
TEST(ArgsTest, EnableIntegrationsDropsRepeats) {
    x::errors::Error err;
    const auto cfg = load({"--enable-integrations", "http,ni,http"}, err);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.integrations, (std::vector<std::string>{"http", "ni"}));
}

/// @brief it should remove the integrations that --disable-integrations names.
TEST(ArgsTest, DisableIntegrationsRemovesThose) {
    x::errors::Error err;
    const auto cfg = load({"--disable-integrations", "modbus,ni"}, err);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.integrations, (std::vector<std::string>{"http", "opc"}));
}

/// @brief it should fail when both integration flags are given.
TEST(ArgsTest, RejectBothIntegrationFlags) {
    x::errors::Error err;
    load({"--enable-integrations", "http", "--disable-integrations", "ni"}, err);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(
        err.data.find("cannot be used with --disable-integrations"),
        std::string::npos
    );
}

/// @brief it should read the startup retries from --startup-retries.
TEST(ArgsTest, StartupRetriesFlag) {
    x::errors::Error err;
    const auto cfg = load({"--startup-retries", "3"}, err);
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.retry.startup_retries, 3);
}

/// @brief it should reject negative startup retries.
TEST(ArgsTest, RejectNegativeStartupRetries) {
    x::errors::Error err;
    load({"--startup-retries", "-1"}, err);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
}

/// @brief it should read the startup retries from SYNNAX_DRIVER_STARTUP_RETRIES.
TEST(ArgsTest, StartupRetriesEnv) {
    x::env::set("SYNNAX_DRIVER_STARTUP_RETRIES", "4");
    Config cfg;
    const auto err = cfg.load_env();
    x::env::unset("SYNNAX_DRIVER_STARTUP_RETRIES");
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.retry.startup_retries, 4);
}
}
