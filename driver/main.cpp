// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sys/stat.h>

/// POSIX
#include <unistd.h>

/// Windows-specific headers
#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#ifndef NOMINMAX
#define NOMINMAX
#endif

#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>

/// LabJack only supported on Windows.
#include "driver/labjack/labjack.h"
#endif

/// std
#include <fstream>
#include <iostream>
#include <thread>
#include <condition_variable>
#include <mutex>
#include <array>
#include <filesystem>
#include <system_error>

/// external
#include "nlohmann/json.hpp"
#include "glog/logging.h"

/// internal
#include "driver/config.h"
#include "driver/task/task.h"
#include "driver/opc/opc.h"
#include "driver/meminfo/meminfo.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/ni/ni.h"
#include "driver/sequence/task.h"
#include "driver/daemon.h"

using json = nlohmann::json;

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

namespace fs = std::filesystem;

std::string get_hostname() {
    std::array<char, 256> hostname{};
#ifdef _WIN32
    DWORD size = hostname.size();
    if (GetComputerNameA(hostname.data(), &size) == 0) {
        LOG(WARNING) << "[driver] Failed to get hostname";
        return "unknown";
    }
#else
    if (gethostname(hostname.data(), hostname.size()) != 0) {
        LOG(WARNING) << "[driver] Failed to get hostname";
        return "unknown";
    }
#endif
    return {hostname.data()};
}

std::pair<synnax::Rack, freighter::Error> retrieve_driver_rack(
    configd::Config &config,
    breaker::Breaker &breaker,
    const std::shared_ptr<synnax::Synnax> &client
) {
    std::pair<synnax::Rack, freighter::Error> res;
    if (config.rack_key != 0) {
        LOG(INFO) << "existing rack key found in configuration: " << config.rack_key;
        res = client->hardware.retrieve_rack(config.rack_key);
    } else {
        LOG(INFO) << "no existing rack key found in configuration. Creating a new rack";
        res = client->hardware.create_rack(get_hostname());
    }
    const auto err = res.second;
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return retrieve_driver_rack(config, breaker, client);
    if (err.matches(synnax::NOT_FOUND)) {
        config.rack_key = 0;
        return retrieve_driver_rack(config, breaker, client);
    }
    LOG(INFO) << "[driver] retrieved rack: " << res.first.key << " - " << res.first.
            name;
    return res;
}

const std::string STOP_COMMAND = "STOP";

void input_listener() {
    std::string input;
    while (std::getline(std::cin, input)) {
        if (input == STOP_COMMAND) {
            {
                std::lock_guard lock(mtx);
                should_stop = true;
            }
            cv.notify_one();
            break;
        }
    }
}

void configure_opc(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(opc::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] OPC integration disabled";
        return;
    }
    factories.push_back(std::make_shared<opc::Factory>());
}

void configure_ni(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(ni::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] NI integration disabled";
        return;
    }
    const auto ni_factory = ni::Factory::create();
    factories.push_back(ni_factory);
}

void configure_sequences(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(sequence::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] Sequence integration disabled";
        return;
    }
    factories.push_back(std::make_shared<sequence::Factory>());
}

void configure_labjack(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories
) {
#ifdef _WIN32
    if (
        !config.integration_enabled(labjack::INTEGRATION_NAME) ||
        !labjack::dlls_available()
    ) {
        LOG(INFO) << "[driver] LabJack integration disabled";
        return;
    }
    auto labjack_factory = std::make_shared<labjack::Factory>();
    factories.push_back(labjack_factory);
    return;
#endif
    LOG(INFO) << "[driver] LabJack integration not available on this platform";
}

void cmd_start_standalone(int argc, char *argv[]) {
    std::string config_path = "./synnax-driver-config.json";
    if (argc > 2) // Changed from argc > 1 to account for the command
        config_path = argv[2];

    auto cfg_json = configd::read(config_path);
    LOG(INFO) << "[driver] reading configuration from " << config_path;
    if (cfg_json.empty())
        LOG(INFO) << "[driver] no configuration found at " << config_path <<
                ". We'll just use the default configuration";
    else
        LOG(INFO) << "[driver] loaded configuration from " << config_path;
    auto [cfg, cfg_err] = configd::parse(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[driver] failed to parse configuration: " << cfg_err;
        return;
    }
    VLOG(1) << "[driver] configuration parsed successfully";

    auto [persisted_state, state_err] = configd::load_persisted_state();
    if (state_err) {
        LOG(WARNING) << "[driver] failed to load persisted state: " << state_err;
    } else {
        LOG(INFO) << "peristed state found in storage";
        if (persisted_state.rack_key != 0 && cfg.rack_key == 0) {
            VLOG(1) << "[driver] using persisted rack key: " << persisted_state.
rack_key;
            cfg.rack_key = persisted_state.rack_key;
        }
        if (!persisted_state.connection.host.empty()) {
            cfg.client_config = persisted_state.connection;
            LOG(INFO) << "[driver] using persisted credentials";
        }
    }

    LOG(INFO) << "[driver] starting up";

    // FLAGS_logtostderr = true;
    // if (cfg.debug)
    //     FLAGS_v = 1;
    // google::InitGoogleLogging(argv[0]);

    VLOG(1) << "[driver] connecting to Synnax at " << cfg.client_config.host << ":"
            << cfg.client_config.port;

    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    auto breaker = breaker::Breaker(cfg.breaker_config);
    breaker.start();
    VLOG(1) << "[driver] retrieving meta-data";
    auto [rack, rack_err] = retrieve_driver_rack(cfg, breaker, client);
    breaker.stop();
    if (rack_err) {
        LOG(FATAL) <<
                "[driver] failed to retrieve meta-data - can't proceed without it. Exiting."
                << rack_err;
        return;
    }

    if (auto err = configd::save_persisted_state({
        .rack_key = rack.key,
        .connection = cfg.client_config
    }))
        LOG(WARNING) << "[driver] failed to save persisted state: " << err;

    auto hb_factory = std::make_shared<heartbeat::Factory>();
    std::vector<std::shared_ptr<task::Factory> > factories{hb_factory};
    configure_opc(cfg, factories);
    configure_ni(cfg, factories);
    configure_sequences(cfg, factories);
    configure_labjack(cfg, factories);

    LOG(INFO) << "[driver] starting task manager";

    auto factory = std::make_unique<task::MultiFactory>(std::move(factories));
    auto task_manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config
    );

    std::thread listener(input_listener);

    if (auto err = task_manager->start()) {
        LOG(FATAL) << "[driver] failed to start: " << err;
        return;
    } {
        std::unique_lock lock(mtx);
        cv.wait(lock, [] { return should_stop; });
    }

    LOG(INFO) << "[driver] received stop command. Shutting down";
    task_manager->stop();
    listener.join();
    LOG(INFO) << "[driver] shutdown complete";
}

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

void cmd_login(int argc, char *argv[]) {
    synnax::Config config;
    bool valid_input = false;

    while (!valid_input) {
        // Get host
        config.host = get_secure_input("Host (default: localhost): ");
        if (config.host.empty()) config.host = "localhost";

        // Get port
        std::string port_str = get_secure_input("Port (default: 9090): ");
        if (port_str.empty()) {
            config.port = 9090;
        } else {
            try {
                config.port = static_cast<uint16_t>(std::stoi(port_str));
            } catch (const std::exception &e) {
                LOG(WARNING) <<
                        "Invalid port number. Please enter a valid number between 0 and 65535.";
                continue;
            }
        }

        // Get username
        config.username = get_secure_input("Username: ");
        if (config.username.empty()) {
            LOG(WARNING) << "Username cannot be empty.";
            continue;
        }

        // Get password
        config.password = get_secure_input("Password: ", true);
        if (config.password.empty()) {
            LOG(WARNING) << "Password cannot be empty.";
            continue;
        }

        valid_input = true;
    }

    LOG(INFO) << "Attempting to connect to Synnax at " << config.host << ":" << config.
            port;
    synnax::Synnax client(config);
    if (const auto err = client.auth->authenticate()) {
        LOG(ERROR) << "Failed to authenticate: " << err;
        return;
    }
    LOG(INFO) << "Successfully logged in!";

    auto [existing_state, _] = configd::load_persisted_state();
    configd::PersistedState state{
        .rack_key = existing_state.rack_key,
        .connection = config
    };

    if (auto err = configd::save_persisted_state(state)) {
        LOG(ERROR) << "Failed to save credentials: " << err;
        return;
    }
    LOG(INFO) << "Credentials saved successfully!";
}

void print_usage() {
    std::cout << "Usage: synnax-driver <command> [options]\n"
            << "Commands:\n"
            << "  start    Start the Synnax driver\n"
            << "  login    Log in to Synnax\n"
            << "  install  Install the Synnax driver as a system service\n";
}

void cmd_install_service() {
    daemond::install_service();
}

void cmd_uninstall_service() {
    if (geteuid() != 0) {
        LOG(ERROR) << "Must run as root to uninstall service";
        exit(1);
    }

    try {
        daemond::uninstall_service();
        LOG(INFO) << "Service uninstallation completed successfully";
    } catch (const std::exception &e) {
        LOG(ERROR) << "Failed to uninstall service: " << e.what();
        exit(1);
    }
}

void cmd_start_daemon(int argc, char *argv[]) {
    // Load persisted state to get connection details
    auto [persisted_state, state_err] = configd::load_persisted_state();
    if (state_err) {
        LOG(FATAL) << "[driver] failed to load persisted state: " << state_err;
        return;
    }

    daemond::Config config;
    config.watchdog_interval = 10;
    config.callback = [](const int argc_, char *argv_[]) {
        cmd_start_standalone(argc_, argv_);
    };

    daemond::run(config, argc, argv);
}

int main(const int argc, char *argv[]) {
    FLAGS_logtostderr = true;
    google::InitGoogleLogging(argv[0]);

    if (argc < 2) {
        print_usage();
        return 1;
    }

    const std::string command = argv[1];

    if (command == "start-standalone")cmd_start_standalone(argc, argv);
    else if (command == "start") cmd_start_daemon(argc, argv);
    else if (command == "login") cmd_login(argc, argv);
    else if (command == "install") cmd_install_service();
    else if (command == "uninstall") cmd_uninstall_service();
    else {
        std::cout << "Unknown command: " << command << std::endl;
        print_usage();
        return 1;
    }
    return 0;
}
