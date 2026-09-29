// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/can/types.gen.h"

#include "driver/can/backends/backends.h"
#include "driver/can/canlib/canlib.h"
#include "driver/can/gs_usb/gs_usb.h"
#include "driver/can/nixnet/nixnet.h"
#include "driver/can/pcan/pcan.h"
#include "driver/can/slcan/slcan.h"
#include "driver/can/socketcan/socketcan.h"

namespace driver::can::backends {
Backends load() {
    return {
        {synnax::can::BACKEND_SOCKETCAN, socketcan::load()},
        {synnax::can::BACKEND_PCAN, pcan::load()},
        {synnax::can::BACKEND_GS_USB, gs_usb::load()},
        {synnax::can::BACKEND_SLCAN, std::make_shared<slcan::Backend>()},
        {synnax::can::BACKEND_CANLIB, canlib::load()},
        {synnax::can::BACKEND_NIXNET, nixnet::load()},
    };
}
}
