// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "x/cpp/args/args.h"
#include "x/cpp/cli/cli.h"
#include "x/cpp/log/log.h"
#include "x/cpp/path/path.h"

#include "driver/rack/rack.h"

namespace driver::cmd::sub {
int login(x::args::Parser &args) {
    // Root takes over or hands off the saved state before login reads it. It runs
    // before the prompts, so state that cannot be used fails at once.
    if (const auto err = rack::Config::secure_persisted_state(args)) {
        LOG(ERROR) << err.data;
        return 1;
    }
    synnax::Config config;
    config.host = x::cli::prompt("Host", "localhost");
    config.port = x::cli::prompt<uint16_t>("Port", static_cast<uint16_t>(9090));
    // The service runs in another directory, so a relative path must not be saved.
    if (const auto ca = args.field("ca_cert", std::string()); !ca.empty())
        config.ca_cert = x::path::resolve_relative(ca);
    else
        config.secure = x::cli::confirm("Secure", false);
    {
        const synnax::Synnax probe(config);
        const auto state = probe.connectivity->check();
        if (state.status != synnax::connection::Status::CONNECTED) {
            LOG(ERROR) << "failed to connect: " << state.message;
            return 1;
        }
    }
    LOG(INFO) << x::log::GREEN() << "connection established." << x::log::RESET();

    config.username = x::cli::prompt("Username");
    config.password = x::cli::prompt("Password", std::nullopt, true);

    const auto [cfg, err] = rack::Config::login(args, config);
    // Login can save the API key and then fail, so the key is secured either way.
    const auto sec_err = rack::Config::secure_persisted_state(args);
    if (err) LOG(ERROR) << "failed to enroll the rack: " << err;
    if (sec_err) LOG(ERROR) << "failed to secure the saved API key: " << sec_err;
    if (err || sec_err) return 1;
    LOG(INFO) << x::log::GREEN() << "enrolled rack " << cfg.rack.name << " ("
              << cfg.rack.key << ") and saved its API key" << x::log::RESET();
    LOG(INFO) << "start driver: " << x::log::BLUE() << "synnax-driver start -s"
              << x::log::RESET();
    return 0;
}
}
