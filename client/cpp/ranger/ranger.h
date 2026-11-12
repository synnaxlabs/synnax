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

#include "client/cpp/ranger/actions.gen.h"
#include "client/cpp/ranger/json.gen.h"
#include "client/cpp/ranger/kv/kv.h"
#include "client/cpp/ranger/proto.gen.h"
#include "client/cpp/ranger/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/filter/field.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/ranger/pb/ranger.pb.h"
#include "core/pkg/transport/grpc/ranger/ranger.pb.h"

namespace synnax::ranger {
/// @brief type alias for the transport used to retrieve ranges.
using RetrieveClient = freighter::
    UnaryClient<grpc::ranger::RetrieveRequest, ::api::ranger::pb::RetrieveResponse>;

/// @brief type alias for the transport used to create ranges.
using CreateClient = freighter::
    UnaryClient<grpc::ranger::CreateRequest, grpc::ranger::CreateResponse>;

/// @brief type alias for the transport used to dispatch actions to a range.
using DispatchClient = freighter::
    UnaryClient<grpc::ranger::DispatchRequest, google::protobuf::Empty>;

/// @brief a client for performing operations on the ranges in a Synnax cluster.
class Client {
public:
    Client(
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<DispatchClient> dispatch_client,
        ranger::kv::Client kv_client
    ):
        retrieve_client(std::move(retrieve_client)),
        create_client(std::move(create_client)),
        dispatch_client(std::move(dispatch_client)),
        kv(std::move(kv_client)) {}

    /// @brief retrieves the range with the given key.
    /// @param key the key of the range.
    /// @param options the options of the retrieve.
    /// @returns the range, and an error that matches NOT_FOUND when no range has the
    /// key.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve(const Key &key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the range with the given name.
    /// @param name the name of the range.
    /// @param options the options of the retrieve.
    /// @returns the range, and an error that matches NOT_FOUND when no range has the
    /// name, or MULTIPLE_RESULTS when more than one does.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve(const std::string &name, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the ranges with the given keys.
    /// @param keys the keys of the ranges.
    /// @param options the options of the retrieve.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve(const std::vector<Key> &keys, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the ranges with the given names.
    /// @param names the names of the ranges.
    /// @param options the options of the retrieve.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error> retrieve(
        const std::vector<std::string> &names,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief retrieves the ranges that where selects.
    /// @param where the filter that the ranges match. An empty filter matches all
    /// ranges.
    /// @param options the options of the retrieve.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one range that where selects.
    /// @param where the filter that the range matches.
    /// @param options the options of the retrieve.
    /// @returns the range, and an error that matches NOT_FOUND when no range matches,
    /// MULTIPLE_RESULTS when more than one does, or VALIDATION when options set a
    /// limit or a cursor.
    [[nodiscard]] std::pair<Range, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief creates the given ranges.
    [[nodiscard]] x::errors::Error create(std::vector<Range> &ranges) const;

    /// @brief creates the given range.
    [[nodiscard]] x::errors::Error create(Range &range) const;

    /// @brief creates a range with the given name and time range.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    create(const std::string &name, x::telem::TimeRange time_range) const;

    /// @brief applies the actions to the range with the given key in order, as one
    /// atomic edit.
    /// @param key the key of the range.
    /// @param actions the actions to apply. A set on the time range may hold its
    /// timestamps as integers.
    /// @returns an error that matches NOT_FOUND when no range has the key, or
    /// VALIDATION when the range cannot hold an action.
    [[nodiscard]] x::errors::Error
    dispatch(const Key &key, const std::vector<Action> &actions) const;

    /// @brief sets the end bound of the range with the given key in a single request,
    /// preserving all other fields.
    [[nodiscard]] x::errors::Error
    set_end(const Key &key, x::telem::TimeStamp end) const;

private:
    /// @brief range retrieval transport.
    std::unique_ptr<RetrieveClient> retrieve_client;
    /// @brief create retrieval transport.
    std::unique_ptr<CreateClient> create_client;
    /// @brief range dispatch transport.
    std::unique_ptr<DispatchClient> dispatch_client;
    /// @brief range kv get transport.
    ranger::kv::Client kv;
};
}
