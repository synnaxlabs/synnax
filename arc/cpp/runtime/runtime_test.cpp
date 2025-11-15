// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <chrono>
#include <fstream>
#include <iomanip>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/module/module.h"
#include "runtime.h"

TEST(RuntimeTest, RuntimeStartStop) {
    auto parser = xjson::Parser::from_file_path(
        "/Users/emilianobonilla/Desktop/synnaxlabs/synnax/arc/cpp/runtime/pid.json"
    );
    auto mod = arc::module::Module(parser);
    ASSERT_NIL(parser.error());

    std::cout << "WASM bytes size: " << mod.wasm.size() << std::endl;
    std::cout << "First 8 bytes: ";
    for (size_t i = 0; i < 8 && i < mod.wasm.size(); i++) {
        std::cout << std::hex << std::setw(2) << std::setfill('0') << (int) mod.wasm[i]
                  << " ";
    }
    std::cout << std::dec << std::endl;

    arc::runtime::Config config{
        .mod = mod,
    };
    auto [runtime, err] = arc::runtime::load(config);
    ASSERT_NIL(err);
    auto results = runtime->run(1000 * 2);

    // Write results to CSV file
    std::ofstream csv_file("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/arc/cpp/runtime/runtime_results.csv");
    csv_file << "iteration,elapsed_ns\n";
    for (size_t i = 0; i < results.size(); i++) {
        csv_file << i << "," << results[i].nanoseconds() << "\n";
    }
    csv_file.close();
    std::cout << "Results written to runtime_results.csv" << std::endl;
}
