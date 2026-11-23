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

#include "google/protobuf/empty.pb.h"

#include "client/cpp/ni/chassis/proto.gen.h"
#include "client/cpp/ni/chassis/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"

#include "core/pkg/api/ni/chassis/pb/ni_chassis.pb.h"
#include "core/pkg/transport/grpc/ni/chassis/chassis.pb.h"

namespace synnax::ni::chassis {
/// @brief the transport that creates NI chassis.
using CreateClient = freighter::
    UnaryClient<grpc::ni_chassis::CreateRequest, grpc::ni_chassis::CreateResponse>;

/// @brief the transport that retrieves NI chassis.
using RetrieveClient = freighter::UnaryClient<
    ::api::ni_chassis::pb::RetrieveRequest,
    ::api::ni_chassis::pb::RetrieveResponse>;

/// @brief the transport that writes one field group to an NI chassis.
using UpdateClient = freighter::
    UnaryClient<grpc::ni_chassis::UpdateRequest, google::protobuf::Empty>;

/// @brief the transport that deletes NI chassis.
using DeleteClient = freighter::
    UnaryClient<grpc::ni_chassis::DeleteRequest, google::protobuf::Empty>;

/// @brief creates, retrieves, updates, and deletes NI chassis in a Core.
class Client {
public:
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<UpdateClient> update_client,
        std::unique_ptr<DeleteClient> delete_client
    );

    /// @brief creates the chassis, and sets the key the Core assigns to each one.
    /// @param chassis the chassis to create.
    /// @returns an error if the create failed.
    [[nodiscard]] x::errors::Error create(std::vector<Chassis> &chassis) const;

    /// @brief creates the chassis, and sets the key the Core assigns to it.
    /// @param chassis the chassis to create.
    /// @returns an error if the create failed.
    [[nodiscard]] x::errors::Error create(Chassis &chassis) const;

    /// @brief retrieves the chassis with the given key.
    /// @param key the key of the chassis.
    /// @param options the options of the retrieve.
    /// @returns the chassis, and an error that matches NOT_FOUND when no chassis has
    /// the key.
    [[nodiscard]] std::pair<Chassis, x::errors::Error>
    retrieve(const Key &key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the chassis that where selects.
    /// @param where the filter that the chassis match. An empty filter matches all
    /// chassis.
    /// @param options the options of the retrieve.
    [[nodiscard]] std::pair<std::vector<Chassis>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one chassis that where selects.
    /// @param where the filter that the chassis matches.
    /// @param options the options of the retrieve.
    /// @returns the chassis, and an error that matches NOT_FOUND when no chassis
    /// matches, MULTIPLE_RESULTS when more than one does, or VALIDATION when options
    /// set a limit or a cursor.
    [[nodiscard]] std::pair<Chassis, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief writes one field group to the chassis with the given key, and keeps the
    /// other group as stored.
    /// @param key the key of the chassis.
    /// @param update the field group to write.
    /// @returns an error that matches NOT_FOUND when no chassis has the key.
    [[nodiscard]] x::errors::Error update(const Key &key, const Update &update) const;

    /// @brief deletes the chassis with the given keys.
    /// @param keys the keys of the chassis.
    /// @returns an error if the delete failed.
    [[nodiscard]] x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<UpdateClient> update_client;
    std::unique_ptr<DeleteClient> delete_client;
};
}
