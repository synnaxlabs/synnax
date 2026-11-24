// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iterator>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "absl/crc/crc32c.h"
#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/journal/journal.h"
#include "driver/outbox/outbox.h"
#include "driver/outbox/pb/call.pb.h"

namespace driver::outbox {
namespace {
const std::filesystem::path TESTDATA = "driver/outbox/testdata";

std::string read_file(const std::filesystem::path &path) {
    std::ifstream f(path, std::ios::binary);
    return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}

using Transport = freighter::testutil::UnaryClient<
    ::grpc::status::SetByKeyOrNameRequest,
    ::grpc::status::SetByKeyOrNameResponse>;

/// @brief returns a transport that answers with each error in order, and with the key
/// "key" when the error is nil.
std::shared_ptr<Transport> transport_of(const std::vector<x::errors::Error> &errors) {
    ::grpc::status::SetByKeyOrNameResponse res;
    res.set_key("key");
    return std::make_shared<Transport>(
        std::vector<::grpc::status::SetByKeyOrNameResponse>(errors.size(), res),
        errors
    );
}

/// @brief returns a set of the status named key, which is message by default.
::grpc::status::SetByKeyOrNameRequest
set_of(const std::string &message, const std::string &key = "") {
    ::grpc::status::SetByKeyOrNameRequest req;
    req.set_key_or_name(key.empty() ? message : key);
    req.set_message(message);
    req.set_variant("info");
    req.set_time(42);
    return req;
}

/// @brief returns the payloads of the call records of the segment at path.
std::vector<std::string> payloads_of(const std::filesystem::path &path) {
    auto reader = ASSERT_NIL_P(
        journal::Reader::open(
            {.path = path, .magic = SEGMENT_MAGIC, .types = {CALL_TYPE}}
        )
    );
    std::vector<std::string> payloads;
    while (true) {
        auto record = ASSERT_NIL_P(reader->next());
        if (!record) return payloads;
        payloads.push_back(record->payload);
    }
}

std::vector<std::string> messages_of(const Transport &transport) {
    std::vector<std::string> messages;
    for (const auto &req: transport.sent())
        messages.push_back(req.message());
    return messages;
}
}

class OutboxTest : public ::testing::Test {
protected:
    std::filesystem::path dir;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("outbox_test_" + x::uuid::create().to_string());
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] std::unique_ptr<Outbox> open_outbox(
        const std::shared_ptr<Transport> &transport,
        const std::uint64_t max_bytes = Config{}.max_bytes,
        const x::telem::TimeSpan retry = x::telem::MILLISECOND
    ) const {
        Config cfg{
            .dir = this->dir,
            .statuses = synnax::status::Client(nullptr, nullptr, nullptr, transport),
            .max_bytes = max_bytes,
        };
        cfg.retry.base_interval = retry;
        cfg.retry.max_interval = retry;
        return ASSERT_NIL_P(Outbox::open(cfg));
    }

    /// @brief opens an outbox that passes the result of each segment sync through
    /// on_sync.
    [[nodiscard]] std::unique_ptr<Outbox> open_with(
        const std::shared_ptr<Transport> &transport,
        const x::telem::TimeSpan retry,
        std::function<x::errors::Error(const x::errors::Error &)> on_sync
    ) const {
        Config cfg{
            .dir = this->dir,
            .statuses = synnax::status::Client(nullptr, nullptr, nullptr, transport),
            .on_sync = std::move(on_sync),
        };
        cfg.retry.base_interval = retry;
        cfg.retry.max_interval = retry;
        return ASSERT_NIL_P(Outbox::open(cfg));
    }

    /// @brief opens an outbox whose first send finds the Core unreachable, and that
    /// waits an hour before it retries.
    [[nodiscard]] std::unique_ptr<Outbox>
    open_unreachable(const std::uint64_t max_bytes = Config{}.max_bytes) const {
        return this->open_outbox(
            transport_of({freighter::UNREACHABLE}),
            max_bytes,
            x::telem::HOUR
        );
    }

    [[nodiscard]] std::filesystem::path segment() const {
        return this->dir / "0.calls";
    }

    /// @brief writes a segment that holds one call record with the payload.
    void write_call(const std::string &payload) const {
        std::filesystem::create_directories(this->dir);
        auto writer = ASSERT_NIL_P(
            journal::Writer::open({.path = this->segment(), .magic = SEGMENT_MAGIC})
        );
        ASSERT_NIL(writer->write({.type = CALL_TYPE, .payload = payload}));
        ASSERT_NIL(writer->close());
    }
};

/// @brief it should send queued calls in order and answer each with its result.
TEST_F(OutboxTest, SendsCallsInOrder) {
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    std::vector<std::string> keys;
    std::mutex mu;
    for (const auto *msg: {"first", "second"})
        ASSERT_NIL(outbox->set_status(
            set_of(msg),
            [&](const synnax::status::SetByKeyOrNameResult &res,
                const x::errors::Error &err) {
                ASSERT_NIL(err);
                std::lock_guard lock(mu);
                keys.push_back(res.key);
            }
        ));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"first", "second"}));
    ASSERT_EQ(transport->sent()[0].time(), 42);
    std::lock_guard lock(mu);
    ASSERT_EQ(keys, (std::vector<std::string>{"key", "key"}));
}

/// @brief it should send the calls it held on disk after a reopen.
TEST_F(OutboxTest, SendsCallsAfterReopen) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
    }
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"first", "second"}));
}

/// @brief it should not send a call again after a reopen once the Core took it.
TEST_F(OutboxTest, DoesNotResendSentCalls) {
    {
        const auto outbox = this->open_outbox(transport_of({x::errors::NIL}));
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 0);
}

/// @brief it should send only the unsent calls of a partly sent segment after a reopen.
TEST_F(OutboxTest, SendsOnlyUnsentCallsAfterReopen) {
    {
        const auto outbox = this->open_outbox(
            transport_of({x::errors::NIL, freighter::UNREACHABLE}),
            Config{}.max_bytes,
            x::telem::HOUR
        );
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 1);
    }
    const auto transport = transport_of({x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"second"}));
}

/// @brief it should refuse to open when the saved offset is corrupt.
TEST_F(OutboxTest, RefusesCorruptOffset) {
    {
        const auto outbox = this->open_outbox(transport_of({x::errors::NIL}));
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    auto offset = read_file(this->dir / "offset");
    offset.back() = static_cast<char>(~offset.back());
    {
        std::ofstream file(this->dir / "offset", std::ios::binary | std::ios::trunc);
        file.write(offset.data(), static_cast<std::streamsize>(offset.size()));
    }
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), journal::CORRUPT);
}

/// @brief it should send calls queued after a reopen that found every segment sent.
TEST_F(OutboxTest, SendsCallsQueuedAfterDrainedReopen) {
    {
        const auto outbox = this->open_outbox(transport_of({x::errors::NIL}));
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("third"), nullptr));
    }
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"second", "third"}));
}

/// @brief it should retry a call while the Core is unreachable.
TEST_F(OutboxTest, RetriesWhileUnreachable) {
    const auto transport = transport_of(
        {freighter::UNREACHABLE, freighter::UNREACHABLE, x::errors::NIL}
    );
    const auto outbox = this->open_outbox(transport);
    std::atomic<int> answers = 0;
    ASSERT_NIL(outbox->set_status(
        set_of("first"),
        [&](const synnax::status::SetByKeyOrNameResult &, const x::errors::Error &err) {
            ASSERT_NIL(err);
            ++answers;
        }
    ));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(transport->sent().size(), 3);
    ASSERT_EQ(answers, 1);
    ASSERT_EQ(outbox->dropped(), 0);
}

/// @brief it should retry a call while the Core has no license.
TEST_F(OutboxTest, RetriesWhileUnlicensed) {
    const auto transport = transport_of(
        {synnax::errors::LICENSE_MISSING, x::errors::NIL}
    );
    const auto outbox = this->open_outbox(transport);
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(transport->sent().size(), 2);
    ASSERT_EQ(outbox->dropped(), 0);
}

/// @brief it should drop and count a call that the Core rejects, and send the next.
TEST_F(OutboxTest, DropsRejectedCalls) {
    const auto transport = transport_of({x::errors::VALIDATION, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    std::atomic<bool> rejected = false;
    ASSERT_NIL(outbox->set_status(
        set_of("first"),
        [&](const synnax::status::SetByKeyOrNameResult &, const x::errors::Error &err) {
            rejected = err.matches(x::errors::VALIDATION);
        }
    ));
    ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"first", "second"}));
    ASSERT_TRUE(rejected);
    ASSERT_EQ(outbox->dropped(), 1);
}

/// @brief it should refuse and count calls when it is full.
TEST_F(OutboxTest, RefusesCallsWhenFull) {
    pb::Call call;
    auto *set = call.mutable_set_status();
    set->set_key_or_name(std::string(30, 'a'));
    set->set_message(std::string(30, 'a'));
    set->set_variant("info");
    set->set_time(42);
    const auto call_bytes = journal::RECORD_PREFIX_SIZE + call.ByteSizeLong();
    // Room for two calls, with or without the header of a new segment.
    const auto outbox = this->open_unreachable(
        2 * journal::HEADER_SIZE + 2 * call_bytes + call_bytes / 2
    );
    ASSERT_NIL(outbox->set_status(set_of(std::string(30, 'a')), nullptr));
    ASSERT_NIL(outbox->set_status(set_of(std::string(30, 'b')), nullptr));
    const auto err = outbox->set_status(set_of(std::string(30, 'c')), nullptr);
    ASSERT_OCCURRED_AS(err, FULL);
    ASSERT_FALSE(err.data.empty());
    ASSERT_EQ(outbox->refused(), 1);
    ASSERT_EQ(outbox->size(), 2);
}

/// @brief it should keep the calls before a call that a crash cut off, and count the
/// bytes it lost.
TEST_F(OutboxTest, KeepsCallsBeforeCutOffTail) {
    {
        const auto outbox = this->open_unreachable();
        for (const auto *msg: {"first", "second"})
            ASSERT_NIL(outbox->set_status(set_of(msg), nullptr));
    }
    const auto size = std::filesystem::file_size(this->segment());
    std::filesystem::resize_file(this->segment(), size - 3);
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 1);
    ASSERT_GT(outbox->cut(), 0);
    ASSERT_NIL(outbox->set_status(set_of("third"), nullptr));
    ASSERT_EQ(outbox->size(), 2);
}

/// @brief it should cut lost bytes from their segment, so a later open does not count
/// them again.
TEST_F(OutboxTest, CountsCutBytesOnce) {
    {
        const auto outbox = this->open_unreachable();
        for (const auto *msg: {"first", "second"})
            ASSERT_NIL(outbox->set_status(set_of(msg), nullptr));
    }
    const auto size = std::filesystem::file_size(this->segment());
    std::filesystem::resize_file(this->segment(), size - 3);
    std::uint64_t cut = 0;
    {
        const auto outbox = this->open_unreachable();
        cut = outbox->cut();
        ASSERT_GT(cut, 0);
    }
    ASSERT_EQ(std::filesystem::file_size(this->segment()), size - 3 - cut);
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 1);
    ASSERT_EQ(outbox->cut(), 0);
}

/// @brief it should cut and count a tail that a power loss left zero-filled.
TEST_F(OutboxTest, IgnoresZeroFilledTail) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    }
    {
        std::ofstream file(this->segment(), std::ios::binary | std::ios::app);
        const std::string zeros(64, '\0');
        file.write(zeros.data(), static_cast<std::streamsize>(zeros.size()));
    }
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 1);
    ASSERT_EQ(outbox->cut(), 64);
}

/// @brief it should count a segment that a crash cut off in its file start as cut.
TEST_F(OutboxTest, CutsSegmentWithCutOffFileStart) {
    std::filesystem::create_directories(this->dir);
    std::ofstream(this->segment(), std::ios::binary).write(SEGMENT_MAGIC.data(), 5);
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 0);
    ASSERT_EQ(outbox->cut(), 5);
    ASSERT_FALSE(std::filesystem::exists(this->segment()));
}

/// @brief it should refuse to open a segment from a newer format version.
TEST_F(OutboxTest, RefusesUnknownVersion) {
    std::filesystem::create_directories(this->dir);
    {
        std::string header(SEGMENT_MAGIC.data(), SEGMENT_MAGIC.size());
        header += std::string("\x02\x00", 2);
        header += std::string(journal::HEADER_SIZE - 14, '\0');
        const auto crc = static_cast<std::uint32_t>(absl::ComputeCrc32c(header));
        for (int i = 0; i < 4; i++)
            header.push_back(static_cast<char>(crc >> (8 * i)));
        std::ofstream(this->segment(), std::ios::binary) << header;
    }
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), journal::UNKNOWN_VERSION);
}

/// @brief it should refuse to open, and keep, a call that passes its checksum but does
/// not decode.
TEST_F(OutboxTest, RefusesUndecodableCall) {
    this->write_call(
        std::string(
            "\x0a\x05"
            "ab",
            4
        )
    );
    const auto size = std::filesystem::file_size(this->segment());
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), CORRUPT);
    ASSERT_EQ(std::filesystem::file_size(this->segment()), size);
}

/// @brief it should refuse to open, and keep, a call of a kind it does not know.
TEST_F(OutboxTest, RefusesUnknownCallKind) {
    this->write_call(std::string("\x4a\x00", 2));
    const auto size = std::filesystem::file_size(this->segment());
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), UNKNOWN_VERSION);
    ASSERT_EQ(std::filesystem::file_size(this->segment()), size);
}

/// @brief it should refuse to open a range call, which this Driver cannot send.
TEST_F(OutboxTest, RefusesRangeCall) {
    pb::Call call;
    call.mutable_end_range()->set_key("range");
    this->write_call(call.SerializeAsString());
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), UNKNOWN_VERSION);
}

/// @brief it should remove the temporary files that a crash left, and keep the rest.
TEST_F(OutboxTest, RemovesStaleTemporaryFiles) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    }
    for (const auto *name: {"0.calls.tmp.12", "offset.tmp.34"})
        std::ofstream(this->dir / name) << "partial";
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 1);
    ASSERT_FALSE(std::filesystem::exists(this->dir / "0.calls.tmp.12"));
    ASSERT_FALSE(std::filesystem::exists(this->dir / "offset.tmp.34"));
    ASSERT_TRUE(std::filesystem::exists(this->segment()));
}

/// @brief it should refuse to open a directory that another outbox has open.
TEST_F(OutboxTest, RefusesLockedDirectory) {
    const auto outbox = this->open_unreachable();
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), journal::LOCKED);
}

/// @brief it should send the calls of a v1 segment.
TEST_F(OutboxTest, SendsV1Segment) {
    std::filesystem::create_directories(this->dir);
    std::filesystem::copy_file(TESTDATA / "v1_segment.bin", this->segment());
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"first", "second"}));
    ASSERT_EQ(outbox->cut(), 0);
    for (const auto &req: transport->sent()) {
        EXPECT_EQ(req.key_or_name(), req.message());
        EXPECT_EQ(req.variant(), "info");
        EXPECT_EQ(req.time(), 42);
    }
}

/// @brief it should send only the calls after a v1 offset.
TEST_F(OutboxTest, SendsV1SegmentFromV1Offset) {
    std::filesystem::create_directories(this->dir);
    std::filesystem::copy_file(TESTDATA / "v1_segment.bin", this->segment());
    std::filesystem::copy_file(TESTDATA / "v1_offset.bin", this->dir / "offset");
    const auto transport = transport_of({x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"second"}));
}

/// @brief it should write the v1 segment format.
TEST_F(OutboxTest, WritesV1Segment) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
    }
    // Each file has its own salt, so only the bytes before it match the fixture.
    const auto written = read_file(this->segment());
    const auto fixture = read_file(TESTDATA / "v1_segment.bin");
    ASSERT_EQ(written.size(), fixture.size());
    ASSERT_EQ(written.substr(0, 12), fixture.substr(0, 12));
    ASSERT_EQ(written.substr(16, 12), fixture.substr(16, 12));
    ASSERT_EQ(payloads_of(this->segment()), payloads_of(TESTDATA / "v1_segment.bin"));
}

/// @brief it should remove each segment once the Core took all of its calls.
TEST_F(OutboxTest, RemovesSentSegments) {
    constexpr std::size_t count = 1001;
    const auto transport = transport_of(
        std::vector<x::errors::Error>(count, x::errors::NIL)
    );
    const auto outbox = this->open_outbox(transport);
    for (std::size_t i = 0; i < count; i++)
        ASSERT_NIL(outbox->set_status(set_of(std::to_string(i)), nullptr));
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        outbox->size(),
        0,
        std::chrono::seconds(10),
        std::chrono::milliseconds(5)
    );
    ASSERT_EVENTUALLY_FALSE(std::filesystem::exists(this->segment()));
    ASSERT_TRUE(std::filesystem::exists(this->dir / "1.calls"));
}

/// @brief it should hold a call whose write succeeded before a failed sync, and refuse
/// the next call that the failed segment cannot take.
TEST_F(OutboxTest, MovesToNewSegmentAfterFailedSync) {
    std::atomic<bool> failing = false;
    std::atomic<int> failed_syncs = 0;
    {
        const auto outbox = this->open_with(
            transport_of({freighter::UNREACHABLE}),
            x::telem::HOUR,
            [&](const x::errors::Error &err) {
                if (!failing) return err;
                ++failed_syncs;
                return x::errors::Error(journal::BASE_ERROR, "disk failed");
            }
        );
        failing = true;
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
            failed_syncs.load(),
            1,
            std::chrono::seconds(5),
            std::chrono::milliseconds(10)
        );
        ASSERT_OCCURRED_AS(
            outbox->set_status(set_of("second"), nullptr),
            journal::BASE_ERROR
        );
        ASSERT_NIL(outbox->set_status(set_of("third"), nullptr));
        ASSERT_EQ(outbox->size(), 2);
        failing = false;
    }
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"first", "third"}));
}

/// @brief it should keep sending when it fails to save its offset, and send the calls
/// again after a reopen.
TEST_F(OutboxTest, KeepsSendingAfterFailedOffsetSave) {
    {
        const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
        const auto outbox = this->open_outbox(transport);
        std::filesystem::create_directories(this->dir / "offset" / "blocker");
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    std::filesystem::remove_all(this->dir / "offset");
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 2);
}

/// @brief it should never save an offset past the bytes of a segment that are on the
/// disk.
TEST_F(OutboxTest, SavesOffsetOnlyAfterSync) {
    std::atomic<bool> failing = false;
    {
        const auto outbox = this->open_with(
            transport_of({x::errors::NIL}),
            x::telem::MILLISECOND,
            [&](const x::errors::Error &err) {
                if (!failing) return err;
                return x::errors::Error(journal::BASE_ERROR, "disk failed");
            }
        );
        failing = true;
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    const auto offset = ASSERT_NIL_P(
        journal::read_offset(this->dir / "offset", OFFSET_MAGIC)
    );
    ASSERT_TRUE(offset.has_value());
    EXPECT_EQ(offset.value().segment, 0);
    EXPECT_EQ(offset.value().pos, journal::HEADER_SIZE);
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 1);
}

/// @brief it should sync a call to the disk while it waits to retry a send.
TEST_F(OutboxTest, SyncsDuringRetryWait) {
    std::atomic<int> syncs = 0;
    const auto outbox = this->open_with(
        transport_of({freighter::UNREACHABLE}),
        x::telem::HOUR,
        [&](const x::errors::Error &err) {
            ++syncs;
            return err;
        }
    );
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        syncs.load(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
}

/// @brief it should tell how to recover from a saved offset past the end of its
/// segment.
TEST_F(OutboxTest, RefusesOffsetPastSegmentEnd) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    }
    ASSERT_NIL(
        journal::commit_offset(
            this->dir / "offset",
            OFFSET_MAGIC,
            {.segment = 0, .pos = 1000}
        )
    );
    auto [outbox, err] = Outbox::open({.dir = this->dir});
    ASSERT_OCCURRED_AS(err, journal::CORRUPT);
    EXPECT_TRUE(err.message().contains("Delete"));
}

/// @brief it should treat only the errors of a Core refusal as rejections.
TEST(RejectedTest, ClassifiesErrors) {
    for (const auto &err:
         {x::errors::VALIDATION,
          x::errors::NOT_FOUND,
          x::errors::UNAUTHORIZED,
          x::errors::Error("sy.auth.access_denied", ""),
          x::errors::Error("sy.license.invalid", ""),
          x::errors::Error("arc.runtime", "")})
        EXPECT_TRUE(rejected(err)) << err;
    for (const auto &err:
         {freighter::UNREACHABLE,
          x::errors::Error("sy.auth.expired_token", ""),
          synnax::errors::LICENSE_MISSING,
          synnax::errors::LICENSE_EXPIRED,
          x::errors::INTERNAL,
          x::errors::Error("sy.validationx", ""),
          x::errors::Error("Received RST_STREAM with error code 2", "")})
        EXPECT_FALSE(rejected(err)) << err;
}

/// @brief it should retry a call that fails with an error the Core did not type.
TEST_F(OutboxTest, RetriesUntypedErrors) {
    const auto transport = transport_of(
        {x::errors::Error("Received RST_STREAM with error code 2", ""), x::errors::NIL}
    );
    const auto outbox = this->open_outbox(transport);
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(transport->sent().size(), 2);
    ASSERT_EQ(outbox->dropped(), 0);
}

/// @brief it should report a call that keeps failing, and report again once it is sent.
TEST_F(OutboxTest, ReportsStuckCall) {
    std::mutex mu;
    std::vector<x::errors::Error> reports;
    Config cfg{
        .dir = this->dir,
        .statuses = synnax::status::Client(
            nullptr,
            nullptr,
            nullptr,
            transport_of(
                {freighter::UNREACHABLE, freighter::UNREACHABLE, x::errors::NIL}
            )
        ),
        .stuck_after = x::telem::TimeSpan(0),
        .on_stuck = [&](const x::errors::Error &err) {
            std::lock_guard lock(mu);
            reports.push_back(err);
        },
    };
    cfg.retry.base_interval = x::telem::MILLISECOND;
    cfg.retry.max_interval = x::telem::MILLISECOND;
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    std::lock_guard lock(mu);
    ASSERT_EQ(reports.size(), 2);
    EXPECT_TRUE(reports[0].matches(freighter::UNREACHABLE));
    EXPECT_NIL(reports[1]);
}

/// @brief it should give each send the deadline of the config.
TEST_F(OutboxTest, SendsWithDeadline) {
    const auto transport = transport_of({x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    const auto start = std::chrono::system_clock::now();
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_TRUE(transport->deadlines.at(0).has_value());
    EXPECT_GE(transport->deadlines[0].value(), start + Config{}.send_timeout.chrono());
}

/// @brief it should skip a set when a newer set of the same status is queued.
TEST_F(OutboxTest, SkipsReplacedSets) {
    const auto transport = transport_of(
        {freighter::UNREACHABLE, x::errors::NIL, x::errors::NIL}
    );
    const auto outbox = this->open_outbox(
        transport,
        Config{}.max_bytes,
        x::telem::SECOND
    );
    ASSERT_NIL(outbox->set_status(set_of("1", "a"), nullptr));
    ASSERT_EVENTUALLY_EQ(transport->sent().size(), 1);
    ASSERT_NIL(outbox->set_status(set_of("x", "b"), nullptr));
    ASSERT_NIL(outbox->set_status(set_of("2", "a"), nullptr));
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        outbox->size(),
        0,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"1", "x", "2"}));
}

/// @brief it should skip the replaced sets it loads after a reopen.
TEST_F(OutboxTest, SkipsReplacedSetsAfterReopen) {
    {
        const auto outbox = this->open_unreachable();
        for (const auto *msg: {"1", "2"})
            ASSERT_NIL(outbox->set_status(set_of(msg, "a"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("x", "b"), nullptr));
        ASSERT_NIL(outbox->set_status(set_of("3", "a"), nullptr));
    }
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"x", "3"}));
}

/// @brief it should compact its files to the newest set of each status before they
/// fill, and send only those sets after a reopen.
TEST_F(OutboxTest, CompactsToNewestSets) {
    constexpr std::uint64_t max_bytes = 4096;
    {
        const auto outbox = this->open_unreachable(max_bytes);
        for (int i = 0; i < 300; i++) {
            const auto set = set_of(std::to_string(i), i % 2 == 0 ? "a" : "b");
            ASSERT_EVENTUALLY_NIL_WITH_TIMEOUT(
                outbox->set_status(set, nullptr),
                std::chrono::seconds(30),
                std::chrono::milliseconds(5)
            );
        }
    }
    const auto transport = transport_of(
        std::vector<x::errors::Error>(3, x::errors::NIL)
    );
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    const auto messages = messages_of(*transport);
    ASSERT_GE(messages.size(), 2);
    EXPECT_EQ(messages[messages.size() - 2], "298");
    EXPECT_EQ(messages.back(), "299");
}

/// @brief it should send no set twice when a crash leaves the old segments next to the
/// compacted one.
TEST_F(OutboxTest, SkipsOldSegmentsAfterCrashInCompaction) {
    const auto aside = std::filesystem::path(this->dir.string() + "_aside");
    {
        const auto outbox = this->open_unreachable(4096);
        for (int i = 0; i < 300; i++) {
            if (i == 20) {
                std::filesystem::create_directories(aside);
                // The sync thread can remove a temp file during the copy.
                for (const auto &entry: std::filesystem::directory_iterator(this->dir))
                    if (entry.path().extension() == ".calls")
                        std::filesystem::copy_file(
                            entry.path(),
                            aside / entry.path().filename()
                        );
            }
            ASSERT_EVENTUALLY_NIL_WITH_TIMEOUT(
                outbox->set_status(set_of(std::to_string(i), "a"), nullptr),
                std::chrono::seconds(30),
                std::chrono::milliseconds(5)
            );
        }
    }
    for (const auto &entry: std::filesystem::directory_iterator(aside))
        if (entry.path().extension() == ".calls" &&
            !std::filesystem::exists(this->dir / entry.path().filename()))
            std::filesystem::copy_file(
                entry.path(),
                this->dir / entry.path().filename()
            );
    std::filesystem::remove_all(aside);
    const auto transport = transport_of({x::errors::NIL, x::errors::NIL});
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(messages_of(*transport), (std::vector<std::string>{"299"}));
}

/// @brief it should keep the calls of a compaction whose offset save failed after the
/// offset reached the disk.
TEST_F(OutboxTest, KeepsCompactionWhenOffsetSaveFails) {
    std::atomic<int> saves = 0;
    std::optional<journal::Offset> first;
    {
        Config cfg{
            .dir = this->dir,
            .statuses = synnax::status::Client(
                nullptr,
                nullptr,
                nullptr,
                transport_of({freighter::UNREACHABLE})
            ),
            .max_bytes = 4096,
            .on_save = [&](const x::errors::Error &err) {
                // The first save comes before the compaction writes its segment, and
                // the second ends the compaction.
                const auto n = ++saves;
                if (n == 2) return x::errors::Error(journal::BASE_ERROR, "disk failed");
                if (n == 1)
                    first = journal::read_offset(this->dir / "offset", OFFSET_MAGIC)
                                .first;
                return err;
            },
        };
        cfg.retry.base_interval = x::telem::HOUR;
        cfg.retry.max_interval = x::telem::HOUR;
        const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
        ASSERT_NIL(outbox->set_status(set_of("keep"), nullptr));
        for (int i = 0; saves < 2; i++) {
            ASSERT_LT(i, 100000);
            (void) outbox->set_status(
                set_of(std::to_string(i), i % 2 == 0 ? "a" : "b"),
                nullptr
            );
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }
    }
    ASSERT_TRUE(first.has_value());
    EXPECT_EQ(first->segment, 0);
    EXPECT_EQ(first->pos, journal::HEADER_SIZE);
    const auto transport = transport_of(
        std::vector<x::errors::Error>(1000, x::errors::NIL)
    );
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    const auto messages = messages_of(*transport);
    ASSERT_FALSE(messages.empty());
    EXPECT_EQ(messages.front(), "keep");
}

/// @brief it should compact before newer sets of repeated statuses fill the room that
/// unique statuses leave.
TEST_F(OutboxTest, CompactsWithoutRefusingNewerSets) {
    constexpr std::uint64_t max_bytes = std::uint64_t{8} * 1000 * 1000;
    const auto outbox = this->open_unreachable(max_bytes);
    const auto message = std::string(1000, 'm');
    const auto call_bytes = journal::RECORD_PREFIX_SIZE + 1020;
    const auto unique = std::uint64_t{6700} * 1000 / call_bytes;
    for (std::size_t i = 0; i < unique; i++)
        ASSERT_NIL(
            outbox->set_status(set_of(message, "u" + std::to_string(i)), nullptr)
        );
    for (int i = 0; i < 3000; i++) {
        ASSERT_NIL(
            outbox->set_status(set_of(message, "r" + std::to_string(i % 100)), nullptr)
        );
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
    EXPECT_EQ(outbox->refused(), 0);
}

/// @brief it should refuse a call larger than max_payload and keep taking calls.
TEST_F(OutboxTest, RefusesTooLargeCall) {
    const auto outbox = this->open_unreachable();
    ASSERT_OCCURRED_AS(
        outbox->set_status(set_of(std::string(Config{}.max_payload, 'a')), nullptr),
        journal::TOO_LARGE
    );
    ASSERT_EQ(outbox->size(), 0);
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_EQ(outbox->size(), 1);
}

/// @brief it should remove the segment of a compaction that a crash stopped, and count
/// none of its bytes as lost.
TEST_F(OutboxTest, RemovesSegmentOfStoppedCompaction) {
    {
        const auto outbox = this->open_unreachable();
        for (const auto *msg: {"first", "second"})
            ASSERT_NIL(outbox->set_status(set_of(msg), nullptr));
    }
    ASSERT_NIL(
        journal::commit_offset(
            this->dir / "offset",
            OFFSET_MAGIC,
            {.segment = 0, .pos = journal::HEADER_SIZE}
        )
    );
    const auto partial = this->dir / "5.calls";
    {
        auto writer = ASSERT_NIL_P(
            journal::Writer::open(
                {.path = partial, .magic = SEGMENT_MAGIC, .flags = COMPACTED_FLAG}
            )
        );
        ASSERT_NIL(writer->write(
            {.type = CALL_TYPE, .payload = payloads_of(this->segment()).at(0)}
        ));
        ASSERT_NIL(writer->close());
    }
    std::ofstream(partial, std::ios::binary | std::ios::app).write("\x10\x00\x00", 3);
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 2);
    ASSERT_EQ(outbox->cut(), 0);
    ASSERT_FALSE(std::filesystem::exists(partial));
}

/// @brief it should call on_create with its directory and each file it creates.
TEST_F(OutboxTest, CallsOnCreateForEveryFile) {
    std::mutex mu;
    std::vector<std::filesystem::path> created;
    const auto transport = transport_of(
        std::vector<x::errors::Error>(5, x::errors::NIL)
    );
    {
        Config cfg{
            .dir = this->dir / "data",
            .statuses = synnax::status::Client(nullptr, nullptr, nullptr, transport),
            .segment_calls = 2,
            .on_create = [&](const std::filesystem::path &path) {
                std::lock_guard lock(mu);
                created.push_back(path);
                return x::errors::NIL;
            },
        };
        const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
        for (int i = 0; i < 5; i++)
            ASSERT_NIL(outbox->set_status(set_of(std::to_string(i)), nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    std::lock_guard lock(mu);
    ASSERT_GE(created.size(), 2);
    EXPECT_EQ(created[0], this->dir);
    EXPECT_EQ(created[1], this->dir / "data");
    const auto was_created = [&](const std::filesystem::path &path) {
        return std::ranges::any_of(created, [&](const std::filesystem::path &c) {
            return c == path || c.string().starts_with(path.string() + ".tmp.");
        });
    };
    int files = 0;
    for (const auto &entry: std::filesystem::directory_iterator(this->dir / "data")) {
        files++;
        EXPECT_TRUE(was_created(entry.path())) << entry.path();
    }
    EXPECT_GE(files, 3);
}

/// @brief it should wait for a spare segment once per full tail, and then keep writing
/// to the tail without a wait.
TEST_F(OutboxTest, WaitsForSpareOncePerTail) {
    std::mutex mu;
    std::condition_variable cv;
    bool blocked = false;
    bool released = false;
    Config cfg{
        .dir = this->dir,
        .statuses = synnax::status::Client(
            nullptr,
            nullptr,
            nullptr,
            transport_of({freighter::UNREACHABLE})
        ),
        .segment_calls = 1,
        .sync_interval = 200 * x::telem::MILLISECOND,
        // The first spare holds the sync thread, so no spare opens.
        .on_create = [&](const std::filesystem::path &path) {
            if (!path.filename().string().starts_with("1.calls")) return x::errors::NIL;
            std::unique_lock lock(mu);
            blocked = true;
            cv.notify_all();
            cv.wait(lock, [&] { return released; });
            return x::errors::NIL;
        },
    };
    cfg.retry.base_interval = x::telem::HOUR;
    cfg.retry.max_interval = x::telem::HOUR;
    {
        const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
        {
            std::unique_lock lock(mu);
            cv.wait(lock, [&] { return blocked; });
        }
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        const auto start = std::chrono::steady_clock::now();
        ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
        EXPECT_GE(std::chrono::steady_clock::now() - start, cfg.sync_interval.chrono());
        const auto after = std::chrono::steady_clock::now();
        for (const auto *msg: {"third", "fourth"})
            ASSERT_NIL(outbox->set_status(set_of(msg), nullptr));
        EXPECT_LT(std::chrono::steady_clock::now() - after, cfg.sync_interval.chrono());
        EXPECT_EQ(outbox->size(), 4);
        std::lock_guard lock(mu);
        released = true;
        cv.notify_all();
    }
}

/// @brief it should refuse a call over max_calls, and compact replaced sets to make
/// room for newer ones.
TEST_F(OutboxTest, BoundsCallCount) {
    const auto open = [&](const std::filesystem::path &dir, const std::size_t max) {
        Config cfg{
            .dir = dir,
            .statuses = synnax::status::Client(
                nullptr,
                nullptr,
                nullptr,
                transport_of({freighter::UNREACHABLE})
            ),
            .max_calls = max,
        };
        cfg.retry.base_interval = x::telem::HOUR;
        cfg.retry.max_interval = x::telem::HOUR;
        return ASSERT_NIL_P(Outbox::open(cfg));
    };
    {
        const auto outbox = open(this->dir / "unique", 2);
        for (const auto *key: {"a", "b"})
            ASSERT_NIL(outbox->set_status(set_of(key), nullptr));
        ASSERT_OCCURRED_AS(outbox->set_status(set_of("c"), nullptr), FULL);
    }
    const auto outbox = open(this->dir / "repeated", 8);
    for (const auto *key: {"a", "b"})
        ASSERT_NIL(outbox->set_status(set_of(key), nullptr));
    for (int i = 0; i < 50; i++)
        ASSERT_EVENTUALLY_NIL_WITH_TIMEOUT(
            outbox->set_status(set_of(std::to_string(i), "d"), nullptr),
            std::chrono::seconds(10),
            std::chrono::milliseconds(5)
        );
    ASSERT_LE(outbox->size(), 8);
}

/// @brief it should send the newest set of each status once after a compaction and the
/// removal of the offset file.
TEST_F(OutboxTest, SendsNewestSetsAfterCompactionWithoutOffset) {
    {
        const auto outbox = this->open_unreachable(4096);
        for (int i = 0; i < 300; i++)
            ASSERT_EVENTUALLY_NIL_WITH_TIMEOUT(
                outbox->set_status(
                    set_of(std::to_string(i), i % 2 == 0 ? "a" : "b"),
                    nullptr
                ),
                std::chrono::seconds(30),
                std::chrono::milliseconds(5)
            );
    }
    std::filesystem::remove(this->dir / "offset");
    const auto transport = transport_of(
        std::vector<x::errors::Error>(300, x::errors::NIL)
    );
    const auto outbox = this->open_outbox(transport);
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    const auto messages = messages_of(*transport);
    ASSERT_GE(messages.size(), 2);
    EXPECT_EQ(std::ranges::count(messages, "298"), 1);
    EXPECT_EQ(std::ranges::count(messages, "299"), 1);
    EXPECT_EQ(messages[messages.size() - 2], "298");
    EXPECT_EQ(messages.back(), "299");
}
}
