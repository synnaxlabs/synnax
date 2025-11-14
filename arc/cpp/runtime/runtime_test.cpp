// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <chrono>
#include <thread>
#include <iomanip>

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "runtime.h"
#include "arc/cpp/module/module.h"

TEST(RuntimeTest, RuntimeStartStop) {
    auto parser = xjson::Parser::from_file_path("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/arc/cpp/runtime/pid.json");
    auto mod = arc::module::Module(parser);
    ASSERT_NIL(parser.error());

    std::cout << "WASM bytes size: " << mod.wasm.size() << std::endl;
    std::cout << "First 8 bytes: ";
    for (size_t i = 0; i < 8 && i < mod.wasm.size(); i++) {
        std::cout << std::hex << std::setw(2) << std::setfill('0') << (int)mod.wasm[i] << " ";
    }
    std::cout << std::dec << std::endl;

    arc::runtime::Config config{
        .mod = mod,
    };
    auto [loaded, err] = arc::runtime::load(config);
    ASSERT_NIL(err);
}