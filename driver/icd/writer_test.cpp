// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstdint>
#include <limits>
#include <map>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/errors/errors.h"
#include "driver/icd/writer.h"

namespace driver::icd {
namespace {
/// @brief returns a delimited text field named name at position.
synnax::icd::DelimitedTextField
at(const std::string &name, const std::uint32_t position) {
    synnax::icd::DelimitedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = position;
    return f;
}

/// @brief Config holds what Writer::create takes: the write, the ICD that holds its
/// message, and its channels.
struct Config {
    synnax::icd::WriteMessage wm;
    synnax::icd::ICD ic;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    /// @brief adds f to the message.
    void add(const synnax::icd::TextField &f) {
        std::get<synnax::icd::TextPayload>(this->ic.messages[0].payload)
            .fields.push_back(f);
    }

    /// @brief adds f to the message.
    void add(const synnax::icd::JSONField &f) {
        std::get<synnax::icd::JSONPayload>(this->ic.messages[0].payload)
            .fields.push_back(f);
    }

    /// @brief adds f to the message and binds it to channel key of data type dt.
    template<typename Field>
    void bind(
        const Field &f,
        const synnax::channel::Key key,
        const x::telem::DataType &dt = x::telem::FLOAT64_T
    ) {
        this->add(f);
        synnax::icd::ChannelWriteBinding b;
        b.field = f.key;
        b.channel = key;
        this->wm.bindings.emplace_back(b);
        synnax::channel::Channel ch;
        ch.key = key;
        ch.data_type = dt;
        this->channels.insert_or_assign(key, ch);
    }

    /// @brief binds the field with key to value.
    void fix(const synnax::icd::FieldKey &key, const synnax::icd::WriteValue &value) {
        synnax::icd::ValueWriteBinding b;
        b.field = key;
        b.value = value;
        this->wm.bindings.emplace_back(b);
    }

    [[nodiscard]] std::pair<Writer, x::errors::Error> create() const {
        return Writer::create(this->wm, this->ic, this->channels);
    }
};

/// @brief returns a config whose text message cmd has no fields.
Config message() {
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "cmd";
    m.payload = synnax::icd::TextPayload{};
    Config c{.wm = {.message = m.key}};
    c.ic.name = "valves";
    c.ic.messages = {m};
    return c;
}

/// @brief returns a config whose JSON message cmd has no fields.
Config json_message() {
    auto c = message();
    c.ic.messages[0].payload = synnax::icd::JSONPayload{};
    return c;
}

/// @brief returns a JSON field of type Field named name, at the pointer /name.
template<typename Field>
Field field(const std::string &name) {
    Field f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

/// @brief returns a config whose field a at position 0 takes channel 2, and whose field
/// b at position 1 takes channel 3.
Config setpoints() {
    auto c = message();
    c.bind(at("a", 0), 2);
    c.bind(at("b", 1), 3);
    return c;
}

/// @brief returns a frame that holds s for channel key.
x::telem::Frame frame(const synnax::channel::Key key, x::telem::Series s) {
    return {key, std::move(s)};
}

/// @brief returns a frame that holds s for channel key and t for channel other.
x::telem::Frame frame(
    const synnax::channel::Key key,
    x::telem::Series s,
    const synnax::channel::Key other,
    x::telem::Series t
) {
    x::telem::Frame fr(2);
    fr.emplace(key, std::move(s));
    fr.emplace(other, std::move(t));
    return fr;
}

/// @brief returns the payloads that w sends for fr, and expects no error.
std::vector<std::string> sent(Writer &w, const x::telem::Frame &fr) {
    std::vector<std::string> out;
    const auto [error, refused] = w.write(fr, [&](std::span<const std::uint8_t> p) {
        out.emplace_back(p.begin(), p.end());
        return x::errors::NIL;
    });
    EXPECT_NIL(error);
    EXPECT_NIL(refused);
    return out;
}

/// @brief expects w to send payloads for fr and to refuse a command with text.
void assert_refuses_command(
    Writer &w,
    const x::telem::Frame &fr,
    const std::vector<std::string> &payloads,
    const std::string &text
) {
    std::vector<std::string> out;
    const auto [error, refused] = w.write(fr, [&](std::span<const std::uint8_t> p) {
        out.emplace_back(p.begin(), p.end());
        return x::errors::NIL;
    });
    EXPECT_NIL(error);
    ASSERT_OCCURRED_AS(refused, x::errors::VALIDATION);
    EXPECT_EQ(refused.data, text);
    EXPECT_EQ(out, payloads);
}

/// @brief expects creating a writer for c to fail with text.
void assert_refuses(const Config &c, const std::string &text) {
    const auto err = c.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, text);
}
}

/// @brief it should send one payload for each row of commands, with the last command of
/// a channel that has fewer.
TEST(Writer, WriteRows) {
    auto w = ASSERT_NIL_P(setpoints().create());
    EXPECT_EQ(
        sent(
            w,
            frame(
                2,
                x::telem::Series(std::vector<double>{1, 2, 3}),
                3,
                x::telem::Series(10.0)
            )
        ),
        (std::vector<std::string>{"1,10", "2,10", "3,10"})
    );
    EXPECT_EQ(
        sent(w, frame(3, x::telem::Series(20.0))),
        (std::vector<std::string>{"3,20"})
    );
}

/// @brief it should send nothing until each field with no default has a command.
TEST(Writer, WaitForCommands) {
    auto w = ASSERT_NIL_P(setpoints().create());
    EXPECT_TRUE(sent(w, frame(2, x::telem::Series(1.0))).empty());
    EXPECT_EQ(
        sent(w, frame(3, x::telem::Series(2.0))),
        (std::vector<std::string>{"1,2"})
    );
}

/// @brief it should send the default of a field with no command yet, and the value of a
/// value binding.
TEST(Writer, SendDefaults) {
    auto c = message();
    c.bind(at("a", 0), 2);
    auto b = at("b", 1);
    b.default_ = 5;
    c.add(b);
    const auto d = at("d", 2);
    c.add(d);
    c.fix(d.key, synnax::icd::FloatWriteValue{.value = 7});
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(1.0))),
        (std::vector<std::string>{"1,5,7"})
    );
}

/// @brief it should take the commands of a channel from each of its series in a frame,
/// in frame order.
TEST(Writer, JoinSeriesOfOneChannel) {
    auto w = ASSERT_NIL_P(setpoints().create());
    x::telem::Frame fr(3);
    fr.emplace(2, x::telem::Series(1.0));
    fr.emplace(3, x::telem::Series(10.0));
    fr.emplace(2, x::telem::Series(2.0));
    EXPECT_EQ(sent(w, fr), (std::vector<std::string>{"1,10", "2,10"}));
}

/// @brief it should ignore the channels of other messages.
TEST(Writer, IgnoreOtherChannels) {
    auto w = ASSERT_NIL_P(setpoints().create());
    EXPECT_TRUE(sent(w, frame(9, x::telem::Series(1.0))).empty());
}

/// @brief it should drop a command outside the limits of its field, or one that its
/// field cannot hold, and keep the last good command of the field.
TEST(Writer, DropRefusedCommands) {
    auto c = message();
    auto a = at("a", 0);
    a.min = 0;
    a.max = 10;
    c.bind(a, 2);
    c.bind(at("b", 1), 3);
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(5.0), 3, x::telem::Series(1.0))),
        (std::vector<std::string>{"5,1"})
    );
    assert_refuses_command(
        w,
        frame(2, x::telem::Series(11.0)),
        {},
        "message cmd: field a: 11 is above the max 10"
    );
    assert_refuses_command(
        w,
        frame(2, x::telem::Series(-1.0), 3, x::telem::Series(2.0)),
        {"5,2"},
        "message cmd: field a: -1 is below the min 0"
    );
    assert_refuses_command(
        w,
        frame(2, x::telem::Series(std::nan(""))),
        {},
        "message cmd: field a: NaN is not finite"
    );
    assert_refuses_command(
        w,
        frame(3, x::telem::Series(std::numeric_limits<double>::infinity())),
        {},
        "message cmd: field b: Infinity is not finite"
    );
    assert_refuses_command(
        w,
        frame(
            2,
            x::telem::Series(12.0),
            3,
            x::telem::Series(std::numeric_limits<double>::infinity())
        ),
        {},
        "message cmd: field a: 12 is above the max 10"
    );
    EXPECT_EQ(
        sent(w, frame(3, x::telem::Series(3.0))),
        (std::vector<std::string>{"5,3"})
    );
}

/// @brief it should send nothing after the first error of send, and keep the last good
/// command of each field for the next write.
TEST(Writer, StopOnSendError) {
    auto w = ASSERT_NIL_P(setpoints().create());
    int calls = 0;
    const auto [error, refused] = w.write(
        frame(2, x::telem::Series(std::vector<double>{1, 2}), 3, x::telem::Series(1.0)),
        [&](std::span<const std::uint8_t>) {
            calls++;
            return x::errors::Error(x::errors::VALIDATION, "closed");
        }
    );
    ASSERT_OCCURRED_AS(error, x::errors::VALIDATION);
    EXPECT_EQ(error.data, "closed");
    EXPECT_NIL(refused);
    EXPECT_EQ(calls, 1);
    EXPECT_EQ(
        sent(w, frame(3, x::telem::Series(5.0))),
        (std::vector<std::string>{"2,5"})
    );
}

/// @brief it should refuse a value that decodes outside the limits of its field.
TEST(Writer, RefuseValueOutsideLimits) {
    auto c = message();
    c.bind(at("a", 0), 2);
    auto scaled = at("s", 1);
    scaled.scale = 5;
    scaled.max = 10;
    c.add(scaled);
    c.fix(scaled.key, synnax::icd::FloatWriteValue{.value = 3});
    assert_refuses(
        c,
        "message cmd: field s: 3 decodes to 15, which is above the max 10"
    );
    auto fixed = message();
    fixed.bind(at("a", 0), 2);
    auto low = at("l", 1);
    low.min = 0;
    fixed.add(low);
    fixed.fix(low.key, synnax::icd::FloatWriteValue{.value = -2});
    assert_refuses(fixed, "message cmd: field l: -2 is below the min 0");
}

/// @brief it should accept a value that decodes to a limit of its field after float64
/// rounding.
TEST(Writer, AcceptValueThatRoundsToLimit) {
    auto c = message();
    c.bind(at("a", 0), 2);
    auto tenths = at("t", 1);
    tenths.scale = 0.1;
    tenths.max = 0.3;
    c.add(tenths);
    c.fix(tenths.key, synnax::icd::FloatWriteValue{.value = 3});
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(1.0))),
        (std::vector<std::string>{"1,3"})
    );
}

/// @brief it should refuse a write of a message that does not exist or is not text, and
/// one that binds no channels or a channel that does not exist.
TEST(Writer, RefuseInvalidMessage) {
    auto missing = setpoints();
    missing.wm.message = x::uuid::create();
    assert_refuses(
        missing,
        "message " + missing.wm.message.to_string() + " does not exist in ICD valves"
    );
    auto binary = setpoints();
    binary.ic.messages[0].payload = synnax::icd::BinaryPayload{};
    assert_refuses(
        binary,
        "message cmd: the Driver writes only text and JSON messages"
    );
    auto unbound = message();
    auto f = at("a", 0);
    unbound.add(f);
    unbound.fix(f.key, synnax::icd::FloatWriteValue{.value = 1});
    assert_refuses(unbound, "message cmd: binds no channels");
    auto lost = setpoints();
    lost.channels.erase(3);
    assert_refuses(lost, "message cmd: channel 3 does not exist");
}

/// @brief it should refuse a value binding to a field that does not exist, a second
/// binding of a field, a value not of its field's kind, and what the encoder refuses.
TEST(Writer, RefuseInvalidBindings) {
    auto missing = setpoints();
    const auto key = x::uuid::create();
    missing.fix(key, synnax::icd::FloatWriteValue{.value = 1});
    assert_refuses(
        missing,
        "message cmd: field " + key.to_string() + " does not exist"
    );
    auto twice = setpoints();
    const auto a = synnax::icd::base(
        std::get<synnax::icd::TextPayload>(twice.ic.messages[0].payload).fields[0]
    );
    twice.fix(a.key, synnax::icd::FloatWriteValue{.value = 1});
    assert_refuses(twice, "message cmd: field a has more than one binding");
    auto kind = message();
    kind.bind(at("a", 0), 2);
    const auto b = at("b", 1);
    kind.add(b);
    kind.fix(b.key, synnax::icd::IntegerWriteValue{.value = 1});
    assert_refuses(
        kind,
        "message cmd: field b of kind float needs a value of its kind"
    );
    auto open = setpoints();
    open.add(at("c", 2));
    const auto err = open.create().second;
    ASSERT_OCCURRED_AS(err, codec::CONFIG_ERROR);
    EXPECT_EQ(err.data, "message cmd: field c: needs a channel or a default");
}

/// @brief it should send one JSON document for each row of commands, with the last
/// command of a channel that has fewer, a string command included.
TEST(Writer, WriteJSONDocuments) {
    auto c = json_message();
    c.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    c.bind(field<synnax::icd::StringJSONField>("s"), 3, x::telem::STRING_T);
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(
            w,
            frame(
                2,
                x::telem::Series(std::vector<double>{1, 2}),
                3,
                x::telem::Series(std::string("on"))
            )
        ),
        (std::vector<std::string>{R"({"a":1,"s":"on"})", R"({"a":2,"s":"on"})"})
    );
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(3.0))),
        (std::vector<std::string>{R"({"a":3,"s":"on"})"})
    );
}

/// @brief it should send the value of a value binding of each kind as its field's
/// default.
TEST(Writer, SendJSONValues) {
    auto c = json_message();
    c.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    const auto f = field<synnax::icd::FloatJSONField>("f");
    const auto i = field<synnax::icd::IntegerJSONField>("i");
    const auto u = field<synnax::icd::UnsignedJSONField>("u");
    const auto s = field<synnax::icd::StringJSONField>("s");
    const auto b = field<synnax::icd::BooleanJSONField>("b");
    auto t = field<synnax::icd::TimestampJSONField>("t");
    t.time_format = "iso8601";
    const auto j = field<synnax::icd::JSONJSONField>("j");
    const auto id = field<synnax::icd::UuidJSONField>("id");
    const auto by = field<synnax::icd::BytesJSONField>("by");
    for (const synnax::icd::JSONField &x: {
             synnax::icd::JSONField(f),
             synnax::icd::JSONField(i),
             synnax::icd::JSONField(u),
             synnax::icd::JSONField(s),
             synnax::icd::JSONField(b),
             synnax::icd::JSONField(t),
             synnax::icd::JSONField(j),
             synnax::icd::JSONField(id),
             synnax::icd::JSONField(by),
         })
        c.add(x);
    const auto key = ASSERT_NIL_P(
        x::uuid::UUID::parse("2f1e8c0a-6b3d-4e7f-9a5c-1d2e3f405162")
    );
    c.fix(f.key, synnax::icd::FloatWriteValue{.value = 1.5});
    c.fix(i.key, synnax::icd::IntegerWriteValue{.value = -2});
    c.fix(u.key, synnax::icd::UnsignedWriteValue{.value = 3});
    c.fix(s.key, synnax::icd::StringWriteValue{.value = "psi"});
    c.fix(b.key, synnax::icd::BooleanWriteValue{.value = true});
    c.fix(
        t.key,
        synnax::icd::TimestampWriteValue{.value = x::telem::TimeStamp(x::telem::SECOND)}
    );
    c.fix(j.key, synnax::icd::JSONWriteValue{.value = R"({"k":[1]})"});
    c.fix(id.key, synnax::icd::UuidWriteValue{.value = key});
    c.fix(by.key, synnax::icd::BytesWriteValue{.value = "AQI="});
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(1.0))),
        (std::vector<std::string>{
            R"({"a":1,"f":1.5,"i":-2,"u":3,"s":"psi","b":true,)"
            R"("t":"1970-01-01T00:00:01Z","j":{"k":[1]},)"
            R"("id":"2f1e8c0a-6b3d-4e7f-9a5c-1d2e3f405162","by":"AQI="})"
        })
    );
}

/// @brief it should refuse a value of another kind than its JSON field, and a value
/// that the encoder refuses as a default.
TEST(Writer, RefuseInvalidJSONValues) {
    auto kind = json_message();
    kind.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    const auto i = field<synnax::icd::IntegerJSONField>("i");
    kind.add(i);
    kind.fix(i.key, synnax::icd::FloatWriteValue{.value = 1});
    assert_refuses(
        kind,
        "message cmd: field i of kind integer needs a value of its kind"
    );
    auto text = json_message();
    text.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    const auto s = field<synnax::icd::StringJSONField>("s");
    text.add(s);
    text.fix(s.key, synnax::icd::JSONWriteValue{.value = "1"});
    assert_refuses(
        text,
        "message cmd: field s of kind string needs a value of its kind"
    );
    auto invalid = json_message();
    invalid.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    const auto j = field<synnax::icd::JSONJSONField>("j");
    invalid.add(j);
    invalid.fix(j.key, synnax::icd::JSONWriteValue{.value = "{"});
    const auto err = invalid.create().second;
    ASSERT_OCCURRED_AS(err, codec::CONFIG_ERROR);
    EXPECT_EQ(err.data, "message cmd: field j: default: must be valid JSON");
}

/// @brief it should drop a command that its JSON field cannot hold, and keep the last
/// good command of the field.
TEST(Writer, DropRefusedJSONCommands) {
    auto c = json_message();
    c.bind(field<synnax::icd::FloatJSONField>("a"), 2);
    auto w = ASSERT_NIL_P(c.create());
    EXPECT_EQ(
        sent(w, frame(2, x::telem::Series(1.0))),
        (std::vector<std::string>{R"({"a":1})"})
    );
    assert_refuses_command(
        w,
        frame(2, x::telem::Series(std::vector<double>{std::nan(""), 2})),
        {R"({"a":2})"},
        "message cmd: field a: NaN is not a JSON number"
    );
}
}
