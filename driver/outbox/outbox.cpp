// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <charconv>
#include <chrono>
#include <cstdint>
#include <limits>
#include <ranges>
#include <string>
#include <system_error>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"

#include "driver/outbox/outbox.h"

namespace driver::outbox {
namespace {
const std::string SEGMENT_EXT = ".calls";
const std::string OFFSET_FILE = "offset";

x::telem::TimeStamp steady_now() {
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::steady_clock::now().time_since_epoch()
        )
            .count()
    );
}

x::errors::Error fs_error(const std::string &msg, const std::error_code &ec) {
    return x::errors::Error(BASE_ERROR, msg + ": " + ec.message());
}

std::pair<pb::Call, x::errors::Error> decode(const journal::Record &record) {
    pb::Call call;
    if (!call.ParseFromString(record.payload))
        return {call, x::errors::Error(CORRUPT, "call does not decode")};
    // Only a newer Driver writes a kind this one cannot send. Refusing it keeps the
    // call for that Driver.
    if (call.call_case() == pb::Call::CALL_NOT_SET)
        return {call, x::errors::Error(UNKNOWN_VERSION, "call has an unknown kind")};
    if (call.call_case() != pb::Call::kSetStatus)
        return {
            call,
            x::errors::Error(
                UNKNOWN_VERSION,
                "this Driver cannot send call kind " +
                    std::to_string(static_cast<int>(call.call_case()))
            )
        };
    return {call, x::errors::NIL};
}

/// @returns whether err has type, or a type under it. Error::matches also matches a
/// type that only starts with the same letters.
bool is_kind(const x::errors::Error &err, const std::string &type) {
    return err.type == type || err.type.starts_with(type + ".");
}
}

bool rejected(const x::errors::Error &err) {
    // The auth middleware renews a token, and a license can come back.
    for (const auto &type:
         {std::string("sy.auth.invalid_token"),
          std::string("sy.auth.expired_token"),
          std::string("sy.auth.session_ended"),
          synnax::errors::LICENSE_MISSING.type,
          synnax::errors::LICENSE_EXPIRED.type})
        if (is_kind(err, type)) return false;
    for (const auto &type:
         {x::errors::VALIDATION.type,
          x::errors::QUERY.type,
          x::errors::CONTROL.type,
          std::string("sy.auth"),
          synnax::errors::LICENSE.type,
          std::string("arc")})
        if (is_kind(err, type)) return true;
    return false;
}

Outbox::Outbox(Config cfg): cfg(std::move(cfg)) {}

std::filesystem::path Outbox::segment_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + SEGMENT_EXT);
}

std::pair<std::shared_ptr<journal::Writer>, x::errors::Error>
Outbox::open_writer(const std::uint64_t key, const std::uint16_t flags) const {
    // The sync thread syncs the writer, so a write never waits for the disk.
    auto [writer, err] = journal::Writer::open({
        .path = this->segment_path(key),
        .magic = SEGMENT_MAGIC,
        .flags = flags,
        .sync_interval = x::telem::TimeSpan(std::numeric_limits<std::int64_t>::max()),
        .sync_bytes = std::numeric_limits<std::uint64_t>::max(),
        .max_payload = this->cfg.max_payload,
        .on_create = this->cfg.on_create,
        .on_sync = this->cfg.on_sync,
    });
    return {std::move(writer), err};
}

std::pair<std::unique_ptr<Outbox>, x::errors::Error> Outbox::open(Config cfg) {
    if (cfg.compact_share <= 0 || cfg.compact_share > 1)
        return {
            nullptr,
            x::errors::Error(BASE_ERROR, "compact_share is not in (0, 1]")
        };
    std::error_code ec;
    std::vector<std::filesystem::path> new_dirs;
    for (auto d = cfg.dir; !d.empty() && !std::filesystem::exists(d, ec) && !ec;
         d = d.parent_path()) {
        new_dirs.push_back(d);
        if (d == d.parent_path()) break;
    }
    if (ec) return {nullptr, fs_error("failed to stat outbox directory", ec)};
    std::filesystem::create_directories(cfg.dir, ec);
    if (ec) return {nullptr, fs_error("failed to create outbox directory", ec)};
    for (const auto &d: std::views::reverse(new_dirs))
        if (const auto err = cfg.on_create(d)) return {nullptr, err};
    auto [dir_lock, lock_err] = journal::Lock::open(cfg.dir, cfg.on_create);
    if (lock_err) return {nullptr, lock_err};
    auto outbox = std::unique_ptr<Outbox>(new Outbox(std::move(cfg)));
    outbox->dir_lock = std::move(dir_lock);
    std::vector<std::filesystem::path> stale;
    std::filesystem::directory_iterator it(outbox->cfg.dir, ec);
    for (; !ec && it != std::filesystem::directory_iterator(); it.increment(ec)) {
        const auto &path = it->path();
        // A crash while a journal file is created leaves its temporary file.
        if (path.filename().string().contains(".tmp.")) {
            stale.push_back(path);
            continue;
        }
        if (path.extension() != SEGMENT_EXT) continue;
        const auto stem = path.stem().string();
        std::uint64_t key = 0;
        const auto [end, parse_ec] = std::from_chars(
            stem.data(),
            stem.data() + stem.size(),
            key
        );
        if (parse_ec != std::errc() || end != stem.data() + stem.size())
            return {nullptr, x::errors::Error(BASE_ERROR, "invalid segment " + stem)};
        std::error_code size_ec;
        const auto size = std::filesystem::file_size(path, size_ec);
        outbox->segments[key] = size_ec ? 0 : size;
    }
    if (ec) return {nullptr, fs_error("failed to list outbox directory", ec)};
    for (const auto &path: stale)
        if (std::filesystem::remove(path, ec); ec)
            return {nullptr, fs_error("failed to remove " + path.string(), ec)};
    const auto [saved, offset_err] = journal::read_offset(
        outbox->cfg.dir / OFFSET_FILE,
        OFFSET_MAGIC
    );
    if (offset_err) return {nullptr, offset_err};
    outbox->saved = saved;
    const auto offset = saved.value_or(journal::Offset{});
    // New segments go after the saved offset, which outlives its sent segments.
    if (saved.has_value()) outbox->next_key = offset.segment + 1;
    for (auto seg = outbox->segments.begin(); seg != outbox->segments.end();) {
        const auto key = seg->first;
        outbox->next_key = std::max(outbox->next_key, key + 1);
        if (key < offset.segment || (saved.has_value() && key == offset.segment)) {
            ++seg;
            continue;
        }
        auto [reader, err] = journal::Reader::open(
            {.path = outbox->segment_path(key), .magic = SEGMENT_MAGIC}
        );
        if (err) return {nullptr, err};
        const auto compacted = (reader->header.flags & COMPACTED_FLAG) != 0;
        reader.reset();
        // A compaction saves the offset at its segment once the segment is whole.
        if (!compacted) {
            ++seg;
            continue;
        }
        if (err = journal::remove(outbox->segment_path(key)); err)
            return {nullptr, err};
        seg = outbox->segments.erase(seg);
    }
    for (const auto &[key, size]: outbox->segments) {
        if (key < offset.segment) continue;
        const auto pos = key == offset.segment ? offset.pos : 0;
        if (const auto err = outbox->load(key, pos)) return {nullptr, err};
    }
    for (const auto &[key, size]: outbox->segments)
        outbox->bytes += size;
    const auto tail_key = outbox->next_key++;
    auto [tail, tail_err] = outbox->open_writer(tail_key);
    if (tail_err) return {nullptr, tail_err};
    outbox->tail = {.key = tail_key, .writer = tail};
    outbox->writers[tail_key] = tail;
    outbox->segments[tail_key] = tail->pos();
    outbox->bytes += tail->pos();
    outbox->spare_wanted = true;
    outbox->remove_sent();
    outbox->sender = std::thread([ptr = outbox.get()] { ptr->run(); });
    outbox->syncer = std::thread([ptr = outbox.get()] { ptr->run_syncs(); });
    return {std::move(outbox), x::errors::NIL};
}

x::errors::Error Outbox::load(const std::uint64_t key, const std::uint64_t pos) {
    const auto path = this->segment_path(key);
    {
        // Cuts a call that a crash cut off, and syncs what an earlier process wrote.
        auto [writer, err] = this->open_writer(key);
        if (err) return err;
        if (const auto cut = writer->cut(); cut > 0) {
            this->cut_bytes += cut;
            LOG(WARNING) << "[outbox] lost " << cut << " bytes at the end of "
                         << path.string();
        }
        this->segments[key] = writer->pos();
        if (err = writer->close(); err) return err;
    }
    auto [reader, err] = journal::Reader::open({
        .path = path,
        .magic = SEGMENT_MAGIC,
        .types = {CALL_TYPE},
        .pos = pos,
    });
    while (!err) {
        auto [record, next_err] = reader->next();
        if (next_err) {
            err = next_err;
            break;
        }
        if (!record) return x::errors::NIL;
        auto [request, decode_err] = decode(*record);
        if (decode_err) return decode_err;
        this->enqueue({
            .request = std::make_shared<const pb::Call>(std::move(request)),
            .set_status_done = nullptr,
            .segment = key,
            .end = reader->pos(),
            .seq = this->next_seq++,
            .size = journal::RECORD_PREFIX_SIZE + record->payload.size(),
        });
    }
    // The whole segment passed its checks, so only the saved position can be wrong.
    if (pos > 0 && err.matches(journal::CORRUPT))
        return x::errors::Error(
            journal::CORRUPT,
            "the saved send position of the outbox is not a call in " + path.string() +
                ". Delete " + (this->cfg.dir / OFFSET_FILE).string() +
                " to send all of its calls again (" + err.message() + ")"
        );
    return err;
}

void Outbox::enqueue(Call call) {
    if (call.request->call_case() == pb::Call::kSetStatus) {
        const auto [it, inserted] = this->latest.try_emplace(
            call.request->set_status().key_or_name(),
            Newest{.seq = call.seq, .size = call.size}
        );
        if (!inserted) {
            ++this->replaced;
            this->replaced_bytes += it->second.size;
            it->second = {.seq = call.seq, .size = call.size};
        }
    }
    this->queued_bytes += call.size;
    this->calls.push_back(std::move(call));
}

bool Outbox::reclaimable() const {
    // A compaction leaves the calls no newer set replaced, in the compacted segment,
    // the tail, and the spare.
    const auto kept = this->queued_bytes - this->replaced_bytes +
                      3 * journal::HEADER_SIZE;
    if (this->bytes <= kept) return false;
    const auto freed = static_cast<double>(this->bytes - kept);
    const auto free = this->cfg.max_bytes > kept ? this->cfg.max_bytes - kept : 0;
    return freed >=
           std::min(
               this->cfg.compact_share * static_cast<double>(this->cfg.max_bytes),
               static_cast<double>(free) / 2
           );
}

void Outbox::unreplace(const Call &call) {
    --this->replaced;
    this->replaced_bytes -= call.size;
}

bool Outbox::is_replaced(const Call &call) const {
    if (call.request->call_case() != pb::Call::kSetStatus) return false;
    return this->latest.at(call.request->set_status().key_or_name()).seq != call.seq;
}

Outbox::~Outbox() {
    // A failed open never started the threads.
    if (!this->sender.joinable()) return;
    {
        std::lock_guard lock(this->mu);
        this->stopped = true;
    }
    this->cv.notify_all();
    this->sender.join();
    {
        std::lock_guard lock(this->mu);
        this->sync_stopped = true;
    }
    this->sync_cv.notify_all();
    this->syncer.join();
}

x::errors::Error Outbox::set_status(
    const ::grpc::status::SetByKeyOrNameRequest &req,
    SetStatusDone done
) {
    pb::Call request;
    auto *set = request.mutable_set_status();
    set->set_key_or_name(req.key_or_name());
    set->set_message(req.message());
    set->set_variant(req.variant());
    set->set_time(req.time());
    return this->push(std::move(request), std::move(done));
}

x::errors::Error Outbox::push(pb::Call request, SetStatusDone done) {
    const auto payload = request.SerializeAsString();
    if (payload.size() > this->cfg.max_payload)
        return x::errors::Error(
            journal::TOO_LARGE,
            "the call has " + std::to_string(payload.size()) +
                " bytes, and the outbox takes at most " +
                std::to_string(this->cfg.max_payload)
        );
    const auto size = journal::RECORD_PREFIX_SIZE + payload.size();
    auto shared = std::make_shared<const pb::Call>(std::move(request));
    {
        std::unique_lock lock(this->mu);
        if (this->tail.writer == nullptr || this->tail_calls >= this->cfg.segment_calls)
            if (const auto err = this->next_tail(lock)) return err;
        if (this->bytes + size > this->cfg.max_bytes) {
            ++this->refused_count;
            if (this->reclaimable()) {
                this->compact_wanted = true;
                this->sync_cv.notify_one();
            }
            return x::errors::Error(
                FULL,
                "The Driver holds " + std::to_string(this->cfg.max_bytes) +
                    " bytes of calls that the Core has not taken. Check the "
                    "connection to the Core."
            );
        }
        if (const auto err = this->tail.writer->write(
                {.type = CALL_TYPE, .payload = payload}
            )) {
            // The call is not in the segment, or only cut off.
            this->closing.push_back(this->tail);
            this->tail.writer = nullptr;
            this->sync_cv.notify_one();
            return err;
        }
        const auto end = this->tail.writer->pos();
        this->bytes += end - this->segments[this->tail.key];
        this->segments[this->tail.key] = end;
        this->enqueue({
            .request = std::move(shared),
            .set_status_done = std::move(done),
            .segment = this->tail.key,
            .end = end,
            .seq = this->next_seq++,
            .size = size,
        });
        ++this->tail_calls;
        // Compacting before the files are full keeps the newest sets from refusal.
        if (this->bytes >= this->cfg.max_bytes / 4 * 3 && this->reclaimable()) {
            this->compact_wanted = true;
            this->sync_cv.notify_one();
        }
    }
    this->cv.notify_one();
    return x::errors::NIL;
}

x::errors::Error Outbox::next_tail(std::unique_lock<std::mutex> &lock) {
    if (!this->spare.has_value()) {
        this->spare_wanted = true;
        this->sync_cv.notify_one();
        this->spare_cv.wait_for(lock, this->cfg.sync_interval.chrono(), [this] {
            return this->spare.has_value() ||
                   (this->tail.writer != nullptr &&
                    this->tail_calls < this->cfg.segment_calls);
        });
        // A full tail takes more calls when the sync thread is slow to open a spare.
        if (!this->spare.has_value()) {
            if (this->tail.writer != nullptr) return x::errors::NIL;
            return x::errors::Error(BASE_ERROR, "no segment is open to take the call");
        }
    }
    if (this->tail.writer != nullptr) this->closing.push_back(this->tail);
    this->tail = *this->spare;
    this->tail_calls = 0;
    this->spare.reset();
    this->spare_wanted = true;
    this->sync_cv.notify_one();
    return x::errors::NIL;
}

void Outbox::fail_open(const std::uint64_t key) {
    // A failed open can leave the file, which remove_sent removes once it is sent.
    this->segments.try_emplace(key, 0);
    this->failed_key = key;
}

void Outbox::run() {
    std::size_t failures = 0;
    auto failing_since = steady_now();
    bool stuck = false;
    while (true) {
        Call call;
        bool skip = false;
        {
            std::unique_lock lock(this->mu);
            this->cv.wait(lock, [this] {
                return this->stopped || !this->calls.empty();
            });
            if (this->stopped) return;
            call = this->calls.front();
            skip = this->is_replaced(call);
        }
        if (skip) {
            this->commit(call.seq);
            continue;
        }
        const auto err = this->send(call);
        if (err && !rejected(err)) {
            if (++failures == 1) {
                failing_since = steady_now();
                LOG(WARNING) << "[outbox] failed to send a call, retrying: " << err;
            }
            if (!stuck && steady_now() - failing_since >= this->cfg.stuck_after) {
                stuck = true;
                this->cfg.on_stuck(err);
            }
            if (!this->wait_retry(failures)) return;
            continue;
        }
        if (failures > 0) LOG(INFO) << "[outbox] sent a call after a retry";
        if (stuck) this->cfg.on_stuck(x::errors::NIL);
        stuck = false;
        failures = 0;
        if (err) {
            ++this->dropped_count;
            LOG(WARNING) << "[outbox] dropped a call the Core rejected: " << err;
        }
        this->commit(call.seq);
    }
}

bool Outbox::wait_retry(const std::size_t failures) {
    std::unique_lock lock(this->mu);
    this->cv.wait_for(lock, this->cfg.retry.interval(failures).chrono(), [this] {
        return this->stopped;
    });
    return !this->stopped;
}

x::errors::Error Outbox::send(const Call &call) const {
    const auto &set = call.request->set_status();
    auto [res, err] = this->cfg.statuses.set_by_key_or_name(
        set.key_or_name(),
        set.message(),
        set.variant(),
        x::telem::TimeStamp(set.time()),
        this->cfg.send_timeout
    );
    if ((!err || rejected(err)) && call.set_status_done != nullptr)
        call.set_status_done(res, err);
    return err;
}

void Outbox::commit(const std::uint64_t seq) {
    std::lock_guard lock(this->mu);
    // A compaction can drop a replaced call while the send thread skips it, and it can
    // move a call to a new segment.
    if (this->calls.empty() || this->calls.front().seq != seq) return;
    const auto done = std::move(this->calls.front());
    this->calls.pop_front();
    this->queued_bytes -= done.size;
    if (done.request->call_case() == pb::Call::kSetStatus) {
        const auto it = this->latest.find(done.request->set_status().key_or_name());
        if (it->second.seq == done.seq)
            this->latest.erase(it);
        else
            this->unreplace(done);
    }
    this->sent = journal::Offset{.segment = done.segment, .pos = done.end};
    if (done.segment != this->tail.key &&
        (this->calls.empty() || this->calls.front().segment != done.segment)) {
        this->segment_sent = true;
        this->sync_cv.notify_one();
    }
}

void Outbox::run_syncs() {
    auto last_sync = steady_now();
    auto last_save = last_sync;
    while (true) {
        bool stopping = false;
        {
            std::unique_lock lock(this->mu);
            this->sync_cv.wait_for(
                lock,
                (this->cfg.sync_interval / std::int64_t{4}).chrono(),
                [this] {
                    return this->sync_stopped || !this->closing.empty() ||
                           this->segment_sent || this->spare_wanted ||
                           this->compact_wanted;
                }
            );
            stopping = this->sync_stopped;
        }
        this->sync_once(stopping, last_sync, last_save);
        if (stopping) return;
    }
}

void Outbox::sync_once(
    const bool stopping,
    x::telem::TimeStamp &last_sync,
    x::telem::TimeStamp &last_save
) {
    std::vector<Segment> to_close;
    std::shared_ptr<journal::Writer> tail_writer;
    std::optional<std::uint64_t> spare_key;
    bool compacting = false;
    {
        std::lock_guard lock(this->mu);
        std::swap(to_close, this->closing);
        tail_writer = this->tail.writer;
        compacting = this->compact_wanted && !stopping;
        if (stopping) {
            to_close.push_back(this->tail);
            this->tail.writer = nullptr;
            if (this->spare.has_value()) to_close.push_back(*this->spare);
            this->spare.reset();
        } else if (!this->spare.has_value()) {
            if (this->failed_key.has_value() && *this->failed_key > this->tail.key)
                spare_key = this->failed_key;
            else
                spare_key = this->next_key++;
            this->failed_key.reset();
        }
        this->spare_wanted = false;
    }
    // A push can wait for the spare, so it opens first.
    if (spare_key.has_value()) {
        auto [writer, err] = this->open_writer(*spare_key);
        if (err) LOG(ERROR) << "[outbox] failed to open a segment: " << err;
        std::unique_lock lock(this->mu);
        if (err)
            this->fail_open(*spare_key);
        else if (*spare_key > this->tail.key) {
            this->spare = Segment{.key = *spare_key, .writer = writer};
            this->writers[*spare_key] = writer;
            this->bytes += writer->pos();
            this->segments[*spare_key] = writer->pos();
            this->spare_cv.notify_all();
        } else {
            lock.unlock();
            // The send thread or a compaction took a later segment first.
            (void) writer->close();
            if (const auto rm_err = journal::remove(this->segment_path(*spare_key)))
                LOG(WARNING) << "[outbox] failed to remove an unused segment: "
                             << rm_err;
        }
    }
    // A closed segment can be one that a compaction or a send left behind.
    if (this->close_segments(to_close)) this->remove_sent();
    const auto now = steady_now();
    if (!stopping && tail_writer != nullptr &&
        now - last_sync >= this->cfg.sync_interval) {
        last_sync = now;
        // A failed sync also fails the next write, which moves the tail.
        if (const auto err = tail_writer->sync())
            LOG(ERROR) << "[outbox] failed to sync a segment: " << err;
    }
    if (compacting) this->compact();
    std::optional<journal::Offset> offset;
    {
        std::lock_guard lock(this->mu);
        const auto due = stopping || this->segment_sent ||
                         now - last_save >= this->cfg.sync_interval;
        this->segment_sent = false;
        if (due && this->sent.has_value()) {
            offset = this->sent;
            // A saved offset never passes bytes that are not on the disk.
            if (const auto it = this->writers.find(offset->segment);
                it != this->writers.end())
                offset->pos = std::min(offset->pos, it->second->synced_pos());
        }
    }
    if (!offset.has_value() ||
        (this->saved.has_value() && this->saved->segment == offset->segment &&
         this->saved->pos == offset->pos))
        return;
    last_save = now;
    if (const auto err = this->save(*offset)) {
        LOG(ERROR) << "[outbox] failed to save the send position: " << err;
        return;
    }
    this->saved = offset;
    this->remove_sent();
}

bool Outbox::close_segments(const std::vector<Segment> &segments) {
    for (const auto &segment: segments) {
        if (segment.writer == nullptr) continue;
        const auto err = segment.writer->close();
        if (err)
            LOG(ERROR) << "[outbox] failed to close " << this->segment_path(segment.key)
                       << ": " << err;
        std::lock_guard lock(this->mu);
        if (!err) this->writers.erase(segment.key);
    }
    return !segments.empty();
}

x::errors::Error Outbox::save(const journal::Offset &offset) const {
    return this->cfg.on_save(
        journal::commit_offset(
            this->cfg.dir / OFFSET_FILE,
            OFFSET_MAGIC,
            offset,
            this->cfg.on_create
        )
    );
}

void Outbox::compact() {
    std::uint64_t key = 0;
    std::uint64_t tail_key = 0;
    {
        std::lock_guard lock(this->mu);
        this->compact_wanted = false;
        if (!this->reclaimable()) return;
        key = this->next_key++;
        tail_key = this->next_key++;
    }
    // Calls that come during the compaction go to a tail after the new segment.
    auto [tail_writer, tail_err] = this->open_writer(tail_key);
    if (tail_err) {
        LOG(ERROR) << "[outbox] failed to open a segment: " << tail_err;
        std::lock_guard lock(this->mu);
        this->fail_open(tail_key);
        return;
    }
    std::vector<std::pair<std::uint64_t, std::shared_ptr<const pb::Call>>> kept;
    std::uint64_t first_new_seq = 0;
    {
        std::lock_guard lock(this->mu);
        first_new_seq = this->next_seq;
        // The first call can be in flight on the send thread, so it stays.
        for (const auto &call: this->calls)
            if (kept.empty() || !this->is_replaced(call))
                kept.emplace_back(call.seq, call.request);
        if (this->tail.writer != nullptr) this->closing.push_back(this->tail);
        if (this->spare.has_value()) this->closing.push_back(*this->spare);
        this->spare.reset();
        this->spare_wanted = true;
        this->tail = {.key = tail_key, .writer = tail_writer};
        this->tail_calls = 0;
        this->writers[tail_key] = tail_writer;
        this->segments[tail_key] = tail_writer->pos();
        this->bytes += tail_writer->pos();
    }
    this->spare_cv.notify_all();
    auto [writer, err] = this->open_writer(key, COMPACTED_FLAG);
    std::vector<std::uint64_t> ends;
    for (std::size_t i = 0; !err && i < kept.size(); i++) {
        err = writer->write(
            {.type = CALL_TYPE, .payload = kept[i].second->SerializeAsString()}
        );
        ends.push_back(writer->pos());
    }
    const auto size = writer == nullptr ? 0 : writer->pos();
    if (!err) err = writer->close();
    if (err) {
        LOG(ERROR) << "[outbox] failed to compact: " << err;
        writer.reset();
        this->remove_segments({key});
        return;
    }
    // The new segment and the offset at its start are on the disk before the old
    // segments go, so a crash leaves the old calls or the new ones.
    const auto offset = journal::Offset{.segment = key, .pos = journal::HEADER_SIZE};
    if (err = this->save(offset); err) {
        LOG(ERROR) << "[outbox] failed to save the offset of a compaction: " << err;
        // The offset can be on the disk, so the new segment stays, and the next save
        // writes the offset again.
        std::lock_guard lock(this->mu);
        this->segments[key] = size;
        this->bytes += size;
        this->saved.reset();
        return;
    }
    this->saved = offset;
    std::vector<Segment> old;
    std::vector<std::uint64_t> old_keys;
    {
        std::lock_guard lock(this->mu);
        this->segments[key] = size;
        this->bytes += size;
        std::deque<Call> next;
        std::size_t i = 0;
        for (auto &call: this->calls) {
            while (i < kept.size() && kept[i].first < call.seq)
                i++;
            if (i < kept.size() && kept[i].first == call.seq) {
                call.segment = key;
                call.end = ends[i];
                next.push_back(std::move(call));
            } else if (call.seq >= first_new_seq)
                next.push_back(std::move(call));
            else {
                this->unreplace(call);
                this->queued_bytes -= call.size;
            }
        }
        this->calls = std::move(next);
        if (this->sent.has_value() && this->sent->segment < key) this->sent = offset;
        // The old segments stop counting toward max_bytes before their files go.
        for (auto it = this->segments.begin();
             it != this->segments.end() && it->first < key;) {
            old_keys.push_back(it->first);
            this->bytes -= it->second;
            this->writers.erase(it->first);
            it = this->segments.erase(it);
        }
        std::swap(old, this->closing);
    }
    this->close_segments(old);
    this->remove_segments(old_keys);
}

void Outbox::remove_segments(const std::vector<std::uint64_t> &keys) const {
    std::error_code ec;
    for (const auto key: keys)
        // The next open removes a file that fails to remove here.
        if (std::filesystem::remove(this->segment_path(key), ec); ec)
            LOG(WARNING) << "[outbox] failed to remove " << this->segment_path(key)
                         << ": " << ec.message();
    if (const auto err = journal::sync_dir(this->cfg.dir))
        LOG(WARNING) << "[outbox] failed to sync " << this->cfg.dir << ": " << err;
}

void Outbox::remove_sent() {
    std::vector<std::uint64_t> sent_keys;
    {
        std::lock_guard lock(this->mu);
        const auto first = this->calls.empty() ? this->tail.key
                                               : this->calls.front().segment;
        for (const auto &[key, size]: this->segments) {
            if (key >= first) break;
            // A segment the sync thread has not closed yet stays.
            if (std::ranges::any_of(this->closing, [key](const Segment &s) {
                    return s.key == key;
                }))
                continue;
            sent_keys.push_back(key);
        }
    }
    for (const auto key: sent_keys) {
        // A file another process holds open can fail to remove on Windows, and the next
        // save tries again.
        if (const auto err = journal::remove(this->segment_path(key))) {
            LOG(WARNING) << "[outbox] failed to remove a sent segment: " << err;
            continue;
        }
        std::lock_guard lock(this->mu);
        this->bytes -= this->segments[key];
        this->segments.erase(key);
        this->writers.erase(key);
    }
}

std::uint64_t Outbox::dropped() const {
    return this->dropped_count.load();
}

std::uint64_t Outbox::cut() const {
    return this->cut_bytes;
}

std::uint64_t Outbox::refused() const {
    return this->refused_count.load();
}

std::size_t Outbox::size() const {
    std::lock_guard lock(this->mu);
    return this->calls.size();
}
}
