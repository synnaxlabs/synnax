// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

// std
#include <string>

// external
#include "x/cpp/xjson/xjson.h"

struct State {
    std::string key;
    std::string variant;
    json details;

    json to_json() const {
        json j;
        j["key"] = this->key;
        j["variant"] = this->variant;
        j["details"] = this->details;
        return j;
    }
};