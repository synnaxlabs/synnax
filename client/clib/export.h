// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// @brief marks a function for export from the synnax_clib shared library
/// (dllexport/visibility when building it, dllimport when consuming on Windows).
#if defined(_WIN32)
#if defined(SYNNAX_BUILDING_DLL)
#define SYNNAX_EXPORT __declspec(dllexport)
#else
#define SYNNAX_EXPORT __declspec(dllimport)
#endif
#else
#if defined(SYNNAX_BUILDING_DLL)
#define SYNNAX_EXPORT __attribute__((visibility("default")))
#else
#define SYNNAX_EXPORT
#endif
#endif
