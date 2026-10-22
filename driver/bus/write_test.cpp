// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <functional>
#include <limits>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/read.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/bus/write.h"

namespace driver::bus {
using namespace testutil;

namespace {
using Bytes = std::vector<std::uint8_t>;

/// @brief a sink over an in-memory wire.
struct Harness {
    std::shared_ptr<Wire> wire = std::make_shared<Wire>();
    std::unique_ptr<Sink> sink;
    std::mutex mu;
    /// @brief each argument to the sink's set_warning, in order.
    std::vector<x::errors::Error> warnings;

    /// @param bindings maps each command channel to the index of the field it drives in
    /// message.
    Harness(
        const synnax::library::MessageEntry &message,
        const std::vector<std::pair<synnax::channel::Key, std::size_t>> &bindings,
        const std::optional<::synnax::bus::Framing> &framing,
        x::telem::NowFunc now = steady_now
    ) {
        ::synnax::bus::WriteConfig cfg;
        cfg.device = "dev";
        ::synnax::bus::WriteMessage wm{.message = message.key};
        std::vector<synnax::channel::Channel> channels;
        for (const auto &[ch, field]: bindings) {
            wm.fields.push_back({.field = field_key(message, field), .channel = ch});
            channels.push_back(data_channel(ch, 0));
        }
        cfg.messages = {wm};
        x::json::Parser parser(x::json::json::object());
        auto [resolved, err] = WriteConfig::resolve(
            parser,
            cfg,
            framing,
            library({message}),
            channels
        );
        EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
        EXPECT_FALSE(err) << err;
        resolved.device = "dev";
        this->sink = std::make_unique<Sink>(
            std::move(resolved),
            std::make_unique<ConnectionTransmitter>(acquire(this->wire)),
            std::move(now)
        );
        this->sink->set_warning = [this](const x::errors::Error &err) {
            std::lock_guard lock(this->mu);
            this->warnings.push_back(err);
        };
        EXPECT_FALSE(this->sink->start());
    }

    void command(const synnax::channel::Key key, const double value) const {
        x::telem::Frame fr(key, x::telem::Series(value, x::telem::FLOAT64_T));
        ASSERT_NIL(this->sink->write(fr));
    }

    [[nodiscard]] std::size_t count() const { return this->wire->written().size(); }

    /// @returns the last argument to set_warning, or nullopt when there is none.
    std::optional<x::errors::Error> warning() {
        std::lock_guard lock(this->mu);
        if (this->warnings.empty()) return std::nullopt;
        return this->warnings.back();
    }

    /// @returns the message of the last warning, or an empty string when the last call
    /// to set_warning cleared it or there was none.
    std::string message() {
        const auto w = this->warning();
        return w.has_value() ? w->data : "";
    }
};

const auto LONG = std::chrono::seconds(5);
const auto TICK = std::chrono::milliseconds(1);

const ::synnax::bus::Framing TWO_BYTES = ::synnax::bus::FixedFraming{.length = 2};
}

TEST(Sink, SendsAMessageWithoutAPeriodOncePerCommand) {
    const auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    Harness h(m, {{1, 0}}, TWO_BYTES);
    h.command(1, 7);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    h.command(1, 9);
    ASSERT_EVENTUALLY_EQ(h.count(), 2);
    EXPECT_EQ(h.wire->written()[0], (Bytes{0x07, 0x00}));
    EXPECT_EQ(h.wire->written()[1], (Bytes{0x09, 0x00}));
}

TEST(Sink, WaitsForEveryCommandChannelBeforeSending) {
    const auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    Harness h(m, {{1, 0}, {2, 1}}, TWO_BYTES);
    h.command(1, 5);
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(h.count(), 0);
    h.command(2, 6);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(h.wire->written()[0], (Bytes{0x05, 0x06}));
}

TEST(Sink, SendsAPeriodicMessageOnItsPeriodAndFallsSilentOnStop) {
    auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    m.period = 10 * x::telem::MILLISECOND;
    Harness h(m, {{1, 0}}, TWO_BYTES);
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(h.count(), 0);
    h.command(1, 3);
    ASSERT_EVENTUALLY_GE(h.count(), 3);
    for (const auto &w: h.wire->written())
        EXPECT_EQ(w, (Bytes{0x03, 0x00}));
    ASSERT_NIL(h.sink->stop());
    const auto stopped = h.count();
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(h.count(), stopped);
}

TEST(Sink, DoesNotReplayAStaleValueAfterARestart) {
    auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    m.period = 10 * x::telem::MILLISECOND;
    Harness h(m, {{1, 0}}, TWO_BYTES);
    h.command(1, 3);
    ASSERT_EVENTUALLY_GE(h.count(), 1);
    ASSERT_NIL(h.sink->stop());
    ASSERT_NIL(h.sink->start());
    const auto restarted = h.count();
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(h.count(), restarted);
}

TEST(Sink, SendsAPeriodicMessageWithNoFields) {
    auto m = binary_message("heartbeat", {binary_field("a", 0)});
    m.period = 10 * x::telem::MILLISECOND;
    Harness h(m, {}, std::nullopt);
    ASSERT_EVENTUALLY_GE(h.count(), 2);
    EXPECT_EQ(h.wire->written()[0], (Bytes{0x00}));
}

TEST(Sink, WritesTheValueOfAFieldIdentifier) {
    auto id = binary_field("kind", 0);
    auto m = binary_message("cmd", {id, binary_field("v", 8)});
    binary(m).identifier = synnax::library::FieldIdentifier{
        .field = id.key,
        .value = 0x42
    };
    Harness h(m, {{1, 1}}, std::nullopt);
    h.command(1, 7);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(h.wire->written()[0], (Bytes{0x42, 0x07}));
}

TEST(Sink, StartsATextLineWithItsPrefix) {
    const auto m = text_message("set", {delimited_field("v", 1)}, "SET,");
    Harness h(m, {{1, 0}}, ::synnax::bus::DelimiterFraming{});
    h.command(1, 5);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(text(h.wire->written()[0]), "SET,5\n");
}

TEST(Sink, ResendsTheNewestPayloadAfterTheConnectionRecovers) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->write_errs = {x::errors::Error(transport::UNREACHABLE_ERROR, "down")};
    }
    h.command(1, 1);
    ASSERT_EVENTUALLY_EQ(h.message(), "reconnecting to dev: down");
    EXPECT_EQ(h.count(), 0);
    h.command(1, 2);
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(h.count(), 1, LONG, TICK);
    EXPECT_EQ(h.wire->written()[0], Bytes{0x02});
    ASSERT_EVENTUALLY_TRUE(h.warning().has_value() && !*h.warning());
    EXPECT_EQ(h.wire->opens, 2);
}

TEST(Sink, ReturnsASendErrorThatNoResendCanFix) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->write_errs = {x::errors::Error(transport::CRITICAL_ERROR, "removed")};
    }
    h.command(1, 1);
    ASSERT_EVENTUALLY_EQ(h.message(), "removed");
    x::telem::Frame fr(1, x::telem::Series(2.0, x::telem::FLOAT64_T));
    ASSERT_OCCURRED_AS(h.sink->write(fr), transport::CRITICAL_ERROR);
}

TEST(Sink, OpensTheConnectionOnStart) {
    const auto wire = std::make_shared<Wire>();
    Sink sink(
        write_config(binary_message("cmd", {binary_field("a", 0)}), {{1, 0}}, {}),
        std::make_unique<ConnectionTransmitter>(acquire(wire))
    );
    ASSERT_NIL(sink.start());
    EXPECT_EQ(wire->opens, 1);
    ASSERT_NIL(sink.stop());
}

TEST(Sink, FailsToStartWithInvalidDeviceProperties) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs = {x::errors::Error(transport::CONFIG_ERROR, "no such baud")};
    Sink sink(
        write_config(binary_message("cmd", {binary_field("a", 0)}), {{1, 0}}, {}),
        std::make_unique<ConnectionTransmitter>(acquire(wire))
    );
    ASSERT_OCCURRED_AS(sink.start(), transport::CONFIG_ERROR);
}

TEST(Sink, StartsWhileTheDeviceIsUnreachableAndSendsWhenItReturns) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs = {x::errors::Error(transport::UNREACHABLE_ERROR, "no port")};
    Sink sink(
        write_config(binary_message("cmd", {binary_field("a", 0)}), {{1, 0}}, {}),
        std::make_unique<ConnectionTransmitter>(acquire(wire))
    );
    std::mutex mu;
    std::string warning;
    sink.set_warning = [&](const x::errors::Error &err) {
        std::lock_guard lock(mu);
        warning = err.data;
    };
    ASSERT_NIL(sink.start());
    x::telem::Frame fr(1, x::telem::Series(4.0, x::telem::FLOAT64_T));
    ASSERT_NIL(sink.write(fr));
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(wire->written().size(), 1, LONG, TICK);
    EXPECT_EQ(wire->written()[0], Bytes{0x04});
    ASSERT_NIL(sink.stop());
}

TEST(Sink, RefusesAValueOutsideTheRangeOfItsField) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    m.period = 10 * x::telem::MILLISECOND;
    Harness h(m, {{1, 0}}, std::nullopt);
    h.command(1, 7);
    ASSERT_EVENTUALLY_GE(h.count(), 1);
    h.command(1, 300);
    ASSERT_EVENTUALLY_EQ(h.message(), "cmd.a: 300 is outside the range 0 to 255");
    const auto refused = h.count();
    ASSERT_EVENTUALLY_GE(h.count(), refused + 2);
    for (const auto &w: h.wire->written())
        EXPECT_EQ(w, Bytes{0x07});
}

TEST(Sink, RefusesNaN) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    h.command(1, std::numeric_limits<double>::quiet_NaN());
    ASSERT_EVENTUALLY_EQ(h.message(), "cmd.a: NaN is outside the range 0 to 255");
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(h.count(), 0);
}

TEST(Sink, ClearsARefusalWarningOneSecondAfterTheLastRefusal) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Clock clock;
    Harness h(m, {{1, 0}}, std::nullopt, clock.func());
    h.command(1, -1);
    ASSERT_EVENTUALLY_EQ(h.message(), "cmd.a: -1 is outside the range 0 to 255");
    clock.advance(WARNING_HOLD);
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        h.warning().has_value() && !*h.warning(),
        LONG,
        TICK
    );
}

TEST(Sink, WarnsWhichFieldsAMessageWaitsFor) {
    const auto m = binary_message(
        "cmd",
        {binary_field("a", 0), binary_field("b", 8), binary_field("c", 16)}
    );
    Harness h(m, {{1, 0}, {2, 1}, {3, 2}}, std::nullopt);
    h.command(1, 1);
    ASSERT_EVENTUALLY_EQ(h.message(), "cmd waits for b, c");
    h.command(3, 1);
    ASSERT_EVENTUALLY_EQ(h.message(), "cmd waits for b");
    h.command(2, 1);
    ASSERT_EVENTUALLY_TRUE(h.warning().has_value() && !*h.warning());
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
}

TEST(Sink, SchedulesAPeriodicMessageOnTheSteadyClock) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    m.period = 100 * x::telem::MILLISECOND;
    Clock clock;
    Harness h(m, {{1, 0}}, std::nullopt, clock.func());
    h.command(1, 3);
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    EXPECT_EQ(h.count(), 0);
    clock.advance(100 * x::telem::MILLISECOND);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    clock.advance(50 * x::telem::MILLISECOND);
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    EXPECT_EQ(h.count(), 1);
    clock.advance(50 * x::telem::MILLISECOND);
    ASSERT_EVENTUALLY_EQ(h.count(), 2);
    clock.advance(x::telem::SECOND);
    ASSERT_EVENTUALLY_EQ(h.count(), 3);
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    EXPECT_EQ(h.count(), 3);
    clock.advance(100 * x::telem::MILLISECOND);
    ASSERT_EVENTUALLY_EQ(h.count(), 4);
}

TEST(Sink, KeepsOnlyTheNewestPayloadDuringASlowWrite) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->write_delay = 200 * x::telem::MILLISECOND;
    }
    h.command(1, 1);
    ASSERT_EVENTUALLY_TRUE([&] {
        std::lock_guard lock(h.wire->mu);
        return h.wire->started == 1;
    }());
    for (int v = 2; v <= 50; v++)
        h.command(1, v);
    ASSERT_EVENTUALLY_EQ(h.count(), 2);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    const auto written = h.wire->written();
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[0], Bytes{1});
    EXPECT_EQ(written[1], Bytes{50});
}

TEST(Sink, WaitsForAPollOfTheSameDeviceToGetItsReply) {
    const auto wire = std::make_shared<Wire>();
    const auto connections = std::make_shared<Connections>();
    const ::synnax::bus::Framing newline = ::synnax::bus::DelimiterFraming{};
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "MEAS?";
    Source source(
        read_config(
            {volt},
            newline,
            nullptr,
            {.rate = x::telem::Rate(1000), .timeout = x::telem::SECOND}
        ),
        acquire(wire, connections)
    );
    ASSERT_NIL(source.start());
    Sink sink(
        write_config(
            text_message("set", {tagged_field("v", "VOLT ")}),
            {{7, 0}},
            newline
        ),
        std::make_unique<ConnectionTransmitter>(acquire(wire, connections))
    );
    sink.set_warning = [](const x::errors::Error &err) { EXPECT_FALSE(err) << err; };
    ASSERT_NIL(sink.start());
    std::thread reader([&] {
        x::breaker::Breaker breaker{x::breaker::Config{.name = "read"}};
        breaker.start();
        x::telem::Frame fr;
        EXPECT_FALSE(source.read(breaker, fr).error);
        breaker.stop();
    });
    ASSERT_EVENTUALLY_EQ(wire->written().size(), 1);
    x::telem::Frame cmd(7, x::telem::Series(5.0, x::telem::FLOAT64_T));
    ASSERT_NIL(sink.write(cmd));
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    EXPECT_EQ(wire->written().size(), 1);
    wire->push(bytes("12\n"));
    reader.join();
    ASSERT_EVENTUALLY_EQ(wire->written().size(), 2);
    EXPECT_EQ(text(wire->written()[0]), "MEAS?\n");
    EXPECT_EQ(text(wire->written()[1]), "VOLT 5\n");
    ASSERT_NIL(sink.stop());
    ASSERT_NIL(source.stop());
    EXPECT_EQ(wire->opens, 1);
}
}
