// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <system_error>

#include "driver/can/socketcan/socketcan.h"

namespace driver::can::socketcan {
x::errors::Error translate(const int code, const std::string &context) {
    const auto message = context + ": " + std::generic_category().message(code);
    switch (code) {
        // The kernel's CAN_RAW socket reports ENODEV once its interface unregisters,
        // ENXIO on a send after that, and ENETDOWN while the interface is down
        // (net/can/raw.c raw_notify and raw_sendmsg, net/can/af_can.c can_send).
        case ENODEV:
        case ENXIO:
        case ENETDOWN:
            return {UNREACHABLE_ERROR, message};
        // can_send returns ENOBUFS when the driver's transmit queue is full.
        case ENOBUFS:
        case EAGAIN:
#if EAGAIN != EWOULDBLOCK
        case EWOULDBLOCK:
#endif
            return {TEMPORARY_HARDWARE_ERROR, message};
        case EAFNOSUPPORT:
        case EPROTONOSUPPORT:
        case EPERM:
        case EACCES:
        case EINVAL:
        case EMSGSIZE:
            return {CONFIG_ERROR, message};
        default:
            return {CRITICAL_HARDWARE_ERROR, message};
    }
}
}
