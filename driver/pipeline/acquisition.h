// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/pipeline/base.h"

namespace pipeline {
/// @brief an object that reads data from an acquisition computer or another source,
/// returning data as frames.
class Source {
public:
    /// @brief reads the next frame from the source, returning an error if
    /// encountered. The source is in charge of regulating the rate at which frames
    /// are read. It should block using sleep or a similar mechanism. If the source
    /// returns an error matching driver::TEMPORARY_HARDWARE_ERROR, the acquisition
    /// pipeline will trigger a breaker (temporary backoff), and then retry the read
    /// operation. Any other error type will be considered a permanent error and the
    /// pipeline will exit. It's recommended that the caller return a sub-error of
    /// driver::CRITICAL_HARDWARE_ERROR for any error that is not recoverable, as
    /// this improved traceability.
    [[nodiscard]] virtual xerrors::Error
    read(breaker::Breaker &breaker, telem::Frame &data) = 0;

    /// @brief communicates an error encountered by the acquisition pipeline that
    /// caused it to shut down or occurred during commanded shutdown. Note that this
    /// method will be called when the pipeline is stopped due to a call to read()
    /// that returns an error.
    ///
    /// After this method is called, the pipeline will NOT make any further calls to
    /// the source (read, stopped_with_err) until the pipeline is restarted.
    ///
    /// This method may be called even if stop() was called on the pipeline.
    virtual void stopped_with_err(const xerrors::Error &err) {}

    virtual ~Source() = default;
};

/// @brief an interface that writes acquired data over the network (to Synnax during
/// production, and to mock objects during testing).
class Writer {
public:
    /// @brief writes the given frame of telemetry to the writer. Returns a non-nil
    /// error if the write fails, at which point the acquisition pipeline will
    /// close the writer and conditionally trigger a retry (see the close method).
    [[nodiscard]] virtual xerrors::Error write(const telem::Frame &fr) = 0;

    /// @brief closes the writer, returning any error that occurred during normal
    /// operation. If the returned error is of type freighter::UNREACHABLE, the
    /// acquisition pipeline will trigger a breaker (temporary backoff), and then
    /// retry until the configured number of maximum retries is exceeded. Any other
    /// error will be considered permanent and the pipeline will exit.
    [[nodiscard]] virtual xerrors::Error close() = 0;

    virtual ~Writer() = default;
};

/// @brief an interface for a factory that can be used to open writers. In
/// production, this is typically backed by the Synnax client.
class WriterFactory {
public:
    /// @brief opens the writer with the given configuration. If the writer cannot
    /// be opened, the method should return an error. If the error is of type
    /// freighter::UNREACHABLE, a breaker will be triggered (temporary backoff), and
    /// the acquisition pipeline will retry the operation until the configured
    /// number of maximum retries is exceeded. Any other error will be considered
    /// permanent and the pipeline will exit.
    virtual std::pair<std::unique_ptr<Writer>, xerrors::Error>
    open_writer(const synnax::WriterConfig &config) = 0;

    virtual ~WriterFactory() = default;
};

/// @brief an implementation of the pipeline::Writer interface that is backed
/// by a Synnax writer that writes data to a cluster.
class SynnaxWriter final : public pipeline::Writer {
    /// @brief the internal Synnax writer that this writer wraps.
    synnax::Writer internal;

public:
    explicit SynnaxWriter(synnax::Writer internal);

    /// @brief implements pipeline::Writer to write the frame to Synnax.
    [[nodiscard]] xerrors::Error write(const telem::Frame &fr) override;

    /// @brief implements pipeline::Writer to close the writer.
    [[nodiscard]] xerrors::Error close() override;
};

/// @brief an implementation of the pipeline::WriterFactory interface that is
/// backed by an actual synnax client connected to a cluster.
class SynnaxWriterFactory final : public WriterFactory {
    /// @brief the Synnax client to use for opening writers.
    std::shared_ptr<synnax::Synnax> client;

public:
    explicit SynnaxWriterFactory(std::shared_ptr<synnax::Synnax> client);

    /// @brief implements pipeline::WriterFactory to open a Synnax writer.
    [[nodiscard]] std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error>
    open_writer(const synnax::WriterConfig &config) override;
};

/// @brief A pipeline that reads from a source and writes it's data to Synnax. The
/// pipeline should be used as a utility for implementing a broader acquisition
/// task. It implements retry handling on connection loss and temporary hardware
/// errors. The pipeline forks a thread to repeatedly read from the source and write
/// to Synnax.
class Acquisition final : public Base {
    /// @brief a factory used to instantiate Synnax writers to write acquired data
    /// to. This is typically backed by a synnax client, but can be mocked.
    const std::shared_ptr<WriterFactory> factory;
    /// @brief the source that the acquisition pipeline reads from in order to push
    /// new frames to synnax.
    const std::shared_ptr<Source> source;
    /// @brief the configuration for the Synnax writer.
    synnax::WriterConfig writer_config;

    /// @brief the run function passed to the pipeline thread. Automatically catches
    /// standard exceptions to ensure the pipeline does not cause the application to
    /// crash.
    void run() override;

public:
    /// @brief construct an acquisition pipeline that opens writers on a Synnax
    /// database cluster.
    /// @param client the Synnax client to use for writing data.
    /// @param writer_config the configuration for the Synnax writer. This
    /// configuration will have its start time set to the first timestamp read from
    /// the source. The pipeline will also set err_on_unauthorized to true so that
    /// multiple acquisition pipelines cannot write to the same channels at once.
    /// @param source the source to read data from. See the Source interface for
    /// more details on how to correctly implement a source.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// acquisition thread lifecycle and retry requests on connection loss or
    /// temporary hardware errors.
    Acquisition(
        std::shared_ptr<synnax::Synnax> client,
        synnax::WriterConfig writer_config,
        std::shared_ptr<Source> source,
        const breaker::Config &breaker_config
    );

    /// @brief construct an acquisition pipeline that opens writers using a writer
    /// factory.
    /// @param factory the writer factory to use for opening writers.
    /// @param writer_config the configuration for the Synnax writer. This
    /// configuration will have its start time set to the first timestamp read from
    /// the source. The pipeline will also set err_on_unauthorized to true so that
    /// multiple acquisition pipelines cannot write to the same channels at once.
    /// @param source the source to read data from. See the Source interface for
    /// more details on how to correctly implement a source.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// acquisition thread lifecycle and retry requests on connection loss or
    /// temporary
    Acquisition(
        std::shared_ptr<WriterFactory> factory,
        synnax::WriterConfig writer_config,
        std::shared_ptr<Source> source,
        const breaker::Config &breaker_config
    );
};
}
