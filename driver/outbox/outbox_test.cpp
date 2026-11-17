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
#include "x/cpp/color/color.h"
#include "x/cpp/telem/telem.h"
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

using Creates = freighter::testutil::
    UnaryClient<::grpc::ranger::CreateRequest, ::grpc::ranger::CreateResponse>;
using Retrieves = freighter::testutil::
    UnaryClient<::grpc::ranger::RetrieveRequest, ::api::ranger::pb::RetrieveResponse>;
using Dispatches = freighter::testutil::
    UnaryClient<::grpc::ranger::DispatchRequest, google::protobuf::Empty>;

/// @brief returns a range with every field that a create carries set.
synnax::ranger::Range full_range() {
    return {
        .key = x::uuid::UUID::parse("2b6f1c43-3e33-4a4f-9a76-0d8c5d1e7a10").first,
        .name = "run",
        .time_range = x::telem::TimeRange(
            x::telem::TimeStamp(10),
            x::telem::TimeStamp(20)
        ),
        .color = x::color::Color{.r = 1, .g = 2, .b = 3, .a = 4},
        .labels = {x::uuid::UUID::parse("6a1e0d55-8f4b-4c51-b0c2-1f7e9a3d2c84").first},
        .parent = x::uuid::UUID::parse("9d3c7b21-5a6e-4f80-8b19-e4c2a7f05d36").first,
    };
}

/// @brief asserts that the create sends every field of full_range().
void expect_full_range(const ::service::ranger::pb::Range &pb) {
    const auto range = ASSERT_NIL_P(synnax::ranger::Range::from_proto(pb));
    const auto expected = full_range();
    EXPECT_EQ(range.key, expected.key);
    EXPECT_EQ(range.name, expected.name);
    EXPECT_EQ(range.time_range, expected.time_range);
    EXPECT_EQ(range.color, expected.color);
    EXPECT_EQ(range.labels, expected.labels);
    EXPECT_EQ(range.parent, expected.parent);
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

/// @brief it should take past the cap only the first end of a range it took a create
/// of.
TEST_F(OutboxTest, TakesOneRangeEndPastCap) {
    const auto outbox = this->open_unreachable(2);
    ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
    synnax::ranger::Range queued;
    queued.key = x::uuid::create();
    ASSERT_NIL(outbox->create_range(queued, nullptr));
    synnax::ranger::Range refused;
    refused.key = x::uuid::create();
    ASSERT_OCCURRED_AS(outbox->create_range(refused, nullptr), FULL);
    ASSERT_NIL(outbox->end_range(queued.key, x::telem::TimeStamp(1), nullptr));
    ASSERT_OCCURRED_AS(
        outbox->end_range(queued.key, x::telem::TimeStamp(2), nullptr),
        FULL
    );
    ASSERT_OCCURRED_AS(
        outbox->end_range(refused.key, x::telem::TimeStamp(1), nullptr),
        FULL
    );
    ASSERT_EQ(outbox->refused(), 3);
    ASSERT_EQ(outbox->size(), 3);
}

/// @brief it should take past the cap the end of a range whose create it sent, for
/// only the last cap ranges it took a create of.
TEST_F(OutboxTest, TakesEndOfSentCreatePastCap) {
    synnax::ranger::Range first;
    first.key = x::uuid::create();
    synnax::ranger::Range second;
    second.key = x::uuid::create();
    std::vector<::grpc::ranger::CreateResponse> created(2);
    *created[0].add_ranges() = ASSERT_NIL_P(first.to_proto());
    *created[1].add_ranges() = ASSERT_NIL_P(second.to_proto());
    const auto creates = std::make_shared<freighter::testutil::UnaryClient<
        ::grpc::ranger::CreateRequest,
        ::grpc::ranger::CreateResponse>>(
        created,
        std::vector{x::errors::NIL, x::errors::NIL}
    );
    Config cfg{
        .dir = this->dir,
        .statuses = synnax::status::Client(
            nullptr,
            nullptr,
            nullptr,
            transport_of({freighter::UNREACHABLE})
        ),
        .ranges = synnax::ranger::Client(nullptr, creates, nullptr, {}),
        .cap = 1,
    };
    cfg.retry.base_interval = x::telem::HOUR;
    cfg.retry.max_interval = x::telem::HOUR;
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    for (const auto &range: {first, second}) {
        ASSERT_NIL(outbox->create_range(range, nullptr));
        ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    }
    ASSERT_EQ(outbox->dropped(), 0);
    ASSERT_NIL(outbox->set_status(set_of("blocked"), nullptr));
    ASSERT_NIL(outbox->end_range(second.key, x::telem::TimeStamp(1), nullptr));
    ASSERT_OCCURRED_AS(
        outbox->end_range(first.key, x::telem::TimeStamp(1), nullptr),
        FULL
    );
    ASSERT_EQ(outbox->size(), 2);
}

/// @brief it should keep taking ends of loaded creates past the cap after a reopen,
/// and count loaded ends past the cap toward the limit on ends.
TEST_F(OutboxTest, KeepsCapAccountingAfterReopen) {
    synnax::ranger::Range first;
    first.key = x::uuid::create();
    synnax::ranger::Range second;
    second.key = x::uuid::create();
    {
        const auto outbox = this->open_unreachable(3);
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        ASSERT_NIL(outbox->create_range(first, nullptr));
        ASSERT_NIL(outbox->create_range(second, nullptr));
    }
    {
        const auto outbox = this->open_unreachable(2);
        ASSERT_NIL(outbox->end_range(first.key, x::telem::TimeStamp(1), nullptr));
    }
    const auto outbox = this->open_unreachable(1);
    ASSERT_OCCURRED_AS(
        outbox->end_range(second.key, x::telem::TimeStamp(1), nullptr),
        FULL
    );
    ASSERT_EQ(outbox->size(), 4);
}

/// @brief it should run a posted function on the send thread.
TEST_F(OutboxTest, RunsPostedFunctions) {
    const auto outbox = this->open_outbox(transport_of({}));
    std::atomic<bool> ran = false;
    outbox->post([&] { ran = true; });
    ASSERT_EVENTUALLY_TRUE(ran.load());
}

/// @brief it should send a create again when the Core rejects it and then cannot be
/// reached to retrieve its key.
TEST_F(OutboxTest, RetriesCreateWhileRetrieveUnreachable) {
    synnax::ranger::Range range;
    range.key = x::uuid::create();
    range.name = "range";
    const auto pb = ASSERT_NIL_P(range.to_proto());
    const auto creates = std::make_shared<freighter::testutil::UnaryClient<
        ::grpc::ranger::CreateRequest,
        ::grpc::ranger::CreateResponse>>(
        std::vector<::grpc::ranger::CreateResponse>(2),
        std::vector{x::errors::VALIDATION, x::errors::VALIDATION}
    );
    ::api::ranger::pb::RetrieveResponse found;
    *found.add_ranges() = pb;
    const auto retrieves = std::make_shared<freighter::testutil::UnaryClient<
        ::grpc::ranger::RetrieveRequest,
        ::api::ranger::pb::RetrieveResponse>>(
        std::vector{::api::ranger::pb::RetrieveResponse(), found},
        std::vector{freighter::UNREACHABLE, x::errors::NIL}
    );
    Config cfg{
        .dir = this->dir,
        .ranges = synnax::ranger::Client(retrieves, creates, nullptr, {}),
    };
    cfg.retry.base_interval = x::telem::MILLISECOND;
    cfg.retry.max_interval = x::telem::MILLISECOND;
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    std::atomic<bool> done = false;
    ASSERT_NIL(outbox->create_range(range, [&](const x::errors::Error &err) {
        EXPECT_NIL(err);
        done = true;
    }));
    ASSERT_EVENTUALLY_TRUE(done.load());
    ASSERT_EQ(creates->requests.size(), 2);
    ASSERT_EQ(retrieves->requests.size(), 2);
    ASSERT_EQ(outbox->dropped(), 0);
}

/// @brief it should keep range calls across a reopen.
TEST_F(OutboxTest, KeepsRangeCallsAfterReopen) {
    {
        const auto outbox = this->open_unreachable();
        ASSERT_NIL(outbox->set_status(set_of("first"), nullptr));
        synnax::ranger::Range range;
        range.key = x::uuid::create();
        ASSERT_NIL(outbox->create_range(range, nullptr));
        ASSERT_NIL(outbox->end_range(range.key, x::telem::TimeStamp(1), nullptr));
    }
    const auto outbox = this->open_unreachable();
    ASSERT_EQ(outbox->size(), 3);
}

/// @brief it should send every field of a range create, and a range end.
TEST_F(OutboxTest, SendsRangeCalls) {
    ::grpc::ranger::CreateResponse created;
    *created.add_ranges() = ASSERT_NIL_P(full_range().to_proto());
    const auto creates = std::make_shared<Creates>(
        std::vector{created},
        std::vector{x::errors::NIL}
    );
    const auto dispatches = std::make_shared<Dispatches>(
        std::vector<google::protobuf::Empty>(1),
        std::vector{x::errors::NIL}
    );
    Config cfg{
        .dir = this->dir,
        .ranges = synnax::ranger::Client(nullptr, creates, dispatches, {}),
    };
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    std::atomic<int> done = 0;
    const auto count = [&](const x::errors::Error &err) {
        EXPECT_NIL(err);
        ++done;
    };
    ASSERT_NIL(outbox->create_range(full_range(), count));
    ASSERT_NIL(outbox->end_range(full_range().key, x::telem::TimeStamp(30), count));
    ASSERT_EVENTUALLY_EQ(done.load(), 2);
    ASSERT_EQ(creates->requests.size(), 1);
    expect_full_range(creates->requests[0].ranges(0));
    ASSERT_EQ(dispatches->requests.size(), 1);
    ASSERT_EQ(dispatches->requests[0].key(), full_range().key.to_string());
}

/// @brief it should drop a create that the Core rejects when no range has its key.
TEST_F(OutboxTest, DropsRejectedCreate) {
    const auto creates = std::make_shared<Creates>(
        std::vector<::grpc::ranger::CreateResponse>(1),
        std::vector{x::errors::VALIDATION}
    );
    const auto retrieves = std::make_shared<Retrieves>(
        std::vector<::api::ranger::pb::RetrieveResponse>(1),
        std::vector{x::errors::NIL}
    );
    Config cfg{
        .dir = this->dir,
        .ranges = synnax::ranger::Client(retrieves, creates, nullptr, {}),
    };
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    std::atomic<bool> done = false;
    ASSERT_NIL(outbox->create_range(full_range(), [&](const x::errors::Error &err) {
        EXPECT_TRUE(err.matches(x::errors::VALIDATION));
        done = true;
    }));
    ASSERT_EVENTUALLY_TRUE(done.load());
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(outbox->dropped(), 1);
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

/// @brief v1 range calls must stay readable by every later Driver.
TEST_F(OutboxTest, SendsV1RangeSegment) {
    std::filesystem::create_directories(this->dir);
    std::filesystem::copy_file(TESTDATA / "v1_range_segment.bin", this->segment());
    ::grpc::ranger::CreateResponse created;
    *created.add_ranges() = ASSERT_NIL_P(full_range().to_proto());
    const auto creates = std::make_shared<Creates>(
        std::vector{created},
        std::vector{x::errors::NIL}
    );
    const auto dispatches = std::make_shared<Dispatches>(
        std::vector<google::protobuf::Empty>(1),
        std::vector{x::errors::NIL}
    );
    Config cfg{
        .dir = this->dir,
        .ranges = synnax::ranger::Client(nullptr, creates, dispatches, {}),
    };
    const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
    ASSERT_EVENTUALLY_EQ(outbox->size(), 0);
    ASSERT_EQ(outbox->dropped(), 0);
    ASSERT_EQ(creates->requests.size(), 1);
    expect_full_range(creates->requests[0].ranges(0));
    ASSERT_EQ(dispatches->requests.size(), 1);
    ASSERT_EQ(dispatches->requests[0].key(), full_range().key.to_string());
}

/// @brief it should write the v1 format of range calls.
TEST_F(OutboxTest, WritesV1RangeSegment) {
    {
        const auto creates = std::make_shared<Creates>(
            std::vector<::grpc::ranger::CreateResponse>(1),
            std::vector{freighter::UNREACHABLE}
        );
        Config cfg{
            .dir = this->dir,
            .ranges = synnax::ranger::Client(nullptr, creates, nullptr, {}),
        };
        cfg.retry.base_interval = x::telem::HOUR;
        cfg.retry.max_interval = x::telem::HOUR;
        const auto outbox = ASSERT_NIL_P(Outbox::open(cfg));
        ASSERT_NIL(outbox->create_range(full_range(), nullptr));
        ASSERT_NIL(
            outbox->end_range(full_range().key, x::telem::TimeStamp(30), nullptr)
        );
    }
    ASSERT_EQ(read_file(this->segment()), read_file(TESTDATA / "v1_range_segment.bin"));
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
