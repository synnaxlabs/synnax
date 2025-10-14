// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <thread>

#include "x/cpp/breaker/breaker.h"

namespace pipeline {
class Base {
    /// @brief the primary thread that runs the pipeline.
    std::thread thread;

    /// @brief an internal run
    void run_internal() {
        try {
            this->run();
        } catch (const std::exception &e) {
            LOG(ERROR) << "[control] Unhandled standard exception: " << e.what();
        } catch (...) { LOG(ERROR) << "[control] Unhandled unknown exception"; }
    }

protected:
    /// @brief a breaker for managing the lifecycle of the pipeline thread.
    breaker::Breaker breaker;

    explicit Base(const breaker::Config &breaker_config): breaker(breaker_config) {}

public:
    virtual ~Base() = default;

    /// @brief the main run loop for the pipeline.
    virtual void run() = 0;

    /// @brief starts the control pipeline. This method is idempotent.
    /// @returns true if this is the first call to start() ever or the first call to
    /// start() since the pipeline was last stopped.
    /// @returns false otherwise.
    virtual bool start() {
        if (!this->breaker.start()) return false;
        this->thread = std::thread(&Base::run_internal, this);
        return true;
    }

    /// @brief stops the pipeline. This method is idempotent.
    /// @returns true if this is the first call to stop() since the last call to
    /// start().
    /// @returns false if this is an N+1 call to stop() since the last call to
    /// start().
    /// @details this function is safe to call from within the pipeline operation
    /// thread itself. If done so, the pipeline breaker will be stopped, but the
    /// thread will not be joined. If calling stop() from within the pipeline
    /// itself, it's important that stop() be called again before the pipeline is
    /// destructed to properly join the thread.
    virtual bool stop() {
        const auto stopped = this->breaker.stop();
        if (this->thread.get_id() != std::this_thread::get_id() &&
            this->thread.joinable())
            this->thread.join();
        return stopped;
    }

    /// @brief returns true if the pipeline is currently running. This method may
    /// return true if the pipeline is in a transient state, i.e., start has been
    /// called, but the pipeline has not started or failed yet or if stop has been
    /// called, but the pipeline has not stopped yet.
    bool running() const { return this->breaker.running(); }
};
}
