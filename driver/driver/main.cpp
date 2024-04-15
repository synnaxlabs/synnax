// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include <fstream>
#include <iostream>
#include <getopt.h>

#include "nlohmann/json.hpp"
#include "glog/logging.h"

#include "driver/driver/driver.h"
#include "task/task.h"
#include "driver/driver/opc/opc.h"

using json = nlohmann::json;

std::unique_ptr<driver::Driver> d;

std::pair<synnax::Rack, freighter::Error> retrieveDriverRack(
    const driver::Config &config,
    breaker::Breaker &breaker,
    const std::shared_ptr<synnax::Synnax> &client
) {
    auto [rack, err] = client->hardware.retrieveRack(
        config.rack_key != 0 ? config.rack_key : config.rack_name);
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return retrieveDriverRack(config, breaker, client);
    return {rack, err};
}

int main(int argc, char *argv[]) {
    static struct option long_options[] = {
        {"config", required_argument, 0, 'c'},
        {0, 0, 0, 0}
    };

    int option_index = 0;
    int c;
    std::string config_path = "./synnax-driver-config.json";
    // Variable to store the config file path

    // Parse the command line arguments using getopt_long
    while ((c = getopt_long(argc, argv, "c:", long_options, &option_index)) != -1) {
        switch (c) {
            case 'c':
                config_path = optarg; // Assign the optarg to your string variable
                break;
            case '?':
                // getopt_long already printed an error message.
                return 1;
            default:
                std::cerr << "Unknown error while parsing options" << std::endl;
                return 1;
        }
    }

    google::InitGoogleLogging(argv[0]);
    google::SetCommandLineOption("minloglevel", "0");

    auto cfg_json = driver::readConfig(config_path);
    auto [cfg, cfg_err] = driver::parseConfig(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[Driver] failed to parse configuration: " << cfg_err;
        return 1;
    }

    auto breaker = breaker::Breaker(cfg.breaker_config);
    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    LOG(INFO) << "[Driver] retrieving meta-data";
    auto [rack, rack_err] = retrieveDriverRack(cfg, breaker, client);
    if (rack_err) {
        LOG(FATAL) <<
                "[Driver] failed to retrieve meta-data - can't proceed without it. Exiting."
                << rack_err;
        return 1;
    }

    std::unique_ptr<task::Factory> opc_factory = std::make_unique<opc::Factory>();
    std::vector<std::shared_ptr<task::Factory> > factories = {std::move(opc_factory)};
    std::unique_ptr<task::Factory> factory = std::make_unique<task::MultiFactory>(
        std::move(factories)
    );

    d = std::make_unique<driver::Driver>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config
    );
    signal(SIGINT, [](int _) {
        LOG(INFO) << "[Driver] received interrupt signal. shutting down";
        d->stop();
    });
    d->run();
    LOG(INFO) << "[Driver] shutdown complete";
    return 0;
}
