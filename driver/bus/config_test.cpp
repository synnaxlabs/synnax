// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <tuple>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/config.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::bus {
using namespace testutil;

namespace {
/// @brief a read config over one text message with one field on channel 2, stamped on
/// index 1.
struct ReadFixture {
    synnax::library::MessageEntry message = text_message(
        "status",
        {delimited_field("value", 0)}
    );
    ::synnax::bus::ReadConfig cfg;
    std::vector<synnax::channel::Channel> channels = {
        index_channel(1),
        data_channel(2, 1),
    };
    std::optional<::synnax::bus::Framing> framing = ::synnax::bus::DelimiterFraming{};
    ::synnax::bus::PollConfig poll;
    Medium medium = Medium::BYTES;

    ReadFixture() {
        cfg.device = "dev";
        cfg.messages = {{
            .message = message.key,
            .fields = {{.field = field_key(message, 0), .channel = 2, .index = 1}},
        }};
    }

    /// @returns how the fixture splits its byte stream, if it has framing.
    [[nodiscard]] std::optional<Stream> stream() const {
        if (!framing.has_value()) return std::nullopt;
        return Stream{.framing = *framing};
    }

    /// @returns the config resolved against lib, and the error of resolve.
    [[nodiscard]] std::pair<ReadConfig, x::errors::Error>
    resolve(const synnax::library::Library &lib) const {
        x::json::Parser parser(x::json::json::object());
        auto [out, err] = ReadConfig::resolve(
            parser,
            cfg,
            poll,
            stream(),
            lib,
            channels,
            medium
        );
        EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
        return {std::move(out), err};
    }

    /// @returns the first field error of resolving the config against lib.
    [[nodiscard]] std::pair<std::string, std::string>
    first_error(const synnax::library::Library &lib) const {
        x::json::Parser parser(x::json::json::object());
        const auto [_, err] = ReadConfig::resolve(
            parser,
            cfg,
            poll,
            stream(),
            lib,
            channels,
            medium
        );
        if (parser.ok()) return {"", ""};
        const auto first = parser.error_json()["errors"][0];
        return {first["path"], first["message"]};
    }

    [[nodiscard]] std::pair<std::string, std::string> first_error() const {
        return this->first_error(library({message}));
    }
};

/// @returns a client whose port has no Core, for parses that must not reach one.
synnax::Synnax unreachable_client() {
    synnax::Config cfg;
    cfg.port = 1;
    return synnax::Synnax(cfg);
}
}

TEST(Unescape, DecodesEscapedBytes) {
    const auto out = ASSERT_NIL_P(unescape(R"(\xAA\x55a\n\r\t\0\\)"));
    EXPECT_EQ(
        out,
        (std::vector<std::uint8_t>{0xAA, 0x55, 'a', '\n', '\r', '\t', 0, '\\'})
    );
}

TEST(Unescape, RejectsAnUnknownEscape) {
    ASSERT_OCCURRED_AS_P(unescape(R"(\q)"), x::errors::VALIDATION);
}

TEST(Unescape, RejectsATruncatedHexEscape) {
    ASSERT_OCCURRED_AS_P(unescape(R"(\x4)"), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(unescape("\\"), x::errors::VALIDATION);
}

TEST(ReadConfig, ResolvesAValidConfig) {
    ReadFixture f;
    const auto [cfg, err] = f.resolve(library({f.message}));
    ASSERT_NIL(err);
    ASSERT_EQ(cfg.messages.size(), 1);
    ASSERT_EQ(cfg.messages[0].groups.size(), 1);
    EXPECT_EQ(cfg.messages[0].groups[0].index, 1);
    EXPECT_EQ(cfg.messages[0].groups[0].slots, std::vector<std::size_t>{0});
    EXPECT_EQ(cfg.messages[0].channels[0].key, 2);
    EXPECT_EQ(cfg.streamed, std::vector<std::size_t>{0});
    EXPECT_TRUE(cfg.polled.empty());
    EXPECT_NE(cfg.framer, nullptr);
}

TEST(ReadConfig, RejectsAMessageNotInTheLibrary) {
    ReadFixture f;
    const auto [path, msg] = f.first_error(library({}));
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(msg, "message " + f.message.key.to_string() + " is not in library test");
}

TEST(ReadConfig, RejectsAFieldNotInTheMessage) {
    ReadFixture f;
    const auto missing = x::uuid::create();
    f.cfg.messages[0].fields[0].field = missing;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.0.field");
    EXPECT_EQ(msg, "field " + missing.to_string() + " is not in message status");
}

TEST(ReadConfig, RejectsAChannelOnAnotherIndex) {
    ReadFixture f;
    f.channels.push_back(index_channel(3));
    f.channels[1].index = 3;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.0.channel");
    EXPECT_EQ(msg, "channel data_2 has index 3, but field value is stamped on 1");
}

TEST(ReadConfig, RejectsAChannelThatDoesNotExist) {
    ReadFixture f;
    f.cfg.messages[0].fields[0].channel = 42;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.0.channel");
    EXPECT_EQ(msg, "channel 42 does not exist");
}

TEST(ReadConfig, RejectsAStringChannel) {
    ReadFixture f;
    f.channels[1].data_type = x::telem::STRING_T;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.0.channel");
    EXPECT_EQ(
        msg,
        "channel data_2 has data type string, which cannot hold a numeric field"
    );
}

TEST(ReadConfig, RejectsAFieldReadTwice) {
    ReadFixture f;
    f.cfg.messages[0].fields.push_back(f.cfg.messages[0].fields[0]);
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.1.field");
    EXPECT_EQ(msg, "field value is read twice");
}

TEST(ReadConfig, RejectsAChannelUsedTwice) {
    ReadFixture f;
    f.message = text_message(
        "status",
        {delimited_field("a", 0), delimited_field("b", 1)}
    );
    f.cfg.messages = {{
        .message = f.message.key,
        .fields = {
            {.field = field_key(f.message, 0), .channel = 2, .index = 1},
            {.field = field_key(f.message, 1), .channel = 2, .index = 1},
        },
    }};
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.1.channel");
    EXPECT_EQ(msg, "channel 2 is used more than once");
}

TEST(ReadConfig, RejectsAnIndexThatIsNotAnIndexChannel) {
    ReadFixture f;
    f.channels[0].is_index = false;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.fields.0.index");
    EXPECT_EQ(msg, "channel index_1 is not an index channel");
}

TEST(ReadConfig, RejectsACANIdentifierOnAByteStream) {
    ReadFixture f;
    f.message = binary_message(
        "status",
        {binary_field("value", 0)},
        synnax::library::CanIdentifier{.id = 0x10}
    );
    f.cfg.messages[0].message = f.message.key;
    f.cfg.messages[0].fields[0].field = field_key(f.message, 0);
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(
        msg,
        "message status has a can identifier, which a byte stream cannot carry"
    );
}

TEST(ReadConfig, RejectsAFieldIdentifierOnACANBus) {
    ReadFixture f;
    f.message = binary_message("status", {binary_field("value", 0)});
    binary(f.message).identifier = synnax::library::FieldIdentifier{
        .field = field_key(f.message, 0),
        .value = 1,
    };
    f.cfg.messages[0].message = f.message.key;
    f.cfg.messages[0].fields[0].field = field_key(f.message, 0);
    f.framing = std::nullopt;
    f.medium = Medium::CAN;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(
        msg,
        "message status has a field identifier, which a CAN bus cannot carry"
    );
}

TEST(ReadConfig, RejectsATextMessageOnACANBus) {
    ReadFixture f;
    f.framing = std::nullopt;
    f.medium = Medium::CAN;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(msg, "message status has a text payload, which a CAN bus cannot carry");
}

TEST(ReadConfig, RejectsAQueryOnACANBus) {
    ReadFixture f;
    f.message = binary_message(
        "status",
        {binary_field("value", 0)},
        synnax::library::CanIdentifier{.id = 0x10}
    );
    f.message.query = R"(\x01)";
    f.cfg.messages[0].message = f.message.key;
    f.cfg.messages[0].fields[0].field = field_key(f.message, 0);
    f.framing = std::nullopt;
    f.medium = Medium::CAN;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(msg, "message status has a query, which a CAN bus cannot send");
}

TEST(ReadConfig, RejectsARawChannelThatIsNotBytes) {
    ReadFixture f;
    f.channels.push_back(data_channel(9, 0));
    f.cfg.raw = 9;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "raw");
    EXPECT_EQ(msg, "channel data_9 must have data type bytes");
}

TEST(ReadConfig, RejectsAnInvalidFraming) {
    ReadFixture f;
    f.framing = ::synnax::bus::SyncFraming{.sync = "zz"};
    const auto [path, _] = f.first_error();
    EXPECT_EQ(path, "framing");
}

TEST(ReadConfig, RejectsAQueryThatDoesNotFitTheFraming) {
    ReadFixture f;
    f.message.query = "VOLT?";
    f.framing = ::synnax::bus::FixedFraming{.length = 4};
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_TRUE(msg.starts_with("query of status does not fit the framing"));
}

TEST(ReadConfig, RejectsABinaryQueryWithAnUnknownEscape) {
    ReadFixture f;
    f.message = binary_message("status", {binary_field("value", 0)});
    f.message.query = R"(\q)";
    f.cfg.messages[0].message = f.message.key;
    f.cfg.messages[0].fields[0].field = field_key(f.message, 0);
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages.0.message");
    EXPECT_EQ(msg, "unknown escape \\q");
}

TEST(ReadConfig, RequiresAnEnabledMessage) {
    ReadFixture f;
    f.cfg.messages[0].disabled = true;
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "messages");
    EXPECT_EQ(msg, "at least one enabled message is required");
}

TEST(ReadConfig, RequiresADevice) {
    ReadFixture f;
    f.cfg.device = "";
    const auto [path, msg] = f.first_error();
    EXPECT_EQ(path, "device");
    EXPECT_EQ(msg, "this field is required");
}

TEST(ReadConfig, SendsTheFramedQueryOfAPolledMessage) {
    ReadFixture f;
    f.message.query = "VOLT?";
    f.poll = {.rate = x::telem::Rate(4), .timeout = x::telem::SECOND};
    const auto [cfg, err] = f.resolve(library({f.message}));
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.polled, std::vector<std::size_t>{0});
    EXPECT_EQ(cfg.messages[0].period, 250 * x::telem::MILLISECOND);
    EXPECT_EQ(text(cfg.messages[0].query), "VOLT?\n");
}

TEST(ReadConfig, GroupsTheFieldsOfAMessageByIndex) {
    ReadFixture f;
    f.message = binary_message(
        "status",
        {binary_field("a", 0), binary_field("b", 8), binary_field("c", 16)}
    );
    f.cfg.messages = {{
        .message = f.message.key,
        .fields = {
            {.field = field_key(f.message, 0), .channel = 2, .index = 1},
            {.field = field_key(f.message, 1), .channel = 4, .index = 3},
            {.field = field_key(f.message, 2), .channel = 5, .index = 1},
        },
    }};
    f.channels.push_back(index_channel(3));
    f.channels.push_back(data_channel(4, 3));
    f.channels.push_back(data_channel(5, 1));
    const auto [cfg, err] = f.resolve(library({f.message}));
    ASSERT_NIL(err);
    const auto &groups = cfg.messages[0].groups;
    ASSERT_EQ(groups.size(), 2);
    EXPECT_EQ(groups[0].index, 1);
    EXPECT_EQ(groups[0].slots, (std::vector<std::size_t>{0, 2}));
    EXPECT_EQ(groups[1].index, 3);
    EXPECT_EQ(groups[1].slots, std::vector<std::size_t>{1});
}

TEST(ReadConfig, GivesEachFieldWithoutAnIndexItsOwnGroup) {
    ReadFixture f;
    f.message = binary_message("status", {binary_field("a", 0), binary_field("b", 8)});
    f.cfg.messages = {{
        .message = f.message.key,
        .fields = {
            {.field = field_key(f.message, 0), .channel = 4},
            {.field = field_key(f.message, 1), .channel = 5},
        },
    }};
    f.channels = {data_channel(4, 0), data_channel(5, 0)};
    const auto [cfg, err] = f.resolve(library({f.message}));
    ASSERT_NIL(err);
    const auto &groups = cfg.messages[0].groups;
    ASSERT_EQ(groups.size(), 2);
    EXPECT_EQ(groups[0].index, 0);
    EXPECT_EQ(groups[0].slots, std::vector<std::size_t>{0});
    EXPECT_EQ(groups[1].slots, std::vector<std::size_t>{1});
}

TEST(ReadConfig, RejectsAnIndexThatStampsTwoMessages) {
    ReadFixture f;
    const auto other = text_message("other", {delimited_field("v", 0)}, "O,");
    f.cfg.messages.push_back({
        .message = other.key,
        .fields = {{.field = field_key(other, 0), .channel = 4, .index = 1}},
    });
    f.channels.push_back(data_channel(4, 1));
    const auto [path, msg] = f.first_error(library({f.message, other}));
    EXPECT_EQ(path, "messages.1.fields.0.index");
    EXPECT_EQ(msg, "channel 1 is used more than once");
}

TEST(ReadConfig, ExpectsAStreamedMessageEveryLibraryPeriod) {
    ReadFixture f;
    f.message.period = 20 * x::telem::MILLISECOND;
    const auto [cfg, err] = f.resolve(library({f.message}));
    ASSERT_NIL(err);
    EXPECT_EQ(cfg.messages[0].period, 20 * x::telem::MILLISECOND);
}

TEST(ReadConfig, ReportsALayoutTheDriverCannotCompileAsABug) {
    ReadFixture f;
    f.message = binary_message("status", {binary_field("value", 0)});
    binary(f.message).fields[0].bit_length = 65;
    f.cfg.messages[0].message = f.message.key;
    f.cfg.messages[0].fields[0].field = field_key(f.message, 0);
    const auto [_, err] = f.resolve(library({f.message}));
    ASSERT_MATCHES(err, x::errors::INTERNAL);
    EXPECT_TRUE(err.data.starts_with(
        "the Core accepted message status, but the Driver cannot compile it ("
    )) << err.data;
    EXPECT_TRUE(err.data.ends_with("). Please report this as a bug")) << err.data;
}

TEST(WriteConfig, ReportsALayoutTheDriverCannotCompileAsABug) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    binary(m).fields[0].bit_length = 65;
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    cfg.messages = {
        {.message = m.key, .fields = {{.field = field_key(m, 0), .channel = 5}}}
    };
    x::json::Parser parser(x::json::json::object());
    const auto [_, err] = WriteConfig::resolve(
        parser,
        cfg,
        std::nullopt,
        library({m}),
        {data_channel(5, 0)}
    );
    ASSERT_NIL(parser.error());
    ASSERT_MATCHES(err, x::errors::INTERNAL);
}

TEST(ReadConfig, ParseRejectsATaskWithNoLibrary) {
    const auto client = unreachable_client();
    ::synnax::bus::ReadConfig cfg;
    cfg.device = "dev";
    x::json::Parser parser(x::json::json::object());
    const auto [_, err] = ReadConfig::parse(client, parser, cfg, {}, std::nullopt);
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    const auto first = parser.error_json()["errors"][0];
    EXPECT_EQ(first["path"], "library");
    EXPECT_EQ(first["message"], "select a library");
}

TEST(WriteConfig, BindsCommandChannelsToFieldSlots) {
    auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    cfg.messages = {
        {.message = m.key, .fields = {{.field = field_key(m, 1), .channel = 5}}}
    };
    x::json::Parser parser(x::json::json::object());
    const auto [out, err] = WriteConfig::resolve(
        parser,
        cfg,
        std::nullopt,
        library({m}),
        {data_channel(5, 0)}
    );
    ASSERT_NIL(parser.error());
    ASSERT_NIL(err);
    ASSERT_EQ(out.messages.size(), 1);
    ASSERT_EQ(out.messages[0].bindings.size(), 1);
    EXPECT_EQ(out.messages[0].bindings[0].slot, 1);
    EXPECT_EQ(out.messages[0].bindings[0].field, "b");
    EXPECT_EQ(out.commands, std::vector<synnax::channel::Key>{5});
}

TEST(WriteConfig, RejectsAFieldMappedTwice) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    const auto field = field_key(m, 0);
    cfg.messages = {{
        .message = m.key,
        .fields = {{.field = field, .channel = 5}, {.field = field, .channel = 6}},
    }};
    x::json::Parser parser(x::json::json::object());
    std::ignore = WriteConfig::resolve(
        parser,
        cfg,
        std::nullopt,
        library({m}),
        {data_channel(5, 0), data_channel(6, 0)}
    );
    const auto err = parser.error_json()["errors"][0];
    EXPECT_EQ(err["path"], "messages.0.fields.1.field");
    EXPECT_EQ(err["message"], "field a is mapped twice");
}

TEST(WriteConfig, RequiresFieldsOnAMessageWithoutAPeriod) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    cfg.messages = {{.message = m.key}};
    x::json::Parser parser(x::json::json::object());
    std::ignore = WriteConfig::resolve(parser, cfg, std::nullopt, library({m}), {});
    const auto err = parser.error_json()["errors"][0];
    EXPECT_EQ(err["path"], "messages.0.fields");
    EXPECT_EQ(
        err["message"],
        "message cmd has no period, so it needs at least one field"
    );
}

TEST(WriteConfig, RequiresACANIdentifierOnACANBus) {
    auto m = binary_message("cmd", {binary_field("a", 0)});
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    cfg.messages = {
        {.message = m.key, .fields = {{.field = field_key(m, 0), .channel = 5}}}
    };
    x::json::Parser parser(x::json::json::object());
    std::ignore = WriteConfig::resolve(
        parser,
        cfg,
        std::nullopt,
        library({m}),
        {data_channel(5, 0)},
        Medium::CAN
    );
    const auto err = parser.error_json()["errors"][0];
    EXPECT_EQ(err["path"], "messages.0.message");
    EXPECT_EQ(err["message"], "message cmd needs a CAN identifier to send with");
}

TEST(WriteConfig, ParseRejectsATaskWithNoLibrary) {
    const auto client = unreachable_client();
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    x::json::Parser parser(x::json::json::object());
    const auto [_, err] = WriteConfig::parse(client, parser, cfg, std::nullopt);
    ASSERT_MATCHES(err, x::errors::VALIDATION);
    const auto first = parser.error_json()["errors"][0];
    EXPECT_EQ(first["path"], "library");
    EXPECT_EQ(first["message"], "select a library");
}
}
