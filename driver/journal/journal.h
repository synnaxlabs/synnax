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
#include <chrono>
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

/// @brief a record that passes its checksum is not valid, or a read position is
/// outside the records of a file.
const x::errors::Error CORRUPT = BASE_ERROR.sub("corrupt");
/// @brief a file starts with the magic of a different kind of file.
const x::errors::Error UNKNOWN_MAGIC = BASE_ERROR.sub("unknown_magic");
/// @brief a file has a format version this Driver cannot read.
const x::errors::Error UNKNOWN_VERSION = BASE_ERROR.sub("unknown_version");
/// @brief a file has a record type the reader does not know and cannot skip.
const x::errors::Error UNKNOWN_TYPE = BASE_ERROR.sub("unknown_type");

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
    /// @brief the monotonic clock that times syncs.
    x::telem::NowFunc now = [] {
        return x::telem::TimeStamp(
            std::chrono::duration_cast<std::chrono::nanoseconds>(
                std::chrono::steady_clock::now().time_since_epoch()
            )
                .count()
        );
    };
    /// @brief gets the result of each sync of the file and returns the result the
    /// writer uses.
    std::function<x::errors::Error(const x::errors::Error &)> on_sync =
        [](const x::errors::Error &err) { return err; };
};

/// @brief appends records to one file. The writer syncs the file to the disk when
/// sync_interval passed or sync_bytes were written since the last sync, and on close.
/// It checks both only in write and sync_if_due. Not safe for concurrent use.
class Writer {
    WriterConfig cfg;
    std::unique_ptr<File> file;
    std::uint64_t size = 0;
    std::uint64_t cut_bytes = 0;
    std::uint64_t unsynced = 0;
    x::telem::TimeStamp last_sync;
    /// @brief stops writes after a write, sync, or close.
    x::errors::Error failed;
    /// @brief stops syncs, because the OS can drop the pages of a failed sync.
    x::errors::Error sync_failed;

    Writer(WriterConfig cfg, std::unique_ptr<File> file, std::uint64_t size);

public:
    Writer(const Writer &) = delete;
    Writer &operator=(const Writer &) = delete;
    /// @brief closes the writer and drops the error of close.
    ~Writer();

    /// @brief locks the file of the config, and creates it with only a file start
    /// when it is shorter than a file start. The writer cuts the file at its first
    /// record that is cut off or fails its checksum, also in the middle of the file,
    /// and appends after the records before it. cut() reports the bytes it cut.
    /// @returns LOCKED when another writer has the file open. UNKNOWN_MAGIC or
    /// UNKNOWN_VERSION when the file start is not one this writer writes. CORRUPT
    /// when a record that passes its checksum has type 0.
    static std::pair<std::unique_ptr<Writer>, x::errors::Error> open(WriterConfig cfg);

    /// @brief appends a record, and syncs the file when a sync is due.
    /// @returns an error when the file fails to write or sync. Then the writer
    /// refuses all later writes. After a sync error, records written before it can be
    /// lost from the disk while the OS cache still shows them, so a writer opened
    /// again in the same process can keep them. Open the file again only after a
    /// restart.
    x::errors::Error write(const Record &record);

    /// @brief syncs the file when a sync is due. Call it when no writes come, so the
    /// last writes reach the disk in sync_interval.
    x::errors::Error sync_if_due();

    /// @brief syncs the file to the disk.
    x::errors::Error sync();

    /// @brief syncs and closes the file. It also syncs after a write error. Later
    /// calls to the writer return an error.
    x::errors::Error close();

    /// @returns the byte position after the last record written.
    [[nodiscard]] std::uint64_t pos() const;

    /// @returns the number of bytes that open cut from the end of the file. It is not
    /// 0 only after a crash or a power loss, or when the file was damaged.
    [[nodiscard]] std::uint64_t cut() const;
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
    std::uint64_t cut_bytes = 0;

    explicit Reader(ReaderConfig cfg);
    std::pair<std::optional<Record>, x::errors::Error> next_record();

public:
    /// @brief the start of the file, or the default for a file with no full start.
    Header header;

    /// @brief opens the file of the config and reads its start. A file shorter than a
    /// file start holds no records, as Writer::open treats it, and cut() returns its
    /// size.
    /// @returns UNKNOWN_MAGIC or UNKNOWN_VERSION when the start is not one this
    /// reader reads. CORRUPT when pos is outside the records of the file.
    static std::pair<std::unique_ptr<Reader>, x::errors::Error> open(ReaderConfig cfg);

    /// @brief reads the next record. The first record that is cut off or fails its
    /// checksum ends the file, as a crash or a power loss leaves it. A damaged record
    /// in the middle of the file also ends it, and the records after it are lost.
    /// cut() then returns the number of bytes from that record to the end of the file.
    /// @returns the record, or std::nullopt at the end of the file. CORRUPT when a
    /// record that passes its checksum has type 0. UNKNOWN_TYPE when a record has an
    /// unknown type without the SKIPPABLE bit.
    std::pair<std::optional<Record>, x::errors::Error> next();

    /// @returns the byte position after the last record read or skipped.
    [[nodiscard]] std::uint64_t pos() const;

    /// @returns the number of bytes after the last valid record that the reader did
    /// not read, or 0 when every byte of the file is a valid record.
    [[nodiscard]] std::uint64_t cut() const;
};
}
