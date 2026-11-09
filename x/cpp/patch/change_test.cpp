// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/json/json.h"
#include "x/cpp/patch/change.h"
#include "x/cpp/patch/json.gen.h"
#include "x/cpp/patch/types.gen.h"

namespace x::patch {
namespace {
struct Settings {
    int rate = 0;
    int stream = 0;

    static Settings parse(x::json::Parser parser) {
        return Settings{
            .rate = parser.field<int>("rate"),
            .stream = parser.field<int>("stream", 0),
        };
    }

    [[nodiscard]] x::json::json to_json() const {
        return {{"rate", this->rate}, {"stream", this->stream}};
    }

    bool operator==(const Settings &other) const = default;
};

struct SettingsUpdate {
    Change<int> rate;
    Change<int> stream;

    static SettingsUpdate parse(x::json::Parser parser) {
        return SettingsUpdate{
            .rate = Change<int>::parse(parser, "rate"),
            .stream = Change<int>::parse(parser, "stream"),
        };
    }

    [[nodiscard]] x::json::json to_json() const {
        auto j = x::json::json::object();
        this->rate.to_json(j, "rate");
        this->stream.to_json(j, "stream");
        return j;
    }

    bool operator==(const SettingsUpdate &other) const = default;
};

using SettingsField = Struct<SettingsUpdate, Settings>;

/// @brief has the shape of a generated update: an optional scalar, a required struct,
/// and an optional struct.
struct Holder {
    Change<std::string> group;
    std::optional<SettingsField> settings;
    Change<SettingsField> extra;

    static Holder parse(x::json::Parser parser) {
        return Holder{
            .group = Change<std::string>::parse(parser, "group"),
            .settings = parser.field<std::optional<SettingsField>>("settings"),
            .extra = Change<SettingsField>::parse(parser, "extra"),
        };
    }

    [[nodiscard]] x::json::json to_json() const {
        auto j = x::json::json::object();
        this->group.to_json(j, "group");
        if (this->settings.has_value()) j["settings"] = this->settings->to_json();
        this->extra.to_json(j, "extra");
        return j;
    }

    bool operator==(const Holder &other) const = default;
};

void expect_round_trip(const Holder &holder, const std::string &data) {
    const auto j = x::json::json::parse(data);
    EXPECT_EQ(holder.to_json(), j);
    x::json::Parser parser(j);
    EXPECT_EQ(Holder::parse(parser), holder);
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
}
}

/// @brief it should hold no change, a value, or a clear.
TEST(Change, States) {
    const Change<int> none;
    EXPECT_TRUE(none.is_zero());
    EXPECT_EQ(none.get(), nullptr);
    const auto set = Change<int>::to(0);
    EXPECT_FALSE(set.is_zero());
    ASSERT_NE(set.get(), nullptr);
    EXPECT_EQ(*set.get(), 0);
    const auto cleared = Change<int>::clear();
    EXPECT_TRUE(cleared.cleared());
    EXPECT_EQ(cleared.get(), nullptr);
}

/// @brief it should leave out a field with no change.
TEST(Change, Absent) {
    expect_round_trip(Holder{}, "{}");
}

/// @brief it should write and read a cleared field as null.
TEST(Change, Clear) {
    expect_round_trip(
        Holder{.group = Change<std::string>::clear()},
        R"({"group":null})"
    );
}

/// @brief it should write and read a set field as its value.
TEST(Change, Set) {
    expect_round_trip(Holder{.group = Change<std::string>::to("")}, R"({"group":""})");
}

/// @brief it should merge a partial into a required struct.
TEST(Struct, RequiredMerge) {
    expect_round_trip(
        Holder{.settings = SettingsField::merge({.rate = Change<int>::to(5)})},
        R"({"settings":{"rate":5}})"
    );
}

/// @brief it should replace a required struct whole.
TEST(Struct, RequiredReplace) {
    expect_round_trip(
        Holder{.settings = SettingsField::replace({.rate = 1, .stream = 2})},
        R"({"settings":{"set":{"rate":1,"stream":2}}})"
    );
}

/// @brief it should leave out an optional struct with no change.
TEST(Struct, OptionalAbsent) {
    const Holder holder;
    EXPECT_TRUE(holder.extra.is_zero());
    expect_round_trip(holder, "{}");
}

/// @brief it should merge a partial into an optional struct.
TEST(Struct, OptionalMerge) {
    expect_round_trip(
        Holder{
            .extra = Change<SettingsField>::to(
                SettingsField::merge({.stream = Change<int>::to(3)})
            )
        },
        R"({"extra":{"stream":3}})"
    );
}

/// @brief it should replace an optional struct whole.
TEST(Struct, OptionalReplace) {
    expect_round_trip(
        Holder{.extra = Change<SettingsField>::to(SettingsField::replace({.rate = 4}))},
        R"({"extra":{"set":{"rate":4,"stream":0}}})"
    );
}

/// @brief it should clear an optional struct on null.
TEST(Struct, OptionalClear) {
    expect_round_trip(
        Holder{.extra = Change<SettingsField>::clear()},
        R"({"extra":null})"
    );
}

/// @brief it should bind an error for set beside another member.
TEST(Struct, SetBesideOtherMember) {
    x::json::Parser parser(
        x::json::json::parse(R"({"settings":{"set":{"rate":1},"rate":2}})")
    );
    Holder::parse(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_NE(
        parser.error_json().dump().find(
            "an object that holds set cannot hold other members"
        ),
        std::string::npos
    );
}

/// @brief it should bind an error for a struct that is not an object.
TEST(Struct, NotAnObject) {
    x::json::Parser parser(x::json::json::parse(R"({"settings":5})"));
    Holder::parse(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_NE(parser.error_json().dump().find("expected an object"), std::string::npos);
}

/// @brief it should keep an empty set apart from an absent one.
TEST(List, EmptySet) {
    const auto j = x::json::json::parse(R"({"set":[],"move":[{"key":"a"}]})");
    x::json::Parser parser(j);
    const auto list = List<Settings, std::string, SettingsUpdate>::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(list.set.has_value());
    EXPECT_TRUE(list.set->empty());
    EXPECT_FALSE(list.insert.has_value());
    ASSERT_TRUE(list.move.has_value());
    EXPECT_FALSE(list.move->at(0).after.has_value());
    EXPECT_EQ(list.to_json(), j);
}

/// @brief it should carry a partial and its guards.
TEST(Update, RoundTrip) {
    const auto j = x::json::json::parse(
        R"({"value":{"extra":null},"guards":[{"path":["config","type"],"value":"ai"}]})"
    );
    x::json::Parser parser(j);
    const auto update = Update<Holder>::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_TRUE(update.value.extra.cleared());
    ASSERT_EQ(update.guards.size(), 1);
    EXPECT_EQ(update.guards[0].value, "ai");
    EXPECT_EQ(update.to_json(), j);
}
}
