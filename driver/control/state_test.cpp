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
const x::control::Subject ARC{.key = "arc-1", .name = "arc"};
const x::control::Subject OPERATOR{.key = "op-1", .name = "operator"};

TEST(StatesTest, AuthorizedWhenNoState) {
    States states;
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, AcquireTransfer) {
    States states;
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = ARC,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
    );
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ReleaseTransfer) {
    States states;
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = ARC,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
    );
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = x::control::State<
                    synnax::channel::
                        Key>{.subject = ARC, .resource = 1, .authority = 200},
                .to = std::nullopt,
            }}
        }
    );
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, HandoffTransfer) {
    States states;
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = ARC,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
    );
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = x::control::State<
                    synnax::channel::
                        Key>{.subject = ARC, .resource = 1, .authority = 200},
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 250
                },
            }}
        }
    );
    ASSERT_FALSE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, FilterKeepsAuthorized) {
    States states;
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = ARC,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
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
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 250
                },
            }}
        }
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
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 250
                },
            }}
        }
    );
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    auto filtered = states.filter(frame, ARC);
    ASSERT_TRUE(filtered.empty());
}

TEST(StatesTest, MoveFilterAllPass) {
    States states;
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = ARC,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
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
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 250
                },
            }}
        }
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
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 250
                },
            }}
        }
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
    states.set_authority(ARC, 1, 200);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseOverridesLowerAuthority) {
    States states;
    states.set_authority(ARC, 1, 100);
    states.set_authority(OPERATOR, 1, 200);
    ASSERT_FALSE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseIgnoresEqualAuthority) {
    States states;
    states.set_authority(ARC, 1, 200);
    states.set_authority(OPERATOR, 1, 200);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyIncreaseIgnoresLowerAuthority) {
    States states;
    states.set_authority(ARC, 1, 200);
    states.set_authority(OPERATOR, 1, 100);
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
    states.set_authority(ARC, 1, 200);
    states.set_authority(ARC, 2, 200);
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_TRUE(states.all_authorized(frame, ARC));
}

TEST(StatesTest, AllAuthorizedFalseWhenOneUnauthorized) {
    States states;
    states.set_authority(ARC, 1, 200);
    states.set_authority(OPERATOR, 2, 200);
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_FALSE(states.all_authorized(frame, ARC));
}

TEST(StatesTest, ApplyFromSeriesAcquire) {
    States states;
    const std::string json =
        R"({"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]})";
    auto series = x::telem::Series(std::vector<std::string>{json}, x::telem::JSON_T);
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyFromSeriesRelease) {
    States states;
    states.set_authority(ARC, 1, 200);
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    const std::string json =
        R"({"transfers":[{"from":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200},"to":null}]})";
    auto series = x::telem::Series(std::vector<std::string>{json}, x::telem::JSON_T);
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyFromSeriesIgnoresInvalidJson) {
    States states;
    auto series = x::telem::Series(
        std::vector<std::string>{"not valid json"},
        x::telem::JSON_T
    );
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, ApplyFromSeriesIgnoresNonJSONType) {
    States states;
    states.set_authority(ARC, 1, 200);
    auto series = x::telem::Series(std::vector<float>{1.0f});
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

TEST(StatesTest, SetReplacesTheMirror) {
    States states;
    states.set({
        x::control::State<synnax::channel::Key>{
            .subject = ARC,
            .resource = 1,
            .authority = 200,
        },
    });
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    states.set({
        x::control::State<synnax::channel::Key>{
            .subject = OPERATOR,
            .resource = 2,
            .authority = 200,
        },
    });
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(2, ARC));
    ASSERT_TRUE(states.is_authorized(2, OPERATOR));
}

TEST(StatesTest, SetWithNoStatesClearsTheMirror) {
    States states;
    states.set({
        x::control::State<synnax::channel::Key>{
            .subject = ARC,
            .resource = 1,
            .authority = 200,
        },
    });
    states.set({});
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

TEST(StatesTest, ApplyFromSeriesMultipleUpdates) {
    States states;
    const std::string json1 =
        R"({"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]})";
    const std::string json2 =
        R"({"transfers":[{"to":{"resource":2,"subject":{"key":"op-1","name":"operator"},"authority":250}}]})";
    auto series = x::telem::Series(
        std::vector<std::string>{json1, json2},
        x::telem::JSON_T
    );
    states.apply(series);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(2, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(2, ARC));
}

/// @brief it should give a tie to the earlier gate, as the Core does.
TEST(StatesTest, OpenGateGivesATieToTheEarlierGate) {
    States states;
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {200});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should open a gate with absolute authority when it names none.
TEST(StatesTest, OpenGateDefaultsToAbsoluteAuthority) {
    States states;
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {});
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

/// @brief it should hand the channel to the next gate by authority, then by order,
/// when the holder closes after disconnect.
TEST(StatesTest, CloseGateHandsOffToTheNextGate) {
    States states;
    states.disconnect();
    const x::control::Subject third{"third", "third-1"};
    states.open_gate(OPERATOR, {1}, {100});
    states.open_gate(third, {1}, {100});
    states.open_gate(ARC, {1}, {200});
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    states.close_gate(OPERATOR);
    ASSERT_TRUE(states.is_authorized(1, third));
    states.close_gate(third);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

/// @brief it should keep the holder when a gate that does not hold the channel closes.
TEST(StatesTest, CloseGateKeepsAnotherHolder) {
    States states;
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {100});
    states.close_gate(OPERATOR);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should keep a channel with its holder when the holder lowers its
/// authority and no other gate writes it.
TEST(StatesTest, SetAuthorityDecreaseKeepsALoneHolder) {
    States states;
    states.open_gate(ARC, {1}, {200});
    states.set_authority(ARC, 1, 50);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should give a channel the Core reports as uncontrolled to its local gates.
TEST(StatesTest, SetFallsBackToLocalGates) {
    States states;
    states.open_gate(ARC, {1, 2}, {200});
    states.set({x::control::State<synnax::channel::Key>{
        .subject = OPERATOR,
        .resource = 2,
        .authority = 250,
    }});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(2, OPERATOR));
    ASSERT_FALSE(states.is_authorized(2, ARC));
}

/// @brief it should give the channel back to an earlier gate that raises its authority
/// to a tie after disconnect, as the Core does.
TEST(StatesTest, EarlierGateTakesBackATieAfterDisconnect) {
    States states;
    states.disconnect();
    states.open_gate(ARC, {1}, {150});
    states.open_gate(OPERATOR, {1}, {150});
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    states.set_authority(ARC, 1, 150);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

/// @brief it should keep the holder while the Core is connected, so the Core decides
/// who takes the channel next.
TEST(StatesTest, WaitsForTheCoreWhileConnected) {
    States states;
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {100});
    states.set_authority(ARC, 1, 50);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}

/// @brief it should leave a holder that the Core reported in place after disconnect,
/// unless a local gate has more authority.
TEST(StatesTest, KeepsARemoteHolderAfterDisconnect) {
    States states;
    states.set({x::control::State<synnax::channel::Key>{
        .subject = OPERATOR,
        .resource = 1,
        .authority = 200,
    }});
    states.disconnect();
    states.open_gate(ARC, {1}, {150});
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    states.set_authority(ARC, 1, 250);
    ASSERT_TRUE(states.is_authorized(1, ARC));
}
}
