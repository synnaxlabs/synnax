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
#include <iterator>
#include <limits>
#include <ranges>
#include <string>
#include <system_error>
#include <variant>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/color/types.gen.h"
#include "client/cpp/errors/errors.h"
#include "x/cpp/color/color.h"

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
    return {call, x::errors::NIL};
}

std::pair<x::uuid::UUID, x::errors::Error> parse_key(const std::string &key) {
    auto [parsed, err] = x::uuid::UUID::parse(key);
    if (err) return {parsed, x::errors::Error(CORRUPT, "invalid range key " + key)};
    return {parsed, x::errors::NIL};
}

}

bool rejected(const x::errors::Error &err) {
    return synnax::errors::classify(err) == synnax::errors::Kind::REJECTED;
}

Outbox::Outbox(Config cfg): cfg(std::move(cfg)) {}

std::filesystem::path Outbox::segment_path(const std::uint64_t key) const {
    return this->cfg.dir / (std::to_string(key) + SEGMENT_EXT);
}

std::pair<std::shared_ptr<journal::Writer>, x::errors::Error> Outbox::open_writer(
    const std::uint64_t key,
    const std::uint16_t flags,
    const std::optional<std::uint32_t> max_payload
) const {
    // The sync thread syncs the writer, so a write never waits for the disk.
    auto [writer, err] = journal::Writer::open({
        .path = this->segment_path(key),
        .magic = SEGMENT_MAGIC,
        .flags = flags,
        .sync_interval = x::telem::TimeSpan(std::numeric_limits<std::int64_t>::max()),
        .sync_bytes = std::numeric_limits<std::uint64_t>::max(),
        .max_payload = max_payload.value_or(this->cfg.max_payload),
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
        if (const auto err = cfg.on_create(d)) {
            // The next open runs the hook again on the directories it creates.
            for (const auto &created: new_dirs)
                std::filesystem::remove(created, ec);
            return {nullptr, err};
        }
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
        // With no saved offset, the old segments load first, and their newer copies in
        // a compacted segment replace them.
        if (!saved.has_value() || key <= offset.segment) {
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
        const auto size = journal::RECORD_PREFIX_SIZE + record->payload.size();
        const bool over_cap = request.has_end_range() &&
                              this->queued_bytes + size > this->cfg.max_bytes;
        this->enqueue({
            .request = std::make_shared<const pb::Call>(std::move(request)),
            .segment = key,
            .end = reader->pos(),
            .seq = this->next_seq++,
            .size = size,
            .over_cap = over_cap,
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
    if (call.request->has_create_range())
        this->open_range(call.request->create_range().key());
    if (call.request->has_end_range()) this->close_range(call.request->end_range().key());
    if (call.over_cap) ++this->over_cap_ends;
    this->queued_bytes += call.size;
    this->calls.push_back(std::move(call));
}

void Outbox::open_range(const std::string &key) {
    if (this->open_ranges.contains(key)) return;
    this->open_order.push_back(key);
    this->open_ranges[key] = std::prev(this->open_order.end());
    if (this->open_ranges.size() <= this->cfg.max_open_ranges) return;
    this->open_ranges.erase(this->open_order.front());
    this->open_order.pop_front();
}

void Outbox::close_range(const std::string &key) {
    const auto it = this->open_ranges.find(key);
    if (it == this->open_ranges.end()) return;
    this->open_order.erase(it->second);
    this->open_ranges.erase(it);
}

bool Outbox::takes_when_full(const pb::Call &request) const {
    return request.has_end_range() &&
           this->open_ranges.contains(request.end_range().key()) &&
           this->over_cap_ends < this->cfg.max_ends_when_full;
}

bool Outbox::reclaimable() const {
    // A compaction leaves the calls no newer set replaced, in the compacted segment,
    // the tail, and the spare.
    const auto kept = this->queued_bytes - this->replaced_bytes +
                      3 * journal::HEADER_SIZE;
    const auto worth = [this](const double freed, const double kept, const double max) {
        return freed > 0 &&
               freed >= std::min(this->cfg.compact_share * max, (max - kept) / 2);
    };
    const auto kept_calls = this->calls.size() - this->replaced;
    return (this->bytes > kept && worth(
                                      static_cast<double>(this->bytes - kept),
                                      static_cast<double>(kept),
                                      static_cast<double>(this->cfg.max_bytes)
                                  )) ||
           worth(
               static_cast<double>(this->replaced),
               static_cast<double>(kept_calls),
               static_cast<double>(this->cfg.max_calls)
           );
}

bool Outbox::near_full() const {
    return this->bytes >= this->cfg.max_bytes / 4 * 3 ||
           this->calls.size() >= this->cfg.max_calls / 4 * 3;
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
    return this->push(std::move(request), {.set_status_done = std::move(done)});
}

x::errors::Error
Outbox::create_range(const synnax::ranger::Range &range, RangeDone done) {
    pb::Call request;
    auto *create = request.mutable_create_range();
    create->set_key(range.key.to_string());
    create->set_name(range.name);
    create->set_start(range.time_range.start.nanoseconds());
    create->set_end(range.time_range.end.nanoseconds());
    if (range.parent.has_value()) create->set_parent(range.parent->to_string());
    if (range.color.has_value()) {
        if (const auto *named = std::get_if<synnax::color::NamedRef>(&*range.color)) {
            create->set_named_color(named->key.to_string());
        } else {
            const auto &value = std::get<synnax::color::FixedRef>(*range.color).value;
            auto *color = create->mutable_fixed_color();
            color->set_r(value.r);
            color->set_g(value.g);
            color->set_b(value.b);
            color->set_a(value.a);
        }
    }
    for (const auto &label: range.labels)
        create->add_labels(label.to_string());
    return this->push(std::move(request), {.range_done = std::move(done)});
}

x::errors::Error Outbox::end_range(
    const x::uuid::UUID &key,
    const x::telem::TimeStamp end,
    RangeDone done
) {
    pb::Call request;
    auto *end_range = request.mutable_end_range();
    end_range->set_key(key.to_string());
    end_range->set_end(end.nanoseconds());
    return this->push(std::move(request), {.range_done = std::move(done)});
}

x::errors::Error Outbox::push(pb::Call request, Call call) {
    const auto payload = request.SerializeAsString();
    if (payload.size() > this->cfg.max_payload)
        return x::errors::Error(
            journal::TOO_LARGE,
            "the call has " + std::to_string(payload.size()) +
                " bytes, and the outbox takes at most " +
                std::to_string(this->cfg.max_payload)
        );
    const auto size = journal::RECORD_PREFIX_SIZE + payload.size();
    call.request = std::make_shared<const pb::Call>(std::move(request));
    {
        std::unique_lock lock(this->mu);
        if (this->tail.writer == nullptr || this->tail_calls >= this->cfg.segment_calls)
            if (const auto err = this->next_tail(lock)) return err;
        call.over_cap = this->bytes + size > this->cfg.max_bytes ||
                        this->calls.size() >= this->cfg.max_calls;
        // A full outbox still takes the first end of a range, so the range does not
        // stay open.
        if (call.over_cap && !this->takes_when_full(*call.request)) {
            ++this->refused_count;
            if (this->reclaimable()) {
                this->compact_wanted = true;
                this->sync_cv.notify_one();
            }
            return x::errors::Error(
                FULL,
                "The Driver holds " + std::to_string(this->calls.size()) +
                    " calls that the Core has not taken, the most it can hold. "
                    "Check the connection to the Core."
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
        call.segment = this->tail.key;
        call.end = end;
        call.seq = this->next_seq++;
        call.size = size;
        this->enqueue(std::move(call));
        ++this->tail_calls;
        // Compacting before the files are full keeps the newest sets from refusal.
        if (this->near_full() && this->reclaimable()) {
            this->compact_wanted = true;
            this->sync_cv.notify_one();
        }
    }
    this->cv.notify_one();
    return x::errors::NIL;
}

void Outbox::post(std::function<void()> fn) {
    {
        std::lock_guard lock(this->mu);
        this->posted.push_back(std::move(fn));
    }
    this->cv.notify_one();
}

x::errors::Error Outbox::next_tail(std::unique_lock<std::mutex> &lock) {
    if (!this->spare.has_value()) {
        this->spare_wanted = true;
        this->sync_cv.notify_one();
        if (this->tail.writer != nullptr && this->tail_overfull) return x::errors::NIL;
        // The result of an open that started before this push can be stale.
        const auto target = this->opens_done + (this->opening ? 2 : 1);
        this->spare_cv.wait_for(lock, this->cfg.sync_interval.chrono(), [&] {
            return this->spare.has_value() ||
                   (this->tail.writer == nullptr && this->opens_done >= target) ||
                   (this->tail.writer != nullptr &&
                    this->tail_calls < this->cfg.segment_calls);
        });
        // A full tail takes more calls when the sync thread is slow to open a spare.
        if (!this->spare.has_value()) {
            if (this->tail.writer == nullptr && this->open_err) return this->open_err;
            if (this->tail.writer == nullptr)
                return x::errors::Error(
                    BASE_ERROR,
                    "no segment is open to take the call"
                );
            this->tail_overfull = this->tail_calls >= this->cfg.segment_calls;
            return x::errors::NIL;
        }
    }
    if (this->tail.writer != nullptr) this->closing.push_back(this->tail);
    this->tail = *this->spare;
    this->tail_calls = 0;
    this->tail_overfull = false;
    this->spare.reset();
    this->spare_wanted = true;
    this->sync_cv.notify_one();
    return x::errors::NIL;
}

void Outbox::fail_open(const std::uint64_t key, const x::errors::Error &err) {
    // A failed open can leave the file, which remove_sent removes once it is sent.
    this->segments.try_emplace(key, 0);
    this->failed_key = key;
    this->open_err = err;
}

void Outbox::finish_open() {
    ++this->opens_done;
    this->opening = false;
    this->spare_cv.notify_all();
}

void Outbox::run() {
    std::size_t failures = 0;
    auto failing_since = steady_now();
    bool stuck = false;
    while (true) {
        Call call;
        bool skip = false;
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
            if (has_call) {
                call = this->calls.front();
                skip = this->is_replaced(call);
            }
        }
        for (const auto &fn: posted)
            fn();
        if (!has_call) continue;
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
    const auto deadline = std::chrono::steady_clock::now() +
                          this->cfg.retry.interval(failures).chrono();
    std::unique_lock lock(this->mu);
    while (true) {
        this->cv.wait_until(lock, deadline, [this] {
            return this->stopped || !this->posted.empty();
        });
        if (this->stopped) return false;
        if (this->posted.empty()) return true;
        std::vector<std::function<void()>> posted;
        std::swap(posted, this->posted);
        lock.unlock();
        for (const auto &fn: posted)
            fn();
        lock.lock();
    }
}

x::errors::Error Outbox::send(const Call &call) const {
    if (call.request->has_set_status()) return this->send_set(call);
    const auto err = call.request->has_create_range()
                       ? this->send_create(call.request->create_range())
                       : this->send_end(call.request->end_range());
    if ((!err || rejected(err)) && call.range_done != nullptr) call.range_done(err);
    return err;
}

x::errors::Error Outbox::send_set(const Call &call) const {
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

x::errors::Error Outbox::send_create(const pb::CreateRange &create) const {
    auto [key, key_err] = parse_key(create.key());
    if (key_err) return key_err;
    synnax::ranger::Range range{
        .key = key,
        .name = create.name(),
        .time_range = x::telem::TimeRange(
            x::telem::TimeStamp(create.start()),
            x::telem::TimeStamp(create.end())
        ),
    };
    if (create.has_parent()) {
        auto [parent, parent_err] = parse_key(create.parent());
        if (parent_err) return parent_err;
        range.parent = parent;
    }
    if (create.has_named_color()) {
        auto [color_key, color_err] = parse_key(create.named_color());
        if (color_err) return color_err;
        range.color = synnax::color::NamedRef{.key = color_key};
    } else if (create.has_fixed_color()) {
        const auto &color = create.fixed_color();
        range.color = synnax::color::FixedRef{
            .value = x::color::Color{
                .r = static_cast<std::uint8_t>(color.r()),
                .g = static_cast<std::uint8_t>(color.g()),
                .b = static_cast<std::uint8_t>(color.b()),
                .a = static_cast<std::uint8_t>(color.a()),
            },
        };
    }
    for (const auto &label: create.labels()) {
        auto [label_key, label_err] = parse_key(label);
        if (label_err) return label_err;
        range.labels.push_back(label_key);
    }
    const auto err = this->cfg.ranges.create(range, this->cfg.send_timeout);
    if (!err.matches(x::errors::VALIDATION)) return err;
    // The Core rejects a key it holds, which an earlier send of this create made.
    const auto retrieve_err = this->cfg.ranges
                                  .retrieve(range.key, {}, this->cfg.send_timeout)
                                  .second;
    if (!retrieve_err) return x::errors::NIL;
    if (!rejected(retrieve_err)) return retrieve_err;
    return err;
}

x::errors::Error Outbox::send_end(const pb::EndRange &end) const {
    auto [key, err] = parse_key(end.key());
    if (err) return err;
    return this->cfg.ranges
        .set_end(key, x::telem::TimeStamp(end.end()), this->cfg.send_timeout);
}

void Outbox::commit(const std::uint64_t seq) {
    std::lock_guard lock(this->mu);
    // A compaction can drop a replaced call while the send thread skips it, and it can
    // move a call to a new segment.
    if (this->calls.empty() || this->calls.front().seq != seq) return;
    const auto done = std::move(this->calls.front());
    this->calls.pop_front();
    this->queued_bytes -= done.size;
    if (done.over_cap) --this->over_cap_ends;
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
            this->opening = true;
        }
        this->spare_wanted = false;
    }
    // A push can wait for the spare, so it opens first.
    if (spare_key.has_value()) {
        auto [writer, err] = this->open_writer(*spare_key);
        // Each push during a fault retries the open, so the log is limited.
        if (const auto now = steady_now();
            err && now - this->open_logged >= this->cfg.sync_interval) {
            this->open_logged = now;
            LOG(ERROR) << "[outbox] failed to open a segment: " << err;
        }
        std::unique_lock lock(this->mu);
        if (err)
            this->fail_open(*spare_key, err);
        else if (*spare_key > this->tail.key) {
            this->open_err = x::errors::NIL;
            this->spare = Segment{.key = *spare_key, .writer = writer};
            this->writers[*spare_key] = writer;
            this->bytes += writer->pos();
            this->segments[*spare_key] = writer->pos();
        }
        this->finish_open();
        if (!err && *spare_key <= this->tail.key) {
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
        this->fail_open(tail_key, tail_err);
        return;
    }
    std::vector<std::pair<std::uint64_t, std::shared_ptr<const pb::Call>>> kept;
    std::uint64_t first_new_seq = 0;
    std::optional<journal::Offset> start;
    {
        std::lock_guard lock(this->mu);
        first_new_seq = this->next_seq;
        if (!this->saved.has_value() && !this->calls.empty())
            start = journal::Offset{
                .segment = this->calls.front().segment,
                .pos = journal::HEADER_SIZE,
            };
        // The first call can be in flight on the send thread, so it stays.
        for (const auto &call: this->calls)
            if (kept.empty() || !this->is_replaced(call))
                kept.emplace_back(call.seq, call.request);
        if (this->tail.writer != nullptr) this->closing.push_back(this->tail);
        if (this->spare.has_value()) this->closing.push_back(*this->spare);
        this->spare.reset();
        this->spare_wanted = true;
        this->tail = {.key = tail_key, .writer = tail_writer};
        this->open_err = x::errors::NIL;
        this->tail_calls = 0;
        this->tail_overfull = false;
        this->writers[tail_key] = tail_writer;
        this->segments[tail_key] = tail_writer->pos();
        this->bytes += tail_writer->pos();
    }
    this->spare_cv.notify_all();
    // A saved offset lets open remove the segment of a compaction that a crash stops.
    if (start.has_value()) {
        if (const auto err = this->save(*start)) {
            LOG(ERROR) << "[outbox] failed to save the offset before a compaction: "
                       << err;
            return;
        }
        this->saved = start;
    }
    // A call that a larger max_payload of an earlier run took still fits.
    std::size_t largest = this->cfg.max_payload;
    for (const auto &[seq, request]: kept)
        largest = std::max(largest, request->ByteSizeLong());
    auto [writer, err] = this->open_writer(
        key,
        COMPACTED_FLAG,
        static_cast<std::uint32_t>(std::min<std::size_t>(largest, journal::MAX_PAYLOAD))
    );
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
    if (const auto err = x::fs::sync_dir(this->cfg.dir))
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
