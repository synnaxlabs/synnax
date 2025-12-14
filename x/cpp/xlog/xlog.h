// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "glog/logging.h"

namespace xlog {
inline std::string get_color(const std::string &color) {
    return FLAGS_colorlogtostderr ? color : "";
}

inline std::string red() {
    return get_color("\033[1;31m");
}
inline std::string green() {
    return get_color("\033[1;32m");
}
inline std::string reset() {
    return get_color("\033[0m");
}
inline std::string blue() {
    return get_color("\033[1;34m");
}
inline std::string shale() {
    return get_color("\033[1;38;2;112;128;144m");
}

inline std::string bool_to_str(const bool b) {
    return b ? "true" : "false";
}

inline std::string sensitive_string(const std::string &s) {
    return std::string(s.length(), '*');
}
}
