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
    return x::errors::Error(ERROR, msg + ": " + ec.message());
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
    std::uint64_t size = HEADER_SIZE;
    bool torn = false;
    if (!std::filesystem::exists(cfg.path)) {
        if (auto err = create(cfg.path, encode_header(cfg.magic, cfg.flags)))
            return {nullptr, err};
    } else {
        auto [reader, err] = Reader::open({.path = cfg.path, .magic = cfg.magic});
        if (err) return {nullptr, err};
        if (reader->header.version != VERSION)
            return {nullptr, x::errors::Error(UNKNOWN_VERSION, "cannot append")};
        while (true) {
            auto [record, r_err] = reader->next_record();
            if (r_err) return {nullptr, r_err};
            if (!record) break;
        }
        size = reader->pos();
        torn = reader->torn();
        reader.reset();
        std::error_code ec;
        if (torn) std::filesystem::resize_file(cfg.path, size, ec);
        if (ec) return {nullptr, fs_error("failed to cut torn tail", ec)};
    }
    auto [file, err] = File::open(cfg.path);
    if (err) return {nullptr, err};
    if (torn) {
        if (err = cfg.on_sync(file->sync()); err) return {nullptr, err};
    }
    return {
        std::unique_ptr<Writer>(new Writer(std::move(cfg), std::move(file), size)),
        x::errors::NIL
    };
}

x::errors::Error Writer::fail(const x::errors::Error &err) {
    this->failed = err;
    return err;
}

x::errors::Error Writer::write(const Record &record) {
    if (this->failed) return this->failed;
    if ((record.type & ~SKIPPABLE) == 0)
        return x::errors::Error(ERROR, "record type has no value");
    if (record.payload.size() > MAX_PAYLOAD)
        return x::errors::Error(ERROR, "record payload is too large");
    std::string buf;
    encode_record(buf, record);
    if (auto err = this->file->write(buf)) return this->fail(err);
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
    if (this->failed || this->unsynced == 0) return this->failed;
    // A failed fsync can drop the dirty pages, so a retry could report a false
    // success.
    if (auto err = this->cfg.on_sync(this->file->sync())) return this->fail(err);
    this->unsynced = 0;
    this->last_sync = this->cfg.now();
    return x::errors::NIL;
}

x::errors::Error Writer::close() {
    if (this->file == nullptr) return this->failed;
    const auto err = this->sync();
    this->file.reset();
    if (!this->failed) this->failed = x::errors::Error(ERROR, "writer is closed");
    return err;
}

std::uint64_t Writer::pos() const {
    return this->size;
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
        return {nullptr, x::errors::Error(ERROR, "failed to open " + path)};
    std::array<char, HEADER_SIZE> buf{};
    if (reader->size < HEADER_SIZE || !reader->file.read(buf.data(), buf.size()))
        return {nullptr, x::errors::Error(CORRUPT, path + " has no full file start")};
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

bool Reader::zero_after(const std::uint64_t pos) {
    std::array<char, 4096> buf{};
    this->file.seekg(static_cast<std::streamoff>(pos));
    for (auto left = this->size - pos; left > 0;) {
        const auto n = std::min<std::uint64_t>(left, buf.size());
        if (!this->file.read(buf.data(), static_cast<std::streamsize>(n))) return false;
        if (std::any_of(buf.begin(), buf.begin() + n, [](char c) { return c != 0; }))
            return false;
        left -= n;
    }
    return true;
}

std::pair<std::optional<Record>, x::errors::Error> Reader::next_record() {
    auto &pos = this->cfg.pos;
    if (this->torn_tail || pos == this->size) return {std::nullopt, x::errors::NIL};
    std::array<char, RECORD_PREFIX_SIZE> prefix{};
    if (this->size - pos < RECORD_PREFIX_SIZE ||
        !this->file.read(prefix.data(), RECORD_PREFIX_SIZE)) {
        this->torn_tail = true;
        return {std::nullopt, x::errors::NIL};
    }
    const auto len = get<std::uint32_t>(prefix.data());
    const auto end = pos + RECORD_PREFIX_SIZE + len;
    Record record{.type = static_cast<std::uint8_t>(prefix[8])};
    bool valid = end <= this->size && len <= MAX_PAYLOAD;
    if (valid) {
        record.payload.resize(len);
        valid = static_cast<bool>(this->file.read(record.payload.data(), len)) &&
                checksum(prefix.data(), record.type, record.payload) ==
                    get<std::uint32_t>(prefix.data() + 4);
    }
    if (!valid) {
        if (end <= this->size && !this->zero_after(end))
            return {std::nullopt, x::errors::Error(CORRUPT, "record fails checksum")};
        this->torn_tail = true;
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

bool Reader::torn() const {
    return this->torn_tail;
}
}
