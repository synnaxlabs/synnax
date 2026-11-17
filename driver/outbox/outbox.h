// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <filesystem>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "client/cpp/status/status.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/journal/journal.h"
#include "driver/outbox/pb/call.pb.h"

namespace driver::outbox {
/// @brief the default number of calls an outbox can hold.
constexpr std::size_t DEFAULT_CAP = 10000;
/// @brief the magic of an outbox segment file.
constexpr journal::Magic SEGMENT_MAGIC = {'S', 'Y', 'O', 'U', 'T', 'B', 'S', 'G'};
/// @brief the magic of the offset file of an outbox.
constexpr journal::Magic OFFSET_MAGIC = {'S', 'Y', 'O', 'U', 'T', 'B', 'O', 'F'};
/// @brief the record type of a call in a segment file.
constexpr std::uint8_t CALL_TYPE = 1;

const x::errors::Error ERROR = x::errors::Error("outbox", "");
/// @brief the outbox has no room for a call.
const x::errors::Error FULL = ERROR.sub("full");
/// @brief a call in a segment file passes its checksum but does not decode.
const x::errors::Error CORRUPT = ERROR.sub("corrupt");
/// @brief a segment file has a call kind this Driver cannot send.
const x::errors::Error UNKNOWN_VERSION = ERROR.sub("unknown_version");

/// @brief the configuration of an outbox.
struct Config {
    /// @brief the directory that holds only the files of the outbox.
    std::filesystem::path dir;
    /// @brief the client that sends status sets to the Core.
    synnax::status::Client statuses;
    /// @brief the maximum number of calls the outbox can hold.
    std::size_t cap = DEFAULT_CAP;
    /// @brief the backoff between sends while the Core is temporarily unavailable.
    x::breaker::Config retry = {
        .name = "outbox",
        .base_interval = x::telem::SECOND,
        .max_retries = x::breaker::RETRY_INFINITELY,
        .scale = 2,
        .max_interval = 30 * x::telem::SECOND,
    };
};

/// @brief called on the send thread once the Core answers a status set. An error
/// means the outbox dropped the set.
using SetStatusDone = std::function<
    void(const synnax::status::SetByKeyOrNameResult &, const x::errors::Error &)>;

/// @brief a queue of Core calls on disk, which one thread sends in order. The queue
/// keeps its calls across process restarts. It syncs a call to the disk within about
/// a second, so a power loss can lose the calls of the last second. It saves its send
/// position about once a second, so after a crash it can send the calls of the last
/// second again. This is safe, because the Core does not let an older status set
/// replace a newer one. While the Core is temporarily unavailable, the thread retries
/// the first call with no limit. Any other error drops the call. Safe for concurrent
/// use.
class Outbox {
    struct Call {
        pb::Call request;
        /// @brief null for a call loaded from disk.
        SetStatusDone set_status_done;
        std::uint64_t segment;
        /// @brief the byte position after the call in its segment.
        std::uint64_t end;
    };

    Config cfg;
    std::unique_ptr<journal::Lock> dir_lock;
    x::breaker::Breaker breaker;
    mutable std::mutex mu;
    std::condition_variable cv;
    std::condition_variable sync_cv;
    bool stopped = false;
    std::deque<Call> calls;
    std::set<std::uint64_t> segments;
    /// @brief the segments whose write or sync failed, so their calls may not be on
    /// the disk.
    std::set<std::uint64_t> unsynced;
    /// @brief the functions to run on the send thread.
    std::vector<std::function<void()>> posted;
    std::uint64_t tail = 0;
    /// @brief null until the first call of the tail segment.
    std::unique_ptr<journal::Writer> writer;
    std::size_t tail_calls = 0;
    /// @brief the position after the last sent call, when it is not saved yet. Used
    /// only by the send thread.
    std::optional<journal::Offset> unsaved;
    x::telem::TimeStamp last_save;
    std::atomic<std::uint64_t> dropped_count = 0;
    std::atomic<std::uint64_t> refused_count = 0;
    std::uint64_t cut_bytes = 0;
    std::thread sender;
    std::thread syncer;

    explicit Outbox(Config cfg);
    [[nodiscard]] std::filesystem::path segment_path(std::uint64_t key) const;
    x::errors::Error load(std::uint64_t key, std::uint64_t pos);
    x::errors::Error push(pb::Call request, SetStatusDone done);
    /// @returns the byte position after the call in the tail segment.
    std::pair<std::uint64_t, x::errors::Error> append(const pb::Call &request);
    /// @brief starts a new tail segment at the next call. failed marks the old tail
    /// as not synced. Requires mu.
    void end_tail(bool failed);
    /// @brief syncs the tail segment, or only when a sync is due unless forced.
    /// Requires mu.
    x::errors::Error sync_tail(bool forced);
    void run();
    void run_syncs();
    x::errors::Error send(const Call &call) const;
    /// @brief takes the sent call off the queue. Saves the offset when the queue is
    /// empty, the call ends its segment, or a second passed since the last save.
    x::errors::Error commit(const Call &call);
    x::errors::Error save_offset();
    x::errors::Error remove_sent();

public:
    Outbox(const Outbox &) = delete;
    Outbox &operator=(const Outbox &) = delete;

    /// @brief opens the outbox in the directory of the config, creates the directory
    /// when it does not exist, locks it, removes the temporary files a crash left, and
    /// starts the send thread. The first call that is cut off or fails its checksum
    /// ends its segment, and the outbox keeps the calls before it and counts the bytes
    /// from it on in cut().
    /// @returns UNKNOWN_VERSION when a segment has a call kind this Driver cannot
    /// send. CORRUPT when a call does not decode. journal::LOCKED when another
    /// outbox has the directory. The errors of journal::Reader::open and
    /// journal::Reader::next when a segment or the offset is not valid.
    static std::pair<std::unique_ptr<Outbox>, x::errors::Error> open(Config cfg);

    /// @brief stops the send thread and waits for a send in progress.
    ~Outbox();

    /// @brief queues a status set and returns without waiting for the Core.
    /// @param req the set, with its time stamped by the caller.
    /// @param done called once the Core answers the set.
    /// @returns FULL when the outbox has no room, and counts the set in refused().
    x::errors::Error
    set_status(const ::grpc::status::SetByKeyOrNameRequest &req, SetStatusDone done);

    /// @brief runs fn on the send thread before its next send or retry, or at once
    /// when the outbox holds no calls. fn must not call the outbox.
    void post(std::function<void()> fn);

    /// @returns the number of calls the Core rejected, which the outbox dropped.
    [[nodiscard]] std::uint64_t dropped() const;

    /// @returns the number of calls the outbox refused because it was full.
    [[nodiscard]] std::uint64_t refused() const;

    /// @returns the number of bytes open lost at the ends of segments, which a crash,
    /// a power loss, or a damaged disk left. Not 0 means the outbox lost calls.
    [[nodiscard]] std::uint64_t cut() const;

    /// @returns the number of calls the outbox holds.
    [[nodiscard]] std::size_t size() const;
};
}
