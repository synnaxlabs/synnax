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
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/status/status.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/journal/journal.h"
#include "driver/outbox/pb/call.pb.h"

namespace driver::outbox {
/// @brief the magic of an outbox segment file.
constexpr journal::Magic SEGMENT_MAGIC = {'S', 'Y', 'O', 'U', 'T', 'B', 'S', 'G'};
/// @brief the magic of the offset file of an outbox.
constexpr journal::Magic OFFSET_MAGIC = {'S', 'Y', 'O', 'U', 'T', 'B', 'O', 'F'};
/// @brief the record type of a call in a segment file.
constexpr std::uint8_t CALL_TYPE = 1;
/// @brief the segment flag of a segment that a compaction wrote. It holds copies of the
/// calls of the segments before it.
constexpr std::uint16_t COMPACTED_FLAG = 0x0001;

const x::errors::Error BASE_ERROR = x::errors::Error("outbox", "");
/// @brief the outbox has no room for a call.
const x::errors::Error FULL = BASE_ERROR.sub("full");
/// @brief a call in a segment file passes its checksum but does not decode.
const x::errors::Error CORRUPT = BASE_ERROR.sub("corrupt");
/// @brief a segment file has a call kind this Driver cannot send.
const x::errors::Error UNKNOWN_VERSION = BASE_ERROR.sub("unknown_version");

/// @brief the configuration of an outbox.
struct Config {
    /// @brief the directory that holds only the files of the outbox.
    std::filesystem::path dir;
    /// @brief the client that sends status sets to the Core.
    synnax::status::Client statuses;
    /// @brief the most bytes the files of the outbox can hold. A full outbox keeps only
    /// the last status set of each status, and then refuses calls.
    std::uint64_t max_bytes = std::uint64_t{64} * 1024 * 1024;
    /// @brief the largest encoded call, at most journal::MAX_PAYLOAD. The outbox
    /// refuses a larger call.
    std::uint32_t max_payload = 64 * 1024;
    /// @brief the share of max_bytes that a compaction must free before it runs, or
    /// half of the free bytes when the files are close to max_bytes.
    double compact_share = 0.25;
    /// @brief the number of calls in a segment file, after which the next starts.
    std::size_t segment_calls = 1000;
    /// @brief how long one send waits for the Core.
    x::telem::TimeSpan send_timeout = 10 * x::telem::SECOND;
    /// @brief the backoff between sends while the Core does not take a call.
    x::breaker::Config retry = {
        .name = "outbox",
        .base_interval = x::telem::SECOND,
        .max_retries = x::breaker::RETRY_INFINITELY,
        .scale = 2,
        .max_interval = 5 * x::telem::SECOND,
    };
    /// @brief the maximum time between a write and its sync, and between a send and the
    /// save of the send position.
    x::telem::TimeSpan sync_interval = x::telem::SECOND;
    /// @brief how long the first call can fail before the outbox calls on_stuck.
    x::telem::TimeSpan stuck_after = 5 * x::telem::MINUTE;
    /// @brief called on the send thread with the error once the first call failed for
    /// stuck_after, and with nil once the outbox sends it.
    std::function<void(const x::errors::Error &)> on_stuck =
        [](const x::errors::Error &) {};
    /// @brief gets the result of each sync of a segment file and returns the result the
    /// outbox uses.
    std::function<x::errors::Error(const x::errors::Error &)> on_sync =
        [](const x::errors::Error &err) { return err; };
    /// @brief called with each file and directory that the outbox creates, before the
    /// outbox uses it, such as to set its owner.
    journal::OnCreate on_create = journal::no_create_hook;
    /// @brief gets the result of each save of the send position and returns the result
    /// the outbox uses.
    std::function<x::errors::Error(const x::errors::Error &)> on_save =
        [](const x::errors::Error &err) { return err; };
};

/// @brief called on the send thread once the Core answers a status set. An error means
/// the outbox dropped the set. The outbox does not call it for a set that a newer set
/// of the same status replaced. The outbox sends no other call until it returns.
using SetStatusDone = std::function<
    void(const synnax::status::SetByKeyOrNameResult &, const x::errors::Error &)>;

/// @returns whether err means the Core refused a call, so a retry cannot succeed. Every
/// other error, such as a lost connection or a timeout, needs a retry.
[[nodiscard]] bool rejected(const x::errors::Error &err);

/// @brief a queue of Core calls on disk, which one thread sends in order. The queue
/// keeps its calls across process restarts. A second thread syncs the calls to the disk
/// each sync_interval, so a power loss can lose the calls of the last interval. The
/// same thread saves the send position each sync_interval, so after a crash the outbox
/// can send the calls of the last interval again. This is safe, because the Core does
/// not let an older status set replace a newer one. The send thread skips a status set
/// when a newer set of the same status is queued. It drops a call the Core rejected,
/// and retries any other error with no limit. Safe for concurrent use.
class Outbox {
    struct Call {
        /// @brief shared, so a compaction can encode it outside the lock.
        std::shared_ptr<const pb::Call> request;
        /// @brief null for a call loaded from disk.
        SetStatusDone set_status_done;
        std::uint64_t segment = 0;
        /// @brief the byte position after the call in its segment.
        std::uint64_t end = 0;
        /// @brief the order of the call in the outbox.
        std::uint64_t seq = 0;
        /// @brief the byte size of the record of the call.
        std::uint64_t size = 0;
    };

    /// @brief a segment file that a writer appends to.
    struct Segment {
        std::uint64_t key = 0;
        std::shared_ptr<journal::Writer> writer;
    };

    Config cfg;
    std::unique_ptr<journal::Lock> dir_lock;
    mutable std::mutex mu;
    std::condition_variable cv;
    std::condition_variable sync_cv;
    /// @brief notified when the sync thread opens a spare.
    std::condition_variable spare_cv;
    bool stopped = false;
    bool sync_stopped = false;
    std::deque<Call> calls;
    std::uint64_t next_seq = 0;
    /// @brief the newest queued set of a status.
    struct Newest {
        std::uint64_t seq = 0;
        /// @brief the byte size of the record of the set.
        std::uint64_t size = 0;
    };

    /// @brief the newest queued set of each status.
    std::unordered_map<std::string, Newest> latest;
    /// @brief the number of queued sets that a newer set of their status replaced.
    std::size_t replaced = 0;
    /// @brief the byte size of the records of the replaced sets.
    std::uint64_t replaced_bytes = 0;
    /// @brief the byte size of each segment file on the disk.
    std::map<std::uint64_t, std::uint64_t> segments;
    /// @brief the byte size of all segment files.
    std::uint64_t bytes = 0;
    /// @brief the byte size of the records of the queued calls.
    std::uint64_t queued_bytes = 0;
    /// @brief the writers of the segments that are not closed, or that failed. A
    /// segment not in it is on the disk whole.
    std::map<std::uint64_t, std::shared_ptr<journal::Writer>> writers;
    /// @brief the segment that takes new calls. Its writer is null after a failure.
    Segment tail;
    std::size_t tail_calls = 0;
    /// @brief a new segment that the sync thread opened for the next tail.
    std::optional<Segment> spare;
    /// @brief the tail took the spare, so the sync thread opens a new one.
    bool spare_wanted = false;
    /// @brief the files are close to max_bytes, so the sync thread compacts them.
    bool compact_wanted = false;
    /// @brief the key of the next new segment.
    std::uint64_t next_key = 0;
    /// @brief the key of a segment that failed to open, which the next open tries
    /// again, so a lasting fault does not leave a file for each try.
    std::optional<std::uint64_t> failed_key;
    /// @brief the segments that left the tail, which the sync thread closes.
    std::vector<Segment> closing;
    /// @brief the position after the last sent call.
    std::optional<journal::Offset> sent;
    /// @brief the last sent call ended its segment.
    bool segment_sent = false;
    /// @brief the last offset on the disk. Used only by the sync thread.
    std::optional<journal::Offset> saved;
    std::atomic<std::uint64_t> dropped_count = 0;
    std::atomic<std::uint64_t> refused_count = 0;
    std::uint64_t cut_bytes = 0;
    std::thread sender;
    std::thread syncer;

    explicit Outbox(Config cfg);
    [[nodiscard]] std::filesystem::path segment_path(std::uint64_t key) const;
    std::pair<std::shared_ptr<journal::Writer>, x::errors::Error>
    open_writer(std::uint64_t key, std::uint16_t flags = 0) const;
    x::errors::Error load(std::uint64_t key, std::uint64_t pos);
    /// @brief queues a call that is on the disk. Requires mu.
    void enqueue(Call call);
    x::errors::Error push(pb::Call request, SetStatusDone done);
    /// @brief moves the tail to the spare, and waits up to sync_interval for the sync
    /// thread to open one. Keeps a full tail when the wait ends with no spare. Requires
    /// a lock on mu.
    x::errors::Error next_tail(std::unique_lock<std::mutex> &lock);
    /// @brief records a segment that failed to open. Requires mu.
    void fail_open(std::uint64_t key);
    /// @returns whether a compaction frees enough of the files to run. Requires mu.
    [[nodiscard]] bool reclaimable() const;
    /// @brief takes a replaced set off the byte counts. Requires mu.
    void unreplace(const Call &call);
    /// @returns whether a newer set of the same status replaced call. Requires mu.
    [[nodiscard]] bool is_replaced(const Call &call) const;
    void run();
    /// @brief waits for the retry interval after the number of failed sends.
    /// @returns false when the outbox stops.
    bool wait_retry(std::size_t failures);
    x::errors::Error send(const Call &call) const;
    /// @brief takes the call with seq off the queue once it is sent or dropped.
    void commit(std::uint64_t seq);
    void run_syncs();
    /// @brief closes the segments that left the tail, syncs the tail, opens a spare,
    /// compacts, and saves the offset when due. stopping closes the tail and the spare.
    void sync_once(
        bool stopping,
        x::telem::TimeStamp &last_sync,
        x::telem::TimeStamp &last_save
    );
    /// @brief closes segments that left the tail. Only the sync thread calls it.
    /// @returns whether there were any.
    bool close_segments(const std::vector<Segment> &segments);
    /// @brief writes the calls that no newer set replaced to a new segment, saves an
    /// offset at its start, and then removes the old segments.
    void compact();
    /// @brief saves offset as the send position on the disk.
    [[nodiscard]] x::errors::Error save(const journal::Offset &offset) const;
    /// @brief removes the segment files of keys, and syncs the directory once.
    void remove_segments(const std::vector<std::uint64_t> &keys) const;
    /// @brief removes the segments whose calls were all sent, and keeps a segment that
    /// fails to remove for the next try.
    void remove_sent();

public:
    Outbox(const Outbox &) = delete;
    Outbox &operator=(const Outbox &) = delete;

    /// @brief opens the outbox in the directory of the config, creates the directory
    /// when it does not exist, locks it, removes the temporary files a crash left, and
    /// starts the send and sync threads. When a crash or a power loss cut off the last
    /// call of a segment, the outbox keeps the calls before it, cuts the bytes from it
    /// on from the segment, and counts them in cut(). It removes a segment that a crash
    /// left during a compaction.
    /// @returns UNKNOWN_VERSION when a segment has a call kind this Driver cannot send.
    /// CORRUPT when a call does not decode. journal::LOCKED when another outbox has the
    /// directory. journal::CORRUPT when the saved offset is past the end of its
    /// segment. The errors of journal::Writer::open and journal::Reader::next when a
    /// segment or the offset is not valid.
    static std::pair<std::unique_ptr<Outbox>, x::errors::Error> open(Config cfg);

    /// @brief stops the threads, waits for a send in progress, and syncs the calls and
    /// the send position to the disk.
    ~Outbox();

    /// @brief queues a status set and returns without waiting for the Core or the disk.
    /// @param req the set, with its time stamped by the caller.
    /// @param done called once the Core answers the set.
    /// @returns FULL when the outbox has no room, and counts the set in refused().
    /// journal::TOO_LARGE when the encoded set is larger than max_payload. The error of
    /// a failed write, and then the outbox does not hold the set.
    x::errors::Error
    set_status(const ::grpc::status::SetByKeyOrNameRequest &req, SetStatusDone done);

    /// @returns the number of calls the Core rejected, which the outbox dropped.
    [[nodiscard]] std::uint64_t dropped() const;

    /// @returns the number of calls the outbox refused because it was full.
    [[nodiscard]] std::uint64_t refused() const;

    /// @returns the number of bytes open lost at the ends of segments, which a crash, a
    /// power loss, or a damaged disk left. Not 0 means the outbox lost calls.
    [[nodiscard]] std::uint64_t cut() const;

    /// @returns the number of calls the outbox holds.
    [[nodiscard]] std::size_t size() const;
};
}
