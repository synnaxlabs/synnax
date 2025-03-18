// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"
#include "x/cpp/cli/cli.h"

int cmd::sub::login(xargs::Parser &args) {
    synnax::Config config;
    bool valid_input = false;

    while (!valid_input) {
        config.host = cli::prompt("host (default: localhost): ");
        if (config.host.empty()) config.host = "localhost";
        std::string port_str = cli::prompt("port (default: 9090): ");
        if (port_str.empty()) config.port = 9090;
        else {
            try {
                config.port = static_cast<uint16_t>(std::stoi(port_str));
            } catch (const std::exception &e) {
                LOG(WARNING) <<
                        "Invalid port number. Please enter a valid number between 0 and 65535.";
                continue;
            }
        }

        config.username = cli::prompt("username: ");
        if (config.username.empty()) {
            LOG(WARNING) << "Username must be provided.";
            continue;
        }

        config.password = cli::prompt("password: ", true);
        if (config.password.empty()) {
            LOG(WARNING) << "Password must be provided.";
            continue;
        }

        valid_input = true;
    }

    LOG(INFO) << "connecting to Synnax at " << config.host << ":" << config.port;
    const synnax::Synnax client(config);
    if (const auto err = client.auth->authenticate()) {
        LOG(ERROR) << xlog::RED() << "failed to authenticate: " << err << xlog::RESET();
        return 1;
    }
    LOG(INFO) << xlog::GREEN() << "successfully logged in!" << xlog::RESET();
    if (const auto err = rack::Config::save_conn_params(args, config)) {
        LOG(ERROR) << xlog::RED() << "failed to save credentials: " << err <<
                xlog::RESET();
        return 1;
    }
    LOG(INFO) << xlog::GREEN() << "credentials saved successfully!" << xlog::RESET();
    return 0;
}
