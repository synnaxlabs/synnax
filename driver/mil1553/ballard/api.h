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

#include "driver/errors/errors.h"

namespace driver::mil1553::ballard {
// TODO: confirm both library names against the BTIDriver SDK for each platform.
#ifdef _WIN32
/// @brief the BTICard library, which opens and starts cards.
const std::string CARD_LIBRARY_NAME = "BTICARD64.DLL";
/// @brief the BTI1553 library, which runs the bus controller, terminals, and
/// monitor.
const std::string LIBRARY_NAME = "BTI155364.DLL";
#else
const std::string CARD_LIBRARY_NAME = "libbticard.so";
const std::string LIBRARY_NAME = "libbti1553.so";
#endif

const LibraryInfo LIBRARY_INFO = {"Ballard BTIDriver MIL-STD-1553", ""};

// TODO: declare the prototypes and constants from the BTIDriver headers, which are
// not public. The backend needs, at least: BTICard_CardOpen, BTICard_CoreOpen,
// BTICard_CardClose, BTICard_CardStart, BTICard_CardStop, BTICard_ErrDescStr, and
// the BTI1553 bus controller, remote terminal, monitor, and message data functions.
}
