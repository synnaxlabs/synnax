// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/module/module.h"

TEST(RuntimeTest, RuntimeStartStop) {
    auto parser = xjson::Parser::from_file_path("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/arc/cpp/runtime/pid.json");
    auto mod = arc::module::Module(parser);
    ASSERT_NIL(parser.error());
}