// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/can/pcan/prod.h"

namespace driver::can::pcan {
vendor::Library library() {
    return {
        .names = {"libPCBUSB.dylib"},
        .info = {"MacCAN PCBUSB", "https://www.mac-can.com"},
    };
}
}
