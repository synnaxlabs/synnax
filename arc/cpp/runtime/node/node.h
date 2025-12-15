// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"

namespace arc::runtime::node {
struct Context {
    telem::TimeSpan elapsed;
    std::function<void(const std::string &output_param)> mark_changed;
    std::function<void(const xerrors::Error &)> report_error;
    std::function<void(const std::string &stage_key)> activate;
};
class Node {
public:
    virtual ~Node() = default;

    virtual xerrors::Error next(Context &ctx) = 0;
};
}
