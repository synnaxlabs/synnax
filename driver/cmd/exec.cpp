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

// void cmd_start_daemon(int argc, char *argv[]) {
//     daemond::Config config;
//     config.watchdog_interval = 10;
//     config.callback = [](const int argc_, char *argv_[]) {
//         cmd_start_standalone(argc_, argv_);
//     };
//     daemond::run(config, argc, argv);
// }

int cmd::exec(int argc, char *argv[]) {
    FLAGS_logtostderr = 1;
    FLAGS_colorlogtostderr = 1;
    google::InitGoogleLogging(argv[0]);

    return cmd::sub::start(argc, argv);
    if (argc < 2) {
        print_usage();
        return 1;
    }

    const std::string command = argv[1];
    if (command == "start") {
        bool standalone = false;
        for (int i = 2; i < argc; i++) {
            const std::string arg = argv[i];
            if (arg == "--standalone" || arg == "-s") {
                standalone = true;
                break;
            }
        }
        if (standalone) return cmd::sub::start(argc, argv);
        return cmd::sub::service_start(argc, argv);
    }
    if (command == "stop") return cmd::sub::service_stop(argc, argv);
    if (command == "restart") return cmd::sub::service_restart(argc, argv);
    if (command == "login") return cmd::sub::login(argc, argv);
    if (command == "install") return cmd::sub::service_install(argc, argv);
    if (command == "uninstall") return cmd::sub::service_uninstall(argc, argv);
    if (command == "logs") return cmd::sub::service_view_logs(argc, argv);
    if (command == "status") return cmd::sub::service_status(argc, argv);
    if (command == "version") return cmd::sub::version(argc, argv);
    if (command == "clear") return cmd::sub::clear(argc, argv);
    print_usage();
    return 1;
}


