// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/can/gs_usb/prod.h"

namespace driver::can::gs_usb {
vendor::Library library() {
    return {
        .names = {"libusb-1.0.dll"},
        .info = {"libusb", "https://libusb.info"},
    };
}
}
