// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/control/control.h"

namespace x::control {
TEST(SubjectTest, Equality) {
    Subject a{"a", "k1"};
    Subject b{"b", "k1"};
    Subject c{"a", "k2"};
    ASSERT_EQ(a, b);
    ASSERT_NE(a, c);
}

TEST(StateTest, Parse) {
    const std::string json =
        R"({"resource": 42, "subject": {"key": "writer-1", "name": "Writer"}, "authority": 200})";
    x::json::Parser parser(json);
    auto state = State::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(state.resource, 42);
    ASSERT_EQ(state.subject.key, "writer-1");
    ASSERT_EQ(state.subject.name, "Writer");
    ASSERT_EQ(state.authority, 200);
}

TEST(TransferTest, ParseAcquire) {
    const std::string json = R"({
        "from": null,
        "to": {
            "resource": 10,
            "subject": {"key": "s1", "name": "Sub1"},
            "authority": 255
        }
    })";
    x::json::Parser parser(json);
    auto transfer = Transfer::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_FALSE(transfer.from.has_value());
    ASSERT_TRUE(transfer.to.has_value());
    ASSERT_EQ(transfer.to->resource, 10);
    ASSERT_EQ(transfer.to->authority, 255);
}

TEST(TransferTest, ParseRelease) {
    const std::string json = R"({
        "from": {
            "resource": 10,
            "subject": {"key": "s1", "name": "Sub1"},
            "authority": 200
        },
        "to": null
    })";
    x::json::Parser parser(json);
    auto transfer = Transfer::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(transfer.from.has_value());
    ASSERT_EQ(transfer.from->resource, 10);
    ASSERT_FALSE(transfer.to.has_value());
}

TEST(TransferTest, ParseHandoff) {
    const std::string json = R"({
        "from": {
            "resource": 10,
            "subject": {"key": "s1", "name": "Old"},
            "authority": 200
        },
        "to": {
            "resource": 10,
            "subject": {"key": "s2", "name": "New"},
            "authority": 250
        }
    })";
    x::json::Parser parser(json);
    auto transfer = Transfer::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(transfer.from.has_value());
    ASSERT_TRUE(transfer.to.has_value());
    ASSERT_EQ(transfer.from->subject.key, "s1");
    ASSERT_EQ(transfer.to->subject.key, "s2");
}

TEST(UpdateTest, ParseMultipleTransfers) {
    const std::string json = R"({
        "transfers": [
            {
                "from": null,
                "to": {
                    "resource": 1,
                    "subject": {"key": "w1", "name": "Writer1"},
                    "authority": 200
                }
            },
            {
                "from": {
                    "resource": 2,
                    "subject": {"key": "w2", "name": "Writer2"},
                    "authority": 100
                },
                "to": null
            }
        ]
    })";
    x::json::Parser parser(json);
    auto update = Update::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_EQ(update.transfers.size(), 2);
    ASSERT_FALSE(update.transfers[0].from.has_value());
    ASSERT_TRUE(update.transfers[0].to.has_value());
    ASSERT_EQ(update.transfers[0].to->resource, 1);
    ASSERT_TRUE(update.transfers[1].from.has_value());
    ASSERT_FALSE(update.transfers[1].to.has_value());
}

TEST(UpdateTest, ParseEmptyTransfers) {
    const std::string json = R"({"transfers": []})";
    x::json::Parser parser(json);
    auto update = Update::parse(parser);
    ASSERT_TRUE(parser.ok()) << parser.error_json().dump();
    ASSERT_TRUE(update.transfers.empty());
}
}
