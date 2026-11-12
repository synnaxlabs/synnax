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
/// @brief turns off the DewesoftX "EthercatDAQ Filter" on the adapter of each Npcap
/// interface. Asks Windows for administrator access when the process is not elevated,
/// and blocks for at most 45 seconds.
/// @returns errors::FILTER_ERROR when the filter may still be on, with the reason and
/// the manual fix. Always errors::FILTER_ERROR outside Windows.
x::errors::Error disable_dewesoft_filter(const std::vector<std::string> &interfaces);
}
