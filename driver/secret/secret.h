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
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

namespace driver::secret {
/// @brief configures a Reader.
struct Config {
    /// @brief the folder of the rack's file secrets and certificate keys. Empty when
    /// none is set.
    std::filesystem::path secrets_dir;
    /// @brief the rack that the Driver runs.
    synnax::rack::Key rack = 0;
};

/// @brief a certificate chain and its private key, as a Reader read them.
struct Certificate {
    /// @brief the certificate chain as PEM, leaf first, with no other blocks or text.
    std::string pem;
    /// @brief the private key as PEM, or empty when the certificate has none.
    std::string private_key;
};

/// @brief reads secrets and certificates for the integrations of a rack. It opens the
/// files of an item at each read and keeps no value.
class Reader {
public:
    Reader(std::shared_ptr<synnax::Synnax> client, Config cfg);

    /// @brief returns the values of the secrets with the given keys. It reads a file
    /// secret from the secrets folder, without the line breaks at the end of the file.
    /// @param grant a spent grant. When set, it must name each key, and a stored value
    /// comes from it. When not, the Core gives a stored value only when a device or
    /// task on the rack uses the secret.
    /// @returns VALIDATION when the grant does not name a key, NOT_FOUND when no secret
    /// has a key, an access error when no device or task on the rack uses a stored
    /// secret, and VALIDATION for a file secret of another rack, or when no secrets
    /// folder is set.
    [[nodiscard]] std::pair<synnax::secret::Values, x::errors::Error> read(
        const std::vector<synnax::secret::Key> &keys,
        const std::optional<synnax::secret::Grant> &grant = std::nullopt
    ) const;

    /// @brief returns the certificate with the given key and its private key. For a
    /// file certificate, it reads the certificate file, and the key file from the
    /// secrets folder. After it reads both, it saves new details in the Core when the
    /// SHA-256 fingerprint of the certificate differs from the saved one.
    /// @param grant a spent grant, as for read.
    /// @returns VALIDATION for a file certificate of another rack, before any read.
    [[nodiscard]] std::pair<Certificate, x::errors::Error> read_certificate(
        const synnax::certificate::Key &key,
        const std::optional<synnax::secret::Grant> &grant = std::nullopt
    ) const;

private:
    std::shared_ptr<synnax::Synnax> client;
    Config cfg;

    /// @returns the bytes of the file with name in the secrets folder, for the item
    /// that the error names.
    [[nodiscard]] std::pair<std::string, x::errors::Error>
    read_file(const std::string &item, const std::string &name) const;
};
}
