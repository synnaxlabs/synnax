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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/payload/number.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"

namespace driver::icd {
namespace {
const x::telem::TimeStamp RECEIVED(1000);

using x::testutil::bytes;

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

synnax::icd::FloatJSONField float_field(const std::string &name) {
    synnax::icd::FloatJSONField f;
    f.key = x::uuid::create();
    f.name = name;
    f.pointer = "/" + name;
    return f;
}

synnax::icd::TimestampJSONField time_field() {
    synnax::icd::TimestampJSONField f;
    f.key = x::uuid::create();
    f.name = "t";
    f.pointer = "/t";
    f.time_format = "unix_ns";
    return f;
}

/// @brief returns an unsigned binary field named name in the byte at offset that the
/// multiplexor mux selects with value, or that no multiplexor selects.
synnax::icd::UnsignedBinaryField byte_field(
    const std::string &name,
    const std::uint16_t offset,
    const std::optional<synnax::icd::FieldKey> mux = std::nullopt,
    const std::int32_t value = 0
) {
    synnax::icd::UnsignedBinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = offset * 8;
    f.multiplexor = mux;
    if (mux.has_value()) f.multiplex_values = {value};
    return f;
}

/// @brief Config holds what Reader::create takes: the reads, the ICD that holds their
/// messages, and their channels.
struct Config {
    std::vector<synnax::icd::ReadMessage> reads;
    synnax::icd::ICD ic;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    /// @brief adds f to the first message, whose payload is a Payload, and binds it to
    /// ch.
    template<typename Payload = synnax::icd::JSONPayload, typename Field>
    void bind(const Field &f, const synnax::channel::Channel &ch) {
        std::get<Payload>(this->ic.messages[0].payload).fields.push_back(f);
        this->reads[0].bindings.push_back(
            {.field = synnax::icd::base(f).key, .channel = ch.key}
        );
        this->channels.insert_or_assign(ch.key, ch);
    }

    /// @brief adds the reads, messages, and channels of other.
    void join(const Config &other) {
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

/// @brief returns a config that reads a message named name with payload and no
/// bindings.
Config message(const std::string &name, const synnax::icd::Payload &payload) {
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = payload;
    Config e{.reads = {{.message = m.key}}};
    e.ic.name = "sensors";
    e.ic.messages = {m};
    return e;
}

/// @brief returns a config that reads a JSON message named name whose field v writes
/// to ch.
Config reading(
    const std::string &name = "reading",
    const synnax::channel::Channel &ch = channel(2, x::telem::FLOAT64_T)
) {
    auto e = message(name, synnax::icd::JSONPayload{});
    e.bind(float_field("v"), ch);
    return e;
}

/// @brief returns a config whose field t indexes field v.
Config indexed() {
    auto e = reading();
    const auto t = time_field();
    e.bind(t, channel(1, x::telem::TIMESTAMP_T));
    e.reads[0].index = t.key;
    return e;
}

/// @brief returns a config that reads the documents whose type is type, as a message
/// named type whose field v writes to ch.
Config matching(
    const std::string &type,
    const synnax::channel::Channel &ch = channel(2, x::telem::FLOAT64_T)
) {
    auto e = reading(type, ch);
    synnax::icd::StringJSONMatch match;
    match.pointer = "/type";
    match.value = type;
    std::get<synnax::icd::JSONPayload>(e.ic.messages[0].payload).match = match;
    return e;
}

/// @brief returns a config that reads a binary message named frame whose first byte v
/// writes to channel 2.
Config frame() {
    auto e = message("frame", synnax::icd::BinaryPayload{});
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("v", 0),
        channel(2, x::telem::FLOAT64_T)
    );
    return e;
}

/// @brief returns a config that reads a binary message named frame with a multiplexor
/// in its first byte, and returns the key of the multiplexor.
std::pair<Config, synnax::icd::FieldKey> multiplexed() {
    auto e = message("frame", synnax::icd::BinaryPayload{});
    const auto mux = byte_field("mux", 0);
    std::get<synnax::icd::BinaryPayload>(e.ic.messages[0].payload)
        .fields.emplace_back(mux);
    return {e, mux.key};
}
}

/// @brief it should read a binary frame into the channels of its message.
TEST(Reader, ReadsBinaryMessage) {
    auto r = ASSERT_NIL_P(frame().create());
    x::telem::Frame fr(2);
    const std::vector<std::uint8_t> payload{7};
    ASSERT_EQ(ASSERT_NIL_P(r.read(payload, RECEIVED, fr)), 0U);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<double>(2, 0), 7.0);
}

/// @brief it should read a text line into the channels of its message.
TEST(Reader, ReadsTextMessage) {
    auto e = message("line", synnax::icd::TextPayload{});
    synnax::icd::DelimitedTextField v;
    v.key = x::uuid::create();
    v.name = "v";
    v.position = 1;
    e.bind<synnax::icd::TextPayload>(v, channel(2, x::telem::FLOAT64_T));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(2);
    ASSERT_EQ(ASSERT_NIL_P(r.read(bytes("oven, 21.5"), RECEIVED, fr)), 0U);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<double>(2, 0), 21.5);
}

/// @brief it should write an index channel and the channels it indexes only when the
/// multiplexors of the frame select their fields.
TEST(Reader, WritesOnlySelectedIndexGroups) {
    auto [e, mux] = multiplexed();
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("a", 1, mux, 1),
        channel(2, x::telem::FLOAT64_T, 1)
    );
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("b", 1, mux, 2),
        channel(3, x::telem::FLOAT64_T, 4)
    );
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame first(2);
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{1, 7}, RECEIVED, first));
    ASSERT_EQ(*first.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(first.at<double>(2, 0), 7.0);
    x::telem::Frame second(2);
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{2, 9}, RECEIVED, second));
    ASSERT_EQ(*second.channels, (std::vector<std::uint32_t>{4, 3}));
    ASSERT_EQ(second.at<double>(3, 0), 9.0);
}

/// @brief it should write a multiplexed field that has a default in every frame.
TEST(Reader, WritesMultiplexedFieldWithDefault) {
    auto [e, mux] = multiplexed();
    auto a = byte_field("a", 1, mux, 1);
    a.default_ = x::payload::Unsigned{5};
    e.bind<synnax::icd::BinaryPayload>(a, channel(2, x::telem::FLOAT64_T));
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("b", 2),
        channel(3, x::telem::FLOAT64_T)
    );
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(3);
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{2, 7, 9}, RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2, 3}));
    ASSERT_EQ(fr.at<double>(2, 0), 5.0);
    ASSERT_EQ(fr.at<double>(3, 0), 9.0);
}

/// @brief it should add no series for a virtual channel whose field the multiplexors of
/// a frame do not select.
TEST(Reader, AddsNoSeriesForUnselectedVirtualChannel) {
    auto [e, mux] = multiplexed();
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("a", 1, mux, 1),
        channel(2, x::telem::FLOAT64_T, 0)
    );
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("b", 2),
        channel(3, x::telem::FLOAT64_T)
    );
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(2);
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{2, 7, 9}, RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 3}));
}

/// @brief it should reject a binary index field that takes a default.
TEST(Reader, RejectsBinaryIndexFieldWithDefault) {
    auto e = message("frame", synnax::icd::BinaryPayload{});
    auto t = byte_field("t", 0);
    t.default_ = x::payload::Unsigned{5};
    e.bind<synnax::icd::BinaryPayload>(t, channel(1, x::telem::TIMESTAMP_T));
    e.reads[0].index = t.key;
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "message frame: the index field t cannot take a default");
}

/// @brief it should reject a binary index field, since a binary field cannot write
/// timestamps.
TEST(Reader, RejectsBinaryIndexField) {
    auto e = message("frame", synnax::icd::BinaryPayload{});
    const auto t = byte_field("t", 0);
    e.bind<synnax::icd::BinaryPayload>(t, channel(1, x::telem::TIMESTAMP_T));
    e.reads[0].index = t.key;
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, codec::CONFIG_ERROR);
    ASSERT_EQ(
        err.data,
        "message frame: field t: cannot write unsigned values to timestamp channels"
    );
}

/// @brief it should reject two fields on one index channel that different multiplexor
/// values select.
TEST(Reader, RejectsIndexGroupWithDifferentSelections) {
    auto [e, mux] = multiplexed();
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("a", 1, mux, 1),
        channel(2, x::telem::FLOAT64_T)
    );
    e.bind<synnax::icd::BinaryPayload>(
        byte_field("b", 2),
        channel(3, x::telem::FLOAT64_T)
    );
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message frame: fields a and b share index channel 1, but different "
        "multiplexor values select them"
    );
}

/// @brief it should add one sample for each channel of a message, stamped with the time
/// the response arrived.
TEST(Reader, StampsSamplesOnArrival) {
    auto r = ASSERT_NIL_P(reading().create());
    x::telem::Frame fr(2);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 1.5})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(
        fr.series->at(0).values<std::int64_t>(),
        (std::vector{RECEIVED.nanoseconds()})
    );
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{1.5}));
}

/// @brief it should read a string field into a string channel.
TEST(Reader, ReadsStringField) {
    auto e = message("reading", synnax::icd::JSONPayload{});
    synnax::icd::StringJSONField f;
    f.key = x::uuid::create();
    f.name = "s";
    f.pointer = "/s";
    e.bind(f, channel(2, x::telem::STRING_T));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(2);
    ASSERT_NIL_P(r.read(bytes(R"({"s": "outlet valve open"})"), RECEIVED, fr));
    ASSERT_EQ(
        fr.series->at(1).strings(),
        (std::vector<std::string>{"outlet valve open"})
    );
}

/// @brief it should add the samples of a read to the series that the frame holds.
TEST(Reader, AppendsToSeriesInFrame) {
    auto r = ASSERT_NIL_P(frame().create());
    x::telem::Frame fr;
    const auto later = RECEIVED + x::telem::NANOSECOND;
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{7}, RECEIVED, fr, 2));
    ASSERT_NIL_P(r.read(std::vector<std::uint8_t>{9}, later, fr, 2));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(
        fr.series->at(0).values<std::int64_t>(),
        (std::vector{RECEIVED.nanoseconds(), later.nanoseconds()})
    );
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0, 9.0}));
}

/// @brief it should stamp the samples of a message with the value of its index field.
TEST(Reader, StampsSamplesWithIndexField) {
    auto r = ASSERT_NIL_P(indexed().create());
    x::telem::Frame fr(2);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 2, "t": 5})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), 5);
    ASSERT_EQ(fr.at<double>(2, 0), 2.0);
}

/// @brief it should list each channel that it writes, with index channels last.
TEST(Reader, ListsChannelsWithIndexesLast) {
    auto e = indexed();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT64_T, 4));
    e.bind(float_field("x"), channel(5, x::telem::FLOAT64_T, 0));
    const auto r = ASSERT_NIL_P(e.create());
    ASSERT_EQ(r.channels(), (std::vector<synnax::channel::Key>{2, 3, 5, 1, 4}));
}

/// @brief it should write no index for virtual channels.
TEST(Reader, WritesNoIndexForVirtualChannels) {
    auto r = ASSERT_NIL_P(
        reading("reading", channel(2, x::telem::FLOAT64_T, 0)).create()
    );
    x::telem::Frame fr(1);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 3})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{2}));
}

/// @brief it should stamp each index channel of a message once, and write no index for
/// its virtual channels.
TEST(Reader, StampsEachIndexChannelOnArrival) {
    auto e = reading();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT64_T, 4));
    e.bind(float_field("x"), channel(5, x::telem::FLOAT64_T, 0));
    e.bind(float_field("y"), channel(6, x::telem::FLOAT64_T, 4));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(6);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 1, "w": 2, "x": 3, "y": 4})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 4, 2, 3, 5, 6}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<std::int64_t>(4, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<double>(5, 0), 3.0);
}

/// @brief it should stamp only the index channel of the index field with its value.
TEST(Reader, StampsOtherIndexChannelsOnArrival) {
    auto e = indexed();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT64_T, 4));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(4);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 2, "t": 5, "w": 3})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 4, 2, 3}));
    ASSERT_EQ(fr.at<std::int64_t>(1, 0), 5);
    ASSERT_EQ(fr.at<std::int64_t>(4, 0), RECEIVED.nanoseconds());
    ASSERT_EQ(fr.at<double>(3, 0), 3.0);
}

/// @brief it should write the other index groups when a value of one group does not
/// convert, and list the field in its failures.
TEST(Reader, WritesOtherIndexGroupsWhenValueOfOneDoesNotConvert) {
    auto e = reading();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT64_T, 4));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(4);
    ASSERT_EQ(ASSERT_NIL_P(r.read(bytes(R"({"v": 1, "w": {}})"), RECEIVED, fr)), 0U);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.at<double>(2, 0), 1.0);
    ASSERT_EQ(r.failures().size(), 1U);
    ASSERT_EQ(r.failures()[0].binding, 1U);
    ASSERT_OCCURRED_AS(r.failures()[0].error, x::errors::VALIDATION);
    ASSERT_EQ(
        r.failures()[0].error.data,
        "field w: expected a number or boolean, got an object"
    );
}

/// @brief it should drop only a virtual channel whose value does not convert.
TEST(Reader, DropsOnlyVirtualChannelWhoseValueDoesNotConvert) {
    auto e = reading();
    e.bind(float_field("x"), channel(5, x::telem::FLOAT64_T, 0));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(3);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 1, "x": {}})"), RECEIVED, fr));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
}

/// @brief it should write the fill value for a float field that does not decode, and
/// keep its index group.
TEST(Reader, FillsFloatFieldThatDoesNotDecode) {
    auto e = reading();
    e.bind(float_field("w"), channel(3, x::telem::FLOAT32_T, 4));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(4);
    const auto nan = std::numeric_limits<double>::quiet_NaN();
    ASSERT_NIL_P(r.read(bytes(R"({"v": {}})"), RECEIVED, fr, 1, nan));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 4, 2, 3}));
    ASSERT_TRUE(std::isnan(fr.at<double>(2, 0)));
    ASSERT_TRUE(std::isnan(fr.at<float>(3, 0)));
    ASSERT_EQ(r.failures().size(), 2U);
}

/// @brief it should drop the index group of a field on a channel that is not a float
/// when the field does not decode, even with a fill value.
TEST(Reader, DropsGroupOfNonFloatFieldWithFill) {
    auto e = reading();
    synnax::icd::IntegerJSONField w;
    w.key = x::uuid::create();
    w.name = "w";
    w.pointer = "/w";
    e.bind(w, channel(3, x::telem::INT64_T, 4));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame fr(4);
    ASSERT_NIL_P(r.read(bytes(R"({"v": 1})"), RECEIVED, fr, 1, 0.0));
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(r.failures()[0].error.data, "field w: no value at /w");
}

/// @brief it should read a response as the message whose match rule it meets, or else
/// as the message with no rule.
TEST(Reader, ReadsMatchedMessage) {
    auto e = matching("status");
    e.join(reading("other", channel(3, x::telem::FLOAT64_T, 4)));
    auto r = ASSERT_NIL_P(e.create());
    x::telem::Frame matched(2);
    ASSERT_NIL_P(r.read(bytes(R"({"type": "status", "v": 5})"), RECEIVED, matched));
    ASSERT_EQ(*matched.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(matched.at<double>(2, 0), 5.0);
    x::telem::Frame fallback(2);
    ASSERT_NIL_P(r.read(bytes(R"({"type": "event", "v": 4})"), RECEIVED, fallback));
    ASSERT_EQ(*fallback.channels, (std::vector<std::uint32_t>{4, 3}));
}

/// @brief it should add nothing to the frame when the value of its only index group
/// does not convert.
TEST(Reader, AddsNothingWhenValueOfOnlyIndexGroupDoesNotConvert) {
    auto r = ASSERT_NIL_P(reading().create());
    x::telem::Frame fr(2);
    ASSERT_EQ(ASSERT_NIL_P(r.read(bytes(R"({"v": {}})"), RECEIVED, fr)), 0U);
    ASSERT_TRUE(fr.empty());
    ASSERT_EQ(r.failures().size(), 1U);
}

/// @brief it should list no failures after a read of a payload that belongs to no
/// message.
TEST(Reader, ListsNoFailuresAfterReadOfNoMessage) {
    auto r = ASSERT_NIL_P(matching("status").create());
    x::telem::Frame fr(2);
    ASSERT_NIL_P(r.read(bytes(R"({"type": "status", "v": {}})"), RECEIVED, fr));
    ASSERT_EQ(r.failures().size(), 1U);
    ASSERT_NIL_P(r.read(bytes(R"({"type": "event", "v": 1})"), RECEIVED, fr));
    ASSERT_TRUE(r.failures().empty());
}

/// @brief it should add nothing for a payload that belongs to no message.
TEST(Reader, ReturnsNothingForPayloadOfNoMessage) {
    auto r = ASSERT_NIL_P(matching("status").create());
    x::telem::Frame fr(2);
    const auto msg = ASSERT_NIL_P(
        r.read(bytes(R"({"type": "event", "v": 1})"), RECEIVED, fr)
    );
    ASSERT_FALSE(msg.has_value());
    ASSERT_TRUE(fr.empty());
}

/// @brief it should read nothing when it holds no messages.
TEST(Reader, ReadsNothingWithNoMessages) {
    auto r = ASSERT_NIL_P(Config{}.create());
    x::telem::Frame fr;
    ASSERT_FALSE(ASSERT_NIL_P(r.read(bytes("{}"), RECEIVED, fr)).has_value());
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

/// @brief it should reject two messages of different formats.
TEST(Reader, RejectsMessagesOfDifferentFormats) {
    auto e = reading();
    e.join(frame());
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "messages reading and frame have different formats");
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
    e.channels.at(1).name = "time";
    const auto err = e.create().second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "message reading: field t cannot write to index channel time, because only "
        "the index field can. Bind field t to a data channel"
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
    auto &fields = std::get<synnax::icd::JSONPayload>(e.ic.messages[0].payload).fields;
    std::get<synnax::icd::TimestampJSONField>(fields.back())
        .default_ = synnax::icd::ConstantTimestampDefault{
        .value = x::telem::TimeStamp(5)
    };
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
