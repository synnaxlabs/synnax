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
/// @brief the number of calls in a segment, after which the next segment starts.
constexpr std::size_t SEGMENT_CALLS = 1000;
/// @brief the maximum time between a write and the sync that follows it, and between
/// a send and the save of its offset.
const auto SYNC_INTERVAL = x::telem::SECOND;
/// @brief the time between the checks of the sync thread.
const auto SYNC_CHECK_INTERVAL = 250 * x::telem::MILLISECOND;

x::telem::TimeStamp steady_now() {
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::steady_clock::now().time_since_epoch()
        )
            .count()
    );
}

x::errors::Error fs_error(const std::string &msg, const std::error_code &ec) {
    return x::errors::Error(ERROR, msg + ": " + ec.message());
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
}

Outbox::Outbox(Config cfg): cfg(std::move(cfg)) {}

std::filesystem::path Outbox::segment_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + SEGMENT_EXT);
}

std::pair<std::shared_ptr<journal::Writer>, x::errors::Error>
Outbox::open_writer(const std::uint64_t key) const {
    // The sync thread syncs the writer, so a write never waits for the disk.
    auto [writer, err] = journal::Writer::open({
        .path = this->segment_path(key),
        .magic = SEGMENT_MAGIC,
        .sync_interval = x::telem::TimeSpan(std::numeric_limits<std::int64_t>::max()),
        .sync_bytes = std::numeric_limits<std::uint64_t>::max(),
        .on_sync = this->cfg.on_sync,
    });
    return {std::move(writer), err};
}

std::pair<std::unique_ptr<Outbox>, x::errors::Error> Outbox::open(Config cfg) {
    std::error_code ec;
    std::filesystem::create_directories(cfg.dir, ec);
    if (ec) return {nullptr, fs_error("failed to create outbox directory", ec)};
    auto [dir_lock, lock_err] = journal::Lock::open(cfg.dir);
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
            return {nullptr, x::errors::Error(ERROR, "invalid segment " + stem)};
        outbox->segments.insert(key);
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
    for (const auto key: outbox->segments) {
        outbox->next_key = std::max(outbox->next_key, key + 1);
        if (key < offset.segment) continue;
        const auto pos = key == offset.segment ? offset.pos : 0;
        if (const auto err = outbox->load(key, pos)) return {nullptr, err};
    }
    {
        std::lock_guard lock(outbox->mu);
        if (const auto err = outbox->next_tail()) return {nullptr, err};
    }
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
        this->calls.push_back({
            .request = std::move(request),
            .set_status_done = nullptr,
            .segment = key,
            .end = reader->pos(),
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
    {
        std::lock_guard lock(this->mu);
        if (this->calls.size() >= this->cfg.cap) {
            ++this->refused_count;
            return x::errors::Error(
                FULL,
                "The Driver holds " + std::to_string(this->cfg.cap) +
                    " calls that the Core has not taken. Restore the connection to "
                    "the Core."
            );
        }
        if (this->tail.writer == nullptr || this->tail_calls >= SEGMENT_CALLS)
            if (const auto err = this->next_tail()) return err;
        if (const auto err = this->tail.writer->write(
                {.type = CALL_TYPE, .payload = payload}
            )) {
            // The call is not in the segment, or only cut off.
            this->closing.push_back(this->tail);
            this->tail.writer = nullptr;
            this->sync_cv.notify_one();
            return err;
        }
        this->calls.push_back({
            .request = std::move(request),
            .set_status_done = std::move(done),
            .segment = this->tail.key,
            .end = this->tail.writer->pos(),
        });
        ++this->tail_calls;
    }
    this->cv.notify_one();
    return x::errors::NIL;
}

x::errors::Error Outbox::next_tail() {
    if (this->tail.writer != nullptr) this->closing.push_back(this->tail);
    this->tail.writer = nullptr;
    this->tail_calls = 0;
    this->sync_cv.notify_one();
    if (this->spare.has_value()) {
        this->tail = *this->spare;
        this->spare.reset();
        this->spare_wanted = true;
        return x::errors::NIL;
    }
    // Only a burst faster than the sync thread opens a segment here.
    const auto key = this->next_key++;
    auto [writer, err] = this->open_writer(key);
    if (err) return err;
    this->tail = {.key = key, .writer = writer};
    this->writers[key] = writer;
    this->segments.insert(key);
    return x::errors::NIL;
}

void Outbox::run() {
    std::size_t failures = 0;
    while (true) {
        Call call;
        {
            std::unique_lock lock(this->mu);
            this->cv.wait(lock, [this] {
                return this->stopped || !this->calls.empty();
            });
            if (this->stopped) return;
            call = this->calls.front();
        }
        const auto err = this->send(call);
        if (synnax::errors::is_temporarily_unavailable(err)) {
            if (++failures == 1)
                LOG(WARNING) << "[outbox] the Core is unavailable, retrying: " << err;
            if (!this->wait_retry(failures)) return;
            continue;
        }
        if (failures > 0) LOG(INFO) << "[outbox] the Core is available again";
        failures = 0;
        if (err) {
            ++this->dropped_count;
            LOG(WARNING) << "[outbox] dropped a call the Core rejected: " << err;
        }
        this->commit(call);
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
    const auto &set = call.request.set_status();
    auto [res, err] = this->cfg.statuses.set_by_key_or_name(
        set.key_or_name(),
        set.message(),
        set.variant(),
        x::telem::TimeStamp(set.time())
    );
    if (!synnax::errors::is_temporarily_unavailable(err) &&
        call.set_status_done != nullptr)
        call.set_status_done(res, err);
    return err;
}

void Outbox::commit(const Call &call) {
    std::lock_guard lock(this->mu);
    this->calls.pop_front();
    this->sent = journal::Offset{.segment = call.segment, .pos = call.end};
    if (call.segment != this->tail.key &&
        (this->calls.empty() || this->calls.front().segment != call.segment)) {
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
            this->sync_cv.wait_for(lock, SYNC_CHECK_INTERVAL.chrono(), [this] {
                return this->sync_stopped || !this->closing.empty() ||
                       this->segment_sent || this->spare_wanted;
            });
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
    {
        std::lock_guard lock(this->mu);
        std::swap(to_close, this->closing);
        tail_writer = this->tail.writer;
        if (stopping) {
            to_close.push_back(this->tail);
            this->tail.writer = nullptr;
            if (this->spare.has_value()) to_close.push_back(*this->spare);
            this->spare.reset();
        } else if (!this->spare.has_value())
            spare_key = this->next_key++;
        this->spare_wanted = false;
    }
    for (const auto &segment: to_close) {
        if (segment.writer == nullptr) continue;
        const auto err = segment.writer->close();
        if (err)
            LOG(ERROR) << "[outbox] failed to close " << this->segment_path(segment.key)
                       << ": " << err;
        std::lock_guard lock(this->mu);
        if (!err) this->writers.erase(segment.key);
    }
    const auto now = steady_now();
    if (!stopping && tail_writer != nullptr && now - last_sync >= SYNC_INTERVAL) {
        last_sync = now;
        // A failed sync also fails the next write, which moves the tail.
        if (const auto err = tail_writer->sync())
            LOG(ERROR) << "[outbox] failed to sync a segment: " << err;
    }
    if (spare_key.has_value()) {
        auto [writer, err] = this->open_writer(*spare_key);
        if (err) LOG(ERROR) << "[outbox] failed to open a segment: " << err;
        std::unique_lock lock(this->mu);
        if (!err && *spare_key > this->tail.key) {
            this->spare = Segment{.key = *spare_key, .writer = writer};
            this->writers[*spare_key] = writer;
            this->segments.insert(*spare_key);
        } else if (!err) {
            lock.unlock();
            // The send thread opened a later segment first.
            (void) writer->close();
            if (const auto rm_err = journal::remove(this->segment_path(*spare_key)))
                LOG(WARNING) << "[outbox] failed to remove an unused segment: "
                             << rm_err;
        }
    }
    std::optional<journal::Offset> offset;
    {
        std::lock_guard lock(this->mu);
        const auto due = stopping || this->segment_sent ||
                         now - last_save >= SYNC_INTERVAL;
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
    if (const auto err = journal::commit_offset(
            this->cfg.dir / OFFSET_FILE,
            OFFSET_MAGIC,
            *offset
        )) {
        LOG(ERROR) << "[outbox] failed to save the send position: " << err;
        return;
    }
    this->saved = offset;
    this->remove_sent();
}

void Outbox::remove_sent() {
    std::vector<std::uint64_t> sent_keys;
    {
        std::lock_guard lock(this->mu);
        const auto first = this->calls.empty() ? this->tail.key
                                               : this->calls.front().segment;
        for (const auto key: this->segments) {
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
        // A file another process holds open can fail to remove on Windows, and the
        // next save tries again.
        if (const auto err = journal::remove(this->segment_path(key))) {
            LOG(WARNING) << "[outbox] failed to remove a sent segment: " << err;
            continue;
        }
        std::lock_guard lock(this->mu);
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
