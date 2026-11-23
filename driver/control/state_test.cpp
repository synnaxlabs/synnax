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
    states.set({{.subject = ARC, .resource = 1, .authority = 200}});
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

namespace {
/// @brief returns a frame of one sample on channel 1.
x::telem::Frame sample(const double v) {
    return x::telem::Frame(1, x::telem::Series(v));
}
}

/// @brief it should publish nothing while a hold lasts, then publish the held frames
/// of a subject with authority in order, and drop the held frames of one without it.
TEST(StatesTest, HoldPublishesHeldFramesAfterSet) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.hold();
    states.route(sample(1), ARC, publish);
    states.route(sample(2), OPERATOR, publish);
    states.route(sample(3), ARC, publish);
    ASSERT_TRUE(published.empty());
    states.set({{.subject = ARC, .resource = 1, .authority = 200}});
    ASSERT_EQ(published, (std::vector<double>{1, 3}));
    states.route(sample(4), ARC, publish);
    states.route(sample(5), OPERATOR, publish);
    ASSERT_EQ(published, (std::vector<double>{1, 3, 4}));
}

/// @brief it should publish the held frames filtered against the last mirror when the
/// read of a hold fails, and keep the bus open with that mirror.
TEST(StatesTest, ReleaseFiltersHeldAgainstTheLastMirror) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.set({{.subject = ARC, .resource = 1, .authority = 200}});
    states.hold();
    states.route(sample(1), ARC, publish);
    states.route(sample(2), OPERATOR, publish);
    ASSERT_TRUE(published.empty());
    states.release();
    ASSERT_EQ(published, (std::vector<double>{1}));
    states.route(sample(3), ARC, publish);
    states.route(sample(4), OPERATOR, publish);
    ASSERT_EQ(published, (std::vector<double>{1, 3}));
}

/// @brief it should drop the held frames when the read of a hold fails before any read
/// opened the bus, and keep the bus closed.
TEST(StatesTest, ReleaseBeforeTheFirstSetKeepsTheBusClosed) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.close();
    states.hold();
    states.route(sample(1), ARC, publish);
    states.release();
    states.route(sample(2), ARC, publish);
    ASSERT_TRUE(published.empty());
}

/// @brief it should ignore a read that started before a hold, so the read cannot end
/// that hold with a mirror from before the drop.
TEST(StatesTest, SetIgnoresAReadThatAHoldOutdated) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.set({});
    const auto stale = states.begin_read();
    states.hold();
    const auto current = states.begin_read();
    states.route(sample(1), ARC, publish);
    ASSERT_FALSE(states.set({}, stale));
    ASSERT_TRUE(published.empty());
    ASSERT_TRUE(states.set({}, current));
    ASSERT_EQ(published, (std::vector<double>{1}));
}

/// @brief it should mark the mirror stale when a failed read follows a set, until the
/// next set.
TEST(StatesTest, ReleaseAfterASetMarksTheMirrorStale) {
    States states;
    ASSERT_TRUE(states.set({}, states.begin_read()));
    ASSERT_FALSE(states.stale());
    states.hold();
    states.abandon(states.begin_read());
    states.release();
    ASSERT_TRUE(states.stale());
    states.set({});
    ASSERT_FALSE(states.stale());
}

/// @brief it should keep the mirror stale when a read that a hold outdated lands after
/// the release of that hold.
TEST(StatesTest, OutdatedSetKeepsTheMirrorStale) {
    States states;
    states.set({});
    const auto outdated = states.begin_read();
    states.hold();
    states.abandon(states.begin_read());
    states.release();
    ASSERT_FALSE(states.set({}, outdated));
    ASSERT_TRUE(states.stale());
}

/// @brief it should apply the updates that land while a read is open on top of its
/// snapshot, so a read is never ignored for a transfer and a transfer is never undone.
TEST(StatesTest, SetAppliesUpdatesThatLandDuringTheRead) {
    States states;
    const auto read = states.begin_read();
    states.apply(
        x::control::Update<synnax::channel::Key>{
            .transfers = {{
                .from = std::nullopt,
                .to = x::control::State<synnax::channel::Key>{
                    .subject = OPERATOR,
                    .resource = 1,
                    .authority = 200
                },
            }}
        }
    );
    ASSERT_TRUE(states.set(
        {{.subject = ARC, .resource = 1, .authority = 200},
         {.subject = ARC, .resource = 2, .authority = 200}},
        read
    ));
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
    ASSERT_TRUE(states.is_authorized(2, ARC));
}

/// @brief it should ignore a read that began before a stream restart and lands after
/// the read of the new run.
TEST(StatesTest, SetIgnoresAReadThatARestartOutdated) {
    States states;
    const auto old_read = states.begin_read();
    states.outdate_reads();
    const auto new_read = states.begin_read();
    ASSERT_TRUE(
        states.set({{.subject = OPERATOR, .resource = 1, .authority = 200}}, new_read)
    );
    ASSERT_FALSE(
        states.set({{.subject = ARC, .resource = 1, .authority = 200}}, old_read)
    );
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
}

/// @brief it should ignore a read that more than MAX_LOG updates landed during, so the
/// log of updates stays bounded.
TEST(StatesTest, SetIgnoresAReadPastTheLogCap) {
    States states;
    const auto read = states.begin_read();
    for (size_t i = 0; i <= States::MAX_LOG; i++)
        states.apply(
            x::control::Update<synnax::channel::Key>{
                .transfers = {{
                    .from = std::nullopt,
                    .to = x::control::State<synnax::channel::Key>{
                        .subject = OPERATOR,
                        .resource = static_cast<synnax::channel::Key>(i + 1),
                        .authority = 200
                    },
                }}
            }
        );
    ASSERT_FALSE(states.set({}, read));
    ASSERT_TRUE(states.set({}, states.begin_read()));
}

/// @brief it should publish nothing while closed at the start of a run, and open once
/// the first read sets the mirror.
TEST(StatesTest, ClosedUntilTheFirstSet) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.close();
    states.hold();
    states.route(sample(1), ARC, publish);
    ASSERT_TRUE(published.empty());
    states.set({{.subject = ARC, .resource = 1, .authority = 200}});
    ASSERT_TRUE(published.empty());
    states.route(sample(2), ARC, publish);
    ASSERT_EQ(published, (std::vector<double>{2}));
}

/// @brief it should publish the held frames filtered against the last mirror once a
/// hold reaches MAX_HELD, forward later frames with that mirror, and let the read
/// that lands later replace the mirror.
TEST(StatesTest, HoldReleasesAtTheCapWithTheLastMirror) {
    States states;
    std::vector<double> published;
    const Publish publish = [&](const x::telem::Frame &fr) {
        published.push_back(fr.at<double>(1, 0));
    };
    states.set({{.subject = ARC, .resource = 1, .authority = 200}});
    states.hold();
    const auto read = states.begin_read();
    for (size_t i = 0; i < States::MAX_HELD; i++)
        states.route(sample(static_cast<double>(i)), ARC, publish);
    ASSERT_TRUE(published.empty());
    // The frame past the cap releases the held frames, then goes through the mirror.
    states.route(sample(-1), OPERATOR, publish);
    ASSERT_EQ(published.size(), States::MAX_HELD);
    states.route(sample(-2), ARC, publish);
    ASSERT_EQ(published.size(), States::MAX_HELD + 1);
    for (size_t i = 0; i < States::MAX_HELD; i++)
        ASSERT_EQ(published[i], static_cast<double>(i));
    ASSERT_EQ(published.back(), -2);
    ASSERT_TRUE(
        states.set({{.subject = OPERATOR, .resource = 1, .authority = 200}}, read)
    );
    states.route(sample(-3), OPERATOR, publish);
    ASSERT_EQ(published.back(), -3);
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
    states.opened_on_core(ARC);
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

/// @brief it should hand a channel to the next local gate on disconnect when its holder
/// closed while the Core was connected.
TEST(StatesTest, DisconnectHandsOffAChannelAClosedWriterHeld) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.disconnect();
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
}

/// @brief it should hand a channel to the next local gate on disconnect when its holder
/// lowered its authority while the Core was connected.
TEST(StatesTest, DisconnectAppliesADecreaseMadeWhileConnected) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.disconnect();
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

/// @brief it should wait for the Core again after set marks it connected.
TEST(StatesTest, SetWaitsForTheCoreAgain) {
    States states;
    states.disconnect();
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.opened_on_core(ARC);
    states.set({x::control::State<synnax::channel::Key>{
        .subject = ARC,
        .resource = 1,
        .authority = 200,
    }});
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should keep the local rule after reconnect for a writer whose Core writer
/// failed and has not opened again, and wait for the Core once it has.
TEST(StatesTest, KeepsTheLocalRuleUntilAWriterReopens) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.disconnect();
    states.closed_on_core(ARC);
    states.set({});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    states.opened_on_core(OPERATOR);
    states.set_authority(OPERATOR, 1, 50);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    states.close_gate(OPERATOR);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

/// @brief it should hand a channel to the next local gate when a writer that has not
/// opened on the Core again closes after reconnect.
TEST(StatesTest, HandsOffWhenAWriterClosesBeforeItReopens) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.disconnect();
    states.closed_on_core(ARC);
    states.set({});
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

/// @brief it should wait for the Core's transfers after reconnect when a writer reopens
/// on the Core before the disconnect runs.
TEST(StatesTest, KeepsAWriterThatReopenedBeforeTheDisconnect) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.closed_on_core(ARC);
    states.opened_on_core(ARC);
    states.disconnect();
    states.set({x::control::State<synnax::channel::Key>{
        .subject = ARC,
        .resource = 1,
        .authority = 200,
    }});
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should keep waiting for the Core after reconnect for a writer still open
/// on the Core, even when the Core does not list it as a holder.
TEST(StatesTest, SetKeepsAnUnlistedWriterWaitingForTheCore) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.disconnect();
    states.set({});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.close_gate(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

namespace {
x::control::Update<synnax::channel::Key>
transfer_to(const x::control::Subject &subject, const x::control::Authority authority) {
    return {
        .transfers = {{
            .from = std::nullopt,
            .to = x::control::State<synnax::channel::Key>{
                .subject = subject,
                .resource = 1,
                .authority = authority,
            },
        }},
    };
}

x::control::Update<synnax::channel::Key> release_from(
    const x::control::Subject &subject,
    const x::control::Authority authority
) {
    return {
        .transfers = {{
            .from =
                x::control::State<synnax::channel::Key>{
                    .subject = subject,
                    .resource = 1,
                    .authority = authority,
                },
            .to = std::nullopt,
        }},
    };
}

/// @brief opens high at 200 and low at 100 on channel 1, both open on the Core, then
/// loses the Core and both Core writers, as an outage does.
void lose_the_core(
    States &states,
    const x::control::Subject &high,
    const x::control::Subject &low
) {
    states.opened_on_core(high);
    states.opened_on_core(low);
    states.open_gate(high, {1}, {200});
    states.open_gate(low, {1}, {100});
    states.disconnect();
    states.closed_on_core(high);
    states.closed_on_core(low);
}
}

/// @brief it should keep a channel with a higher-authority local writer that has not
/// reopened on the Core when the Core hands it to a lower one that has.
TEST(StatesTest, ReconnectFlipViaTransfer) {
    States states;
    lose_the_core(states, ARC, OPERATOR);
    states.set({});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.opened_on_core(OPERATOR);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.apply(transfer_to(OPERATOR, 100));
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
    states.opened_on_core(ARC);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.apply(transfer_to(ARC, 200));
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should keep a channel with a higher-authority local writer when the read
/// after reconnect names a lower one that reopened on the Core first.
TEST(StatesTest, ReconnectFlipViaSet) {
    States states;
    lose_the_core(states, ARC, OPERATOR);
    states.opened_on_core(OPERATOR);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.set({{.subject = OPERATOR, .resource = 1, .authority = 100}});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should give a channel that the Core releases to the local writers that
/// have not reopened on the Core, by the Core's rule, so only one of them writes it.
TEST(StatesTest, ReleaseLeavesTwoLocalGatesUncontrolled) {
    States states;
    const x::control::Subject console{.key = "console", .name = "console"};
    lose_the_core(states, ARC, OPERATOR);
    states.set({{.subject = console, .resource = 1, .authority = 250}});
    ASSERT_TRUE(states.is_authorized(1, console));
    ASSERT_FALSE(states.is_authorized(1, ARC));
    states.apply(release_from(console, 250));
    ASSERT_TRUE(states.is_authorized(1, ARC));
    ASSERT_FALSE(states.is_authorized(1, OPERATOR));
}

/// @brief it should follow the Core's holder in a tie between writers open on the Core,
/// even when another writer opened earlier locally.
TEST(StatesTest, TieOrderDivergesFromCore) {
    States states;
    const x::control::Subject third{.key = "third", .name = "third"};
    states.disconnect();
    states.open_gate(ARC, {1}, {100});
    states.open_gate(OPERATOR, {1}, {100});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.set({});
    states.opened_on_core(OPERATOR);
    states.apply(transfer_to(OPERATOR, 100));
    states.opened_on_core(ARC);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
    states.open_gate(third, {1}, {50});
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
    states.set_authority(third, 1, 60);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
}

/// @brief it should keep a tie with an earlier local writer that has not reopened on
/// the Core, then follow the Core once that writer reopens and the Core keeps its
/// holder.
TEST(StatesTest, FollowsTheCoreOnceAnEarlierTieReopens) {
    States states;
    states.disconnect();
    states.open_gate(ARC, {1}, {100});
    states.open_gate(OPERATOR, {1}, {100});
    states.set({});
    states.opened_on_core(OPERATOR);
    states.apply(transfer_to(OPERATOR, 100));
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.opened_on_core(ARC);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
}

/// @brief it should give a channel to a local writer that has not reopened on the Core
/// when it has more authority than the Core's holder outside the Driver, and leave a
/// tie with that holder.
TEST(StatesTest, LocalWriterTakesAChannelFromAWeakerRemoteHolder) {
    States states;
    const x::control::Subject console{.key = "console", .name = "console"};
    states.open_gate(ARC, {1}, {200});
    states.set({{.subject = console, .resource = 1, .authority = 150}});
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.set({{.subject = console, .resource = 1, .authority = 200}});
    ASSERT_TRUE(states.is_authorized(1, console));
}

/// @brief it should apply the local rule to a writer's channels once its Core writer
/// closes, so a decrease it made takes effect.
TEST(StatesTest, ClosedOnCoreAppliesTheLocalRule) {
    States states;
    states.opened_on_core(ARC);
    states.open_gate(ARC, {1}, {200});
    states.open_gate(OPERATOR, {1}, {150});
    states.set_authority(ARC, 1, 100);
    ASSERT_TRUE(states.is_authorized(1, ARC));
    states.closed_on_core(ARC);
    ASSERT_TRUE(states.is_authorized(1, OPERATOR));
    ASSERT_FALSE(states.is_authorized(1, ARC));
}

/// @brief it should call the on_connect of each open gate on each set, and never after
/// the gate closes.
TEST(StatesTest, SetCallsOnConnectOfOpenGates) {
    States states;
    size_t calls = 0;
    states.open_gate(ARC, {1}, {200}, [&calls] { calls++; });
    states.disconnect();
    states.set({});
    ASSERT_EQ(calls, 1);
    states.set({});
    ASSERT_EQ(calls, 2);
    states.close_gate(ARC);
    states.set({});
    ASSERT_EQ(calls, 2);
}
}
