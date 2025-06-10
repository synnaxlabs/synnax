// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

/// @brief utility packages for managing status messages.
namespace status {
const std::string VARIANT_SUCCESS = "success";
const std::string VARIANT_ERROR = "error";
const std::string VARIANT_WARNING = "warning";
const std::string VARIANT_INFO = "info";
const std::string VARIANT_DISABLED = "disabled";
const std::string VARIANT_LOADING = "loading";
}
