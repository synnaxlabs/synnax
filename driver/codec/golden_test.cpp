// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <fstream>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/matcher.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;
using json = nlohmann::json;

/// @brief Golden is one message from vectors.json, with its layout rebuilt as a library
/// entry.
struct Golden {
    library::MessageEntry message;
    std::vector<std::string> names;
    json signals;
    json vectors;
};

std::vector<std::uint8_t> from_hex(const std::string &hex) {
    std::vector<std::uint8_t> out;
    for (std::size_t i = 0; i < hex.size(); i += 2)
        out.push_back(
            static_cast<std::uint8_t>(std::stoul(hex.substr(i, 2), nullptr, 16))
        );
    return out;
}

std::vector<Golden> load() {
    std::ifstream file("driver/codec/testdata/vectors.json");
    const auto root = json::parse(file);
    std::vector<Golden> out;
    for (const auto &m: root["messages"]) {
        Golden g;
        g.message.key = x::uuid::create();
        g.message.name = m["name"];
        g.message.length = m["length"].get<std::uint16_t>();
        g.message.identifier = library::CanIdentifier{
            .id = m["id"],
            .extended = m["extended"],
        };
        std::unordered_map<std::string, library::FieldKey> keys;
        for (const auto &s: m["signals"])
            keys[s["name"]] = x::uuid::create();
        for (const auto &s: m["signals"]) {
            library::BinaryField f;
            f.key = keys[s["name"]];
            f.name = s["name"];
            f.start_bit = s["start_bit"];
            f.bit_length = s["bit_length"];
            f.byte_order = s["byte_order"];
            f.signed_ = s["signed"];
            f.float_ = s["float"];
            f.scale = s["scale"];
            f.offset = s["offset"];
            if (!s["multiplexor"].is_null()) {
                f.multiplexor = keys[s["multiplexor"]];
                using V = decltype(f.multiplex_values)::value_type;
                for (const auto &v: s["multiplex_values"])
                    f.multiplex_values.push_back(v.get<V>());
            }
            g.names.push_back(f.name);
            g.message.fields.emplace_back(std::move(f));
        }
        g.signals = m["signals"];
        g.vectors = m["vectors"];
        out.push_back(std::move(g));
    }
    return out;
}

bool exact(const json &signal) {
    return !signal["float"].get<bool>() && signal["scale"] == 1 &&
           signal["offset"] == 0;
}

class GoldenVectors : public ::testing::Test {
protected:
    std::vector<Golden> goldens = load();
};
}

TEST_F(GoldenVectors, CoverEveryMessageInTheDBC) {
    ASSERT_EQ(this->goldens.size(), 13);
    for (const auto &g: this->goldens)
        ASSERT_EQ(g.vectors.size(), 64) << g.message.name;
}

TEST_F(GoldenVectors, DecodeToTheValuesCantoolsDecodes) {
    for (const auto &g: this->goldens) {
        const auto plan = ASSERT_NIL_P(Plan::compile(g.message));
        auto values = plan.values();
        for (const auto &v: g.vectors) {
            const auto payload = from_hex(v["payload"]);
            ASSERT_NIL(plan.decode(payload, values));
            for (std::size_t i = 0; i < g.names.size(); i++) {
                const auto &name = g.names[i];
                const auto &expected = v["values"];
                const auto ctx = g.message.name + "." + name + " " +
                                 v["payload"].get<std::string>();
                if (!expected.contains(name)) {
                    EXPECT_FALSE(values.present(i)) << ctx;
                    continue;
                }
                ASSERT_TRUE(values.present(i)) << ctx;
                const auto &e = expected[name];
                if (!exact(g.signals[i])) {
                    EXPECT_DOUBLE_EQ(values.get(i), e.get<double>()) << ctx;
                    continue;
                }
                if (g.signals[i]["signed"].get<bool>()) {
                    x::telem::Series s(x::telem::INT64_T, 1);
                    ASSERT_EQ(values.write(i, s), 1);
                    EXPECT_EQ(s.at<std::int64_t>(0), e.get<std::int64_t>()) << ctx;
                } else {
                    x::telem::Series s(x::telem::UINT64_T, 1);
                    ASSERT_EQ(values.write(i, s), 1);
                    EXPECT_EQ(s.at<std::uint64_t>(0), e.get<std::uint64_t>()) << ctx;
                }
            }
        }
    }
}

TEST_F(GoldenVectors, EncodeToThePayloadCantoolsEncodes) {
    for (const auto &g: this->goldens) {
        const auto plan = ASSERT_NIL_P(Plan::compile(g.message));
        auto values = plan.values();
        std::vector<std::uint8_t> payload;
        for (const auto &v: g.vectors) {
            values.clear();
            for (std::size_t i = 0; i < g.names.size(); i++) {
                const auto &expected = v["values"];
                if (!expected.contains(g.names[i])) continue;
                const auto &e = expected[g.names[i]];
                if (!exact(g.signals[i]))
                    values.set(i, e.get<double>());
                else if (g.signals[i]["signed"].get<bool>())
                    values.set(i, e.get<std::int64_t>());
                else
                    values.set(i, e.get<std::uint64_t>());
            }
            payload.assign(plan.length(), 0);
            ASSERT_NIL(plan.encode(values, payload));
            EXPECT_EQ(payload, from_hex(v["encoded"]))
                << g.message.name << " " << v["payload"].get<std::string>();
        }
    }
}

TEST_F(GoldenVectors, MatchEachMessageByItsCANIdentifier) {
    std::vector<library::MessageEntry> messages;
    for (const auto &g: this->goldens)
        messages.push_back(g.message);
    const auto matcher = ASSERT_NIL_P(Matcher::compile(messages));
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &id = std::get<library::CanIdentifier>(*messages[i].identifier);
        EXPECT_EQ(matcher.match(id.id, id.extended), i);
        EXPECT_EQ(matcher.match(id.id, !id.extended), std::nullopt);
    }
}
}
