// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <vector>
#include <utility>
#include <memory>
#include "freighter/cpp/freighter.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"
#include "client/cpp/telem/telem.h"
#include "client/cpp/telem/series.h"
#include "client/cpp/telem/control.h"
#include "client/cpp/channel/channel.h"

using namespace synnax;

namespace synnax {
/// @brief type alias for streamer network transport stream.
typedef freighter::Stream<
    api::v1::FrameStreamerRequest,
    api::v1::FrameStreamerResponse
> StreamerStream;

/// @brief typ;e alias for frame writer network transport.
typedef freighter::StreamClient<
    api::v1::FrameStreamerRequest,
    api::v1::FrameStreamerResponse
> StreamerClient;

/// @brief type alias for writer network transports stream.
typedef freighter::Stream<
    api::v1::FrameWriterRequest,
    api::v1::FrameWriterResponse
> WriterStream;

/// @brief type alias for writer network transport.
typedef freighter::StreamClient<
    api::v1::FrameWriterRequest,
    api::v1::FrameWriterResponse
> WriterClient;


/// @brief Frame type.
struct Frame {
    std::unique_ptr<std::vector<ChannelKey> > channels;
    std::unique_ptr<std::vector<synnax::Series> > series;

    Frame() = default;

    Frame(
        std::unique_ptr<std::vector<ChannelKey> > channels,
        std::unique_ptr<std::vector<synnax::Series> > series
    );

    explicit Frame(size_t size);

    explicit Frame(const api::v1::Frame &f);

    Frame(
        ChannelKey chan,
        synnax::Series ser
    );

    void toProto(api::v1::Frame *f) const;

    void add(ChannelKey col, synnax::Series ser) const;

    // implement the ostream operator
    friend std::ostream &operator<<(std::ostream &os, const Frame &f);

    template<typename NumericType>
    NumericType at(synnax::ChannelKey key, int index) const {
        for (size_t i = 0; i < channels->size(); i++) {
            if (channels->at(i) == key)
                return series->at(i).at<NumericType>(index);
        }
        throw std::runtime_error("channel not found");
    }


    [[nodiscard]] size_t size() const { return series->size(); }
};

/// @brief configuration for opening a new streamer.
class StreamerConfig {
public:
    /// @brief the channels to stream.
    std::vector<ChannelKey> channels;

    /// @brief sets the timestamp that that the streamer should forward previous reads
    /// from. On startup, the streamer will read all existing data from the provided
    /// channels starting at this timestamp. If this timestamp is zero or in the future,
    /// no samples will be read.
    synnax::TimeStamp start;

private:
    void toProto(api::v1::FrameStreamerRequest &f) const;

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
    Streamer() = default;

    /// @brief blocks until the next frame is received from the Synnax cluster. This
    /// frame is not guaranteed to contain series for all channels specified when
    /// opening the streamer, but it is guaranteed to contain data for at least one
    /// channel and not contain data for any channels not specified.
    /// @returns the next frame of telemetry received from the Synnax cluster and an error.
    /// If error.ok() is false, then the streamer has failed and must be closed.
    /// @note read is not safe to call concurrently with itself or with close(), but it
    /// is safe to call concurrently with setChannels().
    std::pair<Frame, freighter::Error> read() const;

    /// @brief sets the channels to stream from the Synnax cluster, replacing any
    /// channels set during construction or a previous call to setChannels().
    /// @returns an error. If error.ok() is false, then the streamer has failed and
    /// must be closed.
    /// @param channels - the channels to stream.
    /// @note setChannels is not safe to call concurrently with itself or with close(),
    /// but it is safe to call concurrently with read().
    freighter::Error setChannels(std::vector<ChannelKey> channels) const;

    /// @brief closes the streamer and releases any resources associated with it. If any
    /// errors occurred during the stream, they will be returned. A streamer MUST be
    /// closed after use, or the caller risks leaking resources. Calling any method
    /// on a closed streamer will throw a runtime_error.
    /// @returns an error. error.ok() will be false if the streamer accumulated an error
    /// during operation.
    /// @note close() is not safe to call concurrently with itself or any other streamer
    /// methods.
    freighter::Error close() const;

    /// @brief closes the sending end of the streamer. Subsequence calls to receive()
    /// will exhaust the stream and eventually return an EOF.
    /// @note closeSend() is safe to call concurrently with read(), but not with any
    /// other DB methods.
    void closeSend() const;

private:
    /// @brief true if the streamer has been closed.
    bool closed = false;

    /// @brief true if an error has occurred in the streamer.
    void assertOpen() const;

    /// @brief constructs the streamer from a configured stream and moves ownership.
    explicit Streamer(std::unique_ptr<StreamerStream> stream);

    /// @brief the stream transport for the streamer.
    std::unique_ptr<StreamerStream> stream;

    /// @brief the only class that can construct a streamer.
    friend class FrameClient;
};

enum WriterMode : uint8_t {
    WriterPersistStream = 1,
    WriterPersistOnly = 2,
    WriterStreamOnly = 3
};

/// @brief configuration for opening a new Writer. For more information on writers,
/// see https://docs.synnaxlabs.com/concepts/write.
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
    /// is empty, then all writes are executed with AUTH_ABSOLUTE authority.
    std::vector<synnax::Authority> authorities;

    /// @brief sets identifying information for the writer. The subject's key and name
    /// will be used to identify the writer in control transfer scenarios.
    synnax::ControlSubject subject;

    /// @brief sets whether the writer is configured to persist data, stream it, or both.
    /// Options are:
    ///     - WriterPersistStream: persist data and stream it.
    ///     - WriterPersistOnly: persist data only.
    ///     - WriterStreamOnly: stream data only.
    WriterMode mode;

    /// @brief sets whether auto commit is enabled for the writer. If true, samples will
    /// be made immediately available for reads. If false, samples will be made available
    /// for reads only after a call to Writer::commit().
    bool enable_auto_commit = false;

    /// @brief sets whether the writer returns error if the writer attempts to write to a channel
    /// that it does not have authority to write to. If false, the writer will silently ignore
    bool err_on_unauthorized = false;

    /// @brief sets the interval at which commits will be flushed to disk and durable 
    /// when auto commit is enabled. Setting this value to zero will make all writes 
    /// durable immediately. Lower values will decrease write throughput. Defaults to 
    /// 1s when auto commit is enabled.
    synnax::TimeSpan auto_index_persist_interval = 1 * synnax::SECOND;

private:
    /// @brief binds the configuration fields to it's protobuf representation.
    void toProto(api::v1::FrameWriterConfig *f) const;

    friend class FrameClient;
    friend class Writer;
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
    Writer() = default;

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
    /// For more information, see https://docs.synnaxlabs.com/concepts/write.
    ///
    /// @returns false if an error occurred in the write pipeline. After an error occurs,
    /// the caller must acknowledge the error by calling error() or close() on the writer.
    bool write(const Frame &fr);

    /// @brief commits all pending writes to the Synnax cluster. Commit can be called
    /// multiple times, committing any new writes made since the last commit.
    ///
    /// @returns false if the commit failed. After a commit fails, the caller must
    /// acknowledge the error by calling error() or close() on the writer.
    std::pair<synnax::TimeStamp, bool> commit();

    /// @brief returns any error accumulated during the write process. If no err has
    /// occurred, err.ok() will be true.
    freighter::Error error() const;

    /// @brief closes the writer and releases any resources associated with it. A writer
    /// MUST be closed after use, or the caller risks leaking resources. Calling any
    /// method on a closed writer will throw a runtime_error.
    freighter::Error close() const;

private:
    /// @brief whether an error has occurred in the write pipeline.
    bool err_accumulated = false;
    /// @brief if close() has been called on the writer.e
    bool closed = false;


    /// @brief the stream transport for the writer.
    std::unique_ptr<WriterStream> stream{};

    /// @brief opens a writer to the Synnax cluster.
    explicit Writer(std::unique_ptr<WriterStream> s);

    /// @brief throws a runtime error if the writer is closed.
    void assertOpen() const;

    friend class FrameClient;
};

class FrameClient {
private:
    std::unique_ptr<StreamerClient> streamer_client;
    std::unique_ptr<WriterClient> writer_client;

public:
    FrameClient(std::unique_ptr<StreamerClient> streamer_client,
                std::unique_ptr<WriterClient> writer_client) : streamer_client(
            std::move(streamer_client)),
        writer_client(std::move(writer_client)) {
    }


    /// @brief opens a new frame writer using the given configuration. For information
    /// on configuration parameters, see WriterConfig.
    /// @returns a pair containing the opened writer and an error when ok() is false
    /// if the writer could not be opened. In the case where ok() is false, the writer
    /// will be in an invalid state and does not need to be closed. If ok() is true,
    /// The writer must be closed after use to avoid leaking resources.
    std::pair<Writer, freighter::Error> openWriter(const WriterConfig &config) const;

    /// @brief opens a new frame streamer using the given configuration. For information
    /// on configuration parameters, see StreamerConfig.
    /// @returns a pair containing the opened streamer and an error when ok() is false
    /// if the streamer could not be opened. In the case where ok() is false, the
    /// streamer will be in an invalid state and does not need to be closed. If ok()
    /// is true, the streamer must be closed after use to avoid leaking resources.
    std::pair<Streamer, freighter::Error> openStreamer(
        const StreamerConfig &config) const;
};
}
