// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <functional>
#include <memory>
#include <optional>
#include <set>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/journal/file.h"

/// @brief an on-disk log of typed, checksummed records. A file starts with an 8-byte
/// magic, a u16 format version, and u16 flags. Each record is a u32 payload length, a
/// u32 CRC-32C of the length, type, and payload, a u8 type, and the payload. All
/// integers are little-endian.
namespace driver::journal {
/// @brief identifies the kind of a file. Each kind of file has its own magic.
using Magic = std::array<char, 8>;
/// @brief the format version that this Driver writes.
constexpr std::uint16_t VERSION = 1;
/// @brief the byte size of the file start.
constexpr std::uint64_t HEADER_SIZE = 12;
/// @brief the byte size of the length, checksum, and type before each payload.
constexpr std::uint64_t RECORD_PREFIX_SIZE = 9;
/// @brief the type bit that lets a reader skip a type it does not know.
constexpr std::uint8_t SKIPPABLE = 0x80;
/// @brief the largest payload of one record.
constexpr std::uint32_t MAX_PAYLOAD = 64 * 1024 * 1024;

/// @brief a file has bytes that are not valid records, other than a torn tail.
const x::errors::Error CORRUPT = ERROR.sub("corrupt");
/// @brief a file starts with the magic of a different kind of file.
const x::errors::Error UNKNOWN_MAGIC = ERROR.sub("unknown_magic");
/// @brief a file has a format version this Driver cannot read.
const x::errors::Error UNKNOWN_VERSION = ERROR.sub("unknown_version");
/// @brief a file has a record type the reader does not know and cannot skip.
const x::errors::Error UNKNOWN_TYPE = ERROR.sub("unknown_type");

/// @brief the start of a file.
struct Header {
    /// @brief the format version of the file.
    std::uint16_t version = VERSION;
    /// @brief flags that the owner of the file defines.
    std::uint16_t flags = 0;
};

/// @brief one record of a file.
struct Record {
    /// @brief the type of the payload. Not 0 or SKIPPABLE.
    std::uint8_t type = 0;
    /// @brief the bytes the owner of the file encodes.
    std::string payload;
};

/// @brief the configuration of a writer.
struct WriterConfig {
    /// @brief the file to append to.
    std::filesystem::path path;
    /// @brief the magic of the kind of file.
    Magic magic{};
    /// @brief the flags to write when the writer creates the file.
    std::uint16_t flags = 0;
    /// @brief the maximum time between a write and the sync that follows it.
    x::telem::TimeSpan sync_interval = x::telem::SECOND;
    /// @brief the number of written bytes that starts a sync.
    std::uint64_t sync_bytes = 1000000;
    /// @brief the clock that times syncs.
    x::telem::NowFunc now = x::telem::TimeStamp::now;
    /// @brief gets the result of each sync of the file and returns the result the
    /// writer uses.
    std::function<x::errors::Error(const x::errors::Error &)> on_sync =
        [](const x::errors::Error &err) { return err; };
};

/// @brief appends records to one file. The writer syncs the file to the disk when
/// sync_interval passed or sync_bytes were written since the last sync, and on close.
/// It checks both only in write and sync. Not safe for concurrent use.
class Writer {
    WriterConfig cfg;
    std::unique_ptr<File> file;
    std::uint64_t size = 0;
    std::uint64_t unsynced = 0;
    x::telem::TimeStamp last_sync;
    x::errors::Error failed;

    Writer(WriterConfig cfg, std::unique_ptr<File> file, std::uint64_t size);
    x::errors::Error fail(const x::errors::Error &err);

public:
    Writer(const Writer &) = delete;
    Writer &operator=(const Writer &) = delete;
    /// @brief closes the writer and drops the error of close.
    ~Writer();

    /// @brief opens the file of the config, or creates it with only a file start.
    /// The writer cuts a torn tail from an existing file, and appends after the last
    /// valid record.
    /// @returns UNKNOWN_MAGIC or UNKNOWN_VERSION when the file start is not one this
    /// writer writes. CORRUPT when the file has invalid bytes before its tail.
    static std::pair<std::unique_ptr<Writer>, x::errors::Error> open(WriterConfig cfg);

    /// @brief appends a record, and syncs the file when a sync is due.
    /// @returns an error when the file fails to write or sync. Then the writer
    /// returns that error from all later calls, and the next open cuts the tail.
    x::errors::Error write(const Record &record);

    /// @brief syncs the file when a sync is due. Call it when no writes come, so the
    /// last writes reach the disk in sync_interval.
    x::errors::Error sync_if_due();

    /// @brief syncs the file to the disk.
    x::errors::Error sync();

    /// @brief syncs and closes the file. Later calls to the writer return an error.
    x::errors::Error close();

    /// @returns the byte position after the last record written.
    [[nodiscard]] std::uint64_t pos() const;
};

/// @brief the configuration of a reader.
struct ReaderConfig {
    /// @brief the file to read.
    std::filesystem::path path;
    /// @brief the magic of the kind of file.
    Magic magic{};
    /// @brief the record types the reader returns. The reader skips other types
    /// that have the SKIPPABLE bit.
    std::set<std::uint8_t> types;
    /// @brief the byte position of the first record to read, or 0 for the first
    /// record of the file.
    std::uint64_t pos = 0;
};

/// @brief reads the records of one file, up to the size the file had at open. Not
/// safe for concurrent use.
class Reader {
    friend class Writer;
    ReaderConfig cfg;
    std::ifstream file;
    std::uint64_t size = 0;
    bool torn_tail = false;

    explicit Reader(ReaderConfig cfg);
    std::pair<std::optional<Record>, x::errors::Error> next_record();
    [[nodiscard]] bool zero_after(std::uint64_t pos);

public:
    /// @brief the start of the file.
    Header header;

    /// @brief opens the file of the config and reads its start.
    /// @returns UNKNOWN_MAGIC, UNKNOWN_VERSION, or CORRUPT when the start is not
    /// valid. CORRUPT when pos is outside the records of the file.
    static std::pair<std::unique_ptr<Reader>, x::errors::Error> open(ReaderConfig cfg);

    /// @brief reads the next record. A record that passes its checksum always
    /// returns. A record that is cut off at the end of the file, or that fails its
    /// checksum with only zeros after it, is a torn tail and ends the file.
    /// @returns the record, or std::nullopt at the end of the file. CORRUPT when a
    /// record fails its checksum before the tail, or has type 0. UNKNOWN_TYPE when
    /// a record has an unknown type without the SKIPPABLE bit.
    std::pair<std::optional<Record>, x::errors::Error> next();

    /// @returns the byte position after the last record read or skipped.
    [[nodiscard]] std::uint64_t pos() const;

    /// @returns true when the reader found a torn tail at the end of the file.
    [[nodiscard]] bool torn() const;
};
}
