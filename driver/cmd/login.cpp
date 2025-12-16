// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/cli/cli.h"

#include "driver/cmd/cmd.h"

int cmd::sub::login(xargs::Parser &args) {
    synnax::Config config;
    config.host = cli::prompt("Host", "localhost");
    config.port = cli::prompt<uint16_t>("Port", static_cast<uint16_t>(9090));
    config.username = cli::prompt("Username");
    config.password = cli::prompt("Password", std::nullopt, true);
    if (cli::confirm("Secure", false)) {
        config.ca_cert_file = cli::prompt("Path to CA certificate file");
        config.client_cert_file = cli::prompt("Path to client certificate file");
        config.client_key_file = cli::prompt("Path to client key file");
    }

    LOG(INFO) << "connecting to Synnax using the following parameters: \n" << config;
    const synnax::Synnax client(config);
    if (const auto err = client.auth->authenticate()) {
        LOG(ERROR) << xlog::red() << "failed to authenticate: " << err << xlog::reset();
        return 1;
    }
    LOG(INFO) << xlog::green() << "successfully logged in!" << xlog::reset();
    if (const auto err = rack::Config::save_conn_params(args, config)) {
        LOG(ERROR) << xlog::red() << "failed to save credentials: " << err
                   << xlog::reset();
        return 1;
    }
    LOG(INFO) << xlog::green() << "credentials saved successfully!" << xlog::reset();
    return 0;
}
