// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/view/json.gen.h"
#include "client/cpp/view/proto.gen.h"
#include "client/cpp/view/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/grpc/view/view.pb.h"

namespace synnax::view {

/// @brief Type alias for the transport used to create a view.
using CreateClient = freighter::
    UnaryClient<grpc::view::CreateRequest, grpc::view::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a view.
using RetrieveClient = freighter::
    UnaryClient<grpc::view::RetrieveRequest, grpc::view::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a view.
using DeleteClient = freighter::
    UnaryClient<grpc::view::DeleteRequest, google::protobuf::Empty>;

/// @brief Client for managing views in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new view client with the given transport clients.
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DeleteClient> delete_client
    );

    /// @brief Retrieves a view by its key.
    [[nodiscard]]
    std::pair<View, x::errors::Error> retrieve(const Key &key) const;

    /// @brief Retrieves multiple views by their keys.
    [[nodiscard]]
    std::pair<std::vector<View>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief Creates a view in the cluster.
    [[nodiscard]]
    x::errors::Error create(View &view) const;

    /// @brief Creates multiple views in the cluster.
    [[nodiscard]]
    x::errors::Error create(std::vector<View> &views) const;

    /// @brief Deletes a view by its key.
    [[nodiscard]]
    x::errors::Error del(const Key &key) const;

    /// @brief Deletes multiple views by their keys.
    [[nodiscard]]
    x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DeleteClient> delete_client;
};

}
