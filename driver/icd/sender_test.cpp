// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <map>
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/errors/errors.h"
#include "driver/icd/sender.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::icd {
namespace {
using Writes = std::vector<std::string>;

/// @brief Wire is a transport that keeps the bytes of each write.
class Wire final : public transport::Transport {
public:
    Writes writes;
    /// @brief attempts is the number of writes, including failed ones.
    std::size_t attempts = 0;
    /// @brief err is the error of each write, which then keeps no bytes.
    x::errors::Error err;

    std::pair<transport::Chunk, x::errors::Error>
    read(x::telem::TimeSpan /*timeout*/) override {
        return {{}, x::errors::NIL};
    }

    x::errors::Error write(
        const std::span<const std::uint8_t> data,
        x::telem::TimeSpan /*timeout*/
    ) override {
        this->attempts++;
        if (this->err) return this->err;
        this->writes.emplace_back(data.begin(), data.end());
        return x::errors::NIL;
    }
};

/// @brief Steady is a steady clock that a spec moves by hand.
struct Steady {
    std::chrono::steady_clock::time_point time;

    [[nodiscard]] transport::Clock clock() {
        return [this] { return this->time; };
    }
};

/// @brief Config holds an ICD with one message, the target that sends it, and the
/// channels of its fields.
struct Config {
    synnax::icd::ICD ic;
    synnax::icd::WriteTarget target;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    explicit Config(
        const synnax::icd::Payload &payload = synnax::icd::TextPayload{},
        std::optional<synnax::icd::Framing> framing = synnax::icd::DelimitedFraming{}
    ) {
        this->ic.key = x::uuid::create();
        this->ic.name = "valves";
        this->ic.framing = std::move(framing);
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "cmd";
        m.payload = payload;
        this->ic.messages = {m};
        this->target.icd = this->ic.key;
        this->target.message.message = m.key;
    }

    /// @brief adds f to the message and binds it to a float64 channel with key.
    template<typename Payload, typename Field>
    void bind(Field f, const synnax::channel::Key key) {
        f.key = x::uuid::create();
        std::get<Payload>(this->ic.messages[0].payload).fields.push_back(f);
        synnax::icd::ChannelWriteBinding b;
        b.field = f.key;
        b.channel = key;
        this->target.message.bindings.emplace_back(b);
        synnax::channel::Channel ch;
        ch.key = key;
        ch.data_type = x::telem::FLOAT64_T;
        this->channels.insert_or_assign(key, ch);
    }

    /// @brief adds a text field at position with default to the message, and binds it
    /// to channel key.
    void bind_text(
        const std::uint32_t position,
        const synnax::channel::Key key,
        const std::optional<double> default_ = std::nullopt
    ) {
        synnax::icd::DelimitedTextField f;
        f.name = std::string(1, static_cast<char>('a' + position));
        f.position = position;
        if (default_.has_value()) f.default_ = *default_;
        this->bind<synnax::icd::TextPayload>(f, key);
    }

    [[nodiscard]] std::pair<Sender, x::errors::Error> create(
        const bool datagram = false,
        transport::Clock now = std::chrono::steady_clock::now
    ) const {
        return Sender::create(
            {this->target},
            {this->ic},
            this->channels,
            datagram,
            std::move(now)
        );
    }
};

/// @brief returns a config whose text field a at position 0 takes channel 2, and whose
/// field b at position 1 with default 5 takes channel 3.
Config setpoints() {
    Config cfg;
    cfg.bind_text(0, 2, 0);
    cfg.bind_text(1, 3, 5);
    return cfg;
}

/// @brief returns a config whose text field a at position 0 with default 0 takes
/// channel 2, and whose field b at position 1 with default 5 takes channel 3. It sends
/// every 100 ms.
Config periodic() {
    auto cfg = setpoints();
    cfg.target.period = 100 * x::telem::MILLISECOND;
    return cfg;
}

/// @brief returns a frame that holds s for channel key.
x::telem::Frame frame(const synnax::channel::Key key, x::telem::Series s) {
    return {key, std::move(s)};
}

/// @brief expects creating a sender of cfg on a stream to fail with text.
void assert_refuses(const Config &cfg, const std::string &text) {
    const auto err = cfg.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, text);
}
}

/// @brief it should send one message for each command on a target with no period.
TEST(Sender, SendsMessageForEachCommand) {
    auto s = ASSERT_NIL_P(setpoints().create());
    Wire t;
    ASSERT_TRUE(
        ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(std::vector<double>{1, 2}))))
            .empty()
    );
    ASSERT_EQ(t.writes, (Writes{"1,5\n", "2,5\n"}));
}

/// @brief it should send one message for a frame with commands on several channels.
TEST(Sender, SendsOneMessageForCommandsOnSeveralChannels) {
    auto s = ASSERT_NIL_P(setpoints().create());
    Wire t;
    x::telem::Frame fr(2);
    fr.emplace(2, x::telem::Series(1.0));
    fr.emplace(3, x::telem::Series(2.0));
    ASSERT_NIL_P(s.write(t, fr));
    ASSERT_EQ(t.writes, (Writes{"1,2\n"}));
}

/// @brief it should send nothing on send for a target with no period, and wait for no
/// target.
TEST(Sender, WaitsForeverWithNoPeriod) {
    auto s = ASSERT_NIL_P(setpoints().create());
    Wire t;
    ASSERT_EQ(ASSERT_NIL_P(s.send(t)), x::telem::TimeSpan::max());
    ASSERT_TRUE(t.writes.empty());
}

/// @brief it should send the last command of each field once per period, and the
/// default of a field with no command yet.
TEST(Sender, SendsLastCommandsEachPeriod) {
    Steady clock;
    auto s = ASSERT_NIL_P(periodic().create(false, clock.clock()));
    Wire t;
    ASSERT_EQ(ASSERT_NIL_P(s.send(t)), 100 * x::telem::MILLISECOND);
    ASSERT_EQ(t.writes, (Writes{"0,5\n"}));
    ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(std::vector<double>{1, 2}))));
    ASSERT_EQ(t.writes.size(), 1U);
    clock.time += std::chrono::milliseconds(60);
    ASSERT_EQ(ASSERT_NIL_P(s.send(t)), 40 * x::telem::MILLISECOND);
    ASSERT_EQ(t.writes.size(), 1U);
    clock.time += std::chrono::milliseconds(40);
    ASSERT_NIL_P(s.send(t));
    ASSERT_EQ(t.writes, (Writes{"0,5\n", "2,5\n"}));
}

/// @brief it should send one message after a stall, not one for each missed period.
TEST(Sender, SkipsMissedPeriods) {
    Steady clock;
    auto s = ASSERT_NIL_P(periodic().create(false, clock.clock()));
    Wire t;
    ASSERT_NIL_P(s.send(t));
    clock.time += std::chrono::milliseconds(350);
    ASSERT_NIL_P(s.send(t));
    ASSERT_EQ(ASSERT_NIL_P(s.send(t)), 100 * x::telem::MILLISECOND);
    ASSERT_EQ(t.writes.size(), 2U);
}

/// @brief it should send nothing on a period until a field with no default has its
/// first command.
TEST(Sender, WaitsForFirstCommandOfFieldWithNoDefault) {
    Steady clock;
    Config cfg;
    cfg.bind_text(0, 2);
    cfg.target.period = 100 * x::telem::MILLISECOND;
    auto s = ASSERT_NIL_P(cfg.create(false, clock.clock()));
    Wire t;
    ASSERT_NIL_P(s.send(t));
    ASSERT_TRUE(t.writes.empty());
    ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(3.0))));
    ASSERT_TRUE(t.writes.empty());
    clock.time += std::chrono::milliseconds(100);
    ASSERT_NIL_P(s.send(t));
    ASSERT_EQ(t.writes, (Writes{"3\n"}));
}

/// @brief it should drop a command that its field refuses with a warning, and send the
/// last good command of the field.
TEST(Sender, KeepsLastGoodCommandWhenFieldRefuses) {
    Steady clock;
    Config cfg;
    synnax::icd::DelimitedTextField a;
    a.name = "a";
    a.min = 0;
    a.max = 10;
    cfg.bind<synnax::icd::TextPayload>(a, 2);
    cfg.target.period = 100 * x::telem::MILLISECOND;
    auto s = ASSERT_NIL_P(cfg.create(false, clock.clock()));
    Wire t;
    ASSERT_TRUE(ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(4.0)))).empty());
    const auto warning = ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(11.0))));
    ASSERT_EQ(warning.message, "Skipped invalid command");
    ASSERT_EQ(warning.description, "message cmd: field a: 11 is above the max 10");
    ASSERT_NIL_P(s.send(t));
    ASSERT_EQ(t.writes, (Writes{"4\n"}));
}

/// @brief it should send each message in the framing of its ICD.
TEST(Sender, FramesMessage) {
    Config cfg{synnax::icd::JSONPayload{}, synnax::icd::CobsFraming{}};
    synnax::icd::FloatJSONField a;
    a.name = "a";
    a.pointer = "/a";
    cfg.bind<synnax::icd::JSONPayload>(a, 2);
    auto s = ASSERT_NIL_P(cfg.create());
    Wire t;
    ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(1.0))));
    ASSERT_EQ(t.writes, (Writes{std::string("\x08{\"a\":1}") + '\0'}));
}

/// @brief it should return the error of a framing that cannot hold a message.
TEST(Sender, ReturnsFramingError) {
    Config cfg{synnax::icd::TextPayload{}, synnax::icd::FixedFraming{.length = 2}};
    cfg.bind_text(0, 2);
    auto s = ASSERT_NIL_P(cfg.create());
    Wire t;
    const auto err = s.write(t, frame(2, x::telem::Series(123.0))).second;
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(err.data, "the frame has 3 bytes, not the fixed length of 2");
    ASSERT_TRUE(t.writes.empty());
}

/// @brief it should send a datagram with no framing as the bare message.
TEST(Sender, SendsDatagramWithNoFraming) {
    Config cfg{synnax::icd::TextPayload{}, std::nullopt};
    cfg.bind_text(0, 2);
    auto s = ASSERT_NIL_P(cfg.create(true));
    Wire t;
    ASSERT_NIL_P(s.write(t, frame(2, x::telem::Series(1.0))));
    ASSERT_EQ(t.writes, (Writes{"1"}));
}

/// @brief it should return the error of the transport and send nothing more in the
/// write, but keep the commands of each target.
TEST(Sender, ReturnsTransportErrorOfCommand) {
    const auto a = setpoints();
    const auto b = setpoints();
    auto s = ASSERT_NIL_P(
        Sender::create({a.target, b.target}, {a.ic, b.ic}, a.channels, false)
    );
    Wire t;
    t.err = x::errors::Error(transport::UNREACHABLE_ERROR, "the port closed");
    ASSERT_OCCURRED_AS_P(
        s.write(t, frame(2, x::telem::Series(1.0))),
        transport::UNREACHABLE_ERROR
    );
    ASSERT_EQ(t.attempts, 1U);
    t.err = x::errors::NIL;
    ASSERT_NIL_P(s.write(t, frame(3, x::telem::Series(2.0))));
    ASSERT_EQ(t.writes, (Writes{"1,2\n", "1,2\n"}));
}

/// @brief it should return the error of the transport on a period, and send the message
/// at the next send.
TEST(Sender, KeepsTargetDueAfterTransportError) {
    Steady clock;
    auto s = ASSERT_NIL_P(periodic().create(false, clock.clock()));
    Wire t;
    t.err = x::errors::Error(transport::UNREACHABLE_ERROR, "the port closed");
    ASSERT_OCCURRED_AS_P(s.send(t), transport::UNREACHABLE_ERROR);
    t.err = x::errors::NIL;
    ASSERT_NIL_P(s.send(t));
    ASSERT_EQ(t.writes, (Writes{"0,5\n"}));
}

/// @brief it should refuse a target whose ICD does not exist.
TEST(Sender, RefusesMissingICD) {
    auto cfg = setpoints();
    const auto key = x::uuid::create();
    cfg.target.icd = key;
    assert_refuses(cfg, "target 0: ICD " + key.to_string() + " does not exist");
}

/// @brief it should refuse an ICD with no framing on a stream.
TEST(Sender, RefusesStreamICDWithNoFraming) {
    auto cfg = setpoints();
    cfg.ic.framing = std::nullopt;
    assert_refuses(cfg, "target 0: ICD valves has no framing to split the stream with");
}

/// @brief it should name the target of a message that the Writer refuses.
TEST(Sender, NamesTargetOfRefusedMessage) {
    const Config cfg;
    assert_refuses(cfg, "target 0: message cmd: binds no channels");
}
}
