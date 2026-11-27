// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <initializer_list>
#include <string>

#include "driver/serial/line.h"
#include "driver/transport/errors.h"

namespace driver::serial::line {
namespace {
bool one_of(
    const std::string &value,
    const std::initializer_list<const char *> options
) {
    return std::ranges::any_of(options, [&value](const char *o) { return value == o; });
}

x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(transport::CONFIG_ERROR, message);
}
}

x::errors::Error check(const synnax::serial::Properties &props) {
    if (props.baud_rate == 0) return invalid("baud rate must be greater than zero");
    if (props.data_bits < 5 || props.data_bits > 8)
        return invalid(
            "data bits must be 5, 6, 7, or 8, got " + std::to_string(props.data_bits)
        );
    if (!one_of(
            props.parity,
            {synnax::serial::PARITY_NONE_,
             synnax::serial::PARITY_EVEN_,
             synnax::serial::PARITY_ODD_,
             synnax::serial::PARITY_MARK_,
             synnax::serial::PARITY_SPACE_}
        ))
        return invalid("unknown parity '" + props.parity + "'");
    if (!one_of(
            props.stop_bits,
            {synnax::serial::STOP_BITS_ONE,
             synnax::serial::STOP_BITS_ONE_AND_HALF,
             synnax::serial::STOP_BITS_TWO}
        ))
        return invalid("unknown stop bits '" + props.stop_bits + "'");
    if (!one_of(
            props.flow_control,
            {synnax::serial::FLOW_CONTROL_NONE,
             synnax::serial::FLOW_CONTROL_SOFTWARE,
             synnax::serial::FLOW_CONTROL_HARDWARE}
        ))
        return invalid("unknown flow control '" + props.flow_control + "'");
    return x::errors::NIL;
}
}
