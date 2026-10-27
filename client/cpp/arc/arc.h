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

#include "google/protobuf/empty.pb.h"

#include "client/cpp/arc/builder.gen.h"
#include "client/cpp/arc/proto.gen.h"
#include "client/cpp/arc/types.gen.h"
#include "freighter/cpp/freighter.h"

#include "core/pkg/api/arc/pb/arc.pb.h"
#include "core/pkg/transport/grpc/arc/arc.pb.h"

namespace synnax::arc {

/// @brief Freighter client for creating Arc programs.
using CreateClient = freighter::
    UnaryClient<grpc::arc::CreateRequest, grpc::arc::CreateResponse>;

/// @brief Freighter client for retrieving Arc programs.
using RetrieveClient = freighter::
    UnaryClient<grpc::arc::RetrieveRequest, ::api::arc::pb::RetrieveResponse>;

/// @brief Freighter client for deleting Arc programs.
using DeleteClient = freighter::
    UnaryClient<grpc::arc::DeleteRequest, google::protobuf::Empty>;

/// @brief Options for retrieving Arc programs.
struct RetrieveOptions {
    /// @brief If true, the Core compiles each Arc and sets its program.
    bool include_program = false;
};

/// @brief Client for managing Arc automation programs in a Synnax cluster.
/// @details Provides methods to create, retrieve, and delete Arc programs.
/// Arc programs can contain visual graph representations and/or text-based source code.
class Client {
public:
    /// @brief Constructs an empty Arc client (invalid).
    Client() = default;

    /// @brief Constructs an Arc client with the given transport clients.
    /// @param retrieve_client Client for retrieving Arc programs.
    /// @param create_client Client for creating Arc programs.
    /// @param delete_client Client for deleting Arc programs.
    Client(
        std::shared_ptr<RetrieveClient> retrieve_client,
        std::shared_ptr<CreateClient> create_client,
        std::shared_ptr<DeleteClient> delete_client
    );

    /// @brief Creates a new Arc program in the Synnax cluster.
    /// @param arc The Arc program to create. The key will be assigned by the server.
    /// @modifies arc Assigns a unique key to the Arc program.
    /// @returns An error if the Arc program could not be created.
    [[nodiscard]] x::errors::Error create(Arc &arc) const;

    /// @brief Creates multiple Arc programs in the Synnax cluster.
    /// @details More efficient than calling create() individually and provides
    /// atomicity.
    /// @param arcs Vector of Arc programs to create.
    /// @modifies arcs Assigns unique keys to each Arc program.
    /// @returns An error if the Arc programs could not be created.
    [[nodiscard]] x::errors::Error create(std::vector<Arc> &arcs) const;

    /// @brief Creates a new Arc program with the given name.
    /// @param name Human-readable name for the Arc program.
    /// @returns A pair containing the created Arc program and an error.
    /// In case of error, the returned Arc will be invalid.
    [[nodiscard]] std::pair<Arc, x::errors::Error>
    create(const std::string &name) const;

    /// @brief Retrieves an Arc program by its name.
    /// @param name The name of the Arc program to retrieve.
    /// @param options Optional retrieve options.
    /// @returns A pair containing the retrieved Arc program and an error.
    /// If the Arc program does not exist or multiple programs have the same name,
    /// an error is returned.
    [[nodiscard]] std::pair<Arc, x::errors::Error> retrieve_by_name(
        const std::string &name,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves an Arc program by its key (UUID).
    /// @param key The key of the Arc program to retrieve.
    /// @param options Optional retrieve options.
    /// @returns A pair containing the retrieved Arc program and an error.
    /// If the Arc program does not exist, an error is returned.
    [[nodiscard]] std::pair<Arc, x::errors::Error> retrieve_by_key(
        const x::uuid::UUID &key,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves Arc programs by their names.
    /// @param names Vector of names of Arc programs to retrieve.
    /// @param options Optional retrieve options.
    /// @returns A pair containing a vector of retrieved Arc programs and an error.
    /// If an Arc program with a given name does not exist, it will not be in the
    /// result.
    [[nodiscard]] std::pair<std::vector<Arc>, x::errors::Error> retrieve(
        const std::vector<std::string> &names,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves Arc programs by their keys (UUIDs).
    /// @param keys Vector of keys of Arc programs to retrieve.
    /// @param options Optional retrieve options.
    /// @returns A pair containing a vector of retrieved Arc programs and an error.
    /// If an Arc program with a given key does not exist, it will not be in the result.
    [[nodiscard]] std::pair<std::vector<Arc>, x::errors::Error> retrieve_by_keys(
        const std::vector<x::uuid::UUID> &keys,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves the Arc programs that req selects.
    [[nodiscard]] std::pair<std::vector<Arc>, x::errors::Error>
    retrieve(const RetrieveRequest &req) const;

    /// @brief Retrieves the Arc programs that match every condition in parts, with each
    /// option in parts applied.
    /// @returns the Arc programs, or one Arc program when parts end with arc::one().
    /// One Arc program is an error when no Arc program or more than one Arc program
    /// matches.
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

    /// @brief Deletes an Arc program by its key.
    /// @param key The key of the Arc program to delete.
    /// @returns An error if the Arc program could not be deleted.
    [[nodiscard]] x::errors::Error del(const x::uuid::UUID &key) const;

    /// @brief Deletes multiple Arc programs by their keys.
    /// @param keys Vector of keys of Arc programs to delete.
    /// @returns An error if the Arc programs could not be deleted.
    [[nodiscard]] x::errors::Error del(const std::vector<x::uuid::UUID> &keys) const;

private:
    [[nodiscard]] std::pair<Arc, x::errors::Error>
    retrieve_one(const RetrieveRequest &req, const std::string &query) const;

    /// @brief Client for retrieving Arc programs.
    std::shared_ptr<RetrieveClient> retrieve_client;

    /// @brief Client for creating Arc programs.
    std::shared_ptr<CreateClient> create_client;

    /// @brief Client for deleting Arc programs.
    std::shared_ptr<DeleteClient> delete_client;
};

}
