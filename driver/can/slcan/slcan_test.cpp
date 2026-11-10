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
#include <cerrno>
#include <chrono>
#include <cstdlib>
#include <format>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <poll.h>
#include <sys/ioctl.h>
#include <unistd.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/can/slcan/slcan.h"

namespace driver::can::slcan {
namespace {
/// @brief the firmware an Adapter plays.
enum class Profile {
    /// @brief LAWICEL CAN232, from section 2.0 of its manual: every command gets a
    /// carriage return or a bell, and each command works only in the state it lists.
    LAWICEL,
    /// @brief stock canable2-fw, from src/slcan.c and src/can.c: only V gets an answer,
    /// settings change only off the bus, O always opens in normal mode, and L is
    /// unknown.
    CANABLE,
    /// @brief the CANable 2.5 firmware, from Firmware/Slcan/control.c: every command
    /// but C gets a carriage return or a bell, M1 makes the next O open in bus
    /// monitoring mode, and L sets bus load reports.
    CANABLE_2_5,
    /// @brief a serial device that is not an slcan adapter and never answers.
    MUTE,
};

constexpr auto OK = "\r";
constexpr auto ERR = "\a";

bool is_frame(const std::string &line) {
    return !line.empty() && std::string_view("tTrRdDbB").contains(line[0]);
}

/// @brief plays an slcan adapter on the controller side of a pseudo-terminal.
class Adapter {
public:
    /// @brief what the adapter's controller is doing.
    struct State {
        bool open = false;
        bool silent = false;
        std::string bitrate;
        std::string data_bitrate;
        /// @brief the frame commands the adapter took.
        std::vector<std::string> sent;
    };

    std::string terminal;

private:
    const Profile profile;
    int controller = -1;
    std::atomic<bool> running = true;
    std::thread thread;
    std::mutex mu;
    std::vector<std::string> lines;
    State state;
    std::string previous;
    /// @brief true when the CANable 2.5 firmware opens in bus monitoring mode next.
    bool monitor = false;

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
                std::string answer;
                {
                    std::lock_guard lock(this->mu);
                    this->lines.push_back(line);
                    answer = this->handle(line);
                }
                this->reply(answer);
                line.clear();
            }
        }
    }

    std::string handle(const std::string &line) {
        switch (this->profile) {
            case Profile::LAWICEL:
                return this->lawicel(line);
            case Profile::CANABLE:
                return this->canable(line);
            case Profile::CANABLE_2_5:
                return this->canable_2_5(line);
            case Profile::MUTE:
                return "";
        }
        return "";
    }

    std::string lawicel(const std::string &line) {
        if (line == "V") return "V1013\r";
        if (line.size() == 2 && line[0] == 'S' && line[1] >= '0' && line[1] <= '8') {
            if (this->state.open) return ERR;
            this->state.bitrate = line;
            return OK;
        }
        if (line == "O" || line == "L") {
            if (this->state.open || this->state.bitrate.empty()) return ERR;
            this->state.open = true;
            this->state.silent = line == "L";
            return OK;
        }
        if (line == "C") {
            if (!this->state.open) return ERR;
            this->state.open = false;
            return OK;
        }
        if (line.starts_with('t') || line.starts_with('T') || line.starts_with('r') ||
            line.starts_with('R')) {
            if (!this->state.open || this->state.silent) return ERR;
            this->state.sent.push_back(line);
            return OK;
        }
        return ERR;
    }

    std::string canable(std::string line) {
        // An empty line runs the previous command again, as slcan.c's FIXME says.
        if (line.empty()) line = this->previous;
        this->previous = line;
        if (is_frame(line)) {
            this->state.sent.push_back(line);
            return "";
        }
        if (line == "V")
            return "b1ff43c5 https://github.com/normaldotcom/canable2-fw\r";
        if (line == "C") this->state.open = false;
        if (this->state.open) return "";
        if (line == "O") {
            // can_enable sets FDCAN_MODE_NORMAL, which discards the mode M1 set.
            this->state.open = true;
            this->state.silent = false;
        }
        if (line.size() == 2 && line[0] == 'S' && line[1] >= '0' && line[1] <= '9')
            this->state.bitrate = line;
        if (line == "Y2" || line == "Y5") this->state.data_bitrate = line;
        return "";
    }

    std::string canable_2_5(const std::string &line) {
        if (line.empty()) return OK;
        if (line == "V") return "+Board: MultiBoard\tMCU: STM32G431\tSlcan: 105\r";
        if (line == "C") {
            this->state.open = false;
            this->monitor = false;
            return "";
        }
        if (line.starts_with('L')) return OK;
        if (is_frame(line)) {
            if (!this->state.open || this->state.silent) return ERR;
            this->state.sent.push_back(line);
            return OK;
        }
        if (this->state.open) return ERR;
        if (line.size() == 2 && line[0] == 'S' && line[1] >= '0' && line[1] <= '9') {
            this->state.bitrate = line;
            return OK;
        }
        if (line == "Y2" || line == "Y5") {
            this->state.data_bitrate = line;
            return OK;
        }
        if (line == "M0" || line == "M1") {
            this->monitor = line == "M1";
            return OK;
        }
        if (line == "O") {
            this->state.open = true;
            this->state.silent = this->monitor;
            return OK;
        }
        return ERR;
    }

    /// @brief writes an answer. The host may close its end first, as it does after
    /// sending the close command.
    void reply(const std::string &answer) const {
        if (answer.empty()) return;
        const auto n = ::write(this->controller, answer.data(), answer.size());
        if (std::cmp_not_equal(n, answer.size())) { EXPECT_EQ(errno, EIO); }
    }

public:
    explicit Adapter(const Profile profile):
        profile(profile), controller(::posix_openpt(O_RDWR | O_NOCTTY)) {
        EXPECT_GE(this->controller, 0);
        EXPECT_EQ(::grantpt(this->controller), 0);
        EXPECT_EQ(::unlockpt(this->controller), 0);
        this->terminal = ::ptsname(this->controller);
        this->thread = std::thread([this] { this->run(); });
    }

    ~Adapter() { this->unplug(); }

    /// @brief closes the controller side, as when the adapter is unplugged.
    void unplug() {
        this->running = false;
        if (this->thread.joinable()) this->thread.join();
        if (this->controller >= 0) ::close(this->controller);
        this->controller = -1;
    }

    void write(const std::string &data) const {
        ASSERT_EQ(
            ::write(this->controller, data.data(), data.size()),
            static_cast<ssize_t>(data.size())
        );
    }

    /// @brief waits until the host has read every byte written to it.
    void drained() const {
        const int host = ::open(
            this->terminal.c_str(),
            O_RDONLY | O_NOCTTY | O_NONBLOCK
        );
        ASSERT_GE(host, 0);
        int pending = 1;
        const auto deadline = std::chrono::steady_clock::now() +
                              std::chrono::seconds(5);
        while (pending > 0 && std::chrono::steady_clock::now() < deadline) {
            ASSERT_EQ(::ioctl(host, FIONREAD, &pending), 0);
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
        ::close(host);
        EXPECT_EQ(pending, 0);
        // The host decodes a read after it leaves the terminal's queue.
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
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

    /// @returns the controller's state once the adapter has received n commands.
    State after(const std::size_t n) {
        this->received(n);
        std::lock_guard lock(this->mu);
        return this->state;
    }
};

std::string profile_name(const ::testing::TestParamInfo<Profile> &info) {
    switch (info.param) {
        case Profile::LAWICEL:
            return "LAWICEL";
        case Profile::CANABLE:
            return "CANable";
        case Profile::CANABLE_2_5:
            return "CANable25";
        case Profile::MUTE:
            return "Mute";
    }
    return "";
}

Frame data_frame(const std::uint32_t id, const std::vector<std::uint8_t> &data) {
    Frame f;
    f.id = id;
    f.length = static_cast<std::uint8_t>(data.size());
    std::ranges::copy(data, f.data.begin());
    return f;
}
}

class SlcanTest : public ::testing::Test {
protected:
    std::unique_ptr<Adapter> adapter;
    Backend backend;

    void play(const Profile profile) {
        this->adapter = std::make_unique<Adapter>(profile);
    }

    [[nodiscard]] synnax::can::Properties props(
        const bool fd = false,
        const bool listen_only = false,
        const std::uint32_t bitrate = 500000
    ) const {
        synnax::can::Properties p;
        p.backend = synnax::can::BACKEND_SLCAN;
        p.channel = this->adapter->terminal;
        p.bitrate = bitrate;
        p.fd = fd;
        p.data_bitrate = 2000000;
        p.listen_only = listen_only;
        return p;
    }

    std::unique_ptr<can::Bus> open(const synnax::can::Properties &p) {
        auto [bus, err] = this->backend.open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }

    std::unique_ptr<can::Bus> open() { return this->open(this->props()); }
};

class SlcanFirmwareTest : public SlcanTest,
                          public ::testing::WithParamInterface<Profile> {
protected:
    void SetUp() override { this->play(this->GetParam()); }
};

INSTANTIATE_TEST_SUITE_P(
    Firmware,
    SlcanFirmwareTest,
    ::testing::Values(Profile::LAWICEL, Profile::CANABLE, Profile::CANABLE_2_5),
    profile_name
);

TEST_P(SlcanFirmwareTest, OpensTheAdapterAtTheBitrate) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    const auto state = this->adapter->after(4);
    EXPECT_TRUE(state.open);
    EXPECT_FALSE(state.silent);
    EXPECT_EQ(state.bitrate, "S6");
}

TEST_P(SlcanFirmwareTest, SendsFrames) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    ASSERT_NIL(bus->send(data_frame(0x123, {0x11, 0x22})));
    EXPECT_EQ(this->adapter->after(5).sent, std::vector<std::string>({"t12321122"}));
}

TEST_P(SlcanFirmwareTest, ReceivesFramesSplitAcrossReads) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    this->adapter->received(4);
    const auto before = x::telem::TimeStamp::now();
    this->adapter->write("z\rt1232AB");
    this->adapter->write("CD\rb0109000102030405060708090A0B\r");
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

TEST_P(SlcanFirmwareTest, ClosesTheAdapter) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    this->adapter->received(4);
    ASSERT_NIL(bus->close());
    ASSERT_NIL(bus->close());
    EXPECT_FALSE(this->adapter->after(5).open);
}

TEST_P(SlcanFirmwareTest, WaitsForAnUnpluggedAdapter) {
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    this->adapter->received(4);
    this->adapter->unplug();
    Frame frame;
    const auto [received, err] = bus->receive(frame, x::telem::SECOND);
    EXPECT_FALSE(received);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    ASSERT_MATCHES(bus->send(data_frame(0x1, {})), UNREACHABLE_ERROR);
}

TEST_F(SlcanTest, OpensCANableFirmwareWithoutWaitingForAnswers) {
    this->play(Profile::CANABLE);
    const auto start = std::chrono::steady_clock::now();
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    EXPECT_LT(std::chrono::steady_clock::now() - start, REPLY_TIMEOUT.chrono());
    EXPECT_EQ(
        this->adapter->received(4),
        std::vector<std::string>({"C", "V", "S6", "O"})
    );
}

TEST_F(SlcanTest, RefusesListenOnlyOnCANableFirmware) {
    this->play(Profile::CANABLE);
    const auto [bus, err] = this->backend.open(this->props(false, true));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "CANable firmware cannot open the bus listen only. Flash the CANable 2.5 or "
        "candleLight firmware to listen only"
    );
    EXPECT_EQ(bus, nullptr);
    EXPECT_EQ(this->adapter->received(3), std::vector<std::string>({"C", "V", "C"}));
}

TEST_F(SlcanTest, RunsCANableFirmwareAt750kWithS7) {
    this->play(Profile::CANABLE);
    auto bus = this->open(this->props(false, false, 750000));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(this->adapter->after(4).bitrate, "S7");
}

TEST_F(SlcanTest, RefusesABitrateThatCANableFirmwareLacks) {
    this->play(Profile::CANABLE);
    const auto [bus, err] = this->backend.open(this->props(false, false, 800000));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "CANable firmware cannot run a classic CAN bus at 800000 bit/s. Supported "
        "bitrates: 10000, 20000, 50000, 100000, 125000, 250000, 500000, 750000, "
        "1000000, 83333"
    );
    EXPECT_EQ(this->adapter->received(3), std::vector<std::string>({"C", "V", "C"}));
}

TEST_F(SlcanTest, SetsTheCANFDDataBitrate) {
    this->play(Profile::CANABLE);
    auto bus = this->open(this->props(true));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(
        this->adapter->received(5),
        std::vector<std::string>({"C", "V", "S6", "Y2", "O"})
    );
    EXPECT_EQ(this->adapter->after(5).data_bitrate, "Y2");
}

TEST_F(SlcanTest, OpensLAWICELFirmwareListenOnlyWithL) {
    this->play(Profile::LAWICEL);
    auto bus = this->open(this->props(false, true));
    ASSERT_NE(bus, nullptr);
    const auto state = this->adapter->after(4);
    EXPECT_TRUE(state.open);
    EXPECT_TRUE(state.silent);
    const auto err = bus->send(data_frame(0x123, {1}));
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel " + this->adapter->terminal + " is listen only");
}

TEST_F(SlcanTest, RunsLAWICELFirmwareAt800kWithS7) {
    this->play(Profile::LAWICEL);
    auto bus = this->open(this->props(false, false, 800000));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(this->adapter->after(4).bitrate, "S7");
}

TEST_F(SlcanTest, RefusesABitrateThatLAWICELFirmwareLacks) {
    this->play(Profile::LAWICEL);
    const auto [bus, err] = this->backend.open(this->props(false, false, 750000));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "LAWICEL slcan firmware cannot run a classic CAN bus at 750000 bit/s. "
        "Supported bitrates: 10000, 20000, 50000, 100000, 125000, 250000, 500000, "
        "800000, 1000000"
    );
    EXPECT_TRUE(this->adapter->after(3).bitrate.empty());
}

TEST_F(SlcanTest, ReportsARejectedCommand) {
    this->play(Profile::LAWICEL);
    const auto [bus, err] = this->backend.open(this->props(true));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "the slcan adapter at " + this->adapter->terminal + " rejected the command 'Y2'"
    );
    EXPECT_EQ(bus, nullptr);
    EXPECT_FALSE(this->adapter->after(5).open);
}

TEST_F(SlcanTest, OpensCANable25FirmwareListenOnlyWithM1) {
    this->play(Profile::CANABLE_2_5);
    auto bus = this->open(this->props(false, true));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(
        this->adapter->received(5),
        std::vector<std::string>({"C", "V", "S6", "M1", "O"})
    );
    const auto state = this->adapter->after(5);
    EXPECT_TRUE(state.open);
    EXPECT_TRUE(state.silent);
}

TEST_F(SlcanTest, RunsCANable25FirmwareAt800kWithS7) {
    this->play(Profile::CANABLE_2_5);
    auto bus = this->open(this->props(false, false, 800000));
    ASSERT_NE(bus, nullptr);
    EXPECT_EQ(this->adapter->after(4).bitrate, "S7");
}

TEST_F(SlcanTest, WaitsForADeviceThatDoesNotAnswer) {
    this->play(Profile::MUTE);
    const auto [bus, err] = this->backend.open(this->props());
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "the device at " + this->adapter->terminal +
            " did not answer the slcan command 'V'. Check that it is an slcan adapter"
    );
}

TEST_F(SlcanTest, WaitsForAMissingPort) {
    synnax::can::Properties p;
    p.backend = synnax::can::BACKEND_SLCAN;
    p.channel = "/dev/synnax-no-such-port";
    p.bitrate = 500000;
    const auto [bus, err] = this->backend.open(p);
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data.starts_with("failed to open serial port " + p.channel));
}

TEST_F(SlcanTest, RejectsAnUnsupportedBitrateBeforeConfiguringTheAdapter) {
    this->play(Profile::LAWICEL);
    const auto [bus, err] = this->backend.open(this->props(false, false, 333));
    ASSERT_MATCHES(err, CONFIG_ERROR);
    EXPECT_EQ(this->adapter->received(3), std::vector<std::string>({"C", "V", "C"}));
}

TEST_F(SlcanTest, DropsTheOldestFramesWhenTheInboxIsFull) {
    this->play(Profile::CANABLE);
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    this->adapter->received(4);
    std::string lines;
    for (std::size_t id = 0; id <= MAX_QUEUED; id++)
        lines += std::format("t{:03X}0\r", id);
    this->adapter->write(lines);
    this->adapter->drained();
    Frame frame;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
    EXPECT_EQ(frame.id, 1);
    std::size_t count = 1;
    while (ASSERT_NIL_P(bus->receive(frame, x::telem::TimeSpan::ZERO())))
        count++;
    EXPECT_EQ(count, MAX_QUEUED);
    EXPECT_EQ(frame.id, MAX_QUEUED);
}

TEST_F(SlcanTest, ReturnsFalseWhenNoFrameArrives) {
    this->play(Profile::LAWICEL);
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    Frame frame;
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(frame, 20 * x::telem::MILLISECOND)));
}

TEST_F(SlcanTest, RefusesAnFDFrameOnAClassicBus) {
    this->play(Profile::LAWICEL);
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    auto frame = data_frame(0x123, {1});
    frame.fd = true;
    const auto err = bus->send(frame);
    ASSERT_MATCHES(err, FRAME_ERROR);
    EXPECT_EQ(err.data, "a CAN FD frame cannot be sent on a classic CAN bus");
}

TEST_F(SlcanTest, SendsAndReceivesAtTheSameTime) {
    this->play(Profile::LAWICEL);
    auto bus = this->open();
    ASSERT_NE(bus, nullptr);
    std::thread receiver([&] {
        Frame frame;
        EXPECT_TRUE(ASSERT_NIL_P(bus->receive(frame, x::telem::SECOND)));
        EXPECT_EQ(frame.id, 0x42);
    });
    for (std::uint32_t i = 0; i < 10; i++)
        ASSERT_NIL(bus->send(data_frame(i, {})));
    this->adapter->write("t0420\r");
    receiver.join();
    EXPECT_EQ(this->adapter->after(14).sent.size(), 10);
}

TEST_F(SlcanTest, ListsNoChannelsWhileASerialPortIsPresent) {
    EXPECT_TRUE(ASSERT_NIL_P(this->backend.scan()).empty());
}
}
