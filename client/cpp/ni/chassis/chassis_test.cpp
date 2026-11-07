// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/ni/chassis/json.gen.h"

namespace synnax::ni::chassis {
/// @brief it should keep the field group of an update through JSON.
TEST(Update, RoundTripsGroup) {
    ConfigurationUpdate configuration;
    configuration.name = "PXIe-1071";
    const auto j = to_json(Update(configuration));
    ASSERT_EQ(j["group"], "configuration");
    x::json::Parser parser(j);
    const auto parsed = parse_update(parser);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(std::get<ConfigurationUpdate>(parsed).name, "PXIe-1071");
}
}
