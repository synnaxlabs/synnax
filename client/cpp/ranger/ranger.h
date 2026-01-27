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
#include <vector>

#include "client/cpp/ranger/json.gen.h"
#include "client/cpp/ranger/kv/kv.h"
#include "client/cpp/ranger/proto.gen.h"
#include "client/cpp/ranger/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/ranger/ranger.pb.h"
#include "core/pkg/api/ranger/pb/ranger.pb.h"

namespace synnax::ranger {
/// @brief type alias for the transport used to retrieve ranges.
using RetrieveClient = freighter::
    UnaryClient<grpc::ranger::RetrieveRequest, grpc::ranger::RetrieveResponse>;

/// @brief type alias for the transport used to create ranges.
using CreateClient = freighter::
    UnaryClient<grpc::ranger::CreateRequest, grpc::ranger::CreateResponse>;

/// @brief a client for performing operations on the ranges in a Synnax cluster.
class Client {
public:
    Client(
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<CreateClient> create_client,
        const kv::Client &kv_client
    ):
        retrieve_client(std::move(retrieve_client)),
        create_client(std::move(create_client)),
        kv(kv_client) {}

    /// @brief retrieves the range with the given key.
    /// @param key - the key of the range to retrieve.
    /// @returns a pair containing the created range and an error where ok() is
    /// false if the range could not be retrieved. Use err.message() to get the
    /// error message or err.type to get the error type.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve_by_key(const x::uuid::UUID &key) const;

    /// @brief retrieves the range with the given name.
    /// @param name - the name of the range to retrieve.
    /// @returns a pair containing the created range and an error where ok() is
    /// false if the range could not be retrieved. Use err.message() to get the
    /// error message or err.type to get the error type.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve_by_name(const std::string &name) const;

    /// @brief retrieves the ranges with the given keys.
    /// @param keys - the keys of the ranges to retrieve.
    /// @returns a pair containing the created ranges and an error where ok() is
    /// false if the ranges could not be retrieved. Use err.message() to get the
    /// error message or err.type to get the error type.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve_by_key(const std::vector<x::uuid::UUID> &keys) const;

    /// @brief retrieves the ranges with the given names.
    /// @param names - the names of the ranges to retrieve.
    /// @returns a pair containing the created ranges and an error where ok() is
    /// false if the ranges could not be retrieved. Use err.message() to get the
    /// error message or err.type to get the error type.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve_by_name(const std::vector<std::string> &names) const;

    /// @brief creates the given ranges.
    /// @param ranges - the ranges to create.
    /// @modifies the ranges in the vector to set their keys and default values.
    /// @returns an error where ok() is false if the ranges could not be created.
    /// Use err.message() to get the error message or err.type to get the error
    /// type.
    [[nodiscard]] x::errors::Error create(std::vector<Range> &ranges) const;

    /// @brief creates the given range.
    /// @param range - the range to create.
    /// @modifies the range to set its key and default values.
    /// @returns an error where ok() is false if the range could not be created.
    [[nodiscard]] x::errors::Error create(Range &range) const;

    /// @brief creates a range with the given name and time range.
    /// @param name - the name of the range to create.
    /// @param time_range - the time range of the range to create.
    /// @returns a pair containing the created range and an error where ok() is
    /// false if the range could not be created. Use err.message() to get the error
    /// message or err.type to get the error type.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    create(const std::string &name, x::telem::TimeRange time_range) const;

private:
    /// @brief range retrieval transport.
    std::unique_ptr<RetrieveClient> retrieve_client;
    /// @brief create retrieval transport.
    std::unique_ptr<CreateClient> create_client;
    /// @brief range kv get transport.
    kv::Client kv;

    /// @brief retrieves multiple ranges.
    std::pair<std::vector<Range>, x::errors::Error>
    retrieve_many(grpc::ranger::RetrieveRequest &req) const;
};
}