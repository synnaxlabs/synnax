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
/// @brief the maximum time between a write and the sync that follows it.
const auto SYNC_INTERVAL = x::telem::SECOND;
/// @brief the time between the checks of the sync thread for a due sync.
const auto SYNC_CHECK_INTERVAL = 250 * x::telem::MILLISECOND;

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

Outbox::Outbox(Config cfg): cfg(std::move(cfg)), breaker(this->cfg.retry) {}

std::filesystem::path Outbox::segment_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + SEGMENT_EXT);
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
    for (const auto &entry: std::filesystem::directory_iterator(outbox->cfg.dir, ec)) {
        const auto &path = entry.path();
        // A crash during a journal file replace leaves its temporary file.
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
    const auto offset = saved.value_or(journal::Offset{});
    // New segments go after the saved offset, which outlives its sent segments.
    if (saved.has_value()) outbox->tail = offset.segment + 1;
    for (const auto key: outbox->segments) {
        outbox->tail = std::max(outbox->tail, key + 1);
        if (key < offset.segment) continue;
        const auto pos = key == offset.segment ? offset.pos : 0;
        if (const auto err = outbox->load(key, pos)) return {nullptr, err};
    }
    if (const auto err = outbox->remove_sent()) return {nullptr, err};
    outbox->breaker.start();
    outbox->sender = std::thread([ptr = outbox.get()] {
        ptr->run();
        if (const auto err = ptr->save_offset())
            LOG(ERROR) << "[outbox] failed to save the offset: " << err;
    });
    outbox->syncer = std::thread([ptr = outbox.get()] { ptr->run_syncs(); });
    return {std::move(outbox), x::errors::NIL};
}

x::errors::Error Outbox::load(const std::uint64_t key, const std::uint64_t pos) {
    auto [reader, err] = journal::Reader::open({
        .path = this->segment_path(key),
        .magic = SEGMENT_MAGIC,
        .types = {CALL_TYPE},
        .pos = pos,
    });
    if (err) return err;
    while (true) {
        auto [record, next_err] = reader->next();
        if (next_err) return next_err;
        if (!record) break;
        auto [request, decode_err] = decode(*record);
        if (decode_err) return decode_err;
        this->calls.push_back({
            .request = std::move(request),
            .set_status_done = nullptr,
            .segment = key,
            .end = reader->pos(),
        });
    }
    if (reader->cut() > 0) {
        this->cut_bytes += reader->cut();
        LOG(WARNING) << "[outbox] lost " << reader->cut() << " bytes at the end of "
                     << this->segment_path(key).string();
    }
    return x::errors::NIL;
}

Outbox::~Outbox() {
    // A failed open never started the send thread.
    if (!this->sender.joinable()) return;
    this->breaker.stop();
    {
        std::lock_guard lock(this->mu);
        this->stopped = true;
    }
    this->cv.notify_all();
    this->sync_cv.notify_all();
    this->sender.join();
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
        auto [end, err] = this->append(request);
        if (err) return err;
        this->calls.push_back({
            .request = std::move(request),
            .set_status_done = std::move(done),
            .segment = this->tail,
            .end = end,
        });
        if (++this->tail_calls == SEGMENT_CALLS) {
            const auto close_err = this->writer->close();
            if (close_err)
                LOG(ERROR) << "[outbox] failed to close a segment: " << close_err;
            this->end_tail(static_cast<bool>(close_err));
        }
    }
    this->cv.notify_one();
    return x::errors::NIL;
}

std::pair<std::uint64_t, x::errors::Error> Outbox::append(const pb::Call &request) {
    if (this->writer == nullptr) {
        auto [writer, err] = journal::Writer::open({
            .path = this->segment_path(this->tail),
            .magic = SEGMENT_MAGIC,
            .sync_interval = SYNC_INTERVAL,
        });
        if (err) return {0, err};
        this->writer = std::move(writer);
        this->segments.insert(this->tail);
    }
    const auto err = this->writer->write({
        .type = CALL_TYPE,
        .payload = request.SerializeAsString(),
    });
    if (err) {
        this->end_tail(true);
        return {0, err};
    }
    return {this->writer->pos(), x::errors::NIL};
}

void Outbox::post(std::function<void()> fn) {
    {
        std::lock_guard lock(this->mu);
        this->posted.push_back(std::move(fn));
    }
    this->cv.notify_one();
}

void Outbox::end_tail(const bool failed) {
    if (failed) this->unsynced.insert(this->tail);
    this->writer.reset();
    ++this->tail;
    this->tail_calls = 0;
}

x::errors::Error Outbox::sync_tail(const bool forced) {
    if (this->writer == nullptr) return x::errors::NIL;
    const auto err = forced ? this->writer->sync() : this->writer->sync_if_due();
    if (err) this->end_tail(true);
    return err;
}

void Outbox::run_syncs() {
    std::unique_lock lock(this->mu);
    while (!this->stopped) {
        if (const auto err = this->sync_tail(false))
            LOG(ERROR) << "[outbox] failed to sync a segment: " << err;
        this->sync_cv.wait_for(lock, SYNC_CHECK_INTERVAL.chrono(), [this] {
            return this->stopped;
        });
    }
}

void Outbox::run() {
    while (true) {
        Call call;
        bool has_call = false;
        std::vector<std::function<void()>> posted;
        {
            std::unique_lock lock(this->mu);
            this->cv.wait(lock, [this] {
                return this->stopped || !this->calls.empty() || !this->posted.empty();
            });
            if (this->stopped) return;
            std::swap(posted, this->posted);
            has_call = !this->calls.empty();
            if (has_call) call = this->calls.front();
        }
        for (const auto &fn: posted)
            fn();
        if (!has_call) continue;
        const auto err = this->send(call);
        if (synnax::errors::is_temporarily_unavailable(err)) {
            if (!this->breaker.wait_unlimited(err)) return;
            continue;
        }
        this->breaker.reset_unlimited();
        if (err) {
            ++this->dropped_count;
            LOG(WARNING) << "[outbox] dropped a call the Core rejected: " << err;
        }
        if (const auto commit_err = this->commit(call))
            LOG(ERROR) << "[outbox] failed to commit a sent call: " << commit_err;
    }
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

x::errors::Error Outbox::commit(const Call &call) {
    this->unsaved = journal::Offset{.segment = call.segment, .pos = call.end};
    {
        std::lock_guard lock(this->mu);
        this->calls.pop_front();
        if (!this->calls.empty() && this->calls.front().segment == call.segment &&
            x::telem::TimeStamp::now() - this->last_save < SYNC_INTERVAL)
            return x::errors::NIL;
    }
    return this->save_offset();
}

x::errors::Error Outbox::save_offset() {
    if (!this->unsaved.has_value()) return x::errors::NIL;
    const auto offset = this->unsaved.value();
    this->unsaved.reset();
    {
        std::lock_guard lock(this->mu);
        // A saved offset must never pass segment bytes that are not on the disk, or
        // a power loss leaves an offset past the end of its segment.
        if (this->unsynced.contains(offset.segment)) return x::errors::NIL;
        if (offset.segment == this->tail)
            if (const auto err = this->sync_tail(true)) return err;
    }
    const auto err = journal::commit_offset(
        this->cfg.dir / OFFSET_FILE,
        OFFSET_MAGIC,
        offset
    );
    this->last_save = x::telem::TimeStamp::now();
    if (err) return err;
    std::lock_guard lock(this->mu);
    return this->remove_sent();
}

x::errors::Error Outbox::remove_sent() {
    const auto first = this->calls.empty() ? this->tail : this->calls.front().segment;
    while (!this->segments.empty() && *this->segments.begin() < first) {
        if (const auto err = journal::remove(
                this->segment_path(*this->segments.begin())
            ))
            return err;
        this->unsynced.erase(*this->segments.begin());
        this->segments.erase(this->segments.begin());
    }
    return x::errors::NIL;
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
