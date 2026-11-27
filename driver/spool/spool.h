// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstdint>
#include <filesystem>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/control/control.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/journal/journal.h"

namespace driver::spool {
/// @brief the default byte cap of a spool.
constexpr std::uint64_t DEFAULT_CAP = 10ULL * 1000 * 1000 * 1000;
/// @brief the default number of bytes a spool leaves free on its disk.
constexpr std::uint64_t DEFAULT_MIN_FREE = 1000ULL * 1000 * 1000;
/// @brief the default number of written bytes that starts a sync of a segment.
constexpr std::uint64_t DEFAULT_SYNC_BYTES = 64ULL * 1000 * 1000;
/// @brief the default number of reserved bytes between two checks of the free space.
constexpr std::uint64_t DEFAULT_SPACE_CHECK_BYTES = 64ULL * 1000 * 1000;
/// @brief the magic at the start of each segment file.
constexpr journal::Magic SEGMENT_MAGIC = {'S', 'Y', 'S', 'P', 'L', 'S', 'E', 'G'};
/// @brief the magic at the start of each read offset file.
constexpr journal::Magic OFFSET_MAGIC = {'S', 'Y', 'S', 'P', 'L', 'O', 'F', 'F'};

const x::errors::Error BASE_ERROR = x::errors::Error("spool", "");
/// @brief the spool has no room for a frame.
const x::errors::Error FULL = BASE_ERROR.sub("full");
/// @brief a record passes its checksum but is not a valid header or frame.
const x::errors::Error CORRUPT = BASE_ERROR.sub("corrupt");

/// @returns true when err shows that a segment fails its checks, not that the disk
/// failed. Set such a segment aside with Spool::set_aside.
bool is_bad(const x::errors::Error &err);

/// @returns the number of bytes free to this process on the disk of dir.
std::pair<std::uint64_t, x::errors::Error>
disk_available(const std::filesystem::path &dir);

/// @brief the configuration of a spool.
struct Config {
    /// @brief the directory that holds only the files of the spool.
    std::filesystem::path dir;
    /// @brief the maximum number of bytes the segment files can hold.
    std::uint64_t cap = DEFAULT_CAP;
    /// @brief the number of bytes the spool leaves free on the disk of dir. The spool
    /// is full when a frame would leave less, also below the cap.
    std::uint64_t min_free = DEFAULT_MIN_FREE;
    /// @brief returns the number of bytes free for the spool on the disk of dir.
    std::function<
        std::pair<std::uint64_t, x::errors::Error>(const std::filesystem::path &dir)>
        available = disk_available;
    /// @brief the reserved bytes between two checks of the free disk space.
    std::uint64_t space_check_bytes = DEFAULT_SPACE_CHECK_BYTES;
    /// @brief the maximum time between a write and the sync that follows it.
    x::telem::TimeSpan sync_interval = x::telem::SECOND;
    /// @brief the written bytes that start a sync of a segment.
    std::uint64_t sync_bytes = DEFAULT_SYNC_BYTES;
    /// @brief the largest encoded frame, at most journal::MAX_PAYLOAD. A crash can
    /// damage the last sync_bytes plus one frame of this size of a segment, so a lower
    /// value lets a reader find more damage.
    std::uint32_t max_payload = journal::MAX_PAYLOAD;
    /// @brief called with each file that the spool creates, before the spool uses it.
    x::fs::OnCreate on_create = x::fs::no_create_hook;
    /// @brief gets the result of each sync of a segment and returns the result the
    /// spool uses. Writers call it from their own threads, so it must be safe for
    /// concurrent use.
    std::function<x::errors::Error(const x::errors::Error &)> on_sync =
        [](const x::errors::Error &err) { return err; };
};

/// @brief what a segment holds frames for: one writer of one cluster.
struct Header {
    /// @brief the key of the cluster the frames go to.
    x::uuid::UUID cluster;
    /// @brief the channel keys of the writer.
    std::vector<std::uint32_t> channels;
    /// @brief the index channel of each channel, in the order of channels.
    std::vector<std::uint32_t> indexes;
    /// @brief the data type of each channel, in the order of channels.
    std::vector<x::telem::DataType> data_types;
    /// @brief the control subject of the writer.
    x::control::Subject subject;
    /// @brief the key of the task that writes the frames.
    x::uuid::UUID task;
    /// @brief the key of the rack of the task.
    std::uint32_t rack = 0;
    /// @brief the time the writer opened.
    x::telem::TimeStamp created;
    /// @brief the version of the Driver that writes the frames.
    std::string version;
};

/// @brief a segment file that the spool set aside because it failed its checks.
struct BadSegment {
    /// @brief the path of the file after the spool renamed it.
    std::filesystem::path path;
    /// @brief why the file failed its checks.
    x::errors::Error cause;
};

class Spool;

/// @brief appends frames to one new segment. Not safe for concurrent use. The spool
/// must outlive the writer.
class Writer {
    friend class Spool;
    Spool &spool;
    Header header;
    std::unordered_map<std::uint32_t, x::telem::DataType> types;
    std::uint64_t key;
    std::unique_ptr<journal::Writer> file;
    std::uint64_t size = 0;
    bool refused = false;
    bool closed = false;
    x::errors::Error failed;

    Writer(Spool &spool, Header header, std::uint64_t key);
    x::errors::Error fail(const x::errors::Error &err, std::uint64_t reserved);

public:
    Writer(const Writer &) = delete;
    Writer &operator=(const Writer &) = delete;
    /// @brief closes the writer and drops the error of close.
    ~Writer();

    /// @brief appends the frame to the segment. After the spool refuses one frame, the
    /// writer refuses every later frame, so the segment has no gaps.
    /// @param frame the frame to append. Each of its channels must be in the header,
    /// with the data type of the header.
    /// @returns FULL when the spool has no room for the frame. BASE_ERROR when the
    /// frame does not match the header, the writer is closed, or the encoded frame is
    /// larger than Config::max_payload. After a write fails to store a valid frame, the
    /// writer returns the same error for every later frame. Spool::refused counts each
    /// of these frames. When only the sync after the frame fails, the frame stays in
    /// the segment, and readers can read it.
    x::errors::Error write(const x::telem::Frame &frame);

    /// @brief syncs the segment when a sync is due. Call it when no frames come, so the
    /// last frames reach the disk in one second.
    x::errors::Error sync_if_due();

    /// @brief syncs and closes the segment, so readers can open it. Later writes return
    /// an error, and later calls to close do nothing.
    /// @returns the error of the last sync. Then the frames since the sync before it
    /// can be lost.
    x::errors::Error close();
};

/// @brief reads the frames of one segment from its committed read offset. Not safe for
/// concurrent use. The spool must outlive the reader.
class Reader {
    friend class Spool;
    Spool &spool;
    std::uint64_t key;
    std::unordered_map<std::uint32_t, x::telem::DataType> types;
    std::unique_ptr<journal::Reader> file;
    std::uint64_t pos = 0;
    x::errors::Error failed;

    Reader(Spool &spool, std::uint64_t key);

public:
    /// @brief the header of the segment.
    Header header;

    /// @brief reads the next frame. The first record that a crash or a power loss
    /// damaged ends the segment, and the spool counts its bytes in Spool::cut.
    /// @returns the frame, or std::nullopt at the end of the segment, where the reader
    /// closes its file. CORRUPT when a record is not a valid frame. The errors of
    /// journal::Reader::next when a record is not valid. After an error, every later
    /// read returns it, so the reader never skips a record. Open a new reader to try
    /// again.
    std::pair<std::optional<x::telem::Frame>, x::errors::Error> read();

    /// @brief persists the read offset after the last frame read, so a reader opened
    /// later starts from it.
    /// @returns BASE_ERROR when the spool removed the segment.
    x::errors::Error commit();
};

/// @brief stores frames that a Driver cannot deliver to the Core in segments on disk,
/// and keeps them across restarts. Safe for concurrent use.
class Spool {
    friend class Writer;
    friend class Reader;
    Config cfg;
    std::unique_ptr<journal::Lock> lock;
    mutable std::mutex mu;
    std::uint64_t used = 0;
    /// @brief the most bytes used can grow to, from the free space at the last check.
    std::uint64_t limit = 0;
    /// @brief the bytes reserved since the last check of the free space.
    std::uint64_t unchecked = 0;
    std::uint64_t next_key = 0;
    /// @brief the byte size of each closed segment, by key.
    std::map<std::uint64_t, std::uint64_t> closed;
    std::atomic<std::uint64_t> refused_count = 0;
    std::atomic<std::uint64_t> cut_count = 0;
    /// @brief the keys of the segments whose cut bytes are in cut_count.
    std::set<std::uint64_t> cut_segments;
    std::vector<BadSegment> bad;

    Spool(Config cfg, std::unique_ptr<journal::Lock> lock);
    [[nodiscard]] std::filesystem::path segment_path(std::uint64_t key) const;
    [[nodiscard]] std::filesystem::path offset_path(std::uint64_t key) const;
    x::errors::Error reserve(std::uint64_t bytes);
    void release(std::uint64_t bytes);
    void close(std::uint64_t key, std::uint64_t size);
    void count_cut(std::uint64_t key, std::uint64_t bytes);
    [[nodiscard]] bool is_closed(std::uint64_t key) const;
    x::errors::Error remove_files(std::uint64_t key);
    void forget(std::uint64_t key);
    x::errors::Error set_aside_files(
        const std::filesystem::path &segment,
        const x::errors::Error &cause
    );
    [[nodiscard]] x::errors::Error remove_temp_files() const;
    [[nodiscard]] x::errors::Error remove_orphan_offsets() const;

public:
    /// @brief opens the spool in the directory of the config, and creates the directory
    /// when it does not exist. Removes segments that a crash left without a header,
    /// read offsets without a segment, and temp files. Sets aside each segment that
    /// fails its checks, as set_aside does.
    /// @returns x::fs::LOCKED when another process has the spool open. An error when
    /// the spool cannot read or change its directory.
    static std::pair<std::unique_ptr<Spool>, x::errors::Error> open(Config cfg);

    /// @brief starts a new segment for the header. The writer creates the file at its
    /// first frame.
    std::unique_ptr<Writer> open_writer(Header header);

    /// @returns the keys of the closed segments, oldest first.
    [[nodiscard]] std::vector<std::uint64_t> segments() const;

    /// @brief opens a reader on a closed segment.
    /// @param key a key from segments().
    /// @returns BASE_ERROR when the segment is not closed. CORRUPT when the header or
    /// the read offset of the segment is not valid. A journal error when a file of the
    /// segment is not valid.
    std::pair<std::unique_ptr<Reader>, x::errors::Error> open_reader(std::uint64_t key);

    /// @brief deletes a closed segment and its read offset. On Windows, a reader of the
    /// segment must first read to the end or be destroyed.
    /// @param key a key from segments().
    x::errors::Error remove(std::uint64_t key);

    /// @brief renames a closed segment to `<key>.seg.bad` and its read offset to
    /// `<key>.offset.bad`, or to `.bad.<n>` when that name is taken, so the spool no
    /// longer lists it, and adds it to bad_segments(). The spool never deletes these
    /// files. Call it when open_reader or a reader of the segment returns an error that
    /// is_bad accepts. On Windows, the readers of the segment must first be destroyed.
    /// @param key a key from segments().
    /// @param cause why the segment fails its checks.
    x::errors::Error set_aside(std::uint64_t key, const x::errors::Error &cause);

    /// @returns the segments set aside since the spool opened, oldest first. Report
    /// them in a status with the message "Set aside N spool segments", and the
    /// directory and causes in the description.
    [[nodiscard]] std::vector<BadSegment> bad_segments() const;

    /// @returns the number of frames that writes did not store because the spool was
    /// full, the frame was too large, or the writer failed.
    [[nodiscard]] std::uint64_t refused() const;

    /// @returns the number of bytes of segments that a crash, a power loss, a full
    /// disk, or damage to the disk made unreadable, since the spool opened.
    [[nodiscard]] std::uint64_t cut() const;
};
}
