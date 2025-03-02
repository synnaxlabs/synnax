// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/breaker/breaker.h"

namespace pipeline {
class Base {
    /// @brief the primary thread that runs the control pipeline.
    std::thread thread;

    /// @brief an internal run
    void run_internal() {
        try {
            this->run();
        } catch (const std::exception &e) {
            LOG(ERROR) << "[control] Unhandled standard exception: " << e.what();
        } catch (...) {
            LOG(ERROR) << "[control] Unhandled unknown exception";
        }
        breaker.mark_stopped();
    }

protected:
    /// @brief a breaker for managing the lifecycle of the pipeline thread.
    breaker::Breaker breaker;

    explicit Base(const breaker::Config &breaker_config) : breaker(breaker_config) {
    }

public:
    virtual ~Base() = default;

    /// @brief the main run loop for the pipeline.
    virtual void run() = 0;

    /// @brief starts the control pipeline if it has not already been started. Start is
    /// idempotent, and is safe to call multiple times without stopping the pipeline.
    virtual bool start() {
        if (!this->breaker.start()) return false;
        this->thread = std::thread(&Base::run_internal, this);
        return true;
    }

    /// @brief stops the control pipeline if it has not already been stopped. Stop is
    /// idempotent, and is safe to call multiple times without starting the pipeline
    /// again.
    virtual bool stop() {
        auto stopped = this->breaker.stop();
        if (this->thread.joinable()) this->thread.join();
        return stopped;
    }

    /// @brief returns true if the pipeline is currently running. This method may return
    /// true if the pipeline is in a transient state i.e. start has been called but the
    /// pipeline has not started or failed yet or if stop has been called but the pipeline
    /// has not stopped yet.
    bool running() const { return this->breaker.running(); }
};
}
