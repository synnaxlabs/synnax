// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <charconv>
#include <cstring>
#include <optional>
#include <string>
#include <system_error>
#include <tuple>
#include <unordered_map>
#include <vector>

#include "x/cpp/telem/series.h"

#include "driver/spool/spool.h"
#include "driver/spool/spool.pb.h"

namespace driver::spool {
namespace {
const std::string SEGMENT_EXT = ".seg";
const std::string OFFSET_EXT = ".offset";
/// @brief marks the temp files that a crash in journal file creation can leave.
const std::string TEMP_INFIX = ".tmp.";
constexpr std::uint8_t HEADER_TYPE = 1;
constexpr std::uint8_t FRAME_TYPE = 2;

using Types = std::unordered_map<std::uint32_t, x::telem::DataType>;

x::errors::Error fs_error(const std::string &msg, const std::error_code &ec) {
    return x::errors::Error(BASE_ERROR, msg + ": " + ec.message());
}

std::string bytes_of(const x::uuid::UUID &id) {
    return {reinterpret_cast<const char *>(id.data()), x::uuid::UUID::size()};
}

std::pair<x::uuid::UUID, x::errors::Error>
parse_uuid(const std::string &bytes, const std::string &field) {
    if (bytes.size() != x::uuid::UUID::size())
        return {{}, x::errors::Error(CORRUPT, field + " is not a UUID")};
    std::array<std::uint8_t, x::uuid::UUID::size()> arr{};
    std::memcpy(arr.data(), bytes.data(), arr.size());
    return {x::uuid::UUID(arr), x::errors::NIL};
}

/// @returns why the header is not valid, or an empty string when it is.
std::string check(const Header &header) {
    if (header.indexes.size() != header.channels.size())
        return "header has " + std::to_string(header.indexes.size()) + " indexes for " +
               std::to_string(header.channels.size()) + " channels";
    if (header.data_types.size() != header.channels.size())
        return "header has " + std::to_string(header.data_types.size()) +
               " data types for " + std::to_string(header.channels.size()) +
               " channels";
    for (const auto &dt: header.data_types)
        if (!dt.is_variable() && dt.density() == 0)
            return "header has unknown data type " + dt.name();
    return "";
}

Types types_of(const Header &header) {
    Types types;
    for (std::size_t i = 0; i < header.channels.size(); i++)
        types.emplace(header.channels[i], header.data_types[i]);
    return types;
}

std::pair<std::string, x::errors::Error> encode_header(const Header &header) {
    if (const auto msg = check(header); !msg.empty())
        return {"", x::errors::Error(BASE_ERROR, msg)};
    pb::Header pb;
    pb.set_cluster(bytes_of(header.cluster));
    for (std::size_t i = 0; i < header.channels.size(); i++) {
        pb.add_channels(header.channels[i]);
        pb.add_indexes(header.indexes[i]);
        pb.add_data_types(header.data_types[i].name());
    }
    auto *subject = pb.mutable_subject();
    subject->set_key(header.subject.key);
    subject->set_name(header.subject.name);
    subject->set_group(header.subject.group);
    pb.set_task(bytes_of(header.task));
    pb.set_rack(header.rack);
    pb.set_created(header.created.nanoseconds());
    pb.set_version(header.version);
    return {pb.SerializeAsString(), x::errors::NIL};
}

std::pair<Header, x::errors::Error> decode_header(const std::string &payload) {
    pb::Header pb;
    if (!pb.ParseFromString(payload))
        return {{}, x::errors::Error(CORRUPT, "header is not valid")};
    Header header{
        .channels = {pb.channels().begin(), pb.channels().end()},
        .indexes = {pb.indexes().begin(), pb.indexes().end()},
        .subject =
            {.key = pb.subject().key(),
             .name = pb.subject().name(),
             .group = pb.subject().group()},
        .rack = pb.rack(),
        .created = x::telem::TimeStamp(pb.created()),
        .version = pb.version(),
    };
    for (const auto &dt: pb.data_types())
        header.data_types.emplace_back(dt);
    auto [cluster, err] = parse_uuid(pb.cluster(), "cluster");
    if (err) return {{}, err};
    header.cluster = cluster;
    auto [task, t_err] = parse_uuid(pb.task(), "task");
    if (t_err) return {{}, t_err};
    header.task = task;
    if (const auto msg = check(header); !msg.empty())
        return {{}, x::errors::Error(CORRUPT, msg)};
    return {std::move(header), x::errors::NIL};
}

std::pair<std::string, x::errors::Error>
encode_frame(const x::telem::Frame &frame, const Types &types) {
    pb::Frame pb;
    const auto n = frame.channels == nullptr ? 0 : frame.channels->size();
    for (std::size_t i = 0; i < n; i++) {
        const auto key = frame.channels->at(i);
        const auto &series = frame.series->at(i);
        const auto it = types.find(key);
        if (it == types.end())
            return {
                "",
                x::errors::Error(
                    BASE_ERROR,
                    "channel " + std::to_string(key) + " is not in the header"
                )
            };
        if (series.data_type() != it->second)
            return {
                "",
                x::errors::Error(
                    BASE_ERROR,
                    "channel " + std::to_string(key) + " has data type " +
                        series.data_type().name() + ", not " + it->second.name()
                )
            };
        pb.add_channels(key);
        auto *s = pb.add_series();
        s->set_data(reinterpret_cast<const char *>(series.data()), series.byte_size());
        s->set_alignment(series.alignment.uint64());
        s->set_start(series.time_range.start.nanoseconds());
        s->set_end(series.time_range.end.nanoseconds());
    }
    return {pb.SerializeAsString(), x::errors::NIL};
}

/// @returns true when data holds whole samples of the data type.
bool whole_samples(const std::string &data, const x::telem::DataType &dt) {
    if (!dt.is_variable()) return data.size() % dt.density() == 0;
    const auto *ptr = reinterpret_cast<const std::byte *>(data.data());
    for (std::size_t left = data.size(); left > 0;) {
        if (left < ::VARIABLE_LENGTH_PREFIX_SIZE) return false;
        const auto len = ::read_u32_le(ptr);
        left -= ::VARIABLE_LENGTH_PREFIX_SIZE;
        if (len > left) return false;
        left -= len;
        ptr += ::VARIABLE_LENGTH_PREFIX_SIZE + len;
    }
    return true;
}

std::pair<x::telem::Frame, x::errors::Error>
decode_frame(const std::string &payload, const Types &types) {
    pb::Frame pb;
    if (!pb.ParseFromString(payload))
        return {x::telem::Frame(), x::errors::Error(CORRUPT, "frame is not valid")};
    if (pb.channels_size() != pb.series_size())
        return {
            x::telem::Frame(),
            x::errors::Error(CORRUPT, "frame has a series count mismatch")
        };
    x::telem::Frame frame(static_cast<std::size_t>(pb.channels_size()));
    for (int i = 0; i < pb.channels_size(); i++) {
        const auto key = pb.channels(i);
        const auto it = types.find(key);
        if (it == types.end())
            return {
                x::telem::Frame(),
                x::errors::Error(
                    CORRUPT,
                    "frame channel " + std::to_string(key) + " is not in the header"
                )
            };
        auto *s = pb.mutable_series(i);
        if (!whole_samples(s->data(), it->second))
            return {
                x::telem::Frame(),
                x::errors::Error(
                    CORRUPT,
                    "series of channel " + std::to_string(key) +
                        " is not whole samples of " + it->second.name()
                )
            };
        // Series has no public constructor from raw bytes.
        x::telem::pb::Series wire;
        wire.set_data_type(it->second.name());
        wire.set_data(std::move(*s->mutable_data()));
        wire.set_alignment(s->alignment());
        wire.mutable_time_range()->set_start(s->start());
        wire.mutable_time_range()->set_end(s->end());
        auto [series, err] = x::telem::Series::from_proto(wire);
        if (err) return {x::telem::Frame(), err};
        frame.emplace(key, std::move(series));
    }
    return {std::move(frame), x::errors::NIL};
}

std::pair<std::unique_ptr<journal::Reader>, x::errors::Error>
open_segment(const std::filesystem::path &path, const std::uint64_t pos = 0) {
    return journal::Reader::open(
        {.path = path,
         .magic = SEGMENT_MAGIC,
         .types = {HEADER_TYPE, FRAME_TYPE},
         .pos = pos}
    );
}

/// @brief reads the first record of a segment.
/// @returns std::nullopt when a crash left the segment without a header.
std::pair<std::optional<Header>, x::errors::Error> read_header(journal::Reader &file) {
    auto [record, err] = file.next();
    if (err || !record) return {std::nullopt, err};
    if (record->type != HEADER_TYPE)
        return {
            std::nullopt,
            x::errors::Error(CORRUPT, "segment does not start with a header")
        };
    auto [header, h_err] = decode_header(record->payload);
    if (h_err) return {std::nullopt, h_err};
    return {std::move(header), x::errors::NIL};
}
}

Writer::Writer(Spool &spool, Header header, const std::uint64_t key):
    spool(spool), header(std::move(header)), key(key) {
    if (check(this->header).empty()) this->types = types_of(this->header);
}

Writer::~Writer() {
    this->close();
}

x::errors::Error Writer::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    if (!this->failed) this->failed = x::errors::Error(BASE_ERROR, "writer is closed");
    if (this->size == 0) return x::errors::NIL;
    const auto err = this->file->close();
    this->file.reset();
    this->spool.close(this->key, this->size);
    return err;
}

x::errors::Error Writer::write(const x::telem::Frame &frame) {
    if (this->failed) return this->failed;
    if (this->refused) {
        ++this->spool.refused_count;
        return FULL;
    }
    std::string header_payload;
    std::uint64_t bytes = 0;
    if (this->size == 0) {
        auto [encoded, err] = encode_header(this->header);
        if (err) return err;
        header_payload = std::move(encoded);
        bytes += journal::HEADER_SIZE + journal::RECORD_PREFIX_SIZE +
                 header_payload.size();
    }
    auto [payload, err] = encode_frame(frame, this->types);
    if (err) return err;
    bytes += journal::RECORD_PREFIX_SIZE + payload.size();
    if (!this->spool.reserve(bytes)) {
        this->refused = true;
        ++this->spool.refused_count;
        return FULL;
    }
    if (this->size == 0) {
        auto [file, o_err] = journal::Writer::open({
            .path = this->spool.segment_path(this->key),
            .magic = SEGMENT_MAGIC,
            .sync_interval = this->spool.cfg.sync_interval,
            .on_sync = this->spool.cfg.on_sync,
        });
        if (o_err) return this->fail(o_err, bytes);
        this->file = std::move(file);
        if (const auto w_err = this->file->write(
                {.type = HEADER_TYPE, .payload = std::move(header_payload)}
            ))
            return this->fail(w_err, bytes);
    }
    if (const auto w_err = this->file->write(
            {.type = FRAME_TYPE, .payload = std::move(payload)}
        ))
        return this->fail(w_err, bytes);
    this->size += bytes;
    return x::errors::NIL;
}

x::errors::Error Writer::sync_if_due() {
    if (this->failed || this->file == nullptr) return this->failed;
    this->failed = this->file->sync_if_due();
    return this->failed;
}

x::errors::Error
Writer::fail(const x::errors::Error &err, const std::uint64_t reserved) {
    this->failed = err;
    // The journal appends a record before it syncs, so after a failed sync the frame
    // is in the segment.
    const auto end = this->size + reserved;
    if (this->file != nullptr && this->file->pos() == end) {
        this->size = end;
        return this->failed;
    }
    this->spool.release(reserved);
    if (this->size > 0) return this->failed;
    // A segment without frames is useless to readers.
    this->file.reset();
    if (const auto rm_err = journal::remove(this->spool.segment_path(this->key)))
        this->failed = x::errors::Error(
            rm_err,
            err.message() + ", then failed to remove the segment: " + rm_err.data
        );
    return this->failed;
}

Reader::Reader(Spool &spool, const std::uint64_t key): spool(spool), key(key) {}

std::pair<std::optional<x::telem::Frame>, x::errors::Error> Reader::read() {
    if (this->file == nullptr) return {std::nullopt, x::errors::NIL};
    auto [record, err] = this->file->next();
    if (err) return {std::nullopt, err};
    if (!record) {
        this->spool.count_cut(this->key, this->file->cut());
        // Closed, so the spool can remove the segment on Windows.
        this->file.reset();
        return {std::nullopt, x::errors::NIL};
    }
    if (record->type != FRAME_TYPE)
        return {std::nullopt, x::errors::Error(CORRUPT, "segment has a second header")};
    auto [frame, d_err] = decode_frame(record->payload, this->types);
    if (d_err) return {std::nullopt, d_err};
    this->pos = this->file->pos();
    return {std::move(frame), x::errors::NIL};
}

x::errors::Error Reader::commit() {
    return journal::commit_offset(
        this->spool.offset_path(this->key),
        OFFSET_MAGIC,
        {.segment = this->key, .pos = this->pos}
    );
}

Spool::Spool(Config cfg, std::unique_ptr<journal::Lock> lock):
    cfg(std::move(cfg)), lock(std::move(lock)) {}

std::pair<std::unique_ptr<Spool>, x::errors::Error> Spool::open(Config cfg) {
    std::error_code ec;
    std::filesystem::create_directories(cfg.dir, ec);
    if (ec) return {nullptr, fs_error("failed to create spool directory", ec)};
    auto [lock, l_err] = journal::Lock::open(cfg.dir);
    if (l_err) return {nullptr, l_err};
    auto spool = std::unique_ptr<Spool>(new Spool(std::move(cfg), std::move(lock)));
    if (const auto err = spool->remove_temp_files()) return {nullptr, err};
    std::vector<std::uint64_t> keys;
    for (const auto &entry: std::filesystem::directory_iterator(spool->cfg.dir, ec)) {
        const auto &path = entry.path();
        if (path.extension() != SEGMENT_EXT) continue;
        const auto stem = path.stem().string();
        std::uint64_t key = 0;
        const auto [end, parse_ec] = std::from_chars(
            stem.data(),
            stem.data() + stem.size(),
            key
        );
        if (parse_ec != std::errc() || end != stem.data() + stem.size())
            return {nullptr, x::errors::Error(CORRUPT, "invalid segment " + stem)};
        keys.push_back(key);
    }
    if (ec) return {nullptr, fs_error("failed to list spool directory", ec)};
    for (const auto key: keys) {
        const auto path = spool->segment_path(key);
        auto [file, err] = open_segment(path);
        if (err) return {nullptr, err};
        auto [header, h_err] = read_header(*file);
        if (h_err) return {nullptr, h_err};
        if (!header) {
            spool->cut_count += file->cut();
            file.reset();
            if (const auto rm_err = spool->remove_files(key)) return {nullptr, rm_err};
            continue;
        }
        const auto size = std::filesystem::file_size(path, ec);
        if (ec) return {nullptr, fs_error("failed to stat segment", ec)};
        spool->used += size;
        spool->closed[key] = size;
        spool->next_key = std::max(spool->next_key, key + 1);
    }
    if (const auto err = spool->remove_orphan_offsets()) return {nullptr, err};
    return {std::move(spool), x::errors::NIL};
}

std::filesystem::path Spool::segment_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + SEGMENT_EXT);
}

std::filesystem::path Spool::offset_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + OFFSET_EXT);
}

bool Spool::reserve(const std::uint64_t bytes) {
    std::lock_guard lock(this->mu);
    if (this->used + bytes > this->cfg.cap) return false;
    this->used += bytes;
    return true;
}

void Spool::release(const std::uint64_t bytes) {
    std::lock_guard lock(this->mu);
    this->used -= bytes;
}

void Spool::close(const std::uint64_t key, const std::uint64_t size) {
    std::lock_guard lock(this->mu);
    this->closed[key] = size;
}

x::errors::Error Spool::remove_files(const std::uint64_t key) {
    if (const auto err = journal::remove(this->segment_path(key))) return err;
    return journal::remove(this->offset_path(key));
}

x::errors::Error Spool::remove_temp_files() const {
    std::error_code ec;
    std::vector<std::filesystem::path> temps;
    for (const auto &entry: std::filesystem::directory_iterator(this->cfg.dir, ec))
        if (entry.path().filename().string().contains(TEMP_INFIX))
            temps.push_back(entry.path());
    if (ec) return fs_error("failed to list spool directory", ec);
    for (const auto &path: temps)
        if (const auto err = journal::remove(path)) return err;
    return x::errors::NIL;
}

x::errors::Error Spool::remove_orphan_offsets() const {
    std::error_code ec;
    std::vector<std::filesystem::path> orphans;
    for (const auto &entry: std::filesystem::directory_iterator(this->cfg.dir, ec)) {
        auto segment = entry.path();
        if (segment.extension() != OFFSET_EXT) continue;
        segment.replace_extension(SEGMENT_EXT);
        if (!std::filesystem::exists(segment)) orphans.push_back(entry.path());
    }
    if (ec) return fs_error("failed to list spool directory", ec);
    for (const auto &path: orphans)
        if (const auto err = journal::remove(path)) return err;
    return x::errors::NIL;
}

std::unique_ptr<Writer> Spool::open_writer(Header header) {
    std::lock_guard lock(this->mu);
    return std::unique_ptr<Writer>(
        new Writer(*this, std::move(header), this->next_key++)
    );
}

std::vector<std::uint64_t> Spool::segments() const {
    std::lock_guard lock(this->mu);
    std::vector<std::uint64_t> keys;
    keys.reserve(this->closed.size());
    for (const auto &[key, _]: this->closed)
        keys.push_back(key);
    return keys;
}

std::pair<std::unique_ptr<Reader>, x::errors::Error>
Spool::open_reader(const std::uint64_t key) {
    const auto path = this->segment_path(key);
    auto [file, err] = open_segment(path);
    if (err) return {nullptr, err};
    auto [header, h_err] = read_header(*file);
    if (h_err) return {nullptr, h_err};
    if (!header) return {nullptr, x::errors::Error(CORRUPT, "segment has no header")};
    auto reader = std::unique_ptr<Reader>(new Reader(*this, key));
    reader->types = types_of(*header);
    reader->header = std::move(*header);
    reader->pos = file->pos();
    auto [offset, o_err] = journal::read_offset(this->offset_path(key), OFFSET_MAGIC);
    if (o_err) return {nullptr, o_err};
    if (offset) {
        if (offset->segment != key)
            return {
                nullptr,
                x::errors::Error(
                    CORRUPT,
                    "read offset is for segment " + std::to_string(offset->segment)
                )
            };
        if (offset->pos < reader->pos)
            return {nullptr, x::errors::Error(CORRUPT, "read offset is in the header")};
        std::tie(file, err) = open_segment(path, offset->pos);
        if (err) return {nullptr, err};
        reader->pos = offset->pos;
    }
    reader->file = std::move(file);
    return {std::move(reader), x::errors::NIL};
}

x::errors::Error Spool::remove(const std::uint64_t key) {
    {
        std::lock_guard lock(this->mu);
        if (!this->closed.contains(key))
            return x::errors::Error(
                BASE_ERROR,
                "segment " + std::to_string(key) + " is not closed"
            );
    }
    if (const auto err = this->remove_files(key)) return err;
    std::lock_guard lock(this->mu);
    const auto it = this->closed.find(key);
    if (it == this->closed.end()) return x::errors::NIL;
    this->used -= it->second;
    this->closed.erase(it);
    this->cut_segments.erase(key);
    return x::errors::NIL;
}

void Spool::count_cut(const std::uint64_t key, const std::uint64_t bytes) {
    std::lock_guard lock(this->mu);
    if (this->cut_segments.insert(key).second) this->cut_count += bytes;
}

std::uint64_t Spool::refused() const {
    return this->refused_count.load();
}

std::uint64_t Spool::cut() const {
    return this->cut_count.load();
}
}
