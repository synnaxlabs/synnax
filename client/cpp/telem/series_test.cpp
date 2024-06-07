// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>
#include "client/cpp/synnax.h"
#include "x/go/telem/x/go/telem/telem.pb.h"
#include <iostream>

///// @brief create basic int series.
TEST(TestSeries, testConstruction)
{
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const synnax::Series s{vals};
    ASSERT_EQ(s.data_type, synnax::UINT8);
    const auto v = s.uint8();
    ASSERT_EQ(v.size(), vals.size());
    for (size_t i = 0; i < vals.size(); i++) ASSERT_EQ(v[i], vals[i]);
}

//// @brief it should correctly initialize and parse a string series.
TEST(TestSeries, testString)
{
    const std::vector<std::string> vals = {"hello", "world"};
    const Series s{vals};
    ASSERT_EQ(s.data_type, synnax::STRING);
    const auto v = s.string();
    for (size_t i = 0; i < vals.size(); i++) ASSERT_EQ(v[i], vals[i]);
}

//// @brief it should correctly serialize and deserialize the series from protoubuf.
TEST(TestSeries, testProto)
{
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const Series s{vals};
    const auto s2 = new telem::PBSeries();
    s.to_proto(s2);
    const Series s3{*s2};
    const auto v = s3.uint8();
    for (size_t i = 0; i < vals.size(); i++) ASSERT_EQ(v[i], vals[i]);
    delete s2;
}

TEST(TestSeries, testAt) {
    const std::vector<uint8_t> vals = {1, 2, 3, 4, 5};
    const Series s{vals};
    ASSERT_EQ(s.at<uint8_t>(0), 1);
    ASSERT_EQ(s.at<uint8_t>(1), 2);
    ASSERT_EQ(s.at<uint8_t>(2), 3);
    ASSERT_EQ(s.at<uint8_t>(3), 4);
    ASSERT_EQ(s.at<uint8_t>(4), 5);
}