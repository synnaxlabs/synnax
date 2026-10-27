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

#include "client/cpp/ranger/builder.gen.h"
#include "client/cpp/ranger/json.gen.h"
#include "client/cpp/ranger/kv/kv.h"
#include "client/cpp/ranger/proto.gen.h"
#include "client/cpp/ranger/types.gen.h"
#include "freighter/cpp/freighter.h"
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

/// @brief type alias for the transport used to set the end bound of a range.
using SetEndClient = freighter::
    UnaryClient<grpc::ranger::SetEndRequest, google::protobuf::Empty>;

/// @brief a client for performing operations on the ranges in a Synnax cluster.
class Client {
public:
    Client(
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<SetEndClient> set_end_client,
        ranger::kv::Client kv_client
    ):
        retrieve_client(std::move(retrieve_client)),
        create_client(std::move(create_client)),
        set_end_client(std::move(set_end_client)),
        kv(std::move(kv_client)) {}

    /// @brief retrieves the ranges that req selects.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve(const RetrieveRequest &req) const;

    /// @brief Retrieves the ranges that match every condition in parts, with each
    /// option in parts applied.
    /// @returns the ranges, or one range when parts end with ranger::one(). One range
    /// is an error when no range or more than one range matches.
    template<typename... Parts>
        requires x::filter::parts<Where, RetrieveRequest, Parts...>
    [[nodiscard]] auto retrieve(Parts... parts) const {
        const auto req = x::filter::request<Where, RetrieveRequest>(
            std::move(parts)...
        );
        if constexpr (x::filter::returns_one<Parts...>)
            return this->retrieve_one(req, x::filter::describe(req.where));
        else
            return this->retrieve(req);
    }

    /// @brief retrieves the range with the given key.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve_by_key(const Key &key) const;

    /// @brief retrieves the range with the given name.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve_by_name(const std::string &name) const;

    /// @brief retrieves the ranges with the given keys.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve_by_key(const std::vector<Key> &keys) const;

    /// @brief retrieves the ranges with the given names.
    [[nodiscard]] std::pair<std::vector<Range>, x::errors::Error>
    retrieve_by_name(const std::vector<std::string> &names) const;

    /// @brief creates the given ranges.
    [[nodiscard]] x::errors::Error create(std::vector<Range> &ranges) const;

    /// @brief creates the given range.
    [[nodiscard]] x::errors::Error create(Range &range) const;

    /// @brief creates a range with the given name and time range.
    [[nodiscard]] std::pair<Range, x::errors::Error>
    create(const std::string &name, x::telem::TimeRange time_range) const;

    /// @brief sets the end bound of the range with the given key in a single request,
    /// preserving all other fields.
    [[nodiscard]] x::errors::Error
    set_end(const Key &key, x::telem::TimeStamp end) const;

private:
    [[nodiscard]] std::pair<Range, x::errors::Error>
    retrieve_one(const RetrieveRequest &req, const std::string &query) const;

    /// @brief range retrieval transport.
    std::unique_ptr<RetrieveClient> retrieve_client;
    /// @brief create retrieval transport.
    std::unique_ptr<CreateClient> create_client;
    /// @brief set-end transport.
    std::unique_ptr<SetEndClient> set_end_client;
    /// @brief range kv get transport.
    ranger::kv::Client kv;
};
}
