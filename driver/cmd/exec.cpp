// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

void print_usage() {
    std::cout << "Usage: synnax-driver <command> [options]\n"
            << "Commands:\n"
            << "  start           Start the Synnax driver service\n"
            << "    --standalone  Run in standalone mode (not as a service)\n"
            << "    -s           Short form for --standalone\n"
            << "  stop            Stop the Synnax driver service\n"
            << "  restart         Restart the Synnax driver service\n"
            << "  login           Log in to Synnax\n"
            << "  install         Install the Synnax driver as a system service\n"
            << "  uninstall       Uninstall the Synnax driver service\n"
            << "  logs            View the driver logs\n"
            << "  version         Display the driver version\n"
            << "  clear           Clear the persisted state\n";
}

int cmd::exec(const int argc, char *argv[]) {
    FLAGS_logtostderr = true;
    FLAGS_colorlogtostderr = true;
    google::InitGoogleLogging(argv[0]);
    auto args = xargs::Parser(argc, argv);
    const std::string command = args.at(1, "command name required");
    if (args.error()) {
        print_usage();
        return 1;
    }
    if (command == "start") {
        if (args.flag("--standalone", "-s")) return cmd::sub::start(args);
        return cmd::sub::service_start(args);
    }
    if (command == "stop") return cmd::sub::service_stop(args);
    if (command == "restart") return cmd::sub::service_restart(args);
    if (command == "login") return cmd::sub::login(args);
    if (command == "install") return cmd::sub::service_install(args);
    if (command == "uninstall") return cmd::sub::service_uninstall(args);
    if (command == "logs") return cmd::sub::service_view_logs(args);
    if (command == "status") return cmd::sub::service_status(args);
    if (command == "version") return cmd::sub::version(args);
    if (command == "clear") return cmd::sub::clear(args);
    print_usage();
    return 1;
}


