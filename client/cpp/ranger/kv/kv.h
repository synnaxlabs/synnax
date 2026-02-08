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

#include "google/protobuf/empty.pb.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/grpc/v1/ranger.pb.h"

namespace synnax::kv {
/// @brief type alias for the transport used to get range-scoped key-values.
using GetClient = freighter::
    UnaryClient<api::v1::RangeKVGetRequest, api::v1::RangeKVGetResponse>;

/// @brief type alias for the transport used to set range-scoped key-values.
using SetClient = freighter::
    UnaryClient<api::v1::RangeKVSetRequest, google::protobuf::Empty>;

/// @brief type alias for the transport used to delete range-scoped key-values.
using DeleteClient = freighter::
    UnaryClient<api::v1::RangeKVDeleteRequest, google::protobuf::Empty>;

/// @brief a range-scoped key-value store for storing metadata and configuration
/// about a range.
class Client {
    std::string range_key;
    std::shared_ptr<GetClient> get_client;
    std::shared_ptr<SetClient> set_client;
    std::shared_ptr<DeleteClient> delete_client;

public:
    Client() = default;

    Client(
        std::shared_ptr<GetClient> get_client,
        std::shared_ptr<SetClient> set_client,
        std::shared_ptr<DeleteClient> delete_client
    ):
        get_client(std::move(get_client)),
        set_client(std::move(set_client)),
        delete_client(std::move(delete_client)) {}

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

    /// @brief returns a copy of this client scoped to the given range key.
    Client scope_to_range(const std::string &range_key_) const {
        auto c = *this;
        c.range_key = range_key_;
        return c;
    }
};
}
