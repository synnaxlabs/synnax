// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <string>
#include <iostream>

/// external
#include "glog/logging.h"

/// internal
#include "driver/cmd/cmd.h"

std::string get_secure_input(const std::string &prompt, bool hide_input = false) {
    std::string input;
#ifdef _WIN32
        HANDLE h_stdin = GetStdHandle(STD_INPUT_HANDLE);
        DWORD mode;
        GetConsoleMode(h_stdin, &mode);
        if (hide_input) {
            SetConsoleMode(h_stdin, mode & (~ENABLE_ECHO_INPUT));
        }
#else
    if (hide_input) {
        system("stty -echo");
    }
#endif

    std::cout << prompt;
    std::getline(std::cin, input);

    if (hide_input) {
        std::cout << std::endl;
#ifdef _WIN32
            SetConsoleMode(h_stdin, mode);
#else
        system("stty echo");
#endif
    }
    return input;
}

int cmd::priv::login(int argc, char **argv) {
    // synnax::Config config;
    // bool valid_input = false;
    //
    // while (!valid_input) {
    //     // Get host
    //     config.host = get_secure_input("Host (default: localhost): ");
    //     if (config.host.empty()) config.host = "localhost";
    //
    //     // Get port
    //     std::string port_str = get_secure_input("Port (default: 9090): ");
    //     if (port_str.empty()) {
    //         config.port = 9090;
    //     } else {
    //         try {
    //             config.port = static_cast<uint16_t>(std::stoi(port_str));
    //         } catch (const std::exception &e) {
    //             LOG(WARNING) <<
    //                     "Invalid port number. Please enter a valid number between 0 and 65535.";
    //             continue;
    //         }
    //     }
    //
    //     // Get username
    //     config.username = get_secure_input("Username: ");
    //     if (config.username.empty()) {
    //         LOG(WARNING) << "Username cannot be empty.";
    //         continue;
    //     }
    //
    //     // Get password
    //     config.password = get_secure_input("Password: ", true);
    //     if (config.password.empty()) {
    //         LOG(WARNING) << "Password cannot be empty.";
    //         continue;
    //     }
    //
    //     valid_input = true;
    // }
    //
    // LOG(INFO) << "Attempting to connect to Synnax at " << config.host << ":" << config.
    //         port;
    // synnax::Synnax client(config);
    // if (const auto err = client.auth->authenticate()) {
    //     LOG(ERROR) << "Failed to authenticate: " << err;
    //     return;
    // }
    // LOG(INFO) << "Successfully logged in!";
    //
    // auto [existing_state, load_err] = driver::load_persisted_state();
    // if (load_err) {
    //     LOG(ERROR) << "Failed to load persisted state: " << load_err;
    //     return;
    // }
    // driver::PersistedState state{
    //     .rack_key = existing_state.rack_key,
    //     .connection = config
    // };
    //
    // if (auto err = driver::save_persisted_state(state)) {
    //     LOG(ERROR) << "Failed to save credentials: " << err;
    //     return;
    // }
    // LOG(INFO) << "Credentials saved successfully!";
    return 0;
}
