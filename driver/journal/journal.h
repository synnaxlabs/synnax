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
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/journal/file.h"

/// @brief an on-disk log of typed, checksummed records. A file starts with an 8-byte
/// magic, a u16 format version, u16 flags, a random u32 salt, the u64 sync_bytes and
/// u32 max_payload of the writer that created it, and a u32 CRC-32C of the bytes before
/// it. Each record is a u32 payload length, a u32 CRC-32C of the length, type, and
/// payload seeded with the salt, a u8 type, and the payload. All integers are
/// little-endian.
namespace driver::journal {
/// @brief identifies the kind of a file. Each kind of file has its own magic.
using Magic = std::array<char, 8>;
/// @brief the format version that this Driver writes.
constexpr std::uint16_t VERSION = 1;
/// @brief the byte size of the file start.
constexpr std::uint64_t HEADER_SIZE = 32;
/// @brief the byte size of the length, checksum, and type before each payload.
constexpr std::uint64_t RECORD_PREFIX_SIZE = 9;
/// @brief the flag bits that a reader must know to read a file. A reader ignores the
/// other flag bits when it does not know them.
constexpr std::uint16_t REQUIRED_FLAGS = 0xFF00;
/// @brief the type bit that lets a reader skip a type it does not know.
constexpr std::uint8_t SKIPPABLE = 0x80;
/// @brief the largest max_payload of a writer.
constexpr std::uint32_t MAX_PAYLOAD = 64 * 1024 * 1024;
/// @brief the default number of written bytes that starts a sync.
constexpr std::uint64_t DEFAULT_SYNC_BYTES = 1000000;
/// @brief the default largest payload of one record.
constexpr std::uint32_t DEFAULT_MAX_PAYLOAD = 4 * 1024 * 1024;

/// @brief a file is damaged where a crash or a power loss cannot have left it, or a
/// read position is outside the file.
const x::errors::Error CORRUPT = BASE_ERROR.sub("corrupt");
/// @brief a file starts with the magic of a different kind of file.
const x::errors::Error UNKNOWN_MAGIC = BASE_ERROR.sub("unknown_magic");
/// @brief a file has a format version newer than this Driver reads.
const x::errors::Error UNKNOWN_VERSION = BASE_ERROR.sub("unknown_version");
/// @brief a file has a format version older than this Driver writes. A reader can read
/// it, but a writer cannot append to it.
const x::errors::Error OLD_VERSION = BASE_ERROR.sub("old_version");
/// @brief a file has a required flag the reader does not know, so a newer Driver wrote
/// it.
const x::errors::Error UNKNOWN_FLAG = BASE_ERROR.sub("unknown_flag");
/// @brief a file has a record type the reader does not know and cannot skip.
const x::errors::Error UNKNOWN_TYPE = BASE_ERROR.sub("unknown_type");
/// @brief a record payload is larger than the max_payload of the file.
const x::errors::Error TOO_LARGE = BASE_ERROR.sub("too_large");

/// @brief the start of a file.
struct Header {
    /// @brief the format version of the file.
    std::uint16_t version = VERSION;
    /// @brief flags that the owner of the file defines.
    std::uint16_t flags = 0;
    /// @brief the random seed of the record checksums of the file, so a record from
    /// another file fails its checksum.
    std::uint32_t salt = 0;
    /// @brief the sync_bytes of the writer that created the file.
    std::uint64_t sync_bytes = DEFAULT_SYNC_BYTES;
    /// @brief the max_payload of the writer that created the file.
    std::uint32_t max_payload = DEFAULT_MAX_PAYLOAD;
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
    /// @brief the magic of the kind of file. Not all zeros.
    Magic magic{};
    /// @brief the flags to write when the writer creates the file.
    std::uint16_t flags = 0;
    /// @brief the maximum time between a write and the sync that follows it.
    x::telem::TimeSpan sync_interval = x::telem::SECOND;
    /// @brief the number of written bytes that starts a sync. The file keeps the value
    /// of the writer that created it, and a writer of an existing file uses the smaller
    /// value.
    std::uint64_t sync_bytes = DEFAULT_SYNC_BYTES;
    /// @brief the largest payload of one record, at most MAX_PAYLOAD. The file keeps it
    /// as it keeps sync_bytes. A crash can damage only the last sync_bytes plus one
    /// record of a file, so a lower value lets a reader find more damage.
    std::uint32_t max_payload = DEFAULT_MAX_PAYLOAD;
    /// @brief the monotonic clock that times syncs.
    x::telem::NowFunc now = [] {
        return x::telem::TimeStamp(
            std::chrono::duration_cast<std::chrono::nanoseconds>(
                std::chrono::steady_clock::now().time_since_epoch()
            )
                .count()
        );
    };
    /// @brief called with each file that open creates, before open writes it.
    OnCreate on_create = no_create_hook;
    /// @brief gets the result of each sync of the file and returns the result the
    /// writer uses.
    std::function<x::errors::Error(const x::errors::Error &)> on_sync =
        [](const x::errors::Error &err) { return err; };
};

/// @brief appends records to one file. The writer syncs the file to the disk when
/// sync_interval passed or sync_bytes were written since the last sync, and on close.
/// It checks both only in write and sync_if_due. One thread can sync while one other
/// thread writes. close is not safe for concurrent use.
class Writer {
    WriterConfig cfg;
    /// @brief guards the state of the writer, but not a sync in progress.
    mutable std::mutex mu;
    std::mutex sync_mu;
    std::unique_ptr<File> file;
    std::uint64_t size = 0;
    std::uint64_t cut_bytes = 0;
    std::uint64_t unsynced = 0;
    std::uint64_t synced = 0;
    std::uint32_t salt = 0;
    x::telem::TimeStamp last_sync;
    /// @brief stops writes after a write, sync, or close.
    x::errors::Error failed;
    /// @brief stops syncs, because the OS can drop the pages of a failed sync.
    x::errors::Error sync_failed;

    Writer(
        WriterConfig cfg,
        std::unique_ptr<File> file,
        std::uint64_t size,
        std::uint32_t salt
    );

public:
    Writer(const Writer &) = delete;
    Writer &operator=(const Writer &) = delete;
    /// @brief closes the writer and drops the error of close.
    ~Writer();

    /// @brief locks the file of the config, and syncs it. When the file is shorter than
    /// a file start, or is a file start of zeros, the writer replaces it with a file
    /// start, and cut() reports the bytes it replaced. When a crash or a power loss
    /// damaged the end of the file, the writer cuts the file at the damage, as
    /// Reader::next finds it, and appends after the records before it. cut() reports
    /// the bytes it cut.
    /// @returns LOCKED when another writer has the file open or creates it.
    /// UNKNOWN_MAGIC, UNKNOWN_VERSION, OLD_VERSION, or UNKNOWN_FLAG when the file start
    /// is not one this writer writes. CORRUPT when the file start fails its checksum.
    /// The errors of Reader::next, and then the file is unchanged. An error when
    /// max_payload is more than MAX_PAYLOAD.
    static std::pair<std::unique_ptr<Writer>, x::errors::Error> open(WriterConfig cfg);

    /// @brief appends a record, and syncs the file when a sync is due.
    /// @returns TOO_LARGE when the payload is more than max_payload, and then the
    /// writer keeps writing. An error when the file fails to write or sync. Then the
    /// writer refuses all later writes. After a sync error, records written before it
    /// can be lost from the disk while the OS cache still shows them, so a writer
    /// opened again in the same process can keep them. Open the file again only after a
    /// restart.
    x::errors::Error write(const Record &record);

    /// @brief syncs the file when a sync is due. Call it when no writes come, so the
    /// last writes reach the disk in sync_interval.
    x::errors::Error sync_if_due();

    /// @brief syncs the file to the disk.
    x::errors::Error sync();

    /// @brief syncs and closes the file. It also syncs after a write error. Later calls
    /// to the writer return an error.
    x::errors::Error close();

    /// @returns the byte position after the last record written.
    [[nodiscard]] std::uint64_t pos() const;

    /// @returns the byte position after the last record on the disk. A power loss keeps
    /// the records before it. Read a file that a writer appends to only up to it.
    [[nodiscard]] std::uint64_t synced_pos() const;

    /// @returns the number of bytes that open cut from the end of the file. It is not 0
    /// only after a crash or a power loss, or when the file was damaged.
    [[nodiscard]] std::uint64_t cut() const;
};

/// @brief the configuration of a reader.
struct ReaderConfig {
    /// @brief the file to read.
    std::filesystem::path path;
    /// @brief the magic of the kind of file.
    Magic magic{};
    /// @brief the record types the reader returns. The reader skips other types that
    /// have the SKIPPABLE bit.
    std::set<std::uint8_t> types;
    /// @brief the REQUIRED_FLAGS bits the reader knows.
    std::uint16_t flags = 0;
    /// @brief the byte position of the first record to read, or 0 for the first record
    /// of the file.
    std::uint64_t pos = 0;
    /// @brief the byte position to stop at, or std::nullopt for the size of the file at
    /// open. Set it to Writer::synced_pos() to read a file that a writer appends to.
    /// The bytes before it are on the disk, so damage in them is CORRUPT.
    std::optional<std::uint64_t> end;
};

/// @brief reads the records of one file, up to the size the file had at open or the end
/// of the config. Not safe for concurrent use.
class Reader {
    friend class Writer;
    ReaderConfig cfg;
    std::unique_ptr<File> file;
    std::uint64_t size = 0;
    std::uint64_t cut_bytes = 0;
    std::unique_ptr<char[]> buf;
    std::uint64_t buf_pos = 0;
    std::size_t buf_len = 0;

    explicit Reader(ReaderConfig cfg);
    std::pair<std::optional<Record>, x::errors::Error> next_record();
    x::errors::Error read(std::uint64_t pos, char *dst, std::size_t size);
    std::pair<std::optional<Record>, x::errors::Error> damaged();

public:
    /// @brief the start of the file, or the default for a file with no full start.
    Header header;

    /// @brief opens the file of the config and reads its start. A file shorter than a
    /// file start, or a file start of zeros, holds no records, as Writer::open treats
    /// it, and cut() returns its size. Other handles can rename, replace, or remove the
    /// file while it is open.
    /// @returns UNKNOWN_MAGIC, UNKNOWN_VERSION, or UNKNOWN_FLAG when the start is not
    /// one this reader reads. CORRUPT when the start fails its checksum, the version is
    /// 0, or pos or end is outside the file.
    static std::pair<std::unique_ptr<Reader>, x::errors::Error> open(ReaderConfig cfg);

    /// @brief reads the next record. A record that is cut off or fails its checksum
    /// ends the file when it starts in the bytes that a crash or a power loss can leave
    /// unsynced: the last sync_bytes plus one record of max_payload of the file start,
    /// or none with an end in the config. cut() then returns the number of bytes from
    /// that record to the end. A pos from the config at such a record also reads as the
    /// end of the file.
    /// @returns the record, or std::nullopt at the end of the file. CORRUPT when a
    /// damaged record starts before those bytes, or when a record that passes its
    /// checksum has type 0. UNKNOWN_TYPE when a record has an unknown type without the
    /// SKIPPABLE bit, and then the reader stays at that record. An error from the file.
    std::pair<std::optional<Record>, x::errors::Error> next();

    /// @returns the byte position after the last record read or skipped.
    [[nodiscard]] std::uint64_t pos() const;

    /// @returns the number of bytes after the last valid record that the reader did not
    /// read, or 0 when every byte of the file is a valid record. With no end in the
    /// config, it has meaning only when no writer appends to the file.
    [[nodiscard]] std::uint64_t cut() const;
};
}
