// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <random>
#include <span>
#include <system_error>
#include <utility>

#include "absl/crc/crc32c.h"

#include "x/cpp/binary/binary.h"
#include "x/cpp/fs/file.h"

#include "driver/journal/journal.h"

namespace driver::journal {
namespace {
template<std::integral T>
void put(std::string &buf, const T value) {
    const auto bytes = x::binary::to_little_endian(value);
    buf.append(reinterpret_cast<const char *>(bytes.data()), bytes.size());
}

template<std::integral T>
T get(const char *data) {
    return x::binary::from_little_endian<T>(std::span<const std::uint8_t, sizeof(T)>(
        reinterpret_cast<const std::uint8_t *>(data),
        sizeof(T)
    ));
}

/// @brief the byte position of the flags in the header.
constexpr std::size_t FLAGS_OFFSET = 10;
/// @brief the byte position of the salt in the header.
constexpr std::size_t SALT_OFFSET = 12;
/// @brief the byte position of the sync_bytes in the header.
constexpr std::size_t SYNC_BYTES_OFFSET = 16;
/// @brief the byte position of the max_payload in the header.
constexpr std::size_t MAX_PAYLOAD_OFFSET = 24;
/// @brief the byte size of the header before its checksum.
constexpr std::size_t HEADER_SUM_OFFSET = HEADER_SIZE - sizeof(std::uint32_t);

std::uint32_t checksum(
    const std::uint32_t salt,
    const char *len,
    const std::uint8_t type,
    const std::string_view payload
) {
    auto crc = absl::ExtendCrc32c(absl::crc32c_t{salt}, {len, sizeof(std::uint32_t)});
    crc = absl::ExtendCrc32c(crc, {reinterpret_cast<const char *>(&type), 1});
    return static_cast<std::uint32_t>(absl::ExtendCrc32c(crc, payload));
}

std::string encode_header(const Magic &magic, const Header &header) {
    std::string buf(magic.data(), magic.size());
    put(buf, header.version);
    put(buf, header.flags);
    put(buf, header.salt);
    put(buf, header.sync_bytes);
    put(buf, header.max_payload);
    put(buf, static_cast<std::uint32_t>(absl::ComputeCrc32c(buf)));
    return buf;
}

void encode_record(std::string &buf, const std::uint32_t salt, const Record &record) {
    const auto start = buf.size();
    put(buf, static_cast<std::uint32_t>(record.payload.size()));
    put(buf, checksum(salt, buf.data() + start, record.type, record.payload));
    buf.push_back(static_cast<char>(record.type));
    buf.append(record.payload);
}

/// @brief the byte size of the buffer a reader reads small records through.
constexpr std::size_t READ_BUFFER_SIZE = std::size_t{64} * 1024;

bool zeros(const std::span<const char> data) {
    return std::ranges::all_of(data, [](const char c) { return c == 0; });
}

x::errors::Error validate(const Magic &magic) {
    if (zeros(magic)) return x::errors::Error(BASE_ERROR, "magic is all zeros");
    return x::errors::NIL;
}
}

Writer::Writer(
    WriterConfig cfg,
    std::unique_ptr<x::fs::File> file,
    const std::uint64_t size,
    const std::uint32_t salt
):
    cfg(std::move(cfg)),
    file(std::move(file)),
    size(size),
    synced(size),
    salt(salt),
    last_sync(this->cfg.now()) {}

Writer::~Writer() {
    this->close();
}

std::pair<std::unique_ptr<Writer>, x::errors::Error> Writer::open(WriterConfig cfg) {
    if (auto err = validate(cfg.magic)) return {nullptr, err};
    if (cfg.max_payload > MAX_PAYLOAD)
        return {
            nullptr,
            x::errors::Error(
                BASE_ERROR,
                "max_payload is more than " + std::to_string(MAX_PAYLOAD)
            )
        };
    Header header{
        .flags = cfg.flags,
        .salt = static_cast<std::uint32_t>(std::random_device()()),
        .sync_bytes = cfg.sync_bytes,
        .max_payload = cfg.max_payload,
    };
    bool created = false;
    std::error_code ec;
    if (!std::filesystem::exists(cfg.path, ec)) {
        if (ec)
            return {
                nullptr,
                x::errors::Error(
                    BASE_ERROR,
                    "failed to stat " + cfg.path.string() + ": " + ec.message()
                )
            };
        const auto err = x::fs::create_new(
            cfg.path,
            encode_header(cfg.magic, header),
            cfg.on_create
        );
        if (err && !err.matches(x::fs::EXISTS)) return {nullptr, err};
        created = !err;
    }
    std::unique_ptr<x::fs::File> file;
    while (file == nullptr) {
        auto [f, err] = x::fs::File::open(cfg.path);
        if (err) return {nullptr, err};
        if (err = f->lock(); err) return {nullptr, err};
        auto [removed, removed_err] = f->removed();
        if (removed_err) return {nullptr, removed_err};
        // A create on a volume with no exclusive rename replaced the locked file.
        if (!removed) file = std::move(f);
    }
    auto err = x::errors::NIL;
    // A crash during a create can leave its temp file.
    if (err = x::fs::remove_temps(cfg.path); err) return {nullptr, err};
    auto [reader, r_err] = Reader::open(
        {.path = cfg.path, .magic = cfg.magic, .flags = cfg.flags}
    );
    if (r_err) return {nullptr, r_err};
    std::uint64_t size = HEADER_SIZE;
    std::uint64_t cut = 0;
    // A crash during create_new on a volume with no exclusive rename, or during the
    // write below, leaves a file with no start.
    if (reader->pos() == 0) {
        cut = reader->cut();
        reader.reset();
        if (err = file->truncate(0); err) return {nullptr, err};
        if (err = file->write(encode_header(cfg.magic, header)); err)
            return {nullptr, err};
        if (err = cfg.on_sync(file->sync()); err) return {nullptr, err};
        const auto dir = cfg.path.parent_path();
        if (err = x::fs::sync_dir(dir.empty() ? "." : dir); err) return {nullptr, err};
    } else {
        header = reader->header;
        if (header.version < VERSION)
            return {
                nullptr,
                x::errors::Error(OLD_VERSION, "cannot append to " + cfg.path.string())
            };
        while (true) {
            auto [record, n_err] = reader->next_record();
            if (n_err) return {nullptr, n_err};
            if (!record) break;
        }
        size = reader->pos();
        cut = reader->cut();
        reader.reset();
        if (cut > 0)
            if (err = file->truncate(size); err) return {nullptr, err};
        // An earlier process can leave its writes only in the OS cache.
        if (!created || cut > 0)
            if (err = cfg.on_sync(file->sync()); err) return {nullptr, err};
    }
    // A crash leaves no more unsynced bytes than the file start allows.
    cfg.sync_bytes = std::min(cfg.sync_bytes, header.sync_bytes);
    cfg.max_payload = std::min(cfg.max_payload, header.max_payload);
    auto writer = std::unique_ptr<Writer>(
        new Writer(std::move(cfg), std::move(file), size, header.salt)
    );
    writer->cut_bytes = cut;
    return {std::move(writer), x::errors::NIL};
}

x::errors::Error Writer::write(const Record &record) {
    if ((record.type & ~SKIPPABLE) == 0)
        return x::errors::Error(BASE_ERROR, "record type has no value");
    if (record.payload.size() > this->cfg.max_payload)
        return x::errors::Error(
            TOO_LARGE,
            "the payload has " + std::to_string(record.payload.size()) +
                " bytes, and the file takes at most " +
                std::to_string(this->cfg.max_payload)
        );
    std::string buf;
    encode_record(buf, this->salt, record);
    {
        std::lock_guard lock(this->mu);
        if (this->failed) return this->failed;
        if (auto err = this->file->write(buf)) {
            this->failed = err;
            return err;
        }
        this->size += buf.size();
        this->unsynced += buf.size();
    }
    return this->sync_if_due();
}

x::errors::Error Writer::sync_if_due() {
    {
        std::lock_guard lock(this->mu);
        if (this->failed || this->unsynced == 0) return this->failed;
        if (this->unsynced < this->cfg.sync_bytes &&
            this->cfg.now() - this->last_sync < this->cfg.sync_interval)
            return x::errors::NIL;
    }
    return this->sync();
}

x::errors::Error Writer::sync() {
    std::lock_guard sync_lock(this->sync_mu);
    std::uint64_t target = 0;
    {
        std::lock_guard lock(this->mu);
        if (this->file == nullptr || this->sync_failed || this->unsynced == 0)
            return this->failed;
        target = this->size;
    }
    const auto err = this->cfg.on_sync(this->file->sync());
    std::lock_guard lock(this->mu);
    if (err) {
        this->sync_failed = err;
        if (!this->failed) this->failed = err;
        return err;
    }
    this->unsynced = this->size - target;
    this->synced = target;
    this->last_sync = this->cfg.now();
    return this->failed;
}

x::errors::Error Writer::close() {
    if (this->file == nullptr) return this->failed;
    const auto err = this->sync();
    std::lock_guard lock(this->mu);
    this->file.reset();
    if (!this->failed) this->failed = x::errors::Error(BASE_ERROR, "writer is closed");
    return err;
}

std::uint64_t Writer::pos() const {
    std::lock_guard lock(this->mu);
    return this->size;
}

std::uint64_t Writer::synced_pos() const {
    std::lock_guard lock(this->mu);
    return this->synced;
}

std::uint64_t Writer::cut() const {
    return this->cut_bytes;
}

Reader::Reader(ReaderConfig cfg): cfg(std::move(cfg)) {}

std::pair<std::unique_ptr<Reader>, x::errors::Error> Reader::open(ReaderConfig cfg) {
    if (auto err = validate(cfg.magic)) return {nullptr, err};
    const auto path = cfg.path.string();
    auto reader = std::unique_ptr<Reader>(new Reader(std::move(cfg)));
    auto [file, err] = x::fs::File::open_read(reader->cfg.path);
    if (err) return {nullptr, err};
    reader->file = std::move(file);
    auto [size, size_err] = reader->file->size();
    if (size_err) return {nullptr, size_err};
    reader->size = size;
    if (const auto end = reader->cfg.end; end.has_value()) {
        if (*end > reader->size)
            return {nullptr, x::errors::Error(CORRUPT, "end is outside " + path)};
        reader->size = *end;
    }
    std::array<char, HEADER_SIZE> buf{};
    if (reader->size >= HEADER_SIZE)
        if (err = reader->read(0, buf.data(), buf.size()); err) return {nullptr, err};
    // A crash while a writer writes the start in place can leave zeros.
    if (reader->size < HEADER_SIZE || (reader->size == HEADER_SIZE && zeros(buf))) {
        reader->cfg.pos = 0;
        reader->cut_bytes = reader->size;
        return {std::move(reader), x::errors::NIL};
    }
    const auto &magic = reader->cfg.magic;
    if (!std::equal(magic.begin(), magic.end(), buf.begin()))
        return {nullptr, x::errors::Error(UNKNOWN_MAGIC, path)};
    // A newer version can lay out the rest of the start in another way.
    reader->header.version = get<std::uint16_t>(buf.data() + magic.size());
    if (reader->header.version > VERSION)
        return {
            nullptr,
            x::errors::Error(
                UNKNOWN_VERSION,
                path + " has version " + std::to_string(reader->header.version)
            )
        };
    if (static_cast<std::uint32_t>(
            absl::ComputeCrc32c({buf.data(), HEADER_SUM_OFFSET})
        ) != get<std::uint32_t>(buf.data() + HEADER_SUM_OFFSET))
        return {nullptr, x::errors::Error(CORRUPT, path + " has a damaged start")};
    reader->header.flags = get<std::uint16_t>(buf.data() + FLAGS_OFFSET);
    reader->header.salt = get<std::uint32_t>(buf.data() + SALT_OFFSET);
    reader->header.sync_bytes = get<std::uint64_t>(buf.data() + SYNC_BYTES_OFFSET);
    reader->header.max_payload = get<std::uint32_t>(buf.data() + MAX_PAYLOAD_OFFSET);
    if (reader->header.version == 0)
        return {nullptr, x::errors::Error(CORRUPT, path + " has version 0")};
    if (const auto unknown = reader->header.flags & REQUIRED_FLAGS & ~reader->cfg.flags;
        unknown != 0)
        return {
            nullptr,
            x::errors::Error(
                UNKNOWN_FLAG,
                path + " has flags " + std::to_string(unknown) +
                    " that a newer Driver writes. Update the Driver to read it."
            )
        };
    auto &pos = reader->cfg.pos;
    if (pos == 0) pos = HEADER_SIZE;
    if (pos < HEADER_SIZE || pos > reader->size)
        return {nullptr, x::errors::Error(CORRUPT, "position is outside " + path)};
    return {std::move(reader), x::errors::NIL};
}

x::errors::Error
Reader::read(const std::uint64_t pos, char *dst, const std::size_t size) {
    if (pos >= this->buf_pos && pos + size <= this->buf_pos + this->buf_len) {
        std::copy_n(this->buf.get() + (pos - this->buf_pos), size, dst);
        return x::errors::NIL;
    }
    auto *to = dst;
    auto len = size;
    if (size < READ_BUFFER_SIZE) {
        if (this->buf == nullptr)
            this->buf = std::make_unique_for_overwrite<char[]>(READ_BUFFER_SIZE);
        to = this->buf.get();
        len = std::max<std::size_t>(
            size,
            std::min<std::uint64_t>(READ_BUFFER_SIZE, this->size - pos)
        );
        this->buf_len = 0;
    }
    auto [n, err] = this->file->read(pos, to, len);
    if (err) return err;
    // The size at open says these bytes exist.
    if (n < len)
        return x::errors::Error(
            BASE_ERROR,
            this->cfg.path.string() + " is shorter than at open"
        );
    if (to != dst) {
        this->buf_pos = pos;
        this->buf_len = len;
        std::copy_n(to, size, dst);
    }
    return x::errors::NIL;
}

std::pair<std::optional<Record>, x::errors::Error> Reader::next_record() {
    const auto pos = this->cfg.pos;
    if (this->cut_bytes > 0 || pos == this->size) return {std::nullopt, x::errors::NIL};
    if (this->size - pos < RECORD_PREFIX_SIZE) return this->damaged();
    std::array<char, RECORD_PREFIX_SIZE> prefix{};
    if (auto err = this->read(pos, prefix.data(), prefix.size()))
        return {std::nullopt, err};
    const auto len = get<std::uint32_t>(prefix.data());
    const auto end = pos + RECORD_PREFIX_SIZE + len;
    if (len > this->header.max_payload) return this->damaged();
    if (end > this->size) return this->damaged();
    Record record{.type = static_cast<std::uint8_t>(prefix[8])};
    x::errors::Error err;
    record.payload.resize_and_overwrite(len, [&](char *data, const std::size_t n) {
        err = this->read(pos + RECORD_PREFIX_SIZE, data, n);
        return n;
    });
    if (err) return {std::nullopt, err};
    if (checksum(this->header.salt, prefix.data(), record.type, record.payload) !=
        get<std::uint32_t>(prefix.data() + 4))
        return this->damaged();
    if ((record.type & ~SKIPPABLE) == 0)
        return {std::nullopt, x::errors::Error(CORRUPT, "record type has no value")};
    this->cfg.pos = end;
    return {std::move(record), x::errors::NIL};
}

std::pair<std::optional<Record>, x::errors::Error> Reader::damaged() {
    const auto pos = this->cfg.pos;
    // A writer syncs once sync_bytes are unsynced, after the write of one record.
    std::uint64_t unsynced = 0;
    if (!this->cfg.end.has_value())
        unsynced = this->header.sync_bytes +
                   std::min<std::uint64_t>(
                       RECORD_PREFIX_SIZE + this->header.max_payload,
                       UINT64_MAX - this->header.sync_bytes
                   );
    if (this->size - pos > unsynced)
        return {
            std::nullopt,
            x::errors::Error(
                CORRUPT,
                "the record at " + std::to_string(pos) + " of " +
                    this->cfg.path.string() + " is damaged before its last " +
                    std::to_string(unsynced) + " bytes"
            )
        };
    this->cut_bytes = this->size - pos;
    return {std::nullopt, x::errors::NIL};
}

std::pair<std::optional<Record>, x::errors::Error> Reader::next() {
    while (true) {
        const auto start = this->cfg.pos;
        auto [record, err] = this->next_record();
        if (err || !record || this->cfg.types.contains(record->type))
            return {std::move(record), err};
        if ((record->type & SKIPPABLE) == 0) {
            this->cfg.pos = start;
            return {
                std::nullopt,
                x::errors::Error(UNKNOWN_TYPE, std::to_string(record->type))
            };
        }
    }
}

std::uint64_t Reader::pos() const {
    return this->cfg.pos;
}

std::uint64_t Reader::cut() const {
    return this->cut_bytes;
}
}
