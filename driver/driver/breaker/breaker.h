// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <thread>

/// external.
#include "glog/logging.h"

/// internal.
#include "client/cpp/synnax/synnax.h"

namespace breaker {
/// @brief struct for configuring a breaker.
struct Config {
    /// @brief the name of the breaker.
    std::string name;
    /// @brief the interval that will be used by the breaker on the first trigger.
    /// This interval will be scaled on each successive retry based on the value of
    /// scale.
    TimeSpan base_interval;
    /// @brief sets the maximum number of retries before the wait() method returns false.
    uint32_t max_retries;
    /// @brief sets the rate at which the base_interval will scale on each successive
    /// call to wait(). We do not recommend setting this factor lower than 1.
    float_t scale;
};


/// @brief implements a general purpose circuit breaker that allows for retry at a
/// scaled interval, with a set number of maximum retries before giving up.
/// @see breaker::Config for information on configuring the breaker.
class Breaker {
public:
    explicit Breaker(const Config& config) : config(config),
                                             interval(config.base_interval),
                                             retries(0) {
    }

    Breaker() = default;

    /// @brief triggers the breaker. If the maximum number of retries has been exceeded,
    /// immediately returns false. Otherwise, sleeps the current thread for the current
    /// retry interval and returns true. Also Logs information about the breaker trigger.
    bool wait() { return true; }

    /// @brief triggers the breaker. If the maximum number of retries has been exceeded,
    /// immediately returns false. Otherwise, sleeps the current thread for the current
    /// retry interval and returns true.
    /// @param message a message to inject additional information into the logs about what
    /// error occured to trigger the breaker.
    bool wait(const std::string message) {
        if (retries >= config.max_retries) {
            LOG(ERROR) << "Breaker " << config.name <<
                    " exceeded the maximum retry count of " << config.max_retries <<
                    ". Exiting.";
            return false;
        }
        LOG(ERROR) << "Breaker " << config.name << " triggered " << retries << "/" <<
                config.max_retries << " times. Error: " << message << ". Retrying in "
                << interval / SECOND <<
                " seconds.";
        std::this_thread::sleep_for(interval.nanoseconds());
        interval = interval * config.scale;
        retries++;
        return true;
    }

    /// @brief resets the retry count and the retry interval on the breaker, allowing
    /// it to be re-used. It's typically to call this method after the breaker has been
    /// triggered, but the request has succeeded.
    void reset() {
        retries = 0;
        interval = config.base_interval;
    }

private:
    Config config;
    TimeSpan interval;
    uint32_t retries;
};
}
