// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
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

/// @brief returns a channel of dt with key that index indexes.
synnax::channel::Channel channel(
    const synnax::channel::Key key,
    const x::telem::DataType &dt,
    const synnax::channel::Key index = 1
) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.data_type = dt;
    ch.index = index;
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

/// @brief Endpoint holds what Reader::create takes: the reads of an endpoint, the ICD
/// that holds their messages, and their channels.
struct Endpoint {
    std::vector<icd::ReadMessage> reads;
    icd::ICD ic;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    /// @brief adds f to the first message and binds it to ch.
    void bind(const icd::JSONField &f, const synnax::channel::Channel &ch) {
        std::get<icd::JSONPayload>(this->ic.messages[0].payload).fields.push_back(f);
        this->reads[0].bindings.push_back(
            {.field = icd::base(f).key, .channel = ch.key}
        );
        this->channels.insert_or_assign(ch.key, ch);
    }

    /// @brief adds the reads, messages, and channels of other.
    void join(const Endpoint &other) {
        this->reads.insert(this->reads.end(), other.reads.begin(), other.reads.end());
        this->ic.messages.insert(
            this->ic.messages.end(),
            other.ic.messages.begin(),
            other.ic.messages.end()
        );
        this->channels.insert(other.channels.begin(), other.channels.end());
    }

    [[nodiscard]] std::pair<Reader, x::errors::Error> create() const {
        return Reader::create(this->reads, this->ic, this->channels);
    }
};

/// @brief returns an endpoint that reads a message named name whose field v writes to
/// ch.
Endpoint reading(
    const std::string &name = "reading",
    const synnax::channel::Channel &ch = channel(2, x::telem::FLOAT64_T)
) {
    icd::Message m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = icd::JSONPayload{};
    Endpoint e{.reads = {{.message = m.key}}};
    e.ic.name = "sensors";
    e.ic.messages = {m};
    e.bind(float_field("v"), ch);
    return e;
}

/// @brief returns an endpoint whose field t indexes field v.
Endpoint indexed() {
    auto e = reading();
    const auto t = time_field();
    e.bind(t, channel(1, x::telem::TIMESTAMP_T));
    e.reads[0].index = t.key;
    return e;
}

/// @brief returns an endpoint that reads the documents whose type is type, as a message
/// named type whose field v writes to ch.
Endpoint matching(
    const std::string &type,
    const synnax::channel::Channel &ch = channel(2, x::telem::FLOAT64_T)
) {
    auto e = reading(type, ch);
    icd::StringJSONMatch match;
    match.pointer = "/type";
    match.value = type;
    std::get<icd::JSONPayload>(e.ic.messages[0].payload).match = match;
    return e;
}
}

/// @brief it should stamp the samples of a message with the time the response arrived.
TEST(Reader, StampsSamplesOnArrival) {
    auto r = ASSERT_NIL_P(reading().create());
    x::telem::Frame fr(2);
    ASSERT_NIL(r.read(R"({"v": 1.5})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<double>(2, 0), 1.5);
}

/// @brief it should stamp the samples of a message with the value of its index field.
TEST(Reader, StampsSamplesWithIndexField) {
    auto r = ASSERT_NIL_P(indexed().create());
    x::telem::Frame fr(2);
    ASSERT_NIL(r.read(R"({"v": 2, "t": 5})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), 5);
    ASSERT_EQ(fr.at<double>(2, 0), 2.0);
}

/// @brief it should write no index for virtual channels.
TEST(Reader, WritesNoIndexForVirtualChannels) {
    auto r = ASSERT_NIL_P(
        reading("reading", channel(2, x::telem::FLOAT64_T, 0)).create()
    );
    x::telem::Frame fr(1);
    ASSERT_NIL(r.read(R"({"v": 3})", RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{2}));
}

/// @brief it should read a response as the message whose match rule it meets, or else
/// as the message with no rule.
TEST(Reader, ReadsMatchedMessage) {
    auto e = matching("status");
    e.join(reading("other", channel(3, x::telem::FLOAT64_T, 4)));
    auto r = ASSERT_NIL_P(e.create());
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
    auto r = ASSERT_NIL_P(reading().create());
    x::telem::Frame fr(2);
    const auto err = r.read(R"({"v": "x"})", RECEIVED, fr);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, R"(field v: unknown enum label "x")");
    ASSERT_TRUE(fr.empty());
}

/// @brief it should reject a response that meets no message.
TEST(Reader, RejectsResponseThatMeetsNoMessage) {
    auto r = ASSERT_NIL_P(matching("status").create());
    x::telem::Frame fr(2);
    const auto err = r.read(R"({"type": "event", "v": 1})", RECEIVED, fr);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "the response matches no message");
    ASSERT_TRUE(fr.empty());
}

/// @brief it should reject an endpoint that reads no messages.
TEST(Reader, RejectsNoMessages) {
    const auto err = Endpoint{}.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "the endpoint reads no messages");
}

/// @brief it should reject a read of a message that the ICD does not hold.
TEST(Reader, RejectsMessageNotInICD) {
    auto e = reading();
    const auto key = x::uuid::create();
    e.reads[0].message = key;
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message " + key.to_string() + " does not exist in ICD sensors"
    );
}

/// @brief it should reject a message that is not JSON.
TEST(Reader, RejectsMessageThatIsNotJSON) {
    auto e = reading();
    e.ic.messages[0].payload = icd::BinaryPayload{};
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading is not a JSON message");
}

/// @brief it should reject a message that binds no fields.
TEST(Reader, RejectsMessageThatBindsNoFields) {
    auto e = reading();
    e.reads[0].bindings.clear();
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading binds no fields");
}

/// @brief it should reject a binding whose channel does not exist.
TEST(Reader, RejectsChannelThatDoesNotExist) {
    auto e = reading();
    e.channels.clear();
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: channel 2 does not exist");
}

/// @brief it should reject channels that do not share one index channel.
TEST(Reader, RejectsChannelsWithDifferentIndexes) {
    auto e = reading();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT64_T, 4));
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: every channel must share one index channel");
}

/// @brief it should reject an index field that does not write to an index channel.
TEST(Reader, RejectsIndexFieldOnDataChannel) {
    auto e = indexed();
    e.channels.at(1).is_index = false;
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: the index field must write to an index channel"
    );
}

/// @brief it should reject a field other than the index field on an index channel.
TEST(Reader, RejectsDataFieldOnIndexChannel) {
    auto e = indexed();
    e.reads[0].index.reset();
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: only the index field can write to an index channel"
    );
}

/// @brief it should reject an index field that no binding names.
TEST(Reader, RejectsIndexFieldWithNoChannel) {
    auto e = indexed();
    e.reads[0].bindings.pop_back();
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: the index field has no channel");
}

/// @brief it should reject an index field that takes a default.
TEST(Reader, RejectsIndexFieldWithDefault) {
    auto e = indexed();
    auto &fields = std::get<icd::JSONPayload>(e.ic.messages[0].payload).fields;
    std::get<icd::TimestampJSONField>(fields.back()).default_ = x::telem::TimeStamp(5);
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading: the index field t cannot take a default");
}

/// @brief it should reject two messages that share an index channel.
TEST(Reader, RejectsMessagesThatShareIndexChannel) {
    auto e = matching("status");
    e.join(reading("other", channel(3, x::telem::FLOAT64_T)));
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "messages status and other write to the same channel 1");
}

/// @brief it should reject two messages that write to the same channel.
TEST(Reader, RejectsMessagesThatShareChannel) {
    const auto ch = channel(2, x::telem::FLOAT64_T, 0);
    auto e = matching("status", ch);
    e.join(reading("other", ch));
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "messages status and other write to the same channel 2");
}

/// @brief it should reject a message that binds two fields to one channel.
TEST(Reader, RejectsChannelBoundTwice) {
    auto e = reading();
    e.bind(float_field("w"), channel(2, x::telem::FLOAT64_T));
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading writes to channel 2 twice");
}

/// @brief it should reject a message that binds its index field twice.
TEST(Reader, RejectsIndexFieldBoundTwice) {
    auto e = indexed();
    e.reads[0].bindings.push_back(e.reads[0].bindings.back());
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message reading writes to channel 1 twice");
}

/// @brief it should reject a binding that the decoder rejects.
TEST(Reader, RejectsBindingTheDecoderRejects) {
    auto e = reading();
    e.channels.at(2).data_type = x::telem::STRING_T;
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, codec::CONFIG_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: field v: cannot write float values to string channels"
    );
}
}
