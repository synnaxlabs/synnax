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
#include <functional>
#include <mutex>
#include <string>
#include <system_error>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"
#include <termios.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @brief expects got to be want, with its message.
void expect_same(const x::errors::Error &got, const x::errors::Error &want) {
    EXPECT_TRUE(got.matches(want)) << got;
    EXPECT_EQ(got.data, want.data);
}
}

TEST_F(PtyTest, ReadsWhatThePeerWrites) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const auto before = x::telem::TimeStamp::now();
    this->peer_write("hello");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

TEST_F(PtyTest, AnswersAQueryInARoundTrip) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_NIL(port->write(bytes("MEAS:VOLT?\n"), x::telem::SECOND));
    EXPECT_EQ(this->peer_read(11), "MEAS:VOLT?\n");
    this->peer_write("12.5\n");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "12.5\n");
}

TEST_F(PtyTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(port->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(PtyTest, ReadsAgainAfterATimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_TRUE(ASSERT_NIL_P(port->read(10 * x::telem::MILLISECOND)).data.empty());
    this->peer_write("late");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "late");
}

TEST_F(PtyTest, ReadsAndWritesFromDifferentThreads) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    std::mutex mu;
    std::string got;
    std::atomic running = true;
    std::thread reader([&] {
        while (running) {
            {
                std::lock_guard lock(mu);
                got += as_string(
                    ASSERT_NIL_P(port->read(20 * x::telem::MILLISECOND)).data
                );
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    });
    for (int i = 0; i < 5; i++) {
        {
            std::lock_guard lock(mu);
            ASSERT_NIL(port->write(bytes("Q?\n"), x::telem::SECOND));
        }
        ASSERT_EQ(this->peer_read(3), "Q?\n");
        this->peer_write(std::to_string(i));
        ASSERT_EVENTUALLY_TRUE([&] {
            std::lock_guard lock(mu);
            return got.size() == static_cast<std::size_t>(i + 1);
        }());
    }
    running = false;
    reader.join();
    EXPECT_EQ(got, "01234");
}

TEST_F(PtyTest, FailsAWriteThatTimesOut) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const std::string data(4 * 1024 * 1024, 'x');
    const auto err = port->write(bytes(data), 50 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to write to serial port " + this->path() + ": " +
            std::make_error_code(std::errc::timed_out).message()
    );
}

TEST_F(PtyTest, ClosesAtOnceAndKeepsTheErrorAfterAFailedWrite) {
    const auto failed = ASSERT_NIL_P(Port::open(this->props()));
    const std::string data(4 * 1024 * 1024, 'x');
    const auto err = failed->write(bytes(data), 50 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    expect_same(failed->write(bytes("x"), x::telem::SECOND), err);
    expect_same(failed->read(x::telem::SECOND).second, err);
    const auto next = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("next");
    EXPECT_EQ(as_string(ASSERT_NIL_P(next->read(x::telem::SECOND)).data), "next");
}

TEST_F(PtyTest, ClosesAtOnceAndKeepsTheErrorAfterAFailedRead) {
    const auto failed = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    const auto [chunk, err] = failed->read(x::telem::SECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    expect_same(failed->read(x::telem::SECOND).second, err);
    expect_same(failed->write(bytes("x"), x::telem::SECOND), err);
}

TEST_F(PtyTest, AppliesStopBitsAndHardwareFlowControl) {
    auto p = this->props();
    p.stop_bits = synnax::serial::STOP_BITS_TWO;
    p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
    const auto port = ASSERT_NIL_P(Port::open(p));
    const auto tio = this->termios();
    EXPECT_TRUE(tio.c_cflag & CSTOPB);
    EXPECT_TRUE(tio.c_cflag & CRTSCTS);
    EXPECT_FALSE(tio.c_iflag & (IXON | IXOFF));
}

TEST_F(PtyTest, AppliesSoftwareFlowControl) {
    auto p = this->props();
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    const auto port = ASSERT_NIL_P(Port::open(p));
    const auto tio = this->termios();
    EXPECT_EQ(tio.c_iflag & (IXON | IXOFF), static_cast<tcflag_t>(IXON | IXOFF));
    EXPECT_FALSE(tio.c_cflag & CRTSCTS);
}

TEST_F(PtyTest, RejectsOneAndAHalfStopBits) {
    auto p = this->props();
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_TRUE(err.data.ends_with(" does not support 1.5 stop bits")) << err.data;
}

TEST_F(PtyTest, ReportsTheDeviceAsUnreachableWhenThePeerCloses) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    const auto [chunk, err] = port->read(x::telem::SECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(
        err.data.starts_with("failed to read from serial port " + this->path() + ": ")
    ) << err.data;
}

TEST_F(PtyTest, OpensAgainThroughAStableLinkOnceTheDeviceReturns) {
    const auto link = std::filesystem::path(::testing::TempDir()) / "serial-by-id";
    std::filesystem::remove(link);
    std::filesystem::create_symlink(this->path(), link);
    auto p = this->props();
    p.port = link.string();
    auto first = ASSERT_NIL_P(Port::open(p));
    // udev removes the link on unplug. A link left on the freed terminal could reach a
    // terminal that another process opens.
    this->close_peer();
    std::filesystem::remove(link);
    ASSERT_OCCURRED_AS_P(first->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    first.reset();
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::UNREACHABLE_ERROR);
    this->open_peer();
    std::filesystem::create_symlink(this->path(), link);
    const auto second = ASSERT_NIL_P(Port::open(p));
    std::filesystem::remove(link);
    this->peer_write("back");
    EXPECT_EQ(as_string(ASSERT_NIL_P(second->read(x::telem::SECOND)).data), "back");
}

TEST_F(PtyTest, SendsWrittenBytesBeforeClosing) {
    auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_NIL(port->write(bytes("bye"), x::telem::SECOND));
    std::string got;
    std::thread peer([&] {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
        got = this->peer_read(3);
    });
    port.reset();
    peer.join();
    EXPECT_EQ(got, "bye");
}

TEST_F(PtyTest, RefusesASecondOpenOfAPortInUse) {
    const auto first = ASSERT_NIL_P(Port::open(this->props()));
    const auto [second, err] = Port::open(this->props());
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port " + this->path() +
            ": the port is in use by another program"
    );
}

TEST_F(PtyTest, OpensAgainAfterTheFirstPortCloses) {
    ASSERT_NIL_P(Port::open(this->props())).reset();
    const auto second = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("again");
    EXPECT_EQ(as_string(ASSERT_NIL_P(second->read(x::telem::SECOND)).data), "again");
}

TEST_F(PtyTest, MarksLineErrorsInsteadOfIgnoringThem) {
    for (const auto *parity:
         {synnax::serial::PARITY_NONE_, synnax::serial::PARITY_EVEN_}) {
        SCOPED_TRACE(parity);
        auto p = this->props();
        p.parity = parity;
        const auto port = ASSERT_NIL_P(Port::open(p));
        const auto tio = this->termios();
        EXPECT_TRUE(tio.c_iflag & INPCK);
        EXPECT_TRUE(tio.c_iflag & PARMRK);
        EXPECT_FALSE(tio.c_iflag & (IGNPAR | ISTRIP | IGNBRK | BRKINT));
    }
}

TEST_F(PtyTest, DeliversA0xFFByteOnceWhileMarkingLineErrors) {
    auto p = this->props();
    p.parity = synnax::serial::PARITY_EVEN_;
    const auto port = ASSERT_NIL_P(Port::open(p));
    this->peer_write(std::string("\xFF\x01\xFF", 3));
    std::string got;
    std::size_t errors = 0;
    for (int i = 0; i < 100 && got.size() < 3; i++) {
        const auto chunk = ASSERT_NIL_P(port->read(10 * x::telem::MILLISECOND));
        got += as_string(chunk.data);
        errors += chunk.errors;
    }
    EXPECT_EQ(got, std::string("\xFF\x01\xFF", 3));
    EXPECT_EQ(errors, 0);
}

TEST_F(PtyTest, ReportsNoOverrunsOnAPortThatCountsNone) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("x");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "x");
    EXPECT_EQ(chunk.overruns, 0);
}

TEST(Port, FailsToOpenAMissingPort) {
    synnax::serial::Properties p;
    p.port = "/dev/synnax-missing-port";
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to open serial port " + p.port + ": "))
        << err.data;
}

TEST(Port, RejectsInvalidSettingsBeforeOpening) {
    using P = synnax::serial::Properties;
    const std::vector<std::pair<std::function<void(P &)>, std::string>> cases{
        {[](P &p) { p.port = ""; }, "serial port path is required"},
        {[](P &p) { p.baud_rate = 0; }, "baud rate must be greater than zero"},
        {[](P &p) { p.data_bits = 9; }, "data bits must be 5, 6, 7, or 8, got 9"},
        {[](P &p) { p.data_bits = 4; }, "data bits must be 5, 6, 7, or 8, got 4"},
        {[](P &p) { p.parity = "bogus"; }, "unknown parity 'bogus'"},
        {[](P &p) { p.stop_bits = "3"; }, "unknown stop bits '3'"},
        {[](P &p) { p.flow_control = "bogus"; }, "unknown flow control 'bogus'"},
    };
    for (const auto &[edit, message]: cases) {
        P p;
        p.port = "/dev/synnax-missing-port";
        edit(p);
        const auto [port, err] = Port::open(p);
        ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
        EXPECT_EQ(err.data, message);
    }
}
}
