// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include "driver/driver/driver.h"
#include "task/task.h"
#include "glog/logging.h"
#include "driver/driver/opcua/opcua.h"

std::unique_ptr<driver::Driver> d;

std::pair<synnax::Rack, freighter::Error> retrieveDriverRack(
    breaker::Breaker &breaker,
    const std::shared_ptr<synnax::Synnax> &client
) {
    auto [rack, err] = client->hardware.retrieveRack("sy_node_1_rack");
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return retrieveDriverRack(breaker, client);
    return {rack, err};
}

int main(int argc, char *argv[]) {
    google::InitGoogleLogging(argv[0]);
    google::SetCommandLineOption("logtostderr", "1");
    google::SetCommandLineOption("minloglevel", "0");
    LOG(INFO) << "[Driver] starting up";
    auto cfg = synnax::Config{
        .host = "localhost",
        .port = 9090,
        .username = "synnax",
        .password = "seldon",
    };

    auto client = std::make_shared<synnax::Synnax>(cfg);

    breaker::Config breaker_config{
        "driver",
        synnax::SECOND * 1,
        50,
        1.1
    };

    auto rack_bootup_breaker = breaker::Breaker(breaker_config.child("startup"));

    LOG(INFO) << "[Driver] retrieving meta-data";
    auto [rack, err] = retrieveDriverRack(rack_bootup_breaker, client);
    if (err) {
        LOG(FATAL) << "[Driver] failed to retrieve meta-data - can't proceed without it. Exiting." << err;
        return 1;
    }

    std::unique_ptr<task::Factory> opcua_factory = std::make_unique<opcua::Factory>();
    std::vector<std::shared_ptr<task::Factory> > factories = {std::move(opcua_factory)};
    std::unique_ptr<task::Factory> factory = std::make_unique<task::MultiFactory>(
        std::move(factories)
    );

    d = std::make_unique<driver::Driver>(
        rack,
        client,
        std::move(factory),
        breaker_config
    );
    signal(SIGINT, [](int _) {
        LOG(INFO) << "[Driver] received interrupt signal. shutting down";
        d->stop();
    });
    d->run();
    LOG(INFO) << "[Driver] shutdown complete";
    return 0;
}
