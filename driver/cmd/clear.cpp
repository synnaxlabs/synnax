// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

int cmd::sub::clear(xargs::Parser &args) {
    if (const auto err = rack::Config::clear_persisted_state(args); err) {
        LOG(ERROR) << "failed to clear persisted state: " << err;
        return 1;
    }
    return 0;
}
