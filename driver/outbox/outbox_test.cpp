// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <iterator>
#include <memory>
#include <mutex>
#include <string>
#include <vector>

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

/// @brief returns a transport that answers with each error in order, and with the
/// key "key" when the error is nil.
std::shared_ptr<Transport> transport_of(const std::vector<x::errors::Error> &errors) {
    ::grpc::status::SetByKeyOrNameResponse res;
    res.set_key("key");
    return std::make_shared<Transport>(
        std::vector<::grpc::status::SetByKeyOrNameResponse>(errors.size(), res),
        errors
    );
}

::grpc::status::SetByKeyOrNameRequest set_of(const std::string &message) {
    ::grpc::status::SetByKeyOrNameRequest req;
    req.set_key_or_name("status");
    req.set_message(message);
    req.set_variant("info");
    req.set_time(42);
    return req;
}

std::vector<std::string> messages_of(const Transport &transport) {
    std::vector<std::string> messages;
    messages.reserve(transport.requests.size());
    for (const auto &req: transport.requests)
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
        const std::size_t cap = DEFAULT_CAP,
        const x::telem::TimeSpan retry = x::telem::MILLISECOND
    ) const {
        Config cfg{
            .dir = this->dir,
            .statuses = synnax::status::Client(nullptr, nullptr, nullptr, transport),
            .cap = cap,
        };
        cfg.retry.base_interval = retry;
        cfg.retry.max_interval = retry;
        return ASSERT_NIL_P(Outbox::open(cfg));
    }

    /// @brief opens an outbox whose first send finds the Core unreachable, and that
    /// waits an hour before it retries.
    [[nodiscard]] std::unique_ptr<Outbox>
    open_unreachable(const std::size_t cap = DEFAULT_CAP) const {
        return this
            ->open_outbox(transport_of({freighter::UNREACHABLE}), cap, x::telem::HOUR);
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
    ASSERT_EQ(transport->requests[0].time(), 42);
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

/// @brief it should send only the unsent calls of a partly sent segment after a
/// reopen.
TEST_F(OutboxTest, SendsOnlyUnsentCallsAfterReopen) {
    {
        const auto outbox = this->open_outbox(
            transport_of({x::errors::NIL, freighter::UNREACHABLE}),
            DEFAULT_CAP,
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
    ASSERT_EQ(transport->requests.size(), 3);
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
    ASSERT_EQ(transport->requests.size(), 2);
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
    const auto outbox = this->open_unreachable(2);
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    ASSERT_NIL(outbox->set_status(set_of("second"), nullptr));
    const auto err = outbox->set_status(set_of("third"), nullptr);
    ASSERT_OCCURRED_AS(err, FULL);
    ASSERT_FALSE(err.data.empty());
    ASSERT_EQ(outbox->refused(), 1);
    ASSERT_EQ(outbox->size(), 2);
}

/// @brief it should run a posted function on the send thread.
TEST_F(OutboxTest, RunsPostedFunctions) {
    const auto outbox = this->open_outbox(transport_of({}));
    std::atomic<bool> ran = false;
    outbox->post([&] { ran = true; });
    ASSERT_EVENTUALLY_TRUE(ran.load());
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
        std::ofstream file(this->segment(), std::ios::binary);
        file.write(SEGMENT_MAGIC.data(), SEGMENT_MAGIC.size());
        file.write("\x02\x00\x00\x00", 4);
    }
    ASSERT_OCCURRED_AS_P(Outbox::open({.dir = this->dir}), journal::UNKNOWN_VERSION);
}

/// @brief it should refuse to open, and keep, a call that passes its checksum but
/// does not decode.
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
    for (const auto &req: transport->requests) {
        EXPECT_EQ(req.key_or_name(), "status");
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
    ASSERT_EQ(read_file(this->segment()), read_file(TESTDATA / "v1_segment.bin"));
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
    ASSERT_FALSE(std::filesystem::exists(this->segment()));
    ASSERT_TRUE(std::filesystem::exists(this->dir / "1.calls"));
}
}
