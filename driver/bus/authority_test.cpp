// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/bus/authority.h"

namespace driver::bus {
const x::control::Subject ARC{"arc", "arc-1"};
const x::control::Subject OPERATOR{"operator", "op-1"};

TEST(AuthorityMirrorTest, AuthorizedWhenNoState) {
    AuthorityMirror mirror;
    ASSERT_TRUE(mirror.is_authorized(1, ARC));
}

TEST(AuthorityMirrorTest, AcquireTransfer) {
    AuthorityMirror mirror;
    mirror.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    ASSERT_TRUE(mirror.is_authorized(1, ARC));
    ASSERT_FALSE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, ReleaseTransfer) {
    AuthorityMirror mirror;
    mirror.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    ASSERT_FALSE(mirror.is_authorized(1, OPERATOR));
    mirror.apply(
        {.transfers = {{
             .from = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
             .to = std::nullopt,
         }}}
    );
    ASSERT_TRUE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, HandoffTransfer) {
    AuthorityMirror mirror;
    mirror.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    mirror.apply(
        {.transfers = {{
             .from = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
             .to = x::control::State{
                 .resource = 1,
                 .subject = OPERATOR,
                 .authority = 250
             },
         }}}
    );
    ASSERT_FALSE(mirror.is_authorized(1, ARC));
    ASSERT_TRUE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, FilterKeepsAuthorized) {
    AuthorityMirror mirror;
    mirror.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = mirror.filter(frame, ARC);
    ASSERT_EQ(filtered.size(), 2);
}

TEST(AuthorityMirrorTest, FilterRemovesUnauthorized) {
    AuthorityMirror mirror;
    mirror.apply(
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
    auto filtered = mirror.filter(frame, ARC);
    ASSERT_EQ(filtered.size(), 1);
}

TEST(AuthorityMirrorTest, FilterAllUnauthorized) {
    AuthorityMirror mirror;
    mirror.apply(
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
    auto filtered = mirror.filter(frame, ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(AuthorityMirrorTest, MoveFilterAllPass) {
    AuthorityMirror mirror;
    mirror.apply(
        {.transfers = {{
             .from = std::nullopt,
             .to = x::control::State{.resource = 1, .subject = ARC, .authority = 200},
         }}}
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(99.0)));
    auto filtered = mirror.filter(std::move(frame), ARC);
    ASSERT_EQ(filtered.size(), 2);
    ASSERT_FLOAT_EQ(filtered.at<float>(1, 0), 42.0f);
    ASSERT_FLOAT_EQ(filtered.at<float>(2, 0), 99.0f);
}

TEST(AuthorityMirrorTest, MoveFilterPartialPass) {
    AuthorityMirror mirror;
    mirror.apply(
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
    auto filtered = mirror.filter(std::move(frame), ARC);
    ASSERT_EQ(filtered.size(), 1);
    ASSERT_FLOAT_EQ(filtered.at<float>(2, 0), 99.0f);
}

TEST(AuthorityMirrorTest, MoveFilterNonePass) {
    AuthorityMirror mirror;
    mirror.apply(
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
    auto filtered = mirror.filter(std::move(frame), ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(AuthorityMirrorTest, MoveFilterEmptyFrame) {
    AuthorityMirror mirror;
    x::telem::Frame frame;
    auto filtered = mirror.filter(std::move(frame), ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(AuthorityMirrorTest, ApplyIncreaseUpdatesState) {
    AuthorityMirror mirror;
    mirror.apply_increase(ARC, 1, 200);
    ASSERT_TRUE(mirror.is_authorized(1, ARC));
    ASSERT_FALSE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, ApplyIncreaseOverridesLowerAuthority) {
    AuthorityMirror mirror;
    mirror.apply_increase(ARC, 1, 100);
    mirror.apply_increase(OPERATOR, 1, 200);
    ASSERT_FALSE(mirror.is_authorized(1, ARC));
    ASSERT_TRUE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, ApplyIncreaseIgnoresEqualAuthority) {
    AuthorityMirror mirror;
    mirror.apply_increase(ARC, 1, 200);
    mirror.apply_increase(OPERATOR, 1, 200);
    ASSERT_TRUE(mirror.is_authorized(1, ARC));
    ASSERT_FALSE(mirror.is_authorized(1, OPERATOR));
}

TEST(AuthorityMirrorTest, ApplyIncreaseIgnoresLowerAuthority) {
    AuthorityMirror mirror;
    mirror.apply_increase(ARC, 1, 200);
    mirror.apply_increase(OPERATOR, 1, 100);
    ASSERT_TRUE(mirror.is_authorized(1, ARC));
    ASSERT_FALSE(mirror.is_authorized(1, OPERATOR));
}
}
