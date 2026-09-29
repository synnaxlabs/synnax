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
#include <cstdlib>
#include <mutex>
#include <set>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <poll.h>
#include <unistd.h>

#include "x/cpp/test/test.h"

#include "driver/can/slcan/slcan.h"

namespace driver::can::slcan {
namespace {
/// @brief plays an slcan adapter on the controller side of a pseudo-terminal. It
/// records each command and answers it with a carriage return, or with a bell when the
/// command is in rejected.
class Adapter {
    int controller = -1;
    std::atomic<bool> running = true;
    std::thread thread;
    std::mutex mu;
    std::vector<std::string> lines;

    void run() {
        std::string line;
        while (this->running) {
            pollfd pfd{.fd = this->controller, .events = POLLIN, .revents = 0};
            if (::poll(&pfd, 1, 10) <= 0) continue;
            char buf[256];
            const auto n = ::read(this->controller, buf, sizeof(buf));
            if (n <= 0) continue;
            for (ssize_t i = 0; i < n; i++) {
                if (buf[i] != '\r') {
                    line += buf[i];
                    continue;
                }
                {
                    std::lock_guard lock(this->mu);
                    this->lines.push_back(line);
                }
                if (!this->silent) this->reply(this->rejected.contains(line));
                line.clear();
            }
        }
    }

public:
    std::string terminal;
    std::set<std::string> rejected;
    bool silent = false;

    Adapter() {
        this->controller = ::posix_openpt(O_RDWR | O_NOCTTY);
        EXPECT_GE(this->controller, 0);
        EXPECT_EQ(::grantpt(this->controller), 0);
        EXPECT_EQ(::unlockpt(this->controller), 0);
        this->terminal = ::ptsname(this->controller);
    }

    ~Adapter() { this->unplug(); }

    /// @brief closes the controller side, as when the adapter is unplugged.
    void unplug() {
        this->running = false;
        if (this->thread.joinable()) this->thread.join();
        if (this->controller >= 0) ::close(this->controller);
        this->controller = -1;
    }

    void start() {
        this->thread = std::thread([this] { this->run(); });
    }

    /// @brief answers a command. The host may close its end before the answer, as it
    /// does after sending the close command.
    void reply(const bool rejected) const {
        const char answer = rejected ? '\a' : '\r';
        if (::write(this->controller, &answer, 1) != 1) { EXPECT_EQ(errno, EIO); }
    }

    void write(const std::string &data) const {
        ASSERT_EQ(
            ::write(this->controller, data.data(), data.size()),
            static_cast<ssize_t>(data.size())
        );
    }

    /// @returns the commands received so far once there are at least n, or after one
    /// second.
    std::vector<std::string> received(const std::size_t n) {
        const auto deadline = std::chrono::steady_clock::now() +
                              std::chrono::seconds(1);
        while (std::chrono::steady_clock::now() < deadline) {
            {
                std::lock_guard lock(this->mu);
                if (this->lines.size() >= n) return this->lines;
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
        std::lock_guard lock(this->mu);
        return this->lines;
    }
};

class SlcanTest : public ::testing::Test {
protected:
    Adapter adapter;
    Backend backend;

    [[nodiscard]] synnax::can::Properties
    props(const bool fd = false, const bool listen_only = false) const {
        synnax::can::Properties p;
        p.backend = synnax::can::BACKEND_SLCAN;
        p.channel = this->adapter.terminal;
        p.bitrate = 500000;
        p.fd = fd;
        p.data_bitrate = 2000000;
        p.listen_only = listen_only;
        return p;
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p) {
        this->adapter.start();
        auto [bus, err] = this->backend.open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }
};

Frame data_frame(const std::uint32_t id, const std::vector<std::uint8_t> &data) {
    Frame f;
    f.id = id;
    f.length = static_cast<std::uint8_t>(data.size());
    std::ranges::copy(data, f.data.begin());
    return f;
}
}

TEST_F(SlcanTest, OpensTheAdapterAtTheBitrate) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(this->adapter.received(3), std::vector<std::string>({"C", "S6", "O"}));
}

TEST_F(SlcanTest, SetsTheDataBitrateAndListenOnlyMode) {
    auto bus = this->open(this->props(true, true));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(
        this->adapter.received(4),
        std::vector<std::string>({"C", "S6", "Y2", "L"})
    );
    const auto err = bus->send(data_frame(0x123, {1}));
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel " + this->adapter.terminal + " is listen only");
}

TEST_F(SlcanTest, OpensAnAdapterThatRejectsTheCloseCommand) {
    this->adapter.rejected = {"C"};
    auto bus = this->open(this->props());
    EXPECT_NE(bus, nullptr);
}

TEST_F(SlcanTest, ReportsARejectedCommand) {
    this->adapter.rejected = {"S6"};
    this->adapter.start();
    const auto [bus, err] = this->backend.open(this->props());
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "the slcan adapter at " + this->adapter.terminal + " rejected the command 'S6'"
    );
    EXPECT_EQ(bus, nullptr);
}

TEST_F(SlcanTest, ReportsADeviceThatDoesNotAnswer) {
    this->adapter.silent = true;
    this->adapter.start();
    const auto [bus, err] = this->backend.open(this->props());
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_EQ(
        err.data,
        "the device at " + this->adapter.terminal +
            " did not answer the slcan command 'C'. Check that it is an slcan adapter"
    );
}

TEST_F(SlcanTest, RejectsAnUnsupportedBitrateBeforeTalkingToTheAdapter) {
    auto p = this->props();
    p.bitrate = 333;
    this->adapter.start();
    const auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_TRUE(this->adapter.received(1).empty());
}

TEST_F(SlcanTest, ReceivesFramesSplitAcrossReads) {
    auto bus = this->open(this->props(true));
    ASSERT_NE(bus, nullptr);
    this->adapter.received(4);
    const auto before = x::telem::TimeStamp::now();
    this->adapter.write("z\rt1232AB");
    this->adapter.write("CD\rb0109000102030405060708090A0B\r");
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x123);
    EXPECT_EQ(frame.length, 2);
    EXPECT_EQ(frame.data[0], 0xAB);
    EXPECT_EQ(frame.data[1], 0xCD);
    EXPECT_EQ(frame.clock, Clock::HOST);
    EXPECT_GE(frame.time, before);
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 0x10);
    EXPECT_TRUE(frame.fd);
    EXPECT_TRUE(frame.bitrate_switched);
    EXPECT_EQ(frame.length, 12);
    EXPECT_EQ(frame.data[11], 0x0B);
}

TEST_F(SlcanTest, ReturnsFalseWhenNoFrameArrives) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, 20 * x::telem::MILLISECOND)));
}

TEST_F(SlcanTest, SendsFrames) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    ASSERT_NIL(bus->send(data_frame(0x123, {0x11, 0x22})));
    EXPECT_EQ(this->adapter.received(4).back(), "t12321122");
}

TEST_F(SlcanTest, RefusesAnFDFrameOnAClassicBus) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    auto frame = data_frame(0x123, {1});
    frame.fd = true;
    const auto err = bus->send(frame);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "a CAN FD frame cannot be sent on a classic CAN bus");
}

TEST_F(SlcanTest, SendsAndReceivesAtTheSameTime) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    std::thread receiver([&] {
        Frame frame;
        EXPECT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
        EXPECT_EQ(frame.id, 0x42);
    });
    for (std::uint32_t i = 0; i < 10; i++)
        ASSERT_NIL(bus->send(data_frame(i, {})));
    this->adapter.write("t0420\r");
    receiver.join();
    EXPECT_EQ(this->adapter.received(13).size(), 13);
}

TEST_F(SlcanTest, ClosesTheAdapter) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    ASSERT_NIL(bus->close());
    ASSERT_NIL(bus->close());
    EXPECT_EQ(this->adapter.received(4).back(), "C");
}

TEST_F(SlcanTest, ListsNoChannelsWhileASerialPortIsPresent) {
    EXPECT_TRUE(ASSERT_NIL_P(this->backend.scan()).empty());
}

TEST_F(SlcanTest, ReportsAnUnpluggedAdapter) {
    auto bus = this->open(this->props());
    ASSERT_NE(bus, nullptr);
    this->adapter.received(3);
    this->adapter.unplug();
    Frame frame;
    const auto [received, err] = bus->receive(frame, x::telem::SECOND);
    EXPECT_FALSE(received);
    ASSERT_MATCHES(err, driver::errors::TEMPORARY_HARDWARE_ERROR);
    ASSERT_MATCHES(
        bus->send(data_frame(0x1, {})),
        driver::errors::TEMPORARY_HARDWARE_ERROR
    );
}
}
