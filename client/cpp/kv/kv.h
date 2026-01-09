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

#include "core/pkg/api/grpc/core/pkg/api/grpc/kv/kv.pb.h"

#include "freighter/cpp/freighter.h"

#include "core/pkg/api/grpc/kv/kv.pb.h"

#include "google/protobuf/empty.pb.h"

#include "client/cpp/ontology/id.h"
#include "x/cpp/telem/telem.h"

namespace synnax::kv {
using Key = std::string;

/// @brief type alias for the transport used to get range-scoped key-values.
using GetClient = freighter::UnaryClient<grpc::kv::GetRequest, grpc::kv::GetResponse>;

/// @brief type alias for the transport used to set range-scoped key-values.
using SetClient = freighter::
    UnaryClient<grpc::kv::SetRequest, google::protobuf::Empty>;

/// @brief type alias for the transport used to delete range-scoped key-values.
using DeleteClient = freighter::
    UnaryClient<grpc::kv::DeleteRequest, google::protobuf::Empty>;

/// @brief a range-scoped key-value store for storing metadata and configuration
/// about a range.
class Client {
    std::string range_key;
    std::shared_ptr<GetClient> kv_get_client;
    std::shared_ptr<SetClient> kv_set_client;
    std::shared_ptr<DeleteClient> kv_delete_client;

public:
    Client(
        std::string range_key,
        std::shared_ptr<GetClient> kv_get_client,
        std::shared_ptr<SetClient> kv_set_client,
        std::shared_ptr<DeleteClient> kv_delete_client
    ):
        range_key(std::move(range_key)),
        kv_get_client(std::move(kv_get_client)),
        kv_set_client(std::move(kv_set_client)),
        kv_delete_client(std::move(kv_delete_client)) {}

    /// @brief gets the value of the given key.
    /// @param key - the key to get the value of.
    /// @returns a pair containing the value and an error where ok() is false if the
    /// value could not be retrieved. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<std::string, x::errors::Error>
    get(const std::string &key) const;

    /// @brief sets the value of the given key.
    /// @param key - the key to set the value of.
    /// @param value - the value to set.
    /// @returns an error where ok() is false if the value could not be set.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    /// @note this will overwrite any existing value for the given key.
    [[nodiscard]] x::errors::Error
    set(const std::string &key, const std::string &value) const;

    /// @brief deletes the value of the given key.
    /// @param key - the key to delete the value of.
    /// @returns an error where ok() is false if the value could not be deleted.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    /// @note this operation is idempotent, an will not error if the key does not
    /// exist.
    [[nodiscard]] x::errors::Error del(const std::string &key) const;
};
}
