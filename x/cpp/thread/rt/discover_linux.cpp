// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <fstream>
#include <sstream>
#include <string>
#include <thread>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {
namespace {
/// @brief Parses a kernel CPU list format string (e.g. "2-3,5,7-9") into
/// individual core numbers.
std::vector<int> parse_cpu_list(const std::string &input) {
    std::vector<int> cores;
    std::istringstream stream(input);
    std::string token;
    while (std::getline(stream, token, ',')) {
        if (token.empty()) continue;
        const auto dash = token.find('-');
        if (dash == std::string::npos) {
            cores.push_back(std::stoi(token));
        } else {
            const int start = std::stoi(token.substr(0, dash));
            const int end = std::stoi(token.substr(dash + 1));
            for (int i = start; i <= end; i++) cores.push_back(i);
        }
    }
    return cores;
}
}

std::vector<int> discover_rt_cores() {
    std::ifstream file("/sys/devices/system/cpu/isolated");
    if (file.is_open()) {
        std::string content;
        std::getline(file, content);
        auto cores = parse_cpu_list(content);
        if (!cores.empty()) {
            std::sort(cores.begin(), cores.end());
            LOG(INFO) << "[rt.manager] discovered " << cores.size()
                      << " isolated cores from kernel";
            return cores;
        }
    }
    const auto hw = static_cast<int>(std::thread::hardware_concurrency());
    if (hw <= 1) return {};
    const int n = std::min(4, std::max(1, hw / 4));
    std::vector<int> cores;
    for (int i = hw - n; i < hw; i++) cores.push_back(i);
    LOG(INFO) << "[rt.manager] no isolated cores, using highest " << n
              << " of " << hw << " cores";
    return cores;
}
}
