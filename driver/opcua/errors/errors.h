// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::opcua::errors {
const x::errors::Error CRITICAL = driver::errors::CRITICAL_HARDWARE_ERROR.sub("opc");
const x::errors::Error TEMPORARY = driver::errors::TEMPORARY_HARDWARE_ERROR.sub("opc");
const x::errors::Error UNREACHABLE = CRITICAL.sub("unreachable");
const x::errors::Error NO_CONNECTION = UNREACHABLE.sub("no_connection");

/// @brief security-related errors for certificate and encryption issues
const x::errors::Error SECURITY = CRITICAL.sub("security");
const x::errors::Error ENCRYPTION_CONFIG_FAILED = SECURITY.sub("encryption_config");
/// @brief data that holds no X.509 certificate.
const x::errors::Error INVALID_CERTIFICATE = SECURITY.sub("invalid_certificate");
/// @brief client certificate files that cannot be read, or that do not match.
const x::errors::Error CLIENT_CERTIFICATE_INVALID = SECURITY.sub(
    "client_certificate_invalid"
);
/// @brief the Driver certificate cannot be loaded, made, or saved.
const x::errors::Error DRIVER_CERTIFICATE_FAILED = SECURITY.sub("driver_certificate");
/// @brief the server rejected the certificate that the client presented.
const x::errors::Error CLIENT_CERTIFICATE_REJECTED = SECURITY.sub(
    "client_certificate_rejected"
);
/// @brief the server presented a certificate other than the trusted one.
const x::errors::Error SERVER_CERTIFICATE_UNTRUSTED = SECURITY.sub(
    "server_certificate_untrusted"
);
/// @brief the server certificate is outside its validity period.
const x::errors::Error SERVER_CERTIFICATE_TIME_INVALID = SECURITY.sub(
    "server_certificate_time_invalid"
);

/// @brief specific OPC UA error codes mapped to error types
const x::errors::Error INVALID_ENDPOINT = CRITICAL.sub("BadTcpEndpointUrlInvalid");
const x::errors::Error IDENTITY_TOKEN_REJECTED = CRITICAL.sub(
    "BadIdentityTokenRejected"
);

x::errors::Error parse(const UA_StatusCode &status);
}
