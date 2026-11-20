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
#include <utility>
#include <vector>

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/proto.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/icd/pb/icd.pb.h"
#include "core/pkg/transport/grpc/icd/icd.pb.h"

namespace synnax::icd {
/// @brief the transport used to create ICDs.
using CreateClient = freighter::
    UnaryClient<grpc::icd::CreateRequest, grpc::icd::CreateResponse>;

/// @brief the transport used to retrieve ICDs.
using RetrieveClient = freighter::
    UnaryClient<::api::icd::pb::RetrieveRequest, ::api::icd::pb::RetrieveResponse>;

/// @brief Client for creating and retrieving ICDs in a Core.
class Client {
public:
    /// @brief constructs a client that sends its requests on the given transports.
    /// @param create_client the transport that creates ICDs.
    /// @param retrieve_client the transport that retrieves ICDs.
    Client(
        std::shared_ptr<CreateClient> create_client,
        std::shared_ptr<RetrieveClient> retrieve_client
    ):
        create_client(std::move(create_client)),
        retrieve_client(std::move(retrieve_client)) {}

    /// @brief retrieves the ICD with the given key.
    /// @param key the key of the ICD to retrieve.
    /// @returns the ICD, and an error that matches NOT_FOUND when no ICD has the key.
    [[nodiscard]] std::pair<ICD, x::errors::Error> retrieve(const Key &key) const;

    /// @brief retrieves the ICDs with the given keys.
    /// @param keys the keys of the ICDs to retrieve.
    /// @returns the ICDs whose keys exist, in no set order.
    [[nodiscard]] std::pair<std::vector<ICD>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief creates the ICD, and sets the keys the Core assigns to it, its messages,
    /// and its fields.
    /// @param ic the ICD to create.
    /// @returns an error that matches VALIDATION when an ICD with the key exists.
    [[nodiscard]] x::errors::Error create(ICD &ic) const;

private:
    std::shared_ptr<CreateClient> create_client;
    std::shared_ptr<RetrieveClient> retrieve_client;

    [[nodiscard]] std::pair<std::vector<ICD>, x::errors::Error>
    retrieve(const Where &where) const;
};
}
