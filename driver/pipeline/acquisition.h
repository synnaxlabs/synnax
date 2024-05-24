// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <thread>
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"
#include "driver/task/task.h"

namespace pipeline {
class Source {
public:
    virtual ~Source() = default;
    virtual std::pair<Frame, freighter::Error> read() = 0;
    virtual freighter::Error start() = 0;
    virtual freighter::Error stop() = 0;
};

/// @brief A pipeline that reads from a source and writes it's data to Synnax. The pipeline is intentionally designed
/// to handle transient hardware and network failures by re-trying operations at a scaled interval.
class Acquisition {
public:
    /// @brief starts the acquisition pipeline, returning immediately.
    void start();

    /// @brief stops the acquisition pipeline, blocking until the pipeline has stopped.
    void stop();

    Acquisition() = default;

    Acquisition(
        std::shared_ptr<task::Context> ctx,
        WriterConfig writer_config,
        std::shared_ptr<Source> source,
        const breaker::Config &breaker_config
    );

    ~Acquisition() {
        breaker.close();
    }

    Acquisition(const Acquisition &copy);
    Acquisition(Acquisition &&move);
    Acquisition& operator=(const Acquisition& copy);
private:
    /// @brief context for issuing state updates to the task.
    std::shared_ptr<task::Context> ctx;

    /// @brief tracks whether the acquisition thread is running.
    volatile bool running = false;
    std::unique_ptr<std::thread> thread;

    /// @brief configuration for the Synnax writer.
    WriterConfig writer_config;

    /// @brief daq interface
    std::shared_ptr<Source> source;

    /// @brief breaker
    breaker::Breaker breaker;

    void run();
};
}