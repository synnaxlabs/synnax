// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <string>

#include "x/cpp/errors/errors.h"

namespace driver::ethercat::soem {
/// @brief opens path with flags. Returns a file descriptor, or -1 with errno set.
using Open = std::function<int(const char *path, int flags)>;

/// @brief checks that the process can open a macOS BPF device, which SOEM needs to send
/// raw frames on ifname.
/// @param ifname the interface named in the returned error.
/// @param open opens a device. The check closes the descriptor it returns.
/// @returns errors::PERMISSION_DENIED when the process lacks permission, or
/// errors::INTERFACE_ERROR when every device is busy or a device fails to open for
/// another reason.
x::errors::Error check_bpf_access(const std::string &ifname, const Open &open);
}
