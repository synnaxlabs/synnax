// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/control/state.h"

namespace driver::control {
const x::control::Subject ARC{"arc", "arc-1"};
const x::control::Subject OPERATOR{"operator", "op-1"};

TEST(StatesTest, AuthorizedWhenNoState) {
    States states;
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, AcquireTransfer) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ReleaseTransfer) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    states.apply(
        {.transfers = {{
             .from = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
             .to = std::nullopt,
         }}}
    );
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, HandoffTransfer) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    states.apply(
        {.transfers = {{
             .from = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    ASSERT_FALSE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, FilterKeepsAuthorized) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = states.filter(frame, ARC);
    ASSERT_EQ(filtered.size(), 2);
}

TEST(StatesTest, FilterRemovesUnauthorized) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = states.filter(frame, ARC);
    ASSERT_EQ(filtered.size(), 1);
}

TEST(StatesTest, FilterAllUnauthorized) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    auto filtered = states.filter(frame, ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(StatesTest, MoveFilterAllPass) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = states.filter(std::move(frame), ARC);
    ASSERT_EQ(filtered.size(), 2);
    ASSERT_FLOAT_EQ(filtered.at<float>(1, 0), 42.0f);
    ASSERT_FLOAT_EQ(filtered.at<float>(2, 0), 99.0f);
}

TEST(StatesTest, MoveFilterPartialPass) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = states.filter(std::move(frame), ARC);
    ASSERT_EQ(filtered.size(), 1);
    ASSERT_FLOAT_EQ(filtered.at<float>(2, 0), 99.0f);
}

TEST(StatesTest, MoveFilterNonePass) {
    States states;
    states.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    auto filtered = states.filter(std::move(frame), ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(StatesTest, MoveFilterEmptyFrame) {
    States states;
    x::telem::Frame frame;
    auto filtered = states.filter(std::move(frame), ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(StatesTest, ApplyIncreaseUpdatesState) {
    States states;
    states.apply_increase(ARC, 1, 200);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseOverridesLowerAuthority) {
    States states;
    states.apply_increase(ARC, 1, 100);
    states.apply_increase(OPERATOR, 1, 200);
    ASSERT_FALSE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseIgnoresEqualAuthority) {
    States states;
    states.apply_increase(ARC, 1, 200);
    states.apply_increase(OPERATOR, 1, 200);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseIgnoresLowerAuthority) {
    States states;
    states.apply_increase(ARC, 1, 200);
    states.apply_increase(OPERATOR, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, AllAuthorizedWhenNoState) {
    States states;
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_TRUE(states.all_authorized(frame, ARC));
}

TEST(StatesTest, AllAuthorizedWhenSubjectHoldsAll) {
    States states;
    states.apply_increase(ARC, 1, 200);
    states.apply_increase(ARC, 2, 200);
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_TRUE(states.all_authorized(frame, ARC));
}

TEST(StatesTest, AllAuthorizedFalseWhenOneUnauthorized) {
    States states;
    states.apply_increase(ARC, 1, 200);
    states.apply_increase(OPERATOR, 2, 200);
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_FALSE(states.all_authorized(frame, ARC));
}

TEST(StatesTest, ApplyFromSeriesAcquire) {
    States states;
    const std::string json =
        R"({"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]})";
    auto series = x::telem::Series(std::vector<std::string>{json}, x::telem::STRING_T);
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyFromSeriesRelease) {
    States states;
    states.apply_increase(ARC, 1, 200);
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    const std::string json =
        R"({"transfers":[{"from":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200},"to":null}]})";
    auto series = x::telem::Series(std::vector<std::string>{json}, x::telem::STRING_T);
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyFromSeriesIgnoresInvalidJson) {
    States states;
    auto series = x::telem::Series(
        std::vector<std::string>{"not valid json"}, x::telem::STRING_T
    );
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, ApplyFromSeriesIgnoresNonStringType) {
    States states;
    states.apply_increase(ARC, 1, 200);
    auto series = x::telem::Series(std::vector<float>{1.0f});
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, ApplyFromSeriesMultipleUpdates) {
    States states;
    const std::string json1 =
        R"({"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]})";
    const std::string json2 =
        R"({"transfers":[{"to":{"resource":2,"subject":{"key":"op-1","name":"operator"},"authority":250}}]})";
    auto series = x::telem::Series(
        std::vector<std::string>{json1, json2}, x::telem::STRING_T
    );
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(2, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(2, ARC));
}
}
