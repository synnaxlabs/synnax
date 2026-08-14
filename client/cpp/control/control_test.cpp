// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::control {
/// @brief it should return the state of a channel a writer holds.
TEST(ControlTests, testRetrieveHeld) {
    auto client = new_test_client();
    const auto ch = create_virtual_channel(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::framer::WriterConfig{
            std::vector{ch.key},
            x::telem::TimeStamp::now(),
            std::vector{x::control::AUTHORITY_ABSOLUTE},
            x::control::Subject{"philadelphia"},
        }
    ));
    const auto states = ASSERT_NIL_P(client.control.retrieve(std::vector{ch.key}));
    ASSERT_EQ(states.size(), 1);
    ASSERT_EQ(states[0].resource, ch.key);
    ASSERT_EQ(states[0].subject.name, "philadelphia");
    ASSERT_EQ(states[0].authority, x::control::AUTHORITY_ABSOLUTE);
    ASSERT_NIL(writer.close());
}

/// @brief it should omit a channel that no subject controls.
TEST(ControlTests, testRetrieveUnheld) {
    auto client = new_test_client();
    const auto ch = create_virtual_channel(client);
    const auto states = ASSERT_NIL_P(client.control.retrieve(std::vector{ch.key}));
    ASSERT_TRUE(states.empty());
}

/// @brief it should return the state of every controlled channel when no keys are
/// given.
TEST(ControlTests, testRetrieveAll) {
    auto client = new_test_client();
    const auto ch = create_virtual_channel(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::framer::WriterConfig{
            std::vector{ch.key},
            x::telem::TimeStamp::now(),
            std::vector{x::control::AUTHORITY_ABSOLUTE},
            x::control::Subject{"denver"},
        }
    ));
    const auto states = ASSERT_NIL_P(client.control.retrieve());
    bool found = false;
    for (const auto &state: states)
        if (state.resource == ch.key) found = true;
    ASSERT_TRUE(found);
    ASSERT_NIL(writer.close());
}
}
