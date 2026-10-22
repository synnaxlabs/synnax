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
#include <memory>
#include <mutex>
#include <string>
#include <thread>

#include "gtest/gtest.h"
#include <termios.h>

#include "x/cpp/test/test.h"

#include "driver/bus/testutil/contract.h"
#include "driver/bus/transport.h"
#include "driver/serial/port.h"
#include "driver/serial/pty.h"

namespace driver::serial {
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

TEST_F(PtyTest, ReadsAfterAWriteFromAnotherThread) {
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

TEST_F(PtyTest, RejectsOneAndAHalfStopBits) {
    auto p = this->props();
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
}

TEST_F(PtyTest, RejectsRS485OnAPortWithoutIt) {
    auto p = this->props();
    p.rs485 = true;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
}

TEST_F(PtyTest, ReportsTheDeviceAsUnreachableWhenThePeerCloses) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    ASSERT_OCCURRED_AS_P(port->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
}

TEST_F(PtyTest, OpensAgainThroughAStableLinkOnceTheDeviceReturns) {
    const auto link = std::filesystem::path(::testing::TempDir()) / "serial-by-id";
    std::filesystem::remove(link);
    std::filesystem::create_symlink(this->props().port, link);
    auto p = this->props();
    p.port = link.string();
    const auto first = ASSERT_NIL_P(Port::open(p));
    this->close_peer();
    ASSERT_OCCURRED_AS_P(first->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
    first->close(x::telem::SECOND);
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::UNREACHABLE_ERROR);
    this->open_peer();
    std::filesystem::remove(link);
    std::filesystem::create_symlink(this->props().port, link);
    const auto second = ASSERT_NIL_P(Port::open(p));
    this->peer_write("back");
    EXPECT_EQ(as_string(ASSERT_NIL_P(second->read(x::telem::SECOND)).data), "back");
}

TEST_F(PtyTest, FailsToReadAfterClose) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    port->close(x::telem::SECOND);
    ASSERT_OCCURRED_AS_P(port->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
}

TEST_F(PtyTest, SendsWrittenBytesBeforeClosing) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_NIL(port->write(bytes("bye"), x::telem::SECOND));
    std::string got;
    std::thread peer([&] {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
        got = this->peer_read(3);
    });
    port->close(x::telem::SECOND);
    peer.join();
    EXPECT_EQ(got, "bye");
}

TEST_F(PtyTest, RefusesASecondOpenOfAPortInUse) {
    const auto first = ASSERT_NIL_P(Port::open(this->props()));
    const auto [second, err] = Port::open(this->props());
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port " + this->props().port +
            ": the port is in use by another program"
    );
}

TEST_F(PtyTest, OpensAgainAfterTheFirstOpenCloses) {
    {
        const auto first = ASSERT_NIL_P(Port::open(this->props()));
        first->close(x::telem::SECOND);
    }
    const auto second = ASSERT_NIL_P(Port::open(this->props()));
    this->peer_write("again");
    EXPECT_EQ(as_string(ASSERT_NIL_P(second->read(x::telem::SECOND)).data), "again");
}

TEST_F(PtyTest, MarksLineErrorsInsteadOfIgnoringThem) {
    for (const auto &parity:
         {synnax::serial::PARITY_NONE_, synnax::serial::PARITY_EVEN_}) {
        auto p = this->props();
        p.parity = parity;
        const auto port = ASSERT_NIL_P(Port::open(p));
        const auto tio = this->termios();
        EXPECT_TRUE(tio.c_iflag & INPCK) << parity;
        EXPECT_TRUE(tio.c_iflag & PARMRK) << parity;
        EXPECT_FALSE(tio.c_iflag & (IGNPAR | ISTRIP | IGNBRK | BRKINT)) << parity;
        port->close(x::telem::SECOND);
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
    EXPECT_EQ(chunk.overruns, 0);
}

TEST(Port, FailsToOpenAMissingPort) {
    synnax::serial::Properties p;
    p.port = "/dev/synnax-missing-port";
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::UNREACHABLE_ERROR);
}

TEST(Port, RejectsInvalidSettingsBeforeOpening) {
    const auto invalid = [](auto edit) {
        synnax::serial::Properties p;
        p.port = "/dev/synnax-missing-port";
        edit(p);
        return Port::open(p);
    };
    using P = synnax::serial::Properties;
    ASSERT_OCCURRED_AS_P(invalid([](P &p) { p.port = ""; }), transport::CONFIG_ERROR);
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.baud_rate = 0; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.data_bits = 9; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.parity = "bogus"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.stop_bits = "3"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.flow_control = "bogus"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) {
            p.rs485 = true;
            p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
        }),
        transport::CONFIG_ERROR
    );
}

namespace {
/// @brief plays a serial device on a pseudo-terminal behind a stable link, as udev's
/// by-id links stay put while a device comes and goes.
class PtyRig final : public bus::testutil::Rig {
    Pty pty;
    std::filesystem::path link = std::filesystem::path(::testing::TempDir()) /
                                 "contract-serial";

    void point() const {
        std::filesystem::remove(this->link);
        std::filesystem::create_symlink(this->pty.path(), this->link);
    }

    static bus::Opener open(const std::string &path) {
        synnax::serial::Properties p;
        p.port = path;
        return bus::opener<Port>(p);
    }

public:
    PtyRig() {
        this->pty.open();
        this->point();
    }

    ~PtyRig() override { std::filesystem::remove(this->link); }

    bus::Opener opener() override { return open(this->link.string()); }

    bus::Opener absent() override { return open("/dev/synnax-missing-port"); }

    void send(const std::string &data) override { this->pty.write(data); }

    std::string receive(const std::size_t n) override { return this->pty.read(n); }

    void hang_up() override { this->pty.close(); }

    void restore() override {
        this->pty.open();
        this->point();
    }
};

const auto PTY = ::testing::Values(
    bus::testutil::Case{.name = "pty", .rig = [] { return std::make_unique<PtyRig>(); }}
);
}

using bus::testutil::StreamContract;
using bus::testutil::TransportContract;
INSTANTIATE_TEST_SUITE_P(Serial, TransportContract, PTY, bus::testutil::name);
INSTANTIATE_TEST_SUITE_P(Serial, StreamContract, PTY, bus::testutil::name);
}
