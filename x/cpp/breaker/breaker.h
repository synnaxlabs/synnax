// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cassert>
#include <condition_variable>
#include <iomanip>
#include <memory>

#include "glog/logging.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::breaker {
/// @brief tells the breaker to retry infinitely.
constexpr int RETRY_INFINITELY = -1;

/// @brief struct for configuring a breaker.
struct Config {
    /// @brief the name of the breaker.
    std::string name;
    /// @brief the interval that will be used by the breaker on the first trigger.
    /// This interval will be scaled on each successive retry based on the value of
    /// scale.
    telem::TimeSpan base_interval = 1 * telem::SECOND;
    /// @brief sets the maximum number of retries before the wait() method returns
    /// false.
    int max_retries = 50;
    /// @brief sets the rate at which the base_interval will scale on each
    /// successive call to wait(). We do not recommend setting this factor lower
    /// than 1.
    float scale = 1.1f;
    /// @brief the maximum amount of time to wait for a retry.
    telem::TimeSpan max_interval = 1 * telem::MINUTE;

    [[nodiscard]] Config child(const std::string &name) const {
        return Config{
            this->name + "." + name,
            base_interval,
            max_retries,
            scale,
            max_interval
        };
    }
};

/// @brief implements a general purpose circuit breaker that allows for retry at a
/// scaled interval, with a set number of maximum retries before giving up.
/// @see breaker::Config for information on configuring the breaker.
class Breaker {
    /// @brief configuration parameters for the breaker.
    Config config;
    /// @brief current retry interval.
    telem::TimeSpan interval;
    /// @brief the current number of retries.
    size_t retries;
    /// @brief a flag to indicate if the breaker is currently running.
    std::atomic<bool> is_running;
    /// @brief a condition variable used to notify the breaker to shut down
    /// immediately.
    std::condition_variable shutdown_cv;
    /// @brief used to protect the condition variable.
    std::mutex mu;

public:
    explicit Breaker(const Config &config):
        config(config), interval(config.base_interval), retries(0), is_running(false) {}

    Breaker():
        Breaker(
            Config{
                "default",
                telem::TimeSpan(1 * telem::SECOND),
                10,
                1.1f,
                telem::TimeSpan(1 * telem::MINUTE)
            }
        ) {}

    ~Breaker() {
        if (!this->running()) return;
        // Very important that we do not use GLOG here, as it can cause problems
        // in destructors.
        std::cerr << "breaker " << this->config.name
                  << " was not stopped before destruction" << std::endl;
        assert(false && "breaker was not stopped before destruction");
    }

    /// @brief marks the breaker as stopped.
    bool mark_stopped() { return this->is_running.exchange(false); }

    /// @brief triggers the breaker. If the maximum number of retries has been
    /// exceeded, immediately returns false. Otherwise, sleeps the current thread
    /// for the current retry interval and returns true. Also Logs information about
    /// the breaker trigger.
    bool wait() { return wait(""); }

    /// @brief triggers the breaker and logs the provided error as its message.
    /// @see wait() for more information.
    bool wait(const errors::Error &err) { return this->wait(err.message()); }

    /// @brief triggers the breaker. If the maximum number of retries has been
    /// exceeded, immediately returns false. Otherwise, sleeps the current thread
    /// for the current retry interval and returns true.
    /// @param message a message to inject additional information into the logs
    /// about what error occurred to trigger the breaker.
    bool wait(const std::string &message) {
        if (!this->running()) {
            LOG(ERROR) << "[" << this->config.name << "] breaker not started. Exiting.";
            return false;
        }
        this->retries++;
        if (this->config.max_retries != -1 &&
            this->retries > static_cast<size_t>(this->config.max_retries)) {
            LOG(ERROR) << "[" << this->config.name
                       << "] exceeded the maximum retry count of "
                       << this->config.max_retries << ". Exiting."
                       << "Error: " << message << ".";
            reset();
            return false;
        }

        const std::string retry_count_msg = this->config.max_retries == -1
                                              ? std::to_string(this->retries) + "/∞"
                                              : std::to_string(this->retries) + "/" +
                                                    std::to_string(
                                                        this->config.max_retries
                                                    );

        LOG(ERROR) << "[" << this->config.name << "] failed " << retry_count_msg
                   << " times. " << "Retrying in " << std::fixed << std::setprecision(1)
                   << this->interval.seconds() << " seconds. " << "Error: " << message;
        std::unique_lock lock(this->mu);
        shutdown_cv.wait_for(lock, this->interval.chrono());
        if (!this->running()) {
            LOG(INFO) << "[" << this->config.name << "] is shutting down. Exiting.";
            reset();
            return false;
        }
        this->interval = this->interval * this->config.scale;
        if (this->interval > this->config.max_interval)
            this->interval = this->config.max_interval;
        return true;
    }

    /// @brief waits for the given time duration. If the breaker stopped before the
    /// specified time, the method will return immediately to ensure graceful exit
    /// of objects using the breaker.
    /// @param time the time to wait (supports multiple time units).
    void wait_for(const telem::TimeSpan &time) { this->wait_for(time.chrono()); }

    /// @brief waits for the given time duration. If the breaker stopped before the
    /// specified time, the method will return immediately to ensure graceful exit
    /// of objects using the breaker.
    /// @note that this implementation is not performance efficient as it relies on
    /// a condition variable to wake up the thread. It is recommended for longer
    /// sleeps where the breaker may need to be interrupted for shut down.
    /// @param time the time to wait for in nanoseconds.
    void wait_for(const std::chrono::nanoseconds &time) {
        if (!this->running()) return;
        std::unique_lock lock(this->mu);
        this->shutdown_cv.wait_for(lock, time);
    }

    /// @brief starts the breaker, using it as a signaling mechanism for a thread to
    /// operate. A breaker that is started must be stopped before it is destroyed.
    /// @throws std::runtime_error inside the destructor if hte breaker is not
    /// stopped.
    /// @returns true if the breaker was not already started, and false if it was.
    bool start() { return !this->is_running.exchange(true); }

    /// @brief shuts down the breaker, preventing any further retries.
    /// @returns true if the breaker was running and is now stopped, and false if it
    /// was already stopped.
    bool stop() {
        if (!this->mark_stopped()) return false;
        std::lock_guard lock(this->mu);
        this->shutdown_cv.notify_all();
        return true;
    }

    /// @brief returns the current retry cont of the breaker, which is the number of
    /// times wait() has been called. Note that accessing this field is not
    /// thread-safe, and should only be treated as a rough estimate of the number of
    /// retries.
    [[nodiscard]]
    size_t retry_count() const {
        return this->retries;
    }

    /// @brief returns true if the breaker is currently running (i.e. start() has
    /// been called, but stop() has not been called yet.
    [[nodiscard]] bool running() const { return this->is_running; }

    /// @brief resets the retry count and the retry interval on the breaker,
    /// allowing it to be re-used. It's typically to call this method after the
    /// breaker has been triggered, but the request has succeeded.
    void reset() {
        this->retries = 0;
        this->interval = this->config.base_interval;
    }
};

inline Config default_config(const std::string &name) {
    return Config{.name = name};
}
}
