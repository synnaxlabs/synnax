// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "meminfo.h"
#include <iostream>
#include <fstream>
#include <string>

std::uint32_t meminfo::getUsage() {
    std::ifstream status("/proc/self/status", std::ios_base::in);
    std::string line;
    std::string key;
    std::uint32_t memory = 0;

    while (std::getline(status, line)) {
        if (line.substr(0, 6) == "VmRSS:") {
            std::istringstream iss(line);
            iss >> key >> memory; // "VmRSS" and the memory value
            return memory * 1024; // Convert from kB to Bytes
        }
    }

    std::cerr << "Failed to read memory usage information.\n";
    return 0;
}
