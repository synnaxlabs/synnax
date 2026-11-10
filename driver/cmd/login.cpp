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

#include "driver/rack/rack.h"

namespace driver::cmd::sub {
int login(x::args::Parser &args) {
    synnax::Config config;
    config.host = x::cli::prompt("Host", "localhost");
    config.port = x::cli::prompt<uint16_t>("Port", static_cast<uint16_t>(9090));
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
    if (err) {
        LOG(ERROR) << "failed to enroll the rack: " << err;
        return 1;
    }
    LOG(INFO) << x::log::GREEN() << "enrolled rack " << cfg.rack.name << " ("
              << cfg.rack.key << ") and saved its API key" << x::log::RESET();
    LOG(INFO) << "start driver: " << x::log::BLUE() << "synnax-driver start -s"
              << x::log::RESET();
    return 0;
}
}
