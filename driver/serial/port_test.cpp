// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cerrno>
#include <chrono>
#include <cstddef>
#include <filesystem>
#include <functional>
#include <mutex>
#include <string>
#include <system_error>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <sys/ioctl.h>
#include <termios.h>
#include <unistd.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
#ifdef __linux__
/// @returns true when the terminal at path refuses programs other than its holder. A
/// program without root cannot open it at all, and root can read its exclusive flag.
bool exclusive(const std::string &path) {
    const int fd = ::open(path.c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK);
    if (fd < 0) return errno == EBUSY;
    int flag = 0;
    const bool read = ::ioctl(fd, TIOCGEXCL, &flag) == 0;
    ::close(fd);
    return read && flag != 0;
}
#endif

/// @brief expects got to be want, with its message.
void expect_same(const x::errors::Error &got, const x::errors::Error &want) {
    EXPECT_TRUE(got.matches(want)) << got;
    EXPECT_EQ(got.data, want.data);
}
}

/// @brief it should read the bytes that the device writes, stamped with the time they
/// arrived.
TEST_F(PtyTest, ReadsWhatThePeerWrites) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const auto before = x::telem::TimeStamp::now();
    this->peer_write("hello");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

/// @brief it should write a query and read the answer of the device.
TEST_F(PtyTest, AnswersAQueryInARoundTrip) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_NIL(port->write(bytes("MEAS:VOLT?\n"), x::telem::SECOND));
    EXPECT_EQ(this->peer_read(11), "MEAS:VOLT?\n");
    this->peer_write("12.5\n");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "12.5\n");
}

/// @brief it should return an empty chunk when nothing arrives before the timeout.
TEST_F(PtyTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(port->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

/// @brief it should read again after a read times out.
TEST_F(PtyTest, ReadsAgainAfterATimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_TRUE(ASSERT_NIL_P(port->read(10 * x::telem::MILLISECOND)).data.empty());
    this->peer_write("late");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "late");
}

/// @brief it should read and write from different threads that take turns.
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

/// @brief it should fail a write that does not finish before the timeout.
TEST_F(PtyTest, FailsAWriteThatTimesOut) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const std::string data(std::size_t{4} * 1024 * 1024, 'x');
    const auto err = port->write(bytes(data), 50 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to write to serial port " + this->path() + ": " +
            std::make_error_code(std::errc::timed_out).message()
    );
}

/// @brief it should close at once after a failed write and return its error on each
/// later call.
TEST_F(PtyTest, ClosesAtOnceAndKeepsTheErrorAfterAFailedWrite) {
    const auto failed = ASSERT_NIL_P(Port::open(this->props()));
    const std::string data(std::size_t{4} * 1024 * 1024, 'x');
    const auto err = failed->write(bytes(data), 50 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    expect_same(failed->write(bytes("x"), x::telem::SECOND), err);
    expect_same(failed->read(x::telem::SECOND).second, err);
    const auto next = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("next");
    EXPECT_EQ(as_string(ASSERT_NIL_P(next->read(x::telem::SECOND)).data), "next");
}

/// @brief it should close at once after a failed read and return its error on each
/// later call.
TEST_F(PtyTest, ClosesAtOnceAndKeepsTheErrorAfterAFailedRead) {
    const auto failed = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    const auto [chunk, err] = failed->read(x::telem::SECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    expect_same(failed->read(x::telem::SECOND).second, err);
    expect_same(failed->write(bytes("x"), x::telem::SECOND), err);
}

/// @brief it should apply two stop bits and hardware flow control to the terminal.
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

/// @brief it should apply software flow control to the terminal.
TEST_F(PtyTest, AppliesSoftwareFlowControl) {
    auto p = this->props();
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    const auto port = ASSERT_NIL_P(Port::open(p));
    const auto tio = this->termios();
    EXPECT_EQ(tio.c_iflag & (IXON | IXOFF), static_cast<tcflag_t>(IXON | IXOFF));
    EXPECT_FALSE(tio.c_cflag & CRTSCTS);
}

/// @brief it should refuse 1.5 stop bits.
TEST_F(PtyTest, RejectsOneAndAHalfStopBits) {
    auto p = this->props();
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_TRUE(err.data.ends_with(" does not support 1.5 stop bits")) << err.data;
}

/// @brief it should report the device as unreachable when it hangs up.
TEST_F(PtyTest, ReportsTheDeviceAsUnreachableWhenThePeerCloses) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    const auto [chunk, err] = port->read(x::telem::SECOND);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(
        err.data.starts_with("failed to read from serial port " + this->path() + ": ")
    ) << err.data;
}

/// @brief it should open again through a stable link once the device returns.
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

/// @brief it should send the written bytes before it closes.
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

/// @brief it should refuse a second open of a port in use and keep the first port
/// exclusive.
TEST_F(PtyTest, RefusesASecondOpenOfAPortInUse) {
    const auto first = ASSERT_NIL_P(Port::open(this->props()));
    const auto [second, err] = Port::open(this->props());
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port " + this->path() +
            ": the port is in use by another program"
    );
#ifdef __linux__
    // A macOS pseudo-terminal lets programs without root open past TIOCEXCL.
    EXPECT_TRUE(exclusive(this->path()));
#endif
}

/// @brief it should refuse settings outside the schema and release the port.
TEST_F(PtyTest, RefusesSettingsOutsideTheSchemaAndReleasesThePort) {
    auto p = this->props();
    p.data_bits = 9;
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "data bits must be 5, 6, 7, or 8, got 9");
    ASSERT_NIL_P(Port::open(this->props()));
}

/// @brief it should open again after the first port closes.
TEST_F(PtyTest, OpensAgainAfterTheFirstPortCloses) {
    ASSERT_NIL_P(Port::open(this->props())).reset();
    const auto second = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("again");
    EXPECT_EQ(as_string(ASSERT_NIL_P(second->read(x::telem::SECOND)).data), "again");
}

/// @brief it should drop the bytes that arrived before it applied the settings.
TEST_F(PtyTest, FlushesBytesThatArrivedBeforeTheSettings) {
    this->peer_write("stale");
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("fresh");
    EXPECT_EQ(as_string(ASSERT_NIL_P(port->read(x::telem::SECOND)).data), "fresh");
}

/// @brief it should mark line errors instead of ignoring them.
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

/// @brief it should deliver a 0xFF byte once while it marks line errors.
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

/// @brief it should report no overruns on a port that counts none.
TEST_F(PtyTest, ReportsNoOverrunsOnAPortThatCountsNone) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("x");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "x");
    EXPECT_EQ(chunk.overruns, 0);
}

/// @brief it should fail to open a port that does not exist.
TEST(Port, FailsToOpenAMissingPort) {
    synnax::serial::Properties p;
    p.port = "/dev/synnax-missing-port";
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to open serial port " + p.port + ": "))
        << err.data;
}

/// @brief it should refuse an empty port path.
TEST(Port, RejectsAnEmptyPath) {
    const auto [port, err] = Port::open(synnax::serial::Properties{});
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "serial port path is required");
}
}
