// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>
#include <string>

#include "x/cpp/args/args.h"
#include "x/cpp/crash/crash.h"
#include "x/cpp/log/log.h"

#include "absl/log/globals.h"
#include "absl/log/log.h"
#include "driver/cmd/cmd.h"

namespace driver::cmd {
// Subcommands are defined in sibling files, one per command.
namespace sub {
int clear(x::args::Parser &args);
int internal_start(x::args::Parser &args);
int login(x::args::Parser &args);
int service_install(x::args::Parser &args);
int service_restart(x::args::Parser &args);
int service_start(x::args::Parser &args);
int service_status(x::args::Parser &args);
int service_stop(x::args::Parser &args);
int service_uninstall(x::args::Parser &args);
int service_view_logs(x::args::Parser &args);
int start(x::args::Parser &args);
int version(x::args::Parser &args);
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
        if (args.flag("--standalone", "-s")) return sub::start(args);
        return sub::service_start(args);
    }
    // Run by the service manager (systemd ExecStart); omitted from the usage text.
    if (command == "internal-start") return sub::internal_start(args);
    if (command == "stop") return sub::service_stop(args);
    if (command == "restart") return sub::service_restart(args);
    if (command == "login") return sub::login(args);
    if (command == "install") return sub::service_install(args);
    if (command == "uninstall") return sub::service_uninstall(args);
    if (command == "logs") return sub::service_view_logs(args);
    if (command == "status") return sub::service_status(args);
    if (command == "version") return sub::version(args);
    if (command == "clear") return sub::clear(args);
    print_usage();
    return 1;
}
}
