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
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::x509 {
/// @brief data that holds no X.509 certificate.
const errors::Error INVALID = errors::SY.sub("x509.invalid");
/// @brief a private key that cannot be read, or that does not match its certificate.
const errors::Error KEY_INVALID = errors::SY.sub("x509.key_invalid");

/// @brief an X.509 certificate.
class Certificate {
public:
    Certificate() = default;

    /// @brief parses a PEM or DER encoded certificate. Of a certificate chain, only
    /// the first certificate is kept.
    /// @returns INVALID if the data holds no certificate.
    static std::pair<Certificate, errors::Error> parse(std::string_view data);

    /// @returns true if the data starts with a PEM certificate header.
    [[nodiscard]] static bool is_pem(std::string_view data);

    /// @returns the DER encoded certificate.
    [[nodiscard]] const std::string &der() const { return this->der_bytes; }
    /// @returns the PEM encoded certificate.
    [[nodiscard]] std::string pem() const;
    /// @returns the SHA-1 hash of the DER bytes as 40 uppercase hex characters.
    [[nodiscard]] std::string thumbprint() const;
    /// @returns the distinguished name of the subject, such as "CN=Server, O=Org".
    [[nodiscard]] const std::string &subject() const { return this->subject_name; }
    /// @returns the first URI in the subject alternative name, or an empty string.
    [[nodiscard]] const std::string &uri() const { return this->san_uri; }
    /// @returns the start of the validity period.
    [[nodiscard]] telem::TimeStamp not_before() const { return this->valid_from; }
    /// @returns the end of the validity period.
    [[nodiscard]] telem::TimeStamp not_after() const { return this->valid_to; }
    /// @returns true if the time is inside the validity period.
    [[nodiscard]] bool valid_at(telem::TimeStamp time) const;

    /// @brief checks that a private key parses without a password and matches the
    /// certificate.
    /// @param key the PEM or DER encoded private key.
    /// @returns KEY_INVALID with the reason as a plain message.
    [[nodiscard]] errors::Error check_key(const std::string &key) const;

    bool operator==(const Certificate &other) const {
        return this->der_bytes == other.der_bytes;
    }

private:
    std::string der_bytes;
    std::string subject_name;
    std::string san_uri;
    telem::TimeStamp valid_from;
    telem::TimeStamp valid_to;
};
}
