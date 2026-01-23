// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <set>
#include <utility>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/control.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/v1/framer.pb.h"

namespace synnax {
/// @brief type alias for streamer network transport stream.
using StreamerStream = freighter::
    Stream<api::v1::FrameStreamerRequest, api::v1::FrameStreamerResponse>;

/// @brief type alias for frame writer network transport.
using StreamerClient = freighter::
    StreamClient<api::v1::FrameStreamerRequest, api::v1::FrameStreamerResponse>;

/// @brief type alias for writer network transports stream.
using WriterStream = freighter::
    Stream<api::v1::FrameWriterRequest, api::v1::FrameWriterResponse>;

/// @brief type alias for writer network transport.
using WriterClient = freighter::
    StreamClient<api::v1::FrameWriterRequest, api::v1::FrameWriterResponse>;

const auto FRAMER_ERROR = xerrors::Error("framer");
const xerrors::Error FRAMER_CLOSED = FRAMER_ERROR.sub("closed");
const xerrors::Error WRITER_CLOSED = FRAMER_CLOSED.sub("writer");

/// @brief Bit positions for flags in the frame codec
enum class FlagPosition : uint8_t {
    ZeroAlignments = 5,
    EqualAlignments = 4,
    EqualLengths = 3,
    EqualTimeRanges = 2,
    TimeRangesZero = 1,
    AllChannelsPresent = 0
};

/// @brief Codec flags for optimizing frame encoding/decoding
struct CodecFlags {
    bool equal_lens = true;
    bool equal_time_ranges = true;
    bool time_ranges_zero = true;
    bool all_channels_present = true;
    bool equal_alignments = true;
    bool zero_alignments = true;

    /// @brief Encodes the flags into a byte
    /// @return The encoded flags
    [[nodiscard]] uint8_t encode() const;

    /// @brief Decodes flags from a byte
    /// @param b The byte to decode
    /// @return The decoded flags
    static CodecFlags decode(uint8_t b);
};

/// @brief Codec for encoding and decoding frames efficiently.
/// This implements the Frame Flight Protocol (RFC 0016)
class Codec {
    struct State {
        /// @brief the ordered set of channel keys for the codec.
        std::set<ChannelKey> keys;
        /// @brief the data types for each channel in keys.
        std::unordered_map<ChannelKey, telem::DataType> key_data_types;
        /// @brief whether the codec has any channels with variable density data types.
        bool has_variable_data_types = false;
    };
    /// @brief a cached set of sorting indices for ensuring that encoded/decoded
    /// frames are properly sorted.
    std::vector<std::pair<ChannelKey, size_t>> sorting_indices;

    /// @brief the current sequence number for the codec. This is used to identify
    /// which codec state to use when encoding/decoding frames.
    std::uint32_t seq_num = 0;

    /// @brief the codec state for each sequence number. This is used to identify the
    /// relevant channel keys and data types for encoding/decoding frames.
    std::unordered_map<std::uint32_t, State> states;

    /// @brief used to retrieve channels when updating the codec state.
    ChannelClient channel_client;

    void throw_if_uninitialized() const;

public:
    Codec() = default;

    /// @brief Creates a new codec with the given data types and channels
    /// @param data_types The data types corresponding to each channel
    /// @param channels The channel keys
    Codec(
        const std::vector<ChannelKey> &channels,
        const std::vector<telem::DataType> &data_types
    );

    /// @brief instantiates a dynamic codec that uses the provided function to look up
    /// the relevant channels when update() is called.
    explicit Codec(ChannelClient channel_client):
        channel_client(std::move(channel_client)) {}

    /// @brief updates the codec to use the given channels. If the channels do not
    /// exist, the codec will return a query::NOT_FOUND error.
    xerrors::Error update(const std::vector<ChannelKey> &keys);

    /// @brief Encodes a frame into a byte array
    /// @param frame The frame to encode
    /// @param output The byte array to encode the frame into.
    /// @return The encoded frame as a byte vector
    xerrors::Error encode(const telem::Frame &frame, std::vector<uint8_t> &output);

    /// @brief Decodes a frame from a byte vector.
    /// @param data The byte vector to decode.
    /// @return The decoded frame.
    [[nodiscard]] std::pair<telem::Frame, xerrors::Error>
    decode(const std::vector<std::uint8_t> &data) const;

    /// @brief decodes a frame from the provided byte array and size.
    /// @param data The byte array to decode.
    /// @param size The size of the byte array.
    [[nodiscard]] std::pair<telem::Frame, xerrors::Error>
    decode(const std::uint8_t *data, std::size_t size) const;
};

/// @brief configuration for opening a new streamer.
class StreamerConfig {
public:
    /// @brief the channels to stream.
    std::vector<ChannelKey> channels;
    /// @brief the downsample factor for the streamer.
    int downsample_factor = 1;
    /// @brief enable experimental high-performance codec for the writer.
    bool enable_experimental_codec = true;

private:
    /// @brief binds the configuration fields to it's protobuf representation.
    void to_proto(api::v1::FrameStreamerRequest &f) const;

    friend class FrameClient;
    friend class Streamer;
};

/// @brief used to stream frames of telemetry from a set of channels in real-time.
/// Streamer cannot be constructed directly, and should instead be opened using the
/// FrameClient.
/// @see FrameClient
///
/// @note read() and setChannels() can be called concurrently with one another, but
/// they cannot be called concurrently with close() or with themselves.
class Streamer {
public:
    Streamer() = default;

    /// @brief blocks until the next frame is received from the Synnax cluster. This
    /// frame is not guaranteed to contain series for all channels specified when
    /// opening the streamer, but it is guaranteed to contain data for at least one
    /// channel and not contain data for any channels not specified.
    /// @returns the next frame of telemetry received from the Synnax cluster and an
    /// error. If error.ok() is false, then the streamer has failed and must be
    /// closed.
    /// @note read is not safe to call concurrently with itself or with close(), but
    /// it is safe to call concurrently with setChannels().
    [[nodiscard]] std::pair<telem::Frame, xerrors::Error> read() const;

    /// @brief sets the channels to stream from the Synnax cluster, replacing any
    /// channels set during construction or a previous call to setChannels().
    /// @returns an error. If error.ok() is false, then the streamer has failed and
    /// must be closed.
    /// @param channels - the channels to stream.
    /// @note setChannels is not safe to call concurrently with itself or with
    /// close(), but it is safe to call concurrently with read().
    [[nodiscard]] xerrors::Error set_channels(const std::vector<ChannelKey> &channels);

    /// @brief closes the streamer and releases any resources associated with it. If
    /// any errors occurred during the stream, they will be returned. A streamer
    /// MUST be closed after use, or the caller risks leaking resources. Calling any
    /// method on a closed streamer will throw a runtime_error.
    /// @returns an error. error.ok() will be false if the streamer accumulated an
    /// error during operation.
    /// @note close() is not safe to call concurrently with itself or any other
    /// streamer methods.
    [[nodiscard]] xerrors::Error close() const;

    /// @brief closes the sending end of the streamer. Subsequence calls to
    /// receive() will exhaust the stream and eventually return an EOF.
    /// @note close_send() is safe to call concurrently with read(), but not with
    /// any other DB methods.
    xerrors::Error close_send() const;

private:
    /// @brief true if the streamer has been closed.
    bool closed = false;

    StreamerConfig cfg;

    /// @brief custom framing codec. only used when cfg.enable_experimental_codec is
    /// set to true.
    Codec codec;

    /// @brief throws if methods have been called on the streamer before it is open.
    void assert_open() const;

    /// @brief constructs the streamer from a configured stream and moves ownership.
    explicit Streamer(std::unique_ptr<StreamerStream> stream, StreamerConfig config);

    /// @brief the stream transport for the streamer.
    std::unique_ptr<StreamerStream> stream;

    /// @brief the only class that can construct a streamer.
    friend class FrameClient;
};

/// @brief sets the persistence and streaming mode for a writer.
enum WriterMode : uint8_t {
    /// @brief sets the writer so that it both persists and streams data.
    PersistStream = 1,
    /// @brief sets the writer so that it persists data, but does not stream it.
    /// Typically used in scenarios involving historical writes.
    PersistOnly = 2,
    /// @brief sets the writer so that it streams data, but does not persist it.
    /// @brief typically used in scenarios involving streaming writes.
    StreamOnly = 3
};

/// @brief configuration for opening a new Writer. For more information on writers,
/// see https://docs.synnaxlabs.com/concepts/write.
struct WriterConfig {
    /// @brief The channels to write to.
    std::vector<ChannelKey> channels;

    /// @brief sets the starting timestamp for the first sample in the writer. If
    /// this timestamp overlaps with existing data for ANY of the provided channels,
    /// the writer will fail to open.
    telem::TimeStamp start;

    /// @brief The control authority to set for each channel. If this vector is of
    /// length 1, then the same authority is set for all channels. Otherwise, the
    /// vector must be the same length as the channels vector. If this vector
    /// is empty, then all writes are executed with AUTH_ABSOLUTE authority.
    std::vector<telem::Authority> authorities;

    /// @brief sets identifying information for the writer. The subject's key and
    /// name will be used to identify the writer in control transfer scenarios.
    telem::ControlSubject subject;

    /// @brief sets whether the writer is configured to persist data, stream it, or
    /// both. Options are:
    ///     - WriterPersistStream: persist data and stream it.
    ///     - WriterPersistOnly: persist data only.
    ///     - WriterStreamOnly: stream data only.
    WriterMode mode;

    /// @brief sets whether auto commit is enabled for the writer. If true, samples
    /// will be made immediately available for reads. If false, samples will be made
    /// available for reads only after a call to Writer::commit().
    bool enable_auto_commit = true;

    /// @brief sets whether the writer returns error if the writer attempts to write
    /// to a channel that it does not have authority to write to. If false, the
    /// writer will silently ignore
    bool err_on_unauthorized = false;

    /// @brief sets the interval at which commits will be flushed to disk and
    /// durable when auto commit is enabled. Setting this value to zero will make
    /// all writes durable immediately. Lower values will decrease write throughput.
    /// Defaults to 1s when auto-commit is enabled.
    telem::TimeSpan auto_index_persist_interval = 1 * telem::SECOND;

    /// @brief enable protobuf frame caching for the writer. This allows
    /// the writer to avoid repeated allocation and deallocation of protobuf frames,
    /// releasing significant heap pressure.
    ///
    /// @details IMPORTANT: This option should only be used for writers that write
    /// a frame with the EXACT same dimensions on every write i.e. same number of
    /// channels and series in the same order. Each series must have the same data
    /// type and the same number of samples. BEHAVIOR IS UNDEFINED IF YOU DO NOT
    /// FOLLOW THIS RULE.
    bool enable_proto_frame_caching = false;

    /// @brief enable experimental high-performance codec for the writer.
    bool enable_experimental_codec = true;

private:
    /// @brief binds the configuration fields to it's protobuf representation.
    void to_proto(api::v1::FrameWriterConfig *f) const;

    friend class FrameClient;

    friend class Writer;
};

/// @brief used to write a new domain of telemetry frames to a set of channels in
/// time order. Writer cannot be constructed directly, and should instead be opened
/// using the FrameClient.
///
/// @note The writer uses a streaming protocol heavily optimized for performance.
/// This comes at the cost of higher complexity.
///
/// @note The writer is not safe for concurrent use.
class Writer {
public:
    Writer() = default;

    /// @brief writes the given frame of telemetry to the Synnax cluster.
    /// @param fr the frame to write. This frame must adhere to a set of
    /// constraints:
    ///
    /// 1. The frame must have at most 1 series per channel.
    /// 2. The frame may not have series for any channel not specified in the
    /// WriterConfig when opening the writer.
    /// 3. All series' that are written to the same index must have the same number
    /// of samples.
    /// 4. When writing to an index, the series' for the index must have
    /// monotonically increasing int64 unix epoch timestamps.
    ///
    /// For more information, see
    /// https://docs.synnaxlabs.com/reference/concepts/writes.
    ///
    /// @returns false if an error occurred in the write pipeline. After an error
    /// occurs, the caller must acknowledge the error by calling error() or close() on
    /// the writer.
    xerrors::Error write(const telem::Frame &fr);

    /// @brief changes the authority of all channels in the writer to the given
    /// authority level.
    /// @returns true if the authority was set successfully.
    /// @param auth the authority level to set all channels to.
    [[nodiscard]] xerrors::Error set_authority(const telem::Authority &auth);

    /// @brief changes the authority of the given channel to the given authority
    /// level. This does not affect the authority levels of any other channels in
    /// the writer.
    /// @returns true if the authority was set successfully.
    /// @param key the channel to set the authority of.
    /// @param authority the authority level to set the channel to.
    [[nodiscard]] xerrors::Error
    set_authority(const ChannelKey &key, const telem::Authority &authority);

    /// @brief changes the authority of the given channels to the given authority
    /// levels.
    /// @returns true if the authority was set successfully.
    /// @param keys the channels to set the authority of.
    /// @param authorities the authority levels to set the channels to.
    [[nodiscard]] xerrors::Error set_authority(
        const std::vector<ChannelKey> &keys,
        const std::vector<telem::Authority> &authorities
    );

    /// @brief commits all pending writes to the Synnax cluster. Commit can be
    /// called multiple times, committing any new writes made since the last commit.
    ///
    /// @returns false if the commit failed. After a commit fails, the caller must
    /// acknowledge the error by calling error() or close() on the writer.
    std::pair<telem::TimeStamp, xerrors::Error> commit();

    /// @brief closes the writer and releases any resources associated with it. A
    /// writer MUST be closed after use, or the caller risks leaking resources.
    /// Calling any method on a closed writer will throw a runtime_error.
    [[nodiscard]] xerrors::Error close();

private:
    [[nodiscard]] xerrors::Error close(const xerrors::Error &close_err);

    /// @brief the error accumulated if the writer has closed with an error.
    xerrors::Error close_err = xerrors::NIL;

    /// @brief the configuration used to open the writer.
    WriterConfig cfg;

    /// @brief the custom synnax frame codec for encoding/decoding frames. This codec
    /// is only used when cfg.enable_experimental_codec is true.
    Codec codec;
    /// @brief the data buffer for storing encoded frames.
    std::vector<std::uint8_t> codec_data;

    /// @brief the stream transport for the writer.
    std::unique_ptr<WriterStream> stream;

    /// @brief cached request for reuse during writes
    std::unique_ptr<api::v1::FrameWriterRequest> cached_write_req;
    /// @brief cached frame within the request for reuse
    telem::PBFrame *cached_frame = nullptr;

    /// @brief internal function that waits until an ack is received for a
    /// particular command.
    std::pair<api::v1::FrameWriterResponse, xerrors::Error>
    exec(api::v1::FrameWriterRequest &req, bool ack);

    /// @brief opens a writer to the Synnax cluster.
    explicit Writer(
        std::unique_ptr<WriterStream> s,
        WriterConfig cfg,
        const Codec &codec
    );

    /// @brief initializes the cached request with the frame structure
    xerrors::Error init_request(const telem::Frame &fr);

    friend class FrameClient;
};

class FrameClient {
public:
    FrameClient(
        std::unique_ptr<StreamerClient> streamer_client,
        std::unique_ptr<WriterClient> writer_client,
        ChannelClient channel_client
    ):
        streamer_client(std::move(streamer_client)),
        writer_client(std::move(writer_client)),
        channel_client(std::move(channel_client)) {}

    /// @brief opens a new frame writer using the given configuration. For
    /// information on configuration parameters, see WriterConfig.
    /// @returns a pair containing the opened writer and an error when ok() is false
    /// if the writer could not be opened. In the case where ok() is false, the
    /// writer will be in an invalid state and does not need to be closed. If ok()
    /// is true, The writer must be closed after use to avoid leaking resources.
    [[nodiscard]] std::pair<Writer, xerrors::Error>
    open_writer(const WriterConfig &cfg) const;

    /// @brief opens a new frame streamer using the given configuration. For
    /// information on configuration parameters, see StreamerConfig.
    /// @returns a pair containing the opened streamer and an error when ok() is
    /// false if the streamer could not be opened. In the case where ok() is false,
    /// the streamer will be in an invalid state and does not need to be closed. If
    /// ok() is true, the streamer must be closed after use to avoid leaking
    /// resources.
    [[nodiscard]] std::pair<Streamer, xerrors::Error>
    open_streamer(const StreamerConfig &config) const;

private:
    /// @brief freighter transport implementation for opening streamers to the
    /// Synnax cluster.
    std::unique_ptr<StreamerClient> streamer_client;
    /// @brief freighter transport implementation for opening writers to the Synnax
    /// cluster.
    std::unique_ptr<WriterClient> writer_client;
    /// @brief a utility function used to retrieve information about channels from the
    /// cluster.
    ChannelClient channel_client;
};

}
