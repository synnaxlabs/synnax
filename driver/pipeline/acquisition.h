// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <thread>

/// external
#include "client/cpp/synnax.h"

/// module
#include "x/cpp/breaker/breaker.h"
#include "driver/pipeline/middleware.h"

namespace pipeline {
/// @brief an object that reads data from an acquisition computer or another source,
/// returning data as frames.
class Source {
public:
    /// @brief reads the next frame from the source, returning an error if encountered.
    /// The source is in charge of regulating the rate at which frames are read. It
    /// should block using sleep or a similar mechanism. If the source returns an
    /// error matching driver::TEMPORARY_HARDWARE_ERROR, the acquisition pipeline will
    /// trigger a breaker (temporary backoff), and then retry the read operation. Any
    /// other error type will be considered a permanent error and the pipeline will
    /// exit. It's recommended that the caller return a sub-error of
    /// driver::CRITICAL_HARDWARE_ERROR for any error that is not recoverable, as this
    /// improved traceability.
    virtual std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) = 0;

    /// @brief communicates an error encountered by the acquisition pipeline that caused
    /// it to shut down or occurred during commanded shutdown. Note that this method
    /// will be called when the pipeline is stopped due to a call to read() that returns
    /// an error.
    ///
    /// After this method is called, the pipeline will NOT make any further calls to the
    /// source (read, stopped_with_err) until the pipeline is restarted.
    ///
    /// This method may be called even if stop() was called on the pipeline.
    virtual void stopped_with_err(const xerrors::Error &err) {
    }

    virtual ~Source() = default;
};

/// @brief an interface that writes acquired data over the network (to Synnax during
/// production, and to mock objects during testing).
class Writer {
public:
    /// @brief writes the given frame of telemetry to the writer. If the write fails
    /// or the writer accumulates an error, the writer should return false. When false
    /// is returned, the acquisition pipeline will close the writer and conditionally
    /// trigger a retry (see the close method).
    virtual bool write(synnax::Frame &fr) = 0;

    /// @brief closes the writer, returning any error that occurred during normal
    /// operation. If the returned error is of type freighter::UNREACHABLE, the
    /// acquisition pipeline will trigger a breaker (temporary backoff), and then retry
    /// until the configured number of maximum retries is exceeded. Any other error will
    /// be considered permanent and the pipeline will exit.
    virtual xerrors::Error close() = 0;

    virtual ~Writer() = default;
};

/// @brief an interface for a factory that can be used to open writers. In production,
/// this is typically backed by the Synnax client.
class WriterFactory {
public:
    /// @brief opens the writer with the given configuration. If the writer cannot be
    /// opened, the method should return an error. If the error is of type freighter::UNREACHABLE,
    /// a breaker will be triggered (temporary backoff), and the acquisition pipeline will
    /// retry the operation until the configured number of maximum retries is exceeded. Any
    /// other error will be considered permanent and the pipeline will exit.
    virtual std::pair<std::unique_ptr<Writer>, xerrors::Error> open_writer(
        const WriterConfig &config
    ) = 0;

    virtual ~WriterFactory() = default;
};

/// @brief an implementation of the pipeline::Writer interface that is backed
/// by a Synnax writer that writes data to a cluster.
class SynnaxWriter final : public pipeline::Writer {
    std::unique_ptr<synnax::Writer> internal;

public:
    explicit SynnaxWriter(std::unique_ptr<synnax::Writer> internal);

    bool write(synnax::Frame &fr) override;

    xerrors::Error close() override;
};

/// @brief an implementation of the pipeline::WriterFactory interface that is
/// backed by an actual synnax client connected to a cluster.
class SynnaxWriterFactory final : public WriterFactory {
    std::shared_ptr<synnax::Synnax> client;

public:
    explicit SynnaxWriterFactory(std::shared_ptr<synnax::Synnax> client);

    std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error> open_writer(
        const WriterConfig &config
    ) override;
};

/// @brief A pipeline that reads from a source and writes it's data to Synnax. The pipeline
/// should be used as a utility for implementing a broader acquisition task. It implements
/// retry handling on connection loss and temporary hardware errors. The pipeline
/// forks a thread to repeatedly read from the source and write to Synnax.
class Acquisition {
public:
    Acquisition() = default;

    /// @brief construct an acquisition pipeline that opens writers on a Synnax database
    /// cluster.
    /// @param client the Synnax client to use for writing data.
    /// @param writer_config the configuration for the Synnax writer. This configuration
    /// will have its start time set to the first timestamp read from the source. The
    /// pipeline will also set err_on_unauthorized to true so that multiple acquisition
    /// pipelines cannot write to the same channels at once.
    /// @param source the source to read data from. See the Source interface for more
    /// details on how to correctly implement a source.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// acquisition thread lifecycle and retry requests on connection loss or temporary
    /// hardware errors.
    Acquisition(
        std::shared_ptr<synnax::Synnax> client,
        WriterConfig writer_config,
        std::shared_ptr<Source> source,
        const breaker::Config &breaker_config
    );

    /// @brief construct an acquisition pipeline that opens writers using a writer factory.
    /// @param factory the writer factory to use for opening writers.
    /// @param writer_config the configuration for the Synnax writer. This configuration
    /// will have its start time set to the first timestamp read from the source. The
    /// pipeline will also set err_on_unauthorized to true so that multiple acquisition
    /// pipelines cannot write to the same channels at once.
    /// @param source the source to read data from. See the Source interface for more
    /// details on how to correctly implement a source.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// acquisition thread lifecycle and retry requests on connection loss or temporary
    Acquisition(
        std::shared_ptr<WriterFactory> factory,
        WriterConfig writer_config,
        std::shared_ptr<Source> source,
        const breaker::Config &breaker_config
    );

    /// @brief starts the acquisition pipeline if it has not already been started. start
    /// is safe to call multiple times without stopping the pipeline.
    /// @retruns true if the pipeline was not previously started and was started, and
    /// false if the pipeline was already running.
    bool start();

    /// @brief stops the acquisition pipeline, blocking until the pipeline has stopped.
    /// If the pipeline has already stopped, stop will return immediately.
    /// @returns true if the pipeline was running and was stopped, and false if the
    /// if the pipeline was already stopped.
    bool stop();

    /// @brief adds a middleware to the acquisition pipeline that will be called on each
    /// frame read from source
    void add_middleware(const std::shared_ptr<pipeline::Middleware> &middleware) {
        middleware_chain.add(middleware);
    }

    ~Acquisition();

private:
    std::unique_ptr<std::thread> thread;
    std::shared_ptr<WriterFactory> factory;
    WriterConfig writer_config;
    breaker::Breaker breaker;
    std::shared_ptr<Source> source;
    pipeline::MiddlewareChain middleware_chain;

    void run_internal();

    void ensure_thread_joined() const;

    void run();
};
}
