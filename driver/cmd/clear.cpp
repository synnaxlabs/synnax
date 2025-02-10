// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "glog/logging.h"

/// internal
#include "driver/cmd/cmd.h"
#include "driver/config/config.h"

int cmd::sub::clear(int argc, char **argv) {
    // clear the driver persisted state
    if (const auto err = driver::clear_persisted_state(); err) {
        LOG(ERROR) << "failed to clear persisted state: " << err;
        return 1;
    }
    return 0;
}