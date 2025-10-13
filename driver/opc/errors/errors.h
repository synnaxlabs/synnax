// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "open62541/types.h"

/// module
#include "x/cpp/xerrors/errors.h"

/// internal
#include "driver/errors/errors.h"

namespace opc::errors {
const xerrors::Error CRITICAL = driver::CRITICAL_HARDWARE_ERROR.sub("opc");
const xerrors::Error TEMPORARY = driver::TEMPORARY_HARDWARE_ERROR.sub("opc");
const xerrors::Error UNREACHABLE = CRITICAL.sub("unreachable");
const xerrors::Error NO_CONNECTION = UNREACHABLE.sub("no_connection");

xerrors::Error parse(const UA_StatusCode &status);
}
