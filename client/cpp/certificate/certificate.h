// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/certificate/json.gen.h"
#include "client/cpp/certificate/proto.gen.h"
#include "client/cpp/certificate/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/certificate/pb/certificate.pb.h"
#include "core/pkg/transport/grpc/certificate/certificate.pb.h"

namespace synnax::certificate {
/// @brief Freighter retrieve transport.
using RetrieveClient = freighter::UnaryClient<
    ::api::certificate::pb::RetrieveRequest,
    ::api::certificate::pb::RetrieveResponse>;

/// @brief Freighter set transport.
using SetClient = freighter::
    UnaryClient<grpc::certificate::SetRequest, grpc::certificate::SetResponse>;

/// @brief Freighter refresh transport.
using RefreshClient = freighter::
    UnaryClient<grpc::certificate::RefreshRequest, google::protobuf::Empty>;

/// @brief Client for reading and writing certificates in the Core.
class Client {
public:
    Client() = default;

    Client(
        std::shared_ptr<RetrieveClient> retrieve_client,
        std::shared_ptr<SetClient> set_client,
        std::shared_ptr<RefreshClient> refresh_client
    );

    /// @brief retrieves the certificate with the given key.
    /// @returns the certificate, and an error that matches NOT_FOUND when no
    /// certificate has the key.
    [[nodiscard]] std::pair<Certificate, x::errors::Error>
    retrieve(const Key &key) const;

    /// @brief retrieves the certificates with the given keys.
    /// @returns the certificates for the keys that exist.
    [[nodiscard]] std::pair<std::vector<Certificate>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief creates the certificate, or replaces the one with its key. The Core reads
    /// the details of a stored certificate from its PEM.
    /// @param certificate the certificate to save. On success, it holds the saved
    /// certificate, with its key and details.
    /// @param private_key_pem the PEM private key of a stored certificate, which the
    /// Core saves as a secret that the certificate uses. Empty for none.
    [[nodiscard]] x::errors::Error
    set(Certificate &certificate, const std::string &private_key_pem = "") const;

    /// @brief saves the details that the calling rack read from the files of a file
    /// certificate. Only the rack that the certificate names may call it.
    [[nodiscard]] x::errors::Error
    refresh(const Key &key, const Details &details) const;

private:
    std::shared_ptr<RetrieveClient> retrieve_client;
    std::shared_ptr<SetClient> set_client;
    std::shared_ptr<RefreshClient> refresh_client;
};
}
