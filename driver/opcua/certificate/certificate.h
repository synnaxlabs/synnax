// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/x509/x509.h"

namespace driver::opcua::certificate {
/// @brief the public description of a certificate that the Driver reports to the
/// Core.
struct Info {
    /// @brief the PEM encoded certificate.
    std::string pem;
    /// @brief the SHA-1 hash of the DER bytes as 40 uppercase hex characters.
    std::string thumbprint;
    /// @brief the distinguished name of the subject, such as "CN=Server, O=Org".
    std::string subject;
    /// @brief the end of the validity period.
    x::telem::TimeStamp not_after;

    /// @returns the description as JSON, with not_after as an ISO 8601 UTC string.
    [[nodiscard]] x::json::json to_json() const;
};

/// @returns the public description of the certificate.
[[nodiscard]] Info info(const x::x509::Certificate &cert);

/// @brief a certificate and the private key that signs for it.
struct Identity {
    /// @brief the certificate that a client presents.
    x::x509::Certificate certificate;
    /// @brief the PEM or DER encoded private key.
    std::string private_key;

    /// @brief reads an identity from a certificate file and a private key file.
    /// @returns errors::CLIENT_CERTIFICATE_INVALID if a file cannot be read, the key
    /// needs a password, or the key does not match the certificate.
    static std::pair<Identity, x::errors::Error>
    load(const std::filesystem::path &certificate, const std::filesystem::path &key);
};

/// @brief makes a self-signed RSA 2048 identity for an OPC UA application.
/// @param application_uri the URI for the subject alternative name.
/// @param common_name the common name of the subject.
/// @param host the DNS name for the subject alternative name. Empty omits it.
/// @param validity the length of the validity period, starting now.
std::pair<Identity, x::errors::Error> create(
    const std::string &application_uri,
    const std::string &common_name,
    const std::string &host,
    x::telem::TimeSpan validity
);
}
