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

#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/proto.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/transport/grpc/icd/icd.pb.h"

namespace synnax::icd {

using CreateClient = freighter::
    UnaryClient<grpc::icd::CreateRequest, grpc::icd::CreateResponse>;

using RetrieveClient = freighter::
    UnaryClient<grpc::icd::RetrieveRequest, grpc::icd::RetrieveResponse>;

using DeleteClient = freighter::
    UnaryClient<grpc::icd::DeleteRequest, google::protobuf::Empty>;

class Client {
public:
    /// @brief Constructs a new ICD client with the given transport clients.
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DeleteClient> delete_client
    );

    /// @brief Retrieves an ICD by its key.
    [[nodiscard]]
    std::pair<ICD, x::errors::Error> retrieve(const Key &key) const;

    /// @brief Retrieves multiple ICDs by their keys.
    [[nodiscard]]
    std::pair<std::vector<ICD>, x::errors::Error>
    retrieve(const std::vector<Key> &keys) const;

    /// @brief Creates the ICD, or replaces the ICD that has its key. A replacement must
    /// carry the hash of the ICD as it was read. Sets the keys and the hash the Core
    /// assigns.
    [[nodiscard]]
    x::errors::Error create(ICD &ic) const;

    /// @brief Creates multiple ICDs. Sets the keys and hashes the Core assigns, as the
    /// single-ICD overload does.
    [[nodiscard]]
    x::errors::Error create(std::vector<ICD> &ics) const;

    /// @brief Deletes an ICD by its key.
    [[nodiscard]]
    x::errors::Error del(const Key &key) const;

    /// @brief Deletes multiple ICDs by their keys.
    [[nodiscard]]
    x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DeleteClient> delete_client;
};

}
