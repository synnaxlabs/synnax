// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>

#include "driver/serial/line.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
const std::array<tcflag_t, 4> SIZES{CS5, CS6, CS7, CS8};
}

namespace line {
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings) {
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE_AND_HALF)
        return {transport::CONFIG_ERROR, "Linux does not support 1.5 stop bits"};
    settings.c_cflag &= ~(CBAUD | (CBAUD << IBSHIFT));
    settings.c_cflag |= BOTHER | (BOTHER << IBSHIFT);
    settings.c_ispeed = props.baud_rate;
    settings.c_ospeed = props.baud_rate;
    settings.c_cflag &= ~CSIZE;
    settings.c_cflag |= SIZES.at(props.data_bits - 5);
    settings.c_cflag &= ~(PARENB | PARODD | CMSPAR);
    const auto &parity = props.parity;
    if (parity != synnax::serial::PARITY_NONE_) settings.c_cflag |= PARENB;
    if (parity == synnax::serial::PARITY_ODD_ || parity == synnax::serial::PARITY_MARK_)
        settings.c_cflag |= PARODD;
    if (parity == synnax::serial::PARITY_MARK_ ||
        parity == synnax::serial::PARITY_SPACE_)
        settings.c_cflag |= CMSPAR;
    if (props.stop_bits == synnax::serial::STOP_BITS_TWO)
        settings.c_cflag |= CSTOPB;
    else
        settings.c_cflag &= ~CSTOPB;
    settings.c_iflag &= ~(IXON | IXOFF);
    settings.c_cflag &= ~CRTSCTS;
    if (props.flow_control == synnax::serial::FLOW_CONTROL_SOFTWARE)
        settings.c_iflag |= IXON | IXOFF;
    if (props.flow_control == synnax::serial::FLOW_CONTROL_HARDWARE)
        settings.c_cflag |= CRTSCTS;
    return x::errors::NIL;
}
}
}
