// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <thread>
#include <glog/logging.h>

#include "synnax/synnax.h"

namespace breaker {

struct Config {
    std::string name;
    synnax::TimeSpan base_interval;
    uint32_t max_retries;
    float_t scale;
};


class Breaker {
public:
    Breaker(const Config& config) : config(config), interval(config.base_interval), retries(0) {}

    bool wait() {
        if (retries >= config.max_retries) {
            LOG(WARNING) << "Breaker " << config.name << " exceeded the maximum retry count of " << config.max_retries << ". Exiting.";
            return false;
        }
        LOG(WARNING) << "Breaker " << config.name << " triggered " << retries << "/" << config.max_retries << " times. Retrying in " << interval / synnax::SECOND << " seconds.";
        std::this_thread::sleep_for(std::chrono::nanoseconds(interval.value));
        interval = interval * config.scale;
        retries++;
        return true;
    }

    void reset() {
        retries = 0;
        interval = config.base_interval;
    }
private:
    Config config;
    synnax::TimeSpan interval;
    uint32_t retries;
};
}

