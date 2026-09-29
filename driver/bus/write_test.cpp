// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <functional>
#include <future>
#include <memory>
#include <mutex>
#include <optional>
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
        const std::optional<::synnax::bus::Framing> &framing
    ) {
        ::synnax::bus::WriteConfig cfg;
        cfg.device = "dev";
        ::synnax::bus::WriteMessage wm{.message = message.key};
        std::vector<synnax::channel::Channel> channels;
        for (const auto &[ch, field]: bindings) {
            wm.fields.push_back({.field = key(message.fields[field]), .channel = ch});
            channels.push_back(data_channel(ch, 0));
        }
        cfg.messages = {wm};
        x::json::Parser parser(x::json::json::object());
        auto resolved = WriteConfig::resolve(
            parser,
            cfg,
            framing,
            library({message}),
            channels
        );
        EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
        this->sink = std::make_unique<Sink>(
            std::move(resolved),
            std::make_unique<ConnectionTransmitter>(acquire(this->wire))
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
};

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
    m.identifier = synnax::library::FieldIdentifier{.field = id.key, .value = 0x42};
    Harness h(m, {{1, 1}}, std::nullopt);
    h.command(1, 7);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(h.wire->written()[0], (Bytes{0x42, 0x07}));
}

TEST(Sink, StartsATextLineWithItsToken) {
    const auto m = text_message(
        "set",
        {delimited_field("v", 1)},
        synnax::library::TokenIdentifier{.prefix = "SET,"}
    );
    Harness h(m, {{1, 0}}, ::synnax::bus::DelimiterFraming{});
    h.command(1, 5);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(text(h.wire->written()[0]), "SET,5\n");
}

TEST(Sink, WarnsWhenTheDeviceIsUnreachableAndRecovers) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->open_errs = {transport::UNREACHABLE_ERROR};
    }
    h.command(1, 1);
    ASSERT_EVENTUALLY_TRUE(h.warning().has_value());
    EXPECT_TRUE(h.warning()->matches(transport::UNREACHABLE_ERROR));
    EXPECT_EQ(h.count(), 0);
    h.command(1, 2);
    ASSERT_EVENTUALLY_EQ(h.count(), 1);
    EXPECT_EQ(h.wire->written()[0], Bytes{0x02});
    ASSERT_EVENTUALLY_TRUE(h.warning().has_value() && !*h.warning());
}

TEST(Sink, KeepsOnlyTheNewestUnsentPayloadOfAMessage) {
    const auto m = binary_message("cmd", {binary_field("a", 0)});
    Harness h(m, {{1, 0}}, std::nullopt);
    std::promise<void> release;
    const auto released = release.get_future().share();
    std::atomic<bool> blocked = false;
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->on_write = [&](Wire &, std::span<const std::uint8_t>) {
            if (blocked.exchange(true)) return;
            released.wait();
        };
    }
    h.command(1, 1);
    ASSERT_EVENTUALLY_TRUE(blocked.load());
    for (int v = 2; v <= 50; v++)
        h.command(1, v);
    release.set_value();
    ASSERT_EVENTUALLY_EQ(h.count(), 2);
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    const auto written = h.wire->written();
    ASSERT_EQ(written.size(), 2);
    EXPECT_EQ(written[0], Bytes{1});
    EXPECT_EQ(written[1], Bytes{50});
    std::lock_guard lock(h.wire->mu);
    h.wire->on_write = nullptr;
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
