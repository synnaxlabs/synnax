// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <bitset>

#include "driver/errors/errors.h"
#include "driver/mil1553/backend.h"

namespace driver::mil1553 {
x::errors::Error validate(const synnax::mil1553::Properties &props) {
    const bool rt = props.role == synnax::mil1553::ROLE_REMOTE_TERMINAL;
    if (!rt && props.role != synnax::mil1553::ROLE_BUS_CONTROLLER &&
        props.role != synnax::mil1553::ROLE_MONITOR)
        return x::errors::Error(
            errors::CONFIGURATION_ERROR,
            "unknown MIL-STD-1553 role " + props.role
        );
    if (!rt) {
        if (props.terminals.empty()) return x::errors::NIL;
        return x::errors::Error(
            errors::CONFIGURATION_ERROR,
            "only a remote terminal channel owns terminals"
        );
    }
    if (props.terminals.empty())
        return x::errors::Error(
            errors::CONFIGURATION_ERROR,
            "a remote terminal channel must own at least one terminal"
        );
    std::bitset<codec::mil1553::MAX_RT + 1> seen;
    for (const auto t: props.terminals) {
        if (t > codec::mil1553::MAX_RT)
            return x::errors::Error(
                errors::CONFIGURATION_ERROR,
                "terminal " + std::to_string(t) + " is not from 0 to 30"
            );
        if (seen.test(t))
            return x::errors::Error(
                errors::CONFIGURATION_ERROR,
                "terminal " + std::to_string(t) + " is listed twice"
            );
        seen.set(t);
    }
    return x::errors::NIL;
}
}
