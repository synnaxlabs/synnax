// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

#include "driver/codec/errors.h"
#include "driver/errors/errors.h"
#include "driver/http/message/message.h"

namespace driver::http::message {
namespace {
namespace icd = synnax::icd;

const x::telem::TimeStamp RECEIVED(1000);

synnax::channel::Channel
channel(const synnax::channel::Key key, const x::telem::DataType &dt) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.data_type = dt;
    ch.index = 1;
    ch.is_index = dt == x::telem::TIMESTAMP_T;
    return ch;
}

icd::FloatJSONField float_field(const std::string &name) {
    icd::FloatJSONField f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

icd::TimestampJSONField time_field() {
    icd::TimestampJSONField f;
    f.key = x::uuid::create();
    f.name = "t";
    f.pointer = "/t";
    f.time_format = "unix_ns";
    return f;
}

/// @brief returns a message named name whose field v writes to channel 2.
Bound reading(const std::string &name = "reading") {
    const auto v = float_field("v");
    icd::Message m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = icd::JSONPayload{.fields = {v}};
    return {
        .message = m,
        .bindings = {{.field = v.key, .channel = channel(2, x::telem::FLOAT64_T)}},
    };
}

/// @brief returns a message whose field t indexes field v.
Bound indexed() {
    auto b = reading();
    const auto t = time_field();
    std::get<icd::JSONPayload>(b.message.payload).fields.emplace_back(t);
    b.index = t.key;
    b.bindings.push_back(
        {.field = t.key, .channel = channel(1, x::telem::TIMESTAMP_T)}
    );
    return b;
}

/// @brief returns a message named type that reads the documents whose type is type.
Bound matching(const std::string &type) {
    auto b = reading(type);
    icd::StringJSONMatch match;
    match.pointer = "/type";
    match.value = type;
    std::get<icd::JSONPayload>(b.message.payload).match = match;
    return b;
}

/// @brief returns the error that creating a Reader for b gives.
x::errors::Error create_error(const Bound &b) {
    return Reader::create({b}).second;
}
}

/// @brief it should stamp the samples of a message with the time the response arrived.
TEST(Reader, StampsSamplesOnArrival) {
    auto r = ASSERT_NIL_P(Reader::create({reading()}));
    x::telem::Frame fr(2);
    ASSERT_NIL(r.read(R"({"v": 1.5})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<double>(2, 0), 1.5);
}

/// @brief it should stamp the samples of a message with the value of its index field.
TEST(Reader, StampsSamplesWithIndexField) {
    auto r = ASSERT_NIL_P(Reader::create({indexed()}));
    x::telem::Frame fr(2);
    ASSERT_NIL(r.read(R"({"v": 2, "t": 5})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), 5);
    ASSERT_EQ(fr.at<double>(2, 0), 2.0);
}

/// @brief it should write no index for virtual channels.
TEST(Reader, WritesNoIndexForVirtualChannels) {
    auto b = reading();
    b.bindings[0].channel.index = 0;
    auto r = ASSERT_NIL_P(Reader::create({b}));
    x::telem::Frame fr(1);
    ASSERT_NIL(r.read(R"({"v": 3})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{2}));
}

/// @brief it should read a response as the message whose match rule it meets, or else
/// as the message with no rule.
TEST(Reader, ReadsMatchedMessage) {
    auto status = matching("status");
    auto other = reading("other");
    other.bindings[0].channel.key = 3;
    other.bindings[0].channel.index = 4;
    auto r = ASSERT_NIL_P(Reader::create({status, other}));
    x::telem::Frame matched(2);
    ASSERT_NIL(r.read(R"({"type": "status", "v": 5})", RECEIVED, matched));
    ASSERT_EQ(*matched.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(matched.at<double>(2, 0), 5.0);
    x::telem::Frame fallback(2);
    ASSERT_NIL(r.read(R"({"type": "event", "v": 4})", RECEIVED, fallback));
    ASSERT_EQ(*fallback.channels, (std::vector<std::uint32_t>{4, 3}));
}

/// @brief it should add nothing to the frame when a value does not convert.
TEST(Reader, AddsNothingWhenValueDoesNotConvert) {
    auto r = ASSERT_NIL_P(Reader::create({reading()}));
    x::telem::Frame fr(2);
    const auto err = r.read(R"({"v": "x"})", RECEIVED, fr);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, R"(field v: unknown enum label "x")");
    ASSERT_TRUE(fr.empty());
}

/// @brief it should reject a response that meets no message.
TEST(Reader, RejectsResponseThatMeetsNoMessage) {
    auto r = ASSERT_NIL_P(Reader::create({matching("status")}));
    x::telem::Frame fr(2);
    const auto err = r.read(R"({"type": "event", "v": 1})", RECEIVED, fr);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "the response matches no message");
    ASSERT_TRUE(fr.empty());
}

/// @brief it should reject an endpoint that reads no messages.
TEST(Reader, RejectsNoMessages) {
    const auto err = Reader::create({}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "the endpoint reads no messages");
}

/// @brief it should reject a message that is not JSON.
TEST(Reader, RejectsMessageThatIsNotJSON) {
    auto b = reading();
    b.message.payload = icd::BinaryPayload{};
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading is not a JSON message");
}

/// @brief it should reject a message that binds no fields.
TEST(Reader, RejectsMessageThatBindsNoFields) {
    auto b = reading();
    b.bindings.clear();
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading binds no fields");
}

/// @brief it should reject channels that do not share one index channel.
TEST(Reader, RejectsChannelsWithDifferentIndexes) {
    auto b = reading();
    const auto w = float_field("w");
    std::get<icd::JSONPayload>(b.message.payload).fields.emplace_back(w);
    auto ch = channel(3, x::telem::FLOAT64_T);
    ch.index = 4;
    b.bindings.push_back({.field = w.key, .channel = ch});
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: every channel must share one index channel");
}

/// @brief it should reject an index field that does not write to an index channel.
TEST(Reader, RejectsIndexFieldOnDataChannel) {
    auto b = indexed();
    b.bindings[1].channel.is_index = false;
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: the index field must write to an index channel"
    );
}

/// @brief it should reject a field other than the index field on an index channel.
TEST(Reader, RejectsDataFieldOnIndexChannel) {
    auto b = indexed();
    b.index.reset();
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: only the index field can write to an index channel"
    );
}

/// @brief it should reject an index field that no binding names.
TEST(Reader, RejectsIndexFieldWithNoChannel) {
    auto b = indexed();
    b.bindings.pop_back();
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: the index field has no channel");
}

/// @brief it should reject two messages that share an index channel.
TEST(Reader, RejectsMessagesThatShareIndexChannel) {
    auto status = matching("status");
    auto other = reading("other");
    other.bindings[0].channel.key = 3;
    const auto err = Reader::create({status, other}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "messages status and other write to the same channel 1");
}

/// @brief it should reject two messages that write to the same channel.
TEST(Reader, RejectsMessagesThatShareChannel) {
    auto status = matching("status");
    status.bindings[0].channel.index = 0;
    auto other = reading("other");
    other.bindings[0].channel.index = 0;
    const auto err = Reader::create({status, other}).second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "messages status and other write to the same channel 2");
}

/// @brief it should reject a message that binds two fields to one channel.
TEST(Reader, RejectsChannelBoundTwice) {
    auto b = reading();
    const auto w = float_field("w");
    std::get<icd::JSONPayload>(b.message.payload).fields.emplace_back(w);
    b.bindings.push_back({.field = w.key, .channel = channel(2, x::telem::FLOAT64_T)});
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading writes to channel 2 twice");
}

/// @brief it should reject a message that binds its index field twice.
TEST(Reader, RejectsIndexFieldBoundTwice) {
    auto b = indexed();
    b.bindings.push_back(b.bindings.back());
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading writes to channel 1 twice");
}

/// @brief it should reject a binding that the decoder rejects.
TEST(Reader, RejectsBindingTheDecoderRejects) {
    auto b = reading();
    b.bindings[0].channel.data_type = x::telem::STRING_T;
    const auto err = create_error(b);
    ASSERT_OCCURRED_AS(err, codec::CONFIG_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: field v: cannot write float values to string channels"
    );
}
}
