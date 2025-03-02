// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>
#include "client/cpp/framer/framer.h"

/// @brief it should construct a frame with a pre-allocated size.
TEST(FramerTests, testConstructionFromSize) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    ASSERT_EQ(f.size(), 1);
}

/// @brief it should construct a frame from a single series and channel.
TEST(FramerTests, testConstructionFromSingleSeriesAndChannel) {
    const auto f = synnax::Frame(65537, telem::Series(std::vector<float>{1, 2, 3}, telem::FLOAT32_T));
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.channels->at(0), 65537);
    ASSERT_EQ(f.series->at(0).values<float>()[0], 1);
}

/// @brief it should construct a frame from a proto.
TEST(FramerTests, toProto) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    const auto p = new api::v1::Frame();
    f.to_proto(p);
    ASSERT_EQ(p->keys_size(), 1);
    ASSERT_EQ(p->series_size(), 1);
    const auto f2 = synnax::Frame(*p);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
}

/// @brief test ostream operator.
TEST(FramerTests, ostream) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    std::stringstream ss;
    ss << f;
    ASSERT_EQ(ss.str(),
              "Frame{\n 65537: Series(type: float32, size: 3, cap: 3, data: [1 2 3 ]), \n}");
}

TEST(FramerTests, testClear) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    f.clear();
    ASSERT_EQ(f.size(), 0);
    ASSERT_EQ(f.channels->size(), 0);
    ASSERT_EQ(f.series->size(), 0); 
}

TEST(FramerTests, testReserve) {
    const auto f = synnax::Frame(2);
    f.reserve(10);
    ASSERT_EQ(f.size(), 0);
    ASSERT_EQ(f.channels->size(), 0);
    ASSERT_EQ(f.series->size(), 0);
    f.emplace(65537, telem::Series(std::vector<float>{1, 2, 3}));
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.channels->size(), 1);
    ASSERT_EQ(f.series->size(), 1);
    f.reserve(10);
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.capacity(), 10);
    ASSERT_EQ(f.channels->capacity(), 10);
    ASSERT_EQ(f.series->capacity(), 10);

}

TEST(FramerTests, testDeepCopy) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    const auto f2 = f.deep_copy();
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
    f.emplace(65538, telem::Series(std::vector<float>{4, 5, 6}));
    ASSERT_EQ(f.size(), 2);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
}