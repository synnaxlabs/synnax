// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/pipeline/base.h"

namespace pipeline {
/// @brief an object that writes data to an acquisition computer or other resource.
class Sink {
public:
    /// @brief writes the given frame to the sink, returning an error if one occurs.
    /// If the sink returns an error matching driver::TEMPORARY_HARDWARE_ERROR, the
    /// acquisition pipeline will trigger a breaker (temporary backoff), and then
    /// retry the read operation. Any other error type will be considered a
    /// permanent error and the pipeline will exit.
    virtual xerrors::Error write(const synnax::Frame &frame) = 0;

    /// @brief communicates an error encountered by the control pipeline that
    /// occurred during shut down or occurred during a commanded shutdown.
    ///
    /// After this method is called, the pipeline will NOT make nay further calls to
    /// the source (read, stopped_with_err) until the pipeline is restarted.
    ///
    /// This method may be called even if stop() was called on the pipeline.
    virtual void stopped_with_err(const xerrors::Error &_) {}

    virtual ~Sink() = default;
};

/// @brief an interface that receives data over the network (from Synnax during
/// production, and from mock objects during testing).
class Streamer {
public:
    /// @brief blocks until the next frame of telemetry is available. If an error
    /// is encountered, the streamer should return an error. If the streamer returns
    /// an error matching driver::TEMPORARY_HARDWARE_ERROR, the control pipeline
    /// will trigger a breaker (temporary backoff), and then retry the read
    /// operation. Any other error type will be considered a permanent error and the
    /// pipeline will exit.
    virtual std::pair<synnax::Frame, xerrors::Error> read() = 0;

    /// @brief closes the streamer, returning any error that occurred during normal
    /// operation. If the returned error is of type freighter::UNREACHABLE, the
    /// control pipeline will trigger a breaker (temporary backoff), and then retry
    ///  until the configured number of maximum retries is exceeded. Any other error
    ///  will
    /// be considered permanent and the pipeline will exit.
    virtual xerrors::Error close() = 0;

    /// @brief signals the streamer that the caller is done sending requests, and
    /// that it should being the streamer shutdown process. This mechanism should be
    /// thread safe when concurrently used with read(). When close_send() is called,
    /// read() should return as soon as possible.
    virtual void close_send() = 0;

    virtual ~Streamer() = default;
};

/// @brief an interface for a factory that can be used to open streamers. In
/// production, this is typically backed by the Synnax client.
class StreamerFactory {
public:
    /// @brief opens a streamer with the given configuration, returning the streamer
    /// and an error if one occurs. If the error is of type freighter::UNREACHABLE,
    /// the control pipeline will trigger a breaker (temporary backoff), and then
    /// retry until the configured number of maximum retries is exceeded. Any other
    /// error is considered permanent and the pipeline will exit.
    virtual std::pair<std::unique_ptr<Streamer>, xerrors::Error>
    open_streamer(synnax::StreamerConfig config) = 0;

    virtual ~StreamerFactory() = default;
};

/// @brief an implementation of the pipeline::Streamer interface that is backed
/// by a Synnax streamer that receives data from a cluster.
class SynnaxStreamer final : public Streamer {
    /// @brief the wrapped synnax streamer.
    synnax::Streamer internal;

public:
    /// @brief constructs a new Synnax streamer that wraps the given internal
    /// streamer.
    explicit SynnaxStreamer(synnax::Streamer internal);

    /// @brief implements pipeline::Streamer to read the next frame from the
    /// streamer.
    std::pair<synnax::Frame, xerrors::Error> read() override;

    /// @brief implements pipeline::Streamer to close the streamer.
    xerrors::Error close() override;

    /// @brief implements pipeline::Streamer to close the streamer.
    void close_send() override;
};

/// @brief an implementation of the pipeline::StreamerFactory interface that is
/// backed by an actual synnax client connected to a cluster.
class SynnaxStreamerFactory final : public StreamerFactory {
    /// @brief the Synnax client to use for opening streamers.
    const std::shared_ptr<synnax::Synnax> client;

public:
    /// @brief constructs a new Synnax streamer factory that uses the given Synnax
    /// client.
    explicit SynnaxStreamerFactory(const std::shared_ptr<synnax::Synnax> &client);

    /// @brief implements pipeline::StreamerFactory to open a Synnax streamer.
    std::pair<std::unique_ptr<pipeline::Streamer>, xerrors::Error>
    open_streamer(synnax::StreamerConfig config) override;
};

/// @brief A pipeline that reads incoming data over the network and writes to a
/// sink. The pipeline should be used as a utility for implementing a broader
/// control task. It implements retry handling on connection loss and handles
/// temporary hardware errors. The pipeline forks a thread to repeatedly read from
/// the streamer and write to the sink.
class Control final : public Base {
    /// @brief a factory used to instantiate Synnax streamers to read commands from.
    /// This is typically backed by a Synnax client, but can be mocked.
    std::shared_ptr<StreamerFactory> factory;
    /// @brief the configuration for the Synnax streamer.
    synnax::StreamerConfig config;
    /// @brief the sink that the control pipeline will write frames to.
    std::shared_ptr<Sink> sink;
    /// @brief the current open streamer reading data from the network.
    std::unique_ptr<Streamer> streamer = nullptr;

public:
    /// @brief constructs a new control pipeline that opens streamers on a Synnax
    /// database cluster.
    /// @param client the Synnax client to use for opening streamers.
    /// @param streamer_config the configuration for the Synnax streamer.
    /// @param sink the sink to write data to. See the Sink interface for more
    /// information.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// control thread lifecycle and retry requests on connection loss or temporary
    /// hardware errors.
    /// @param thread_name optional name for the pipeline thread (visible in debuggers).
    Control(
        std::shared_ptr<synnax::Synnax> client,
        synnax::StreamerConfig streamer_config,
        std::shared_ptr<Sink> sink,
        const breaker::Config &breaker_config,
        std::string thread_name = ""
    );

    //// @brief constructs a new control pipeline that opens streamers using the
    /// given
    /// streamer factory.
    /// @param streamer_factory the streamer factory to use for opening streamers.
    /// @param streamer_config the configuration for opening the streamer.
    /// @param sink the sink to write data to. See the Sink interface for more
    /// information.
    /// @param breaker_config the configuration for the breaker used to manage the
    /// control thread lifecycle and retry requests on connection loss or temporary
    /// hardware errors.
    /// @param thread_name optional name for the pipeline thread (visible in debuggers).
    Control(
        std::shared_ptr<StreamerFactory> streamer_factory,
        synnax::StreamerConfig streamer_config,
        std::shared_ptr<Sink> sink,
        const breaker::Config &breaker_config,
        std::string thread_name = ""
    );

    /// @brief stops the control pipeline, blocking until the control thread has
    /// exited. If the pipeline has already stopped, this method will return
    /// immediately.
    bool stop() override;

    void run() override;
};
}
