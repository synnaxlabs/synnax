// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <thread>
#include <mutex>
#include <condition_variable>

#include "glog/logging.h"
#include "client/cpp/synnax.h"
#include "freighter/cpp/freighter.h"


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

    [[nodiscard]] Config child(const std::string &name) const {
        return Config{this->name + "." + name, base_interval, max_retries, scale};
    }
};


/// @brief implements a general purpose circuit breaker that allows for retry at a
/// scaled interval, with a set number of maximum retries before giving up.
/// @see breaker::Config for information on configuring the breaker.
class Breaker {
public:
    explicit Breaker(
        const Config &config
    ) : config(config),
        interval(config.base_interval),
        retries(0),
        is_running(false),
        breaker_shutdown(std::make_unique<std::condition_variable>()) {
    }

    Breaker(): Breaker(Config{
        "default",
        TimeSpan(1 * SECOND),
        10,
        1.1
    }) {
    }

    Breaker(
        const Breaker &other
    ) noexcept: config(other.config),
                interval(other.interval),
                retries(other.retries),
                is_running(other.is_running),
                breaker_shutdown(std::make_unique<std::condition_variable>()) {
        std::cout << "copy constructor called" << std::endl;
    }

    Breaker(Breaker &&other) noexcept : config(other.config),
                                        interval(other.interval),
                                        retries(other.retries),
                                        is_running(other.is_running),
                                        breaker_shutdown(
                                            std::make_unique<
                                                std::condition_variable>()) {
        std::cout << "move constructor called" << std::endl;
    }

    Breaker &operator=(const Breaker &other) noexcept {
        if (this == &other) return *this;
        this->config = other.config;
        this->interval = other.interval;
        this->retries = other.retries;
        this->is_running = other.is_running;
        this->breaker_shutdown = std::make_unique<std::condition_variable>();
        return *this;
    }

    ~Breaker() {
        stop();
        // sleep to allow for the breaker to shutdown.
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }


    /// @brief triggers the breaker. If the maximum number of retries has been exceeded,
    /// immediately returns false. Otherwise, sleeps the current thread for the current
    /// retry interval and returns true. Also Logs information about the breaker trigger.
    bool wait() { return wait(""); }

    bool wait(const freighter::Error &err) { return wait(err.message()); }

    /// @brief triggers the breaker. If the maximum number of retries has been exceeded,
    /// immediately returns false. Otherwise, sleeps the current thread for the current
    /// retry interval and returns true.
    /// @param message a message to inject additional information into the logs about what
    /// error occurred to trigger the breaker.
    bool wait(const std::string &message) {
        if (!running()) {
            LOG(ERROR) << "[" << config.name << "] breaker not started. Exiting.";
            return false;
        }
        retries++;
        if (retries > config.max_retries) {
            LOG(ERROR) << "[" << config.name << "] exceeded the maximum retry count of "
                    << config.max_retries << ". Exiting." << "Error: " << message <<
                    ".";
            reset();
            return false;
        }
        LOG(ERROR) << "[" << config.name << "] failed " << retries << "/" << config.
                max_retries
                << " times. " << "Retrying in " << interval / SECOND << " seconds. " <<
                "Error: " << message << "."; {
            std::unique_lock lock(shutdown_mutex);
            breaker_shutdown->wait_for(lock, interval.chrono());
            if (!running()) {
                LOG(INFO) << "[" << config.name << "] is shutting down. Exiting.";
                reset();
                return false;
            }
        }
        interval = interval * config.scale;
        return true;
    }

    void waitFor(const TimeSpan &time) { this->waitFor(time.chrono()); }

    void waitFor(const std::chrono::nanoseconds &time) {
        if (!running()) return;
        std::unique_lock lock(shutdown_mutex);
        breaker_shutdown->wait_for(lock, time);
    }

    void start() {
        if (running()) return;
        is_running = true;
    }

    /// @brief shuts down the breaker, preventing any further retries.
    void stop() {
        if (!running()) return;
        std::lock_guard lock(shutdown_mutex);
        is_running = false;
        breaker_shutdown->notify_all();
    }

    [[nodiscard]] bool running() const { return is_running; }

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
    volatile bool is_running;
    std::unique_ptr<std::condition_variable> breaker_shutdown;
    std::mutex shutdown_mutex;
};
}
