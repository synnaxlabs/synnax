// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

namespace driver::cmd {
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
    google::InitGoogleLogging(argv[0]);
    auto args = x::args::Parser(argc, argv);
    const bool disable_color = args.flag("--no-color");
    FLAGS_logtostderr = true;
    FLAGS_colorlogtostderr = !disable_color;
    if (args.flag("--debug")) FLAGS_v = 2;
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
