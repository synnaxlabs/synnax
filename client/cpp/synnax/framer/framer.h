// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Std
#include <vector>
#include <utility>
#include <memory>

/// Freighter
#include "freighter/freighter.h"

/// Api Protos
#include "v1/framer.pb.h"

/// Internal
#include "synnax/telem/telem.h"
#include "synnax/telem/series.h"
#include "synnax/telem/control.h"
#include "synnax/channel/channel.h"


using namespace synnax;

namespace synnax {

/// @brief type alias for streamer network transport stream.
typedef freighter::Stream<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest
> StreamerStream;

/// @brief typ;e alias for frame writer network transport.
typedef freighter::StreamClient<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest
> StreamerClient;

/// @brief type alias for writer network transports stream.
typedef freighter::Stream<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest
> WriterStream;

/// @brief type alias for writer network transport.
typedef freighter::StreamClient<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest
> WriterClient;


/// @brief Frame type.
class Frame {
public:
    std::unique_ptr<std::vector<ChannelKey>> columns;
    std::unique_ptr<std::vector<synnax::Series>> series;

    Frame(
            std::unique_ptr<std::vector<ChannelKey>> columns,
            std::unique_ptr<std::vector<synnax::Series>> series
    );

    explicit Frame(size_t size);

    explicit Frame(const api::v1::Frame &f);

    void to_proto(api::v1::Frame *f) const;

    void push_back(ChannelKey col, synnax::Series ser);

    [[nodiscard]] size_t size() const { return series->size(); }
};

class StreamerConfig {
public:
    std::vector<ChannelKey> channels;
    synnax::TimeStamp start;
private:
    void to_proto(api::v1::FrameStreamerRequest *f) const;

    friend class FrameClient;
};

/// @brief used to stream frames of telemetry from a set of channels in real-time.
/// Streamer cannot be constructed directly, and should instead be opened using the
/// FrameClient.
/// @see FrameClient
///
/// @note read() and setChannels() can be called concurrently with one another, but they
/// cannot be called concurrently with close() or with themselves.
class Streamer {
public:
    /// @brief blocks until the next frame is received from the Synnax cluster. This
    /// frame is not guaranteed to contain series for all channels specified when
    /// opening the streamer, but it is guaranteed to contain data for at least one
    /// channel and not contain data for any channels not specified.
    /// @returns the next frame of telemetry received from the Synnax cluster and an error.
    /// If error.ok() is false, then the streamer has failed and must be closed.
    /// @note read is not safe to call concurrently with itself or with close(), but it
    /// is safe to call concurrently with setChannels().
    std::pair<Frame, freighter::Error> read();

    /// @brief sets the channels to stream from the Synnax cluster, replacing any
    /// channels set during construction or a previous call to setChannels().
    /// @returns an error. If error.ok() is false, then the streamer has failed and
    /// must be closed.
    /// @param channels - the channels to stream.
    /// @note setChannels is not safe to call concurrently with itself or with close(),
    /// but it is safe to call concurrently with read().
    freighter::Error setChannels(std::vector<ChannelKey> channels);

    /// @brief closes the streamer and releases any resources associated with it. If any
    /// errors occurred during the stream, they will be returned. A streamer MUST be
    /// closed after use, or the caller risks leaking resources. Calling any method
    /// on a closed streamer will throw a runtime_error.
    /// @returns an error. error.ok() will be false if the streamer accumulated an error
    /// during operation.
    /// @note close() is not safe to call concurrently with itself or any other streamer
    /// methods.
    freighter::Error close();

private:
    Streamer() = default;

    /// @brief true if the streamer has been closed.
    bool closed = false;

    /// @brief true if an error has occurred in the streamer.
    void assert_open() const;

    /// @brief constructs the streamer from a configured stream and moves ownership.
    explicit Streamer(std::unique_ptr<StreamerStream> stream);

    /// @brief the stream transport for the streamer.
    std::unique_ptr<StreamerStream> stream;

    /// @brief the only class that can construct a streamer.
    friend class FrameClient;
};

/// @brief configuration for opening a new Writer. For more information on writers,
/// see https://docs.synnaxlabs.com/concepts/write-domains.
class WriterConfig {
public:
    /// @brief The channels to write to.
    std::vector<ChannelKey> channels;

    /// @brief sets the starting timestamp for the first sample in the writer. If
    /// this timestamp overlaps with existing data for ANY of the provided channels,
    /// the writer will fail to open.
    synnax::TimeStamp start;

    /// @brief The control authority to set for each channel. If this vector is of
    /// length 1, then the same authority is set for all channels. Otherwise, the
    /// vector must be the same length as the channels vector. If this vector
    /// is empty, then all writes are executed with absolute authority.
    std::vector<synnax::Authority> authorities;

    /// @brief sets identifying information for the writer. The subject's key and name
    /// will be used to identify the writer in control transfer scenarios.
    synnax::Subject subject;
private:
    /// @brief binds the configuration fields to it's protobuf representation.
    void to_proto(api::v1::FrameWriterConfig *f) const;

    friend class FrameClient;
};


/// @brief used to write a new domain of telemetry frames to a set of channels in time
/// order. Writer cannot be constructed directly, and should instead be opened using
/// the FrameClient.
///
/// @note The writer uses a streaming protocol heavily optimized for performance. This comes
/// at the cost of higher complexity.
///
/// @note The writer is not safe for concurrent use.
class Writer {
public:
    /// @brief writes the given frame of telemetry to the Synnax cluster.
    /// @param fr the frame to write. This frame must adhere to a set of constraints:
    ///
    /// 1. The frame must have at most 1 series per channel.
    /// 2. The frame may not have series for any channel not specified in the
    ///  WriterConfig when opening the writer.
    /// 3. All series' that are written to the same index must have the same number of
    /// samples.
    /// 4. When writing to an index, the series' for the index must have monotonically
    /// increasing int64 unix epoch timestamps.
    ///
    /// For more information, see https://docs.synnaxlabs.com/concepts/write-domains.
    ///
    /// @returns false if an error occurred in the write pipeline. After an error occurs,
    /// the caller must acknowledge the error by calling error() or close() on the writer.
    bool write(Frame fr);

    /// @brief commits all pending writes to the Synnax cluster. Commit can be called
    /// multiple times, committing any new writes made since the last commit.
    ///
    /// @returns false if the commit failed. After a commit fails, the caller must
    /// acknowledge the error by calling error() or close() on the writer.
    std::pair<synnax::TimeStamp, bool> commit();

    /// @brief returns any error accumulated during the write process. If no err has
    /// occurred, err.ok() will be true.
    freighter::Error error();

    /// @brief closes the writer and releases any resources associated with it. A writer
    /// MUST be closed after use, or the caller risks leaking resources. Calling any
    /// method on a closed writer will throw a runtime_error.
    freighter::Error close();

private:
    /// @brief whether an error has occurred in the write pipeline.
    bool err_accumulated = false;
    /// @brief if close() has been called on the writer.e
    bool closed = false;

    Writer() = default;

    /// @brief the stream transport for the writer.
    std::unique_ptr<WriterStream> stream{};

    /// @brief opens a writer to the Synnax cluster.
    explicit Writer(std::unique_ptr<WriterStream> s);

    /// @brief throws a runtime error if the writer is closed.
    void assert_open() const;

    friend class FrameClient;
};

class FrameClient {
private:
    StreamerClient *streamer_client;
    WriterClient *writer_client;
public:
    FrameClient(StreamerClient *streamer_client, WriterClient *writer_client) :
            streamer_client(streamer_client),
            writer_client(writer_client) {}


    std::pair<Writer, freighter::Error> openWriter(const WriterConfig &config);

    std::pair<Streamer, freighter::Error> openStreamer(const StreamerConfig &config);
};
}
