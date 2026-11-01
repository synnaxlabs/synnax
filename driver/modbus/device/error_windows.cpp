// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>

#include <winsock2.h>

#include "driver/modbus/device/device.h"

namespace driver::modbus::device {
void clear_error() {
    WSASetLastError(0);
}

int last_error() {
    switch (WSAGetLastError()) {
        case 0:
            return errno;
        case WSAECONNABORTED:
            return ECONNABORTED;
        default:
            return ECONNRESET;
    }
}
}
