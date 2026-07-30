// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <functional>
#include <iostream>
#include <memory>
#include <string>

#include "x/cpp/args/args.h"
#include "x/cpp/cli/cli.h"
#include "x/cpp/crash/crash.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/log/log.h"
#include "x/cpp/shutdown/shutdown.h"

#include "absl/log/globals.h"
#include "absl/log/log.h"
#include "core/pkg/version/version.h"
#include "driver/cmd/cmd.h"
#include "driver/daemon/daemon.h"
#include "driver/rack/rack.h"

namespace driver::cmd {
namespace {
std::string version() {
    return "v" + std::string(SYNNAX_DRIVER_VERSION) + " (" +
           std::string(SYNNAX_BUILD_TIMESTAMP) + ")";
}

int version(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "Synnax Driver " << version() << x::log::RESET();
    return 0;
}

int start(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "starting Synnax Driver " << version()
              << x::log::RESET();

    const bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    VLOG(1) << "stdin stop " << (stdin_stop_enabled ? "enabled" : "disabled");

    const bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    VLOG(1) << "sig stop " << (sig_stop_enabled ? "enabled" : "disabled");

    if (args.error()) {
        LOG(ERROR) << "invalid arguments: " << args.error();
        return 1;
    }

    rack::Rack r;

    // Register an early shutdown handler to stop the driver when the process encounters
    // an error.
    auto early_shutdown = std::make_shared<std::atomic<bool>>(false);
    const std::function on_shutdown = [early_shutdown] {
        x::shutdown::signal_shutdown();
        early_shutdown->store(true);
    };

    r.start(args, on_shutdown);

    // Register a signal handler to stop the driver when the process receives a signal.
    x::shutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    if (!early_shutdown->load())
        LOG(INFO) << x::log::BLUE()
                  << "received shutdown signal. Gracefully stopping driver. "
                     "This can take up to 5 seconds. Please be patient"
                  << x::log::RESET();
    else
        LOG(WARNING) << "unexpected early shutdown";
    if (const auto err = r.stop())
        LOG(ERROR) << "stopped with error: " << err;
    else
        LOG(INFO) << x::log::BLUE() << "stopped" << x::log::RESET();
    return 0;
}

int internal_start(const int argc, char *argv[]) {
    daemon::Config config;
    config.callback = [](const int cb_argc, char *cb_argv[]) {
        auto cb_args = x::args::Parser(cb_argc, cb_argv);
        start(cb_args);
    };
    daemon::run(config, argc, argv);
    return 0;
}

int login(x::args::Parser &args) {
    synnax::Config config;
    config.host = x::cli::prompt("Host", "localhost");
    config.port = x::cli::prompt<uint16_t>("Port", static_cast<uint16_t>(9090));
    if (x::cli::confirm("Secure", false)) {
        config.secure = true;
        config.ca_cert_file = x::cli::prompt("Path to CA certificate file", "");
        config.client_cert_file = x::cli::prompt("Path to client certificate file", "");
        config.client_key_file = x::cli::prompt("Path to client key file", "");
    }
    {
        const synnax::Synnax probe(config);
        const auto state = probe.connectivity->check();
        if (state.status != synnax::connection::Status::CONNECTED) {
            LOG(ERROR) << x::log::RED() << "failed to connect: " << state.message
                       << x::log::RESET();
            return 1;
        }
    }
    LOG(INFO) << x::log::GREEN() << "connection established." << x::log::RESET();

    config.username = x::cli::prompt("Username");
    config.password = x::cli::prompt("Password", std::nullopt, true);

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
    LOG(INFO) << "start driver: " << x::log::BLUE() << "synnax-driver start -s"
              << x::log::RESET();
    return 0;
}

int clear(x::args::Parser &args) {
    if (const auto err = rack::Config::clear_persisted_state(args); err) {
        LOG(ERROR) << "failed to clear persisted state: " << err;
        return 1;
    }
    return 0;
}

int exec_svc_cmd(
    const std::function<x::errors::Error()> &cmd,
    const std::string &action,
    const std::string &past_tense = ""
) {
    if (const auto err = cmd()) {
        LOG(ERROR) << "" << x::log::RED() << "Failed to " << action << ": " << err
                   << x::log::RESET();
        return 1;
    }
    if (!past_tense.empty()) {
        LOG(INFO) << "" << x::log::GREEN() << past_tense << " successfully"
                  << x::log::RESET();
    }
    return 0;
}

int service_start(x::args::Parser &args) {
    return exec_svc_cmd(daemon::start_service, "start", "started");
}

int service_stop(x::args::Parser &args) {
    return exec_svc_cmd(daemon::stop_service, "stop", "stopped");
}

int service_restart(x::args::Parser &args) {
    return exec_svc_cmd(daemon::restart_service, "restart", "restarted");
}

int service_install(x::args::Parser &args) {
    return exec_svc_cmd(daemon::install_service, "install", "installed");
}

int service_uninstall(x::args::Parser &args) {
    return exec_svc_cmd(daemon::uninstall_service, "uninstall", "uninstalled");
}

int service_view_logs(x::args::Parser &args) {
    return exec_svc_cmd(daemon::view_logs, "view logs");
}

int service_status(x::args::Parser &args) {
    return exec_svc_cmd(daemon::status, "status");
}

void print_usage() {
    std::cout
        << "Usage: synnax-driver <command> [options]\n"
        << "Commands:\n"
        << "  start                     Start the Driver service\n"
        << "    --standalone/-s         Run in standalone mode (not as a service)\n"
        << "    --debug                 Enable debug logging\n"
        << "    --no-color              Disable color output in logs\n"
        << "    --disable-sig-stop      Prevent SIGINT and SIGTERM from stopping the "
           "Driver\n"
        << "    --disable-stdin-stop    Prevent typing 'STOP' into stdin from stopping "
           "the Driver\n"
        << "  status                    Display the Driver's status\n"
        << "  stop                      Stop the Driver\n"
        << "  restart                   Restart the Driver\n"
        << "  login                     Log in to Synnax\n"
        << "  install                   Install the Driver as a system service\n"
        << "  uninstall                 Uninstall the Driver\n"
        << "  logs                      View the Driver's logs\n"
        << "  version                   Display the Driver's version\n"
        << "  clear                     Clear the persisted state\n";
}
}

int exec(const int argc, char *argv[]) {
    x::crash::install("synnax-driver");
    auto args = x::args::Parser(argc, argv);
    // Color only when stderr is a terminal so ANSI codes never land in redirected log
    // files (systemd journal, /var/log/synnax-driver.log on NI Linux RT).
    x::log::init(!args.flag("--no-color") && x::log::stderr_is_terminal());
    if (args.flag("--debug")) absl::SetGlobalVLogLevel(2);
    VLOG(1) << "debug logging enabled";
    const std::string command = args.at(1, "command name required");
    if (args.error()) {
        print_usage();
        return 1;
    }
    if (command == "start") {
        if (args.flag("--standalone", "-s")) return start(args);
        return service_start(args);
    }
    if (command == "internal-start") return internal_start(argc, argv);
    if (command == "stop") return service_stop(args);
    if (command == "restart") return service_restart(args);
    if (command == "login") return login(args);
    if (command == "install") return service_install(args);
    if (command == "uninstall") return service_uninstall(args);
    if (command == "logs") return service_view_logs(args);
    if (command == "status") return service_status(args);
    if (command == "version") return version(args);
    if (command == "clear") return clear(args);
    print_usage();
    return 1;
}
}
