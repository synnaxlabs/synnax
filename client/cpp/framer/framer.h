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
#include <memory>
#include <vector>

/// internal
#include "client/cpp/channel/channel.h"

/// module
#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/control.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

/// protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

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

/// @brief A frame is a collection of series mapped to their corresponding channel
/// keys.
class Frame {
    /// @brief private copy constructor that deep copies the frame.
    Frame(const Frame &other);

public:
    /// @brief the channels in the frame.
    std::unique_ptr<std::vector<ChannelKey>> channels;
    /// @brief the series in the frame.
    std::unique_ptr<std::vector<telem::Series>> series;

    Frame() = default;

    /// @brief move constructor.
    Frame(Frame &&other) noexcept;

    /// @brief allocates a frame that can hold the given number of series.
    /// @param size the number of series to allocate space for.
    explicit Frame(size_t size);

    /// @brief constructs the frame from its protobuf representation.
    /// @param f the protobuf representation of the frame.
    explicit Frame(const api::v1::Frame &f);

    /// @brief constructs a frame with a single channel and series.
    /// @param chan the channel key corresponding to the given series.
    /// @param ser the series to add to the frame.
    Frame(const ChannelKey &chan, telem::Series &&ser);

    explicit Frame(
        std::unordered_map<ChannelKey, telem::SampleValue> &data,
        size_t cap = 0
    );

    /// @brief binds the frame to the given protobuf representation.
    /// @param f the protobuf representation to bind to. This pb must be non-null.
    void to_proto(api::v1::Frame *f) const;

    /// @brief adds a channel and series to the frame.
    /// @param chan the channel key to add.
    /// @param ser the series to add for the channel key.
    void add(const ChannelKey &chan, telem::Series &ser) const;

    /// @brief adds the given series to the frame for the given channel key. Unlike
    /// add,
    ///  this method moves the series into the frame, rather than copying it.
    /// @param chan the channel key to add.
    /// @param ser the series to add for the channel key.
    void emplace(const ChannelKey &chan, telem::Series &&ser) const;

    /// @brief returns true if the frame has no series.
    [[nodiscard]] bool empty() const;

    friend std::ostream &operator<<(std::ostream &os, const Frame &f);

    /// @brief returns the sample for the given channel and index.
    template<typename NumericType>
    NumericType at(const ChannelKey &key, const int &index) const {
        for (size_t i = 0; i < channels->size(); i++)
            if (channels->at(i) == key) return series->at(i).at<NumericType>(index);
        throw std::runtime_error("channel not found");
    }

    [[nodiscard]] telem::SampleValue at(const ChannelKey &key, const int &index) const;

    /// @brief returns the number of series in the frame.
    [[nodiscard]] size_t size() const { return series != nullptr ? series->size() : 0; }

    [[nodiscard]] size_t length() const {
        if (series == nullptr || series->empty()) return 0;
        return series->at(0).size();
    }

    [[nodiscard]] bool contains(const ChannelKey &key) const {
        return std::find(channels->begin(), channels->end(), key) != channels->end();
    }

    /// @brief returns the number of channel-series pairs that the frame can hold
    /// before resizing.
    [[nodiscard]] size_t capacity() const {
        return channels != nullptr ? channels->capacity() : 0;
    }

    /// @brief clears the frame of all channels and series, making it empty for
    /// reuse.
    void clear() const;

    /// @brief reserves the given number of series in the frame.
    void reserve(const size_t &size);

    /// @brief deep copies the frame, all of its series, and their data. This
    /// function must be used explicitly (instead of through a copy constructor) to
    /// avoid unintentional deep copies.
    [[nodiscard]] Frame deep_copy() const;

    /// @brief implements iterator support for the frame, allowing the caller to
    /// traverse the channel keys and series in the frame.
    struct Iterator {
        using iterator_category = std::forward_iterator_tag;
        using value_type = std::pair<ChannelKey, telem::Series &>;
        using difference_type = std::ptrdiff_t;
        using pointer = value_type *;
        using reference = value_type &;

        Iterator(
            std::vector<ChannelKey> &channels_ref,
            std::vector<telem::Series> &series_ref,
            const size_t pos
        ):
            channels(channels_ref), series(series_ref), pos(pos) {}

        value_type operator*() const { return {channels.at(pos), series.at(pos)}; }

        Iterator &operator++() {
            pos++;
            return *this;
        }

        bool operator!=(const Iterator &other) const { return pos != other.pos; }

        bool operator==(const Iterator &other) const { return pos == other.pos; }

    private:
        std::vector<ChannelKey> &channels;
        std::vector<telem::Series> &series;
        size_t pos;
    };

    [[nodiscard]] Iterator begin() const { return {*channels, *series, 0}; }

    [[nodiscard]] Iterator end() const {
        return {*channels, *series, channels->size()};
    }
};


/// @brief A simple binary writer to help with frame encoding
class BinaryWriter {
    std::vector<uint8_t> &buf;
    size_t offset;

public:
    /// @brief Creates a new binary writer that writes to an existing buffer
    /// @param buffer The buffer to write to
    /// @param size The size to resize the buffer to
    /// @param offset The starting offset
    BinaryWriter(std::vector<uint8_t> &buffer, size_t size, size_t offset = 0):
        buf(buffer), offset(offset) {
        buf.resize(size);
    }

    /// @brief Writes a byte to the buffer
    /// @param value The byte to write
    void uint8(uint8_t value);

    /// @brief Writes a 32-bit unsigned integer to the buffer
    /// @param value The uint32 to write
    void uint32(uint32_t value);

    /// @brief Writes a 64-bit unsigned integer to the buffer
    /// @param value The uint64 to write
    void uint64(uint64_t value);

    /// @brief Writes raw bytes to the buffer
    /// @param data The bytes to write
    /// @param size The number of bytes to write
    void write(const void *data, size_t size);

    /// @brief Returns the buffer
    /// @return The buffer as a byte vector
    std::vector<uint8_t> bytes() const;
};

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
    uint8_t encode() const;

    /// @brief Decodes flags from a byte
    /// @param b The byte to decode
    /// @return The decoded flags
    static CodecFlags decode(uint8_t b);
};

/// @brief Codec for encoding and decoding frames efficiently.
/// This implements the Frame Flight Protocol (RFC 0016)
class Codec {
public:
    Codec() = default;

    /// @brief Creates a new codec with the given data types and channels
    /// @param data_types The data types corresponding to each channel
    /// @param channels The channel keys
    Codec(
        const std::vector<telem::DataType> &data_types,
        const std::vector<ChannelKey> &channels
    );

    /// @brief Creates a new codec from a list of channels
    /// @param channels The channels to create the codec from
    explicit Codec(const std::vector<Channel> &channels);

    /// @brief Encodes a frame into a byte array
    /// @param frame The frame to encode
    /// @param start_offset The starting offset in the output buffer
    /// @return The encoded frame as a byte vector
    void encode(const Frame &frame, size_t start_offset, std::vector<uint8_t> &data);

    /// @brief Decodes a frame from a byte array
    /// @param data The byte array to decode
    /// @return The decoded frame
    Frame decode(const std::vector<uint8_t> &data) const;

    /// @brief Decodes a frame from a stream
    /// @param stream The stream to read from
    /// @return The decoded frame
    Frame decode_stream(std::istream &stream) const;

    std::vector<ChannelKey> keys;
    std::unordered_map<ChannelKey, telem::DataType> key_data_types;
    std::vector<std::pair<ChannelKey, size_t>> sorting_indices;

    /// @brief Reads a time range from a stream
    /// @param stream The stream to read from
    /// @return The time range
    static telem::TimeRange read_time_range(std::istream &stream);

    /// @brief Writes a time range to a binary writer
    /// @param writer The writer to write to
    /// @param tr The time range to write
    static void write_time_range(BinaryWriter &writer, const telem::TimeRange &tr);
};
inline bool get_bit(const uint8_t byte, FlagPosition pos) {
    return byte >> static_cast<uint8_t>(pos) & 1;
}

inline uint8_t set_bit(const uint8_t byte, FlagPosition pos, const bool value) {
    if (value) return byte | 1 << static_cast<uint8_t>(pos);
    return byte & ~(1 << static_cast<uint8_t>(pos));
}

/// @brief configuration for opening a new streamer.
class StreamerConfig {
public:
    /// @brief the channels to stream.
    std::vector<ChannelKey> channels;
    /// @brief the downsample factor for the streamer.
    int downsample_factor = 1;

private:
    /// @brief binds the configuration fields to it's protobuf representation.
    void to_proto(api::v1::FrameStreamerRequest &f) const;

    friend class FrameClient;
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
    [[nodiscard]] std::pair<Frame, xerrors::Error> read() const;

    /// @brief sets the channels to stream from the Synnax cluster, replacing any
    /// channels set during construction or a previous call to setChannels().
    /// @returns an error. If error.ok() is false, then the streamer has failed and
    /// must be closed.
    /// @param channels - the channels to stream.
    /// @note setChannels is not safe to call concurrently with itself or with
    /// close(), but it is safe to call concurrently with read().
    [[nodiscard]] xerrors::Error set_channels(std::vector<ChannelKey> channels) const;

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
    void close_send() const;

private:
    /// @brief true if the streamer has been closed.
    bool closed = false;

    /// @brief throws if methods have been called on the streamer before it is open.
    void assert_open() const;

    /// @brief constructs the streamer from a configured stream and moves ownership.
    explicit Streamer(std::unique_ptr<StreamerStream> stream);

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

inline WriterMode data_saving_writer_mode(const bool data_saving) {
    if (data_saving) return WriterMode::PersistStream;
    return WriterMode::StreamOnly;
}

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
    bool enable_auto_commit = false;

    /// @brief sets whether the writer returns error if the writer attempts to write
    /// to a channel that it does not have authority to write to. If false, the
    /// writer will silently ignore
    bool err_on_unauthorized = false;

    /// @brief sets the interval at which commits will be flushed to disk and
    /// durable when auto commit is enabled. Setting this value to zero will make
    /// all writes durable immediately. Lower values will decrease write throughput.
    /// Defaults to 1s when auto commit is enabled.
    telem::TimeSpan auto_index_persist_interval = 1 * telem::SECOND;

    /// @brief whether to enable protobuf frame caching for the writer. This allows
    /// the writer to avoid repeatedly allocation and deallocating protobuf frames,
    /// releasing significant heap pressure.
    ///
    /// @details IMPORTANT: This option should only be used for writers that write
    /// a frame with the EXACT same dimensions on every write i.e. same number of
    /// channels and series in the same order. Each series must have the same data
    /// type and the same number of samples. BEHAVIOR IS UNDEFINED IF YOU DO NOT
    /// FOLLOW THIS RULE.
    bool enable_proto_frame_caching = false;

    bool enable_experimental_codec = false;
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
    /// occurs, the caller must acknowledge the error by calling error() or close()
    /// on the writer.
    bool write(const Frame &fr);

    /// @brief changes the authority of all channels in the writer to the given
    /// authority level.
    /// @returns true if the authority was set successfully.
    /// @param auth the authority level to set all channels to.
    [[nodiscard]] bool set_authority(const telem::Authority &auth);

    /// @brief changes the authority of the given channel to the given authority
    /// level. This does not affect the authority levels of any other channels in
    /// the writer.
    /// @returns true if the authority was set successfully.
    /// @param key the channel to set the authority of.
    /// @param authority the authority level to set the channel to.
    [[nodiscard]] bool
    set_authority(const ChannelKey &key, const telem::Authority &authority);

    /// @brief changes the authority of the given channels to the given authority
    /// levels.
    /// @returns true if the authority was set successfully.
    /// @param keys the channels to set the authority of.
    /// @param authorities the authority levels to set the channels to.
    [[nodiscard]] bool set_authority(
        const std::vector<ChannelKey> &keys,
        const std::vector<telem::Authority> &authorities
    );

    /// @brief commits all pending writes to the Synnax cluster. Commit can be
    /// called multiple times, committing any new writes made since the last commit.
    ///
    /// @returns false if the commit failed. After a commit fails, the caller must
    /// acknowledge the error by calling error() or close() on the writer.
    std::pair<telem::TimeStamp, bool> commit();

    /// @brief returns any error accumulated during the write process. If no err has
    /// occurred, err.ok() will be true.
    [[nodiscard]] xerrors::Error error();

    /// @brief closes the writer and releases any resources associated with it. A
    /// writer MUST be closed after use, or the caller risks leaking resources.
    /// Calling any method on a closed writer will throw a runtime_error.
    [[nodiscard]] xerrors::Error close() const;

private:
    /// @brief whether an error has occurred in the write pipeline.
    bool err_accumulated = false;
    /// @brief if close() has been called on the writer.
    bool closed = false;

    /// @brief the configuration used to open the writer.
    WriterConfig cfg;
    Codec codec;

    /// @brief the stream transport for the writer.
    std::unique_ptr<WriterStream> stream;

    /// @brief cached request for reuse during writes
    std::unique_ptr<api::v1::FrameWriterRequest> cached_write_req;
    /// @brief cached frame within the request for reuse
    api::v1::Frame *cached_frame = nullptr;

    /// @brief internal function that waits until an ack is received for a
    /// particular command.
    api::v1::FrameWriterResponse ack(api::v1::FrameWriterRequest &req);

    /// @brief opens a writer to the Synnax cluster.
    explicit Writer(std::unique_ptr<WriterStream> s, const WriterConfig &cfg);

    /// @brief initializes the cached request with the frame structure
    void init_request(const Frame &fr);

    /// @brief throws a runtime error if the writer is closed.
    void assert_open() const;

    friend class FrameClient;
};

using RetrieveChannels = std::function<
    std::pair<std::vector<synnax::Channel>, xerrors::Error>(
        std::vector<synnax::ChannelKey>
    )>;

class FrameClient {
public:
    FrameClient(
        std::unique_ptr<StreamerClient> streamer_client,
        std::unique_ptr<WriterClient> writer_client,
        const RetrieveChannels &retrieve_channels
    ):
        streamer_client(std::move(streamer_client)),
        writer_client(std::move(writer_client)),
        retrieve_channels(retrieve_channels) {}

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
    RetrieveChannels retrieve_channels;
};


}
