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
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/secret/json.gen.h"
#include "client/cpp/secret/proto.gen.h"
#include "client/cpp/secret/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/secret/pb/secret.pb.h"
#include "core/pkg/transport/grpc/secret/secret.pb.h"

namespace synnax::secret {
/// @brief Freighter retrieve transport.
using RetrieveClient = freighter::UnaryClient<
    ::api::secret::pb::RetrieveRequest,
    ::api::secret::pb::RetrieveResponse>;

/// @brief Freighter resolve transport.
using ResolveClient = freighter::
    UnaryClient<grpc::secret::ResolveRequest, grpc::secret::ResolveResponse>;

/// @brief the plain text values of secrets by key.
using Values = std::unordered_map<Key, std::string>;

/// @brief what a spent grant gives a rack.
struct Grant {
    /// @brief the keys of the secrets that the grant names, file secrets included.
    std::vector<Key> secrets;
    /// @brief the values of the secrets that the grant names, except a file secret of
    /// a rack that is not the Core's, which the rack reads from its own secrets folder.
    Values values;
};

/// @brief Client for reading secrets in the Core.
class Client {
public:
    Client() = default;

    Client(
        std::shared_ptr<RetrieveClient> retrieve_client,
        std::shared_ptr<ResolveClient> resolve_client
    );

    /// @brief retrieves the secret with the given key, never with its value.
    /// @returns the secret, and an error that matches NOT_FOUND when no secret has the
    /// key.
    [[nodiscard]] std::pair<Secret, x::errors::Error> retrieve(const Key &key) const;

    /// @brief retrieves the secrets with the given keys, never with their values.
    /// @returns the secrets for the keys that exist.
    [[nodiscard]] std::pair<std::vector<Secret>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief returns the values of the stored secrets with the given keys. Only a rack
    /// may call it, and a device or task on the rack must use each secret.
    [[nodiscard]] std::pair<Values, x::errors::Error>
    resolve(const std::vector<Key> &keys) const;

private:
    std::shared_ptr<RetrieveClient> retrieve_client;
    std::shared_ptr<ResolveClient> resolve_client;
};
}
