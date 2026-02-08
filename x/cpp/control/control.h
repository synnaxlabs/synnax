// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "x/go/control/x/go/control/control.pb.h"

namespace x::control {
using Authority = std::uint8_t;
constexpr Authority AUTHORITY_ABSOLUTE = 255;

struct Subject {
    std::string name;
    std::string key;

    void to_proto(::control::ControlSubject *s) const {
        s->set_name(name);
        s->set_key(key);
    }
};
}
