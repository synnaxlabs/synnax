// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <span>
#include <system_error>

#include "absl/crc/crc32c.h"

#include "x/cpp/binary/binary.h"

#include "driver/journal/file.h"
#include "driver/journal/journal.h"

namespace driver::journal {
namespace {
constexpr std::uint8_t OFFSET_TYPE = 1;
constexpr std::uint32_t OFFSET_SIZE = 16;

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

std::uint32_t
checksum(const char *len, const std::uint8_t type, const std::string_view payload) {
    auto crc = absl::ComputeCrc32c({len, sizeof(std::uint32_t)});
    crc = absl::ExtendCrc32c(crc, {reinterpret_cast<const char *>(&type), 1});
    return static_cast<std::uint32_t>(absl::ExtendCrc32c(crc, payload));
}

std::string encode_header(const Magic &magic, const std::uint16_t flags) {
    std::string buf(magic.data(), magic.size());
    put(buf, VERSION);
    put(buf, flags);
    return buf;
}

void encode_record(std::string &buf, const Record &record) {
    const auto start = buf.size();
    put(buf, static_cast<std::uint32_t>(record.payload.size()));
    put(buf, checksum(buf.data() + start, record.type, record.payload));
    buf.push_back(static_cast<char>(record.type));
    buf.append(record.payload);
}

x::errors::Error fs_error(const std::string &msg, const std::error_code &ec) {
    return x::errors::Error(BASE_ERROR, msg + ": " + ec.message());
}
}

Writer::Writer(WriterConfig cfg, std::unique_ptr<File> file, const std::uint64_t size):
    cfg(std::move(cfg)),
    file(std::move(file)),
    size(size),
    last_sync(this->cfg.now()) {}

Writer::~Writer() {
    this->close();
}

std::pair<std::unique_ptr<Writer>, x::errors::Error> Writer::open(WriterConfig cfg) {
    auto [file, err] = File::open(cfg.path);
    if (err) return {nullptr, err};
    if (err = file->lock(); err) return {nullptr, err};
    std::error_code ec;
    const auto start = std::filesystem::file_size(cfg.path, ec);
    if (ec) return {nullptr, fs_error("failed to stat " + cfg.path.string(), ec)};
    // A file shorter than a file start is new, or a crash cut it before any record.
    if (start < HEADER_SIZE) {
        if (err = file->truncate(0); err) return {nullptr, err};
        if (err = file->write(encode_header(cfg.magic, cfg.flags)); err)
            return {nullptr, err};
        if (err = cfg.on_sync(file->sync()); err) return {nullptr, err};
        const auto dir = cfg.path.parent_path();
        if (err = sync_dir(dir.empty() ? "." : dir); err) return {nullptr, err};
    }
    auto [reader, r_err] = Reader::open({.path = cfg.path, .magic = cfg.magic});
    if (r_err) return {nullptr, r_err};
    if (reader->header.version != VERSION)
        return {nullptr, x::errors::Error(UNKNOWN_VERSION, "cannot append")};
    while (true) {
        auto [record, n_err] = reader->next_record();
        if (n_err) return {nullptr, n_err};
        if (!record) break;
    }
    const auto size = reader->pos();
    const auto cut = reader->cut();
    reader.reset();
    if (cut > 0) {
        if (err = file->truncate(size); err) return {nullptr, err};
        if (err = cfg.on_sync(file->sync()); err) return {nullptr, err};
    }
    auto writer = std::unique_ptr<Writer>(
        new Writer(std::move(cfg), std::move(file), size)
    );
    writer->cut_bytes = cut;
    return {std::move(writer), x::errors::NIL};
}

x::errors::Error Writer::write(const Record &record) {
    if (this->failed) return this->failed;
    if ((record.type & ~SKIPPABLE) == 0)
        return x::errors::Error(BASE_ERROR, "record type has no value");
    if (record.payload.size() > MAX_PAYLOAD)
        return x::errors::Error(BASE_ERROR, "record payload is too large");
    std::string buf;
    encode_record(buf, record);
    if (auto err = this->file->write(buf)) {
        this->failed = err;
        return err;
    }
    this->size += buf.size();
    this->unsynced += buf.size();
    return this->sync_if_due();
}

x::errors::Error Writer::sync_if_due() {
    if (this->failed || this->unsynced == 0) return this->failed;
    if (this->unsynced >= this->cfg.sync_bytes ||
        this->cfg.now() - this->last_sync >= this->cfg.sync_interval)
        return this->sync();
    return x::errors::NIL;
}

x::errors::Error Writer::sync() {
    if (this->file == nullptr || this->sync_failed) return this->failed;
    if (this->unsynced > 0) {
        if (auto err = this->cfg.on_sync(this->file->sync())) {
            this->sync_failed = err;
            if (!this->failed) this->failed = err;
            return err;
        }
        this->unsynced = 0;
        this->last_sync = this->cfg.now();
    }
    return this->failed;
}

x::errors::Error Writer::close() {
    if (this->file == nullptr) return this->failed;
    const auto err = this->sync();
    this->file.reset();
    if (!this->failed) this->failed = x::errors::Error(BASE_ERROR, "writer is closed");
    return err;
}

std::uint64_t Writer::pos() const {
    return this->size;
}

std::uint64_t Writer::cut() const {
    return this->cut_bytes;
}

Reader::Reader(ReaderConfig cfg): cfg(std::move(cfg)) {}

std::pair<std::unique_ptr<Reader>, x::errors::Error> Reader::open(ReaderConfig cfg) {
    const auto path = cfg.path.string();
    auto reader = std::unique_ptr<Reader>(new Reader(std::move(cfg)));
    std::error_code ec;
    reader->size = std::filesystem::file_size(reader->cfg.path, ec);
    if (ec) return {nullptr, fs_error("failed to stat " + path, ec)};
    reader->file.open(reader->cfg.path, std::ios::binary);
    if (!reader->file)
        return {nullptr, x::errors::Error(BASE_ERROR, "failed to open " + path)};
    if (reader->size < HEADER_SIZE) {
        reader->cfg.pos = 0;
        reader->cut_bytes = reader->size;
        return {std::move(reader), x::errors::NIL};
    }
    std::array<char, HEADER_SIZE> buf{};
    if (!reader->file.read(buf.data(), buf.size()))
        return {nullptr, x::errors::Error(BASE_ERROR, "failed to read " + path)};
    const auto &magic = reader->cfg.magic;
    if (!std::equal(magic.begin(), magic.end(), buf.begin()))
        return {nullptr, x::errors::Error(UNKNOWN_MAGIC, path)};
    reader->header.version = get<std::uint16_t>(buf.data() + magic.size());
    reader->header.flags = get<std::uint16_t>(buf.data() + magic.size() + 2);
    if (reader->header.version == 0 || reader->header.version > VERSION)
        return {
            nullptr,
            x::errors::Error(
                UNKNOWN_VERSION,
                path + " has version " + std::to_string(reader->header.version)
            )
        };
    auto &pos = reader->cfg.pos;
    if (pos == 0) pos = HEADER_SIZE;
    if (pos < HEADER_SIZE || pos > reader->size)
        return {nullptr, x::errors::Error(CORRUPT, "position is outside " + path)};
    reader->file.seekg(static_cast<std::streamoff>(pos));
    return {std::move(reader), x::errors::NIL};
}

std::pair<std::optional<Record>, x::errors::Error> Reader::next_record() {
    auto &pos = this->cfg.pos;
    if (this->cut_bytes > 0 || pos == this->size) return {std::nullopt, x::errors::NIL};
    std::array<char, RECORD_PREFIX_SIZE> prefix{};
    bool valid = this->size - pos >= RECORD_PREFIX_SIZE &&
                 this->file.read(prefix.data(), RECORD_PREFIX_SIZE);
    const auto len = get<std::uint32_t>(prefix.data());
    const auto end = pos + RECORD_PREFIX_SIZE + len;
    Record record{.type = static_cast<std::uint8_t>(prefix[8])};
    if (valid) valid = len <= MAX_PAYLOAD && end <= this->size;
    if (valid) {
        record.payload.resize(len);
        valid = this->file.read(record.payload.data(), len) &&
                checksum(prefix.data(), record.type, record.payload) ==
                    get<std::uint32_t>(prefix.data() + 4);
    }
    if (!valid) {
        this->cut_bytes = this->size - pos;
        return {std::nullopt, x::errors::NIL};
    }
    if ((record.type & ~SKIPPABLE) == 0)
        return {std::nullopt, x::errors::Error(CORRUPT, "record type has no value")};
    pos = end;
    return {std::move(record), x::errors::NIL};
}

std::pair<std::optional<Record>, x::errors::Error> Reader::next() {
    while (true) {
        auto [record, err] = this->next_record();
        if (err || !record || this->cfg.types.contains(record->type))
            return {std::move(record), err};
        if ((record->type & SKIPPABLE) == 0)
            return {
                std::nullopt,
                x::errors::Error(UNKNOWN_TYPE, std::to_string(record->type))
            };
    }
}

std::uint64_t Reader::pos() const {
    return this->cfg.pos;
}

std::uint64_t Reader::cut() const {
    return this->cut_bytes;
}

std::pair<std::optional<Offset>, x::errors::Error>
read_offset(const std::filesystem::path &path, const Magic &magic) {
    std::error_code ec;
    if (!std::filesystem::exists(path, ec)) {
        if (ec) return {std::nullopt, fs_error("failed to stat " + path.string(), ec)};
        return {std::nullopt, x::errors::NIL};
    }
    auto [reader, err] = Reader::open(
        {.path = path, .magic = magic, .types = {OFFSET_TYPE}}
    );
    if (err) return {std::nullopt, err};
    auto [record, r_err] = reader->next();
    if (r_err) return {std::nullopt, r_err};
    if (!record || record->payload.size() != OFFSET_SIZE)
        return {std::nullopt, x::errors::Error(CORRUPT, path.string())};
    // The file is renamed into place whole, so any bytes after the record are corrupt.
    auto [extra, e_err] = reader->next();
    if (e_err || extra || reader->cut() > 0)
        return {std::nullopt, x::errors::Error(CORRUPT, path.string())};
    return {
        Offset{
            .segment = get<std::uint64_t>(record->payload.data()),
            .pos = get<std::uint64_t>(record->payload.data() + 8),
        },
        x::errors::NIL
    };
}

x::errors::Error commit_offset(
    const std::filesystem::path &path,
    const Magic &magic,
    const Offset &offset
) {
    Record record{.type = OFFSET_TYPE};
    put(record.payload, offset.segment);
    put(record.payload, offset.pos);
    auto buf = encode_header(magic, 0);
    encode_record(buf, record);
    return create(path, buf);
}

x::errors::Error remove(const std::filesystem::path &path) {
    std::error_code ec;
    std::filesystem::remove(path, ec);
    if (ec) return fs_error("failed to remove " + path.string(), ec);
    const auto dir = path.parent_path();
    return sync_dir(dir.empty() ? "." : dir);
}

Lock::Lock(std::unique_ptr<File> file): file(std::move(file)) {}

Lock::~Lock() = default;

std::pair<std::unique_ptr<Lock>, x::errors::Error>
Lock::open(const std::filesystem::path &dir) {
    auto [file, err] = File::open(dir / "LOCK");
    if (err) return {nullptr, err};
    if (err = file->lock(); err.matches(LOCKED))
        return {nullptr, x::errors::Error(LOCKED, dir.string() + " is in use")};
    if (err) return {nullptr, err};
    return {std::unique_ptr<Lock>(new Lock(std::move(file))), x::errors::NIL};
}
}
