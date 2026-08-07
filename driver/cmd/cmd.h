// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

// IT IS ABSOLUTELY MISSION CRITICAL THAT THIS BLOCK IS THE FIRST INCLUDE IN THIS FILE.
// Otherwise, you will see a bunch of linker errors.
#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#endif
// END OF MISSION CRITICAL CODE BLOCK.

namespace driver::cmd {
/// @brief exec runs the CLI command.
int exec(int argc, char **argv);
}
