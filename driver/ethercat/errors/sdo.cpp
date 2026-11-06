// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

extern "C" {
#include "soem/soem.h"
}

#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::errors {
std::string sdo_abort_reason(const uint32_t code) {
    if (code == 0) return "no response";
    return ec_sdoerror2string(code);
}
}
