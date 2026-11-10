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
#include "x/cpp/filter/field.h"

#include "core/pkg/api/icd/pb/icd.pb.h"
#include "core/pkg/transport/grpc/icd/icd.pb.h"

namespace synnax::icd {

using CreateClient = freighter::
    UnaryClient<grpc::icd::CreateRequest, grpc::icd::CreateResponse>;

using RetrieveClient = freighter::
    UnaryClient<grpc::icd::RetrieveRequest, ::api::icd::pb::RetrieveResponse>;

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

    /// @brief retrieves the ICD with the given key.
    /// @param key the key of the ICD.
    /// @param options the options of the retrieve.
    /// @returns the ICD, and an error that matches NOT_FOUND when no ICD has the key.
    [[nodiscard]]
    std::pair<ICD, x::errors::Error>
    retrieve(const Key &key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the ICDs with the given keys.
    /// @param keys the keys of the ICDs.
    /// @param options the options of the retrieve.
    /// @returns the ICDs for the keys that exist, and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<ICD>, x::errors::Error>
    retrieve(const std::vector<Key> &keys, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the ICDs that where selects.
    /// @param where the filter that the ICDs match. An empty filter matches all ICDs.
    /// @param options the options of the retrieve.
    [[nodiscard]]
    std::pair<std::vector<ICD>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one ICD that where selects.
    /// @param where the filter that the ICD matches.
    /// @param options the options of the retrieve.
    /// @returns the ICD, and an error that matches NOT_FOUND when no ICD matches,
    /// MULTIPLE_RESULTS when more than one does, or VALIDATION when options set a limit
    /// or a cursor.
    [[nodiscard]]
    std::pair<ICD, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

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
