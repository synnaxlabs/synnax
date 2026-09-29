// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstddef>
#include <string>
#include <system_error>
#include <utility>

#include <sys/ioctl.h>

#include "driver/serial/native.h"
#include "driver/serial/posix.h"
#include "driver/transport/errors.h"

namespace driver::serial::native {
x::errors::Error failed(const std::string &setting, const std::string &port) {
    return transport::error(
        transport::CONFIG_ERROR,
        "failed to set " + setting + " on " + port,
        {errno, std::generic_category()}
    );
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    int queued = 0;
    if (::ioctl(static_cast<int>(handle), TIOCOUTQ, &queued) != 0)
        return {0, {errno, std::generic_category()}};
    return {static_cast<std::size_t>(queued), {}};
}
}
