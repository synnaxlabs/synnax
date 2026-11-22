// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "x/cpp/errors/errors.h"

namespace driver::ethercat::soem {
/// @brief starts to turn off the DewesoftX "EthercatDAQ Filter" on the adapter of each
/// Npcap interface, and returns without waiting for it. Windows asks for administrator
/// access when the process is not elevated.
/// @param interfaces the Npcap names of the interfaces, such as `\Device\NPF_{...}`.
/// @returns errors::FILTER_ERROR when Windows cannot start the change, with the reason
/// and the manual fix. Always errors::FILTER_ERROR outside Windows.
x::errors::Error disable_dewesoft_filter(const std::vector<std::string> &interfaces);
}
