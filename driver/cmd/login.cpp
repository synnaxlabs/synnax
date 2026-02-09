// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/cli/cli.h"

#include "driver/cmd/cmd.h"

namespace driver::cmd::sub {
int login(x::args::Parser &args) {
    synnax::Config config;
    config.host = x::cli::prompt("Host", "localhost");
    config.port = x::cli::prompt<uint16_t>("Port", static_cast<uint16_t>(9090));
    config.username = x::cli::prompt("Username");
    config.password = x::cli::prompt("Password", std::nullopt, true);
    if (x::cli::confirm("Secure", false)) {
        config.ca_cert_file = x::cli::prompt("Path to CA certificate file");
        config.client_cert_file = x::cli::prompt("Path to client certificate file");
        config.client_key_file = x::cli::prompt("Path to client key file");
    }

    LOG(INFO) << "connecting to Synnax using the following parameters: \n" << config;
    const synnax::Synnax client(config);
    if (const auto err = client.auth->authenticate()) {
        LOG(ERROR) << x::log::RED() << "failed to authenticate: " << err
                   << x::log::RESET();
        return 1;
    }
    LOG(INFO) << x::log::GREEN() << "successfully logged in!" << x::log::RESET();
    if (const auto err = rack::Config::save_conn_params(args, config)) {
        LOG(ERROR) << x::log::RED() << "failed to save credentials: " << err
                   << x::log::RESET();
        return 1;
    }
    LOG(INFO) << x::log::GREEN() << "credentials saved successfully!"
              << x::log::RESET();
    return 0;
}
}
