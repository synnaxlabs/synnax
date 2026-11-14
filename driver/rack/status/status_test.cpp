// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <mutex>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/status/status.h"

namespace driver::rack::status {
/// @brief returns true when owner owns status.
bool owns(
    const synnax::resource::ID &owner,
    const ::service::status::pb::Status &status
) {
    return synnax::resource::ID::from_proto(status.owner()).first == owner;
}

/// @brief a status set transport that fails the first rack heartbeat and the first
/// re-send of task statuses, and echoes every other write.
class ScriptedSetClient final : public synnax::status::SetClient {
    const synnax::resource::ID rack;

public:
    std::atomic<int> beats = 0;
    std::atomic<int> task_writes = 0;

    explicit ScriptedSetClient(synnax::resource::ID rack): rack(std::move(rack)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        // The first task write is the started status, and the second is a re-send.
        if (owns(this->rack, req.statuses(0))) {
            if (this->beats++ == 0) return {{}, freighter::UNREACHABLE};
        } else if (this->task_writes++ == 1)
            return {{}, freighter::UNREACHABLE};
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }
};

/// @brief a status set transport that fails the first lost task status writes, and
/// echoes every other write.
class LostTaskWriteSetClient final : public synnax::status::SetClient {
    const synnax::resource::ID rack;
    const int lost;

public:
    std::atomic<int> beats = 0;
    std::atomic<int> task_writes = 0;

    explicit LostTaskWriteSetClient(synnax::resource::ID rack, const int lost = 1):
        rack(std::move(rack)), lost(lost) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        if (owns(this->rack, req.statuses(0)))
            this->beats++;
        else if (this->task_writes++ < this->lost)
            return {{}, freighter::UNREACHABLE};
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }
};

/// @brief a status set transport that holds the second rack heartbeat past LATE_GAP,
/// and echoes every write.
class SlowSetClient final : public synnax::status::SetClient {
    const synnax::resource::ID rack;

public:
    std::atomic<int> beats = 0;
    std::atomic<int> task_writes = 0;

    explicit SlowSetClient(synnax::resource::ID rack): rack(std::move(rack)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        if (!owns(this->rack, req.statuses(0)))
            this->task_writes++;
        else if (this->beats++ == 1)
            std::this_thread::sleep_for((LATE_GAP + x::telem::SECOND / 2).chrono());
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }
};

/// @brief a status set transport that keeps the last rack heartbeat and echoes every
/// write.
class BeatRecorder final : public synnax::status::SetClient {
    const synnax::resource::ID rack;
    mutable std::mutex mu;
    synnax::rack::Status last;

public:
    std::atomic<int> beats = 0;

    explicit BeatRecorder(synnax::resource::ID rack): rack(std::move(rack)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        if (owns(this->rack, req.statuses(0))) {
            auto [stat, err] = synnax::rack::Status::from_proto(req.statuses(0));
            if (err) return {{}, err};
            std::lock_guard lock(this->mu);
            this->last = std::move(stat);
            this->beats++;
        }
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }

    [[nodiscard]] synnax::rack::Status last_beat() const {
        std::lock_guard lock(this->mu);
        return this->last;
    }
};

/// @brief a connectivity transport that reports a Core clock offset from this host.
class SkewedCheckClient final : public synnax::connection::CheckClient {
public:
    /// @brief how far this host is ahead of the Core.
    std::atomic<std::int64_t> skew = 0;

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::connectivity::CheckResponse, x::errors::Error>
    send(const std::string &, google::protobuf::Empty &) override {
        grpc::connectivity::CheckResponse res;
        res.set_node_version("0.0.0");
        res.set_node_time((x::telem::TimeStamp::now() -
                           x::telem::TimeSpan(this->skew.load()))
                              .nanoseconds());
        return {res, x::errors::NIL};
    }
};

/// @brief it should report nominal driver status via state streamer.
TEST(stateTests, Nominal) {
    x::breaker::Breaker cancel;
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true
    };
    ASSERT_NIL(rack.tasks.create(task));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    auto cmd = synnax::task::Command{
        .task = task.key,
        .type = "start",
    };
    cancel.start();
    hb->exec(cmd, cancel);
    cancel.stop();
    x::defer::defer stop([&hb]() { hb->stop(false); });
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{
            .channels = {ch.key},
        }
    ));
    const auto owner = synnax::rack::resource_id(rack.key).to_json();
    x::json::json j;
    for (int i = 0; i < 50; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        ASSERT_EQ(frm.size(), 1);
        frm.series->at(0).at(-1, j);
        if (j.value("owner", x::json::json()) == owner) break;
    }
    EXPECT_EQ(j.value("owner", x::json::json()), owner);
    EXPECT_EQ(j["variant"], synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(j["message"], "Driver is running");
    ASSERT_NIL(streamer.close());
}

/// @brief it should send the context's current rack name in each heartbeat.
TEST(stateTests, RenamedRack) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    ctx->set_rack_name(rack.name);
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true,
    };
    ASSERT_NIL(rack.tasks.create(task));
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{.channels = {ch.key}}
    ));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    ASSERT_NE(hb, nullptr);
    x::defer::defer stop([&hb]() { hb->stop(false); });
    const auto owner = synnax::rack::resource_id(rack.key).to_json();
    const auto next_beat_name = [&]() -> std::string {
        for (int i = 0; i < 50; i++) {
            auto [frm, err] = streamer.read();
            if (err) return "";
            for (size_t c = 0; c < frm.size(); c++)
                for (const auto &raw: frm.series->at(c).strings())
                    if (const auto j = x::json::json::parse(raw);
                        j.value("owner", x::json::json()) == owner)
                        return j["name"];
        }
        return "";
    };
    ASSERT_EQ(next_beat_name(), "test_rack");

    ctx->set_rack_name("renamed_rack");
    // A beat in flight during the rename may still carry the old name.
    bool renamed = false;
    for (int i = 0; i < 3 && !renamed; i++)
        renamed = next_beat_name() == "renamed_rack";
    ASSERT_TRUE(renamed);
    EXPECT_EQ(next_beat_name(), "renamed_rack");
    EXPECT_EQ(next_beat_name(), "renamed_rack");
    ASSERT_NIL(streamer.close());
}

/// @brief it should report the task as running while the heartbeat is live, and as
/// not running once it stops.
TEST(stateTests, TaskRunningTransitions) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true,
    };
    ASSERT_NIL(rack.tasks.create(task));
    const auto owner = synnax::task::resource_id(task.key).to_json();
    // The started status is written once, so the streamer must be open before the
    // task is configured.
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{.channels = {ch.key}}
    ));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    ASSERT_NE(hb, nullptr);

    bool started = false;
    for (int i = 0; i < 50 && !started; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        for (size_t c = 0; c < frm.size(); c++)
            for (const auto &raw: frm.series->at(c).strings()) {
                const auto j = x::json::json::parse(raw);
                if (j.value("owner", x::json::json()) != owner) continue;
                EXPECT_EQ(j["details"]["running"], true);
                started = true;
            }
    }
    EXPECT_TRUE(started);

    hb->stop(false);

    bool stopped = false;
    for (int i = 0; i < 50 && !stopped; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        for (size_t c = 0; c < frm.size(); c++)
            for (const auto &raw: frm.series->at(c).strings()) {
                const auto j = x::json::json::parse(raw);
                if (j.value("owner", x::json::json()) != owner ||
                    j["details"]["running"] != false)
                    continue;
                EXPECT_EQ(j["message"], "Stopped");
                stopped = true;
            }
    }
    EXPECT_TRUE(stopped);
    ASSERT_NIL(streamer.close());
}

/// @brief it should send task statuses again on the first heartbeat that succeeds after
/// a failed one, try again on the next heartbeat when the re-send fails, and stop once
/// a re-send succeeds.
TEST(stateTests, ResendsTaskStatusesUntilTheyReachTheCore) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<ScriptedSetClient>(
        synnax::rack::resource_id(rack.key)
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 1 fails, the re-send after beat 2 fails, beat 3 re-sends, and beat 4 sends
    // nothing. Beat 5 starts once beat 4 is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        5,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 3);
}

/// @brief it should send task statuses again after a heartbeat that succeeds more than
/// LATE_GAP after the one before it, and only once.
TEST(stateTests, ResendsTaskStatusesAfterALateHeartbeat) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<SlowSetClient>(
        synnax::rack::resource_id(rack.key)
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 2 is late and re-sends, and beat 3 sends nothing. Beat 4 starts once beat 3
    // is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        4,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 2);
}

/// @brief it should send task statuses again, and only once, after a task status write
/// fails while every heartbeat succeeds.
TEST(stateTests, ResendsTaskStatusesAfterAFailedWrite) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<LostTaskWriteSetClient>(
        synnax::rack::resource_id(rack.key)
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 1 re-sends the started status, and beat 2 sends nothing. Beat 3 starts once
    // beat 2 is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        3,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 2);
}

/// @brief it should send task statuses again on the next heartbeat after a re-send
/// fails while every heartbeat succeeds.
TEST(stateTests, RetriesAFailedResend) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<LostTaskWriteSetClient>(
        synnax::rack::resource_id(rack.key),
        2
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 1 fails its re-send, beat 2 repairs it, and beat 3 sends nothing. Beat 4
    // starts once beat 3 is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        4,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 3);
}

/// @brief it should report the variant, message, and description of its health in each
/// heartbeat.
TEST(stateTests, ReportsHealth) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<BeatRecorder>(
        synnax::rack::resource_id(rack.key)
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx,
        Health{
            .variant = synnax::status::VARIANT_ERROR,
            .message = "Driver 0.59.3 cannot work with Core 0.60.1",
            .description = "Install the Driver release that matches the Core.",
        }
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    const auto beat = transport->last_beat();
    EXPECT_EQ(beat.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(beat.message, "Driver 0.59.3 cannot work with Core 0.60.1");
    EXPECT_EQ(beat.description, "Install the Driver release that matches the Core.");
    EXPECT_EQ(beat.owner, synnax::rack::resource_id(rack.key));
    EXPECT_TRUE(beat.details.running);
}

/// @brief it should warn while the clock skew to the Core exceeds the threshold, and
/// return to its health once the skew is back under it.
TEST(stateTests, WarnsOnClockSkew) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto check_transport = std::make_unique<SkewedCheckClient>();
    auto *skewed = check_transport.get();
    skewed->skew = (3 * x::telem::SECOND).nanoseconds();
    client->connectivity = std::make_shared<synnax::connection::Checker>(
        std::move(check_transport),
        x::telem::HOUR,
        "0.0.0",
        "",
        x::telem::SECOND
    );
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<BeatRecorder>(
        synnax::rack::resource_id(rack.key)
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    const auto warned = transport->last_beat();
    EXPECT_EQ(warned.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(warned.message, "Clock out of sync with the Core");
    EXPECT_NE(warned.description.find("ahead of the Core"), std::string::npos);
    EXPECT_NE(warned.description.find("1s limit"), std::string::npos);
    EXPECT_NE(warned.description.find("NTP"), std::string::npos);
    EXPECT_TRUE(warned.details.running);
    skewed->skew = 0;
    client->connectivity->check();
    const auto beats = transport->beats.load();
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        beats + 2,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    const auto cleared = transport->last_beat();
    EXPECT_EQ(cleared.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(cleared.message, "Driver is running");
    EXPECT_EQ(cleared.description, "");
}

/// @brief it should say the clock is behind the Core when the skew is negative, and
/// print the skew without rounding it to the limit.
TEST(stateTests, WarnsWhenClockBehind) {
    const auto health = with_clock_skew(
        Health{},
        synnax::connection::State{
            .clock_skew = x::telem::TimeSpan::ZERO() - x::telem::SECOND -
                          400 * x::telem::MICROSECOND,
            .clock_skew_exceeded = true,
            .clock_skew_threshold = x::telem::SECOND,
        }
    );
    EXPECT_EQ(health.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(health.message, "Clock out of sync with the Core");
    EXPECT_EQ(
        health.description,
        "The clock of this host is 1s 400us behind the Core, more than the 1s limit. "
        "Synchronize both clocks with NTP. See " +
            CLOCK_TROUBLESHOOTING_URL + "."
    );
}

/// @brief it should report a health that is not a success over a clock skew warning.
TEST(stateTests, HealthOverridesClockSkew) {
    const Health error{
        .variant = synnax::status::VARIANT_ERROR,
        .message = "Driver 0.59.3 cannot work with Core 0.60.1",
        .description = "Install the Driver release that matches the Core.",
    };
    const auto health = with_clock_skew(
        error,
        synnax::connection::State{
            .clock_skew = 3 * x::telem::SECOND,
            .clock_skew_exceeded = true,
            .clock_skew_threshold = x::telem::SECOND,
        }
    );
    EXPECT_EQ(health.variant, error.variant);
    EXPECT_EQ(health.message, error.message);
    EXPECT_EQ(health.description, error.description);
}
}
