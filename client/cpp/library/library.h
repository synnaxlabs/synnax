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

#include "client/cpp/library/json.gen.h"
#include "client/cpp/library/proto.gen.h"
#include "client/cpp/library/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/transport/grpc/library/library.pb.h"

namespace synnax::library {

/// @brief Type alias for the transport used to create a library.
using CreateClient = freighter::
    UnaryClient<grpc::library::CreateRequest, grpc::library::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a library.
using RetrieveClient = freighter::
    UnaryClient<grpc::library::RetrieveRequest, grpc::library::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a library.
using DeleteClient = freighter::
    UnaryClient<grpc::library::DeleteRequest, google::protobuf::Empty>;

/// @brief Client for managing libraries in a Core.
class Client {
public:
    /// @brief Constructs a new library client with the given transport clients.
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DeleteClient> delete_client
    );

    /// @brief Retrieves a library by its key.
    [[nodiscard]]
    std::pair<Library, x::errors::Error> retrieve(const Key &key) const;

    /// @brief Retrieves multiple libraries by their keys.
    [[nodiscard]]
    std::pair<std::vector<Library>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief Creates the library, or replaces the library that has its key. Sets the
    /// keys the Core assigns to the library, its entries, and their fields.
    [[nodiscard]]
    x::errors::Error create(Library &lib) const;

    /// @brief Creates multiple libraries. Sets the keys the Core assigns, as the
    /// single-library overload does.
    [[nodiscard]]
    x::errors::Error create(std::vector<Library> &libs) const;

    /// @brief Deletes a library by its key.
    [[nodiscard]]
    x::errors::Error del(const Key &key) const;

    /// @brief Deletes multiple libraries by their keys.
    [[nodiscard]]
    x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DeleteClient> delete_client;
};

}
