// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "open62541/types.h"

/// module
#include "x/cpp/errors/errors.h"

/// internal
#include "driver/errors/errors.h"

namespace driver::opc::errors {
const x::errors::Error CRITICAL = driver::errors::CRITICAL_HARDWARE_ERROR.sub("opc");
const x::errors::Error TEMPORARY = driver::errors::TEMPORARY_HARDWARE_ERROR.sub("opc");
const x::errors::Error UNREACHABLE = CRITICAL.sub("unreachable");
const x::errors::Error NO_CONNECTION = UNREACHABLE.sub("no_connection");

/// @brief security-related errors for certificate and encryption issues
const x::errors::Error SECURITY = CRITICAL.sub("security");
const x::errors::Error MISSING_CERTIFICATE = SECURITY.sub("missing_certificate");
const x::errors::Error INVALID_SECURITY_POLICY = SECURITY.sub("invalid_policy");
const x::errors::Error ENCRYPTION_CONFIG_FAILED = SECURITY.sub("encryption_config");

/// @brief specific OPC UA error codes mapped to error types
const x::errors::Error INVALID_ENDPOINT = CRITICAL.sub("BadTcpEndpointUrlInvalid");
const x::errors::Error IDENTITY_TOKEN_REJECTED = CRITICAL.sub(
    "BadIdentityTokenRejected"
);

x::errors::Error parse(const UA_StatusCode &status);
}
