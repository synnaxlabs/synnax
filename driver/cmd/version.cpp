// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>

/// module
#include "synnax/pkg/version/version.h"

/// internal
#include "driver/cmd/cmd.h"

int cmd::sub::version(int argc, char **argv) {
    std::cout << "Synnax Driver version " << SYNNAX_DRIVER_VERSION << " (" <<
            SYNNAX_BUILD_TIMESTAMP << ")" << std::endl;
    return 0;
}
