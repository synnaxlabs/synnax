// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/icd/retrieve.h"

namespace driver::icd {
namespace {
/// @brief creates an ICD with no messages.
synnax::icd::ICD create_icd(const synnax::Synnax &client) {
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "Plant";
    EXPECT_NIL(client.icds.create(ic));
    return ic;
}

/// @brief creates a virtual float64 channel.
synnax::channel::Channel create_channel(const synnax::Synnax &client) {
    synnax::channel::Channel ch{
        .name = make_unique_channel_name("cmd"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true,
    };
    EXPECT_NIL(client.channels.create(ch));
    return ch;
}
}

/// @brief it should retrieve the ICDs and channels with the given keys.
TEST(Retrieve, RetrieveICDsAndChannels) {
    const auto client = new_test_client();
    const auto ic = create_icd(client);
    const auto ch = create_channel(client);
    const auto r = ASSERT_NIL_P(retrieve(client, {ic.key}, {ch.key}));
    ASSERT_EQ(r.icds.size(), 1);
    EXPECT_EQ(r.icds[0].key, ic.key);
    ASSERT_EQ(r.channels.size(), 1);
    EXPECT_EQ(r.channels.at(ch.key).name, ch.name);
}

/// @brief it should skip a key with no ICD or channel.
TEST(Retrieve, SkipMissingKeys) {
    const auto client = new_test_client();
    const auto ic = create_icd(client);
    const auto ch = create_channel(client);
    const auto r = ASSERT_NIL_P(
        retrieve(client, {ic.key, x::uuid::create()}, {ch.key, ch.key + 1000000})
    );
    ASSERT_EQ(r.icds.size(), 1);
    EXPECT_EQ(r.icds[0].key, ic.key);
    ASSERT_EQ(r.channels.size(), 1);
    EXPECT_TRUE(r.channels.contains(ch.key));
}

/// @brief it should retrieve nothing for no keys.
TEST(Retrieve, RetrieveNothingForNoKeys) {
    const auto client = new_test_client();
    create_icd(client);
    create_channel(client);
    const auto r = ASSERT_NIL_P(retrieve(client, {}, {}));
    EXPECT_TRUE(r.icds.empty());
    EXPECT_TRUE(r.channels.empty());
}
}
