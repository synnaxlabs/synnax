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

#include "freighter/cpp/freighter.h"

#include "arc/cpp/graph/graph.h"
#include "arc/cpp/module/module.h"
#include "arc/cpp/text/text.h"
#include "core/pkg/api/grpc/arc/arc.pb.h"

namespace synnax {

/// @brief Freighter client for creating Arc programs.
using ArcCreateClient = freighter::
    UnaryClient<grpc::arc::CreateRequest, grpc::arc::CreateResponse>;

/// @brief Freighter client for retrieving Arc programs.
using ArcRetrieveClient = freighter::
    UnaryClient<grpc::arc::RetrieveRequest, grpc::arc::RetrieveResponse>;

/// @brief Freighter client for deleting Arc programs.
using ArcDeleteClient = freighter::
    UnaryClient<grpc::arc::DeleteRequest, google::protobuf::Empty>;

class ArcClient;

/// @brief Options for retrieving Arc programs.
struct RetrieveOptions {
    /// @brief If true, compiles the Arc text to a module with IR and WASM bytecode.
    bool compile = false;

    /// @brief If true, includes the runtime status of the Arc program.
    bool include_status = false;

    /// @brief Maximum number of results to return (0 = unlimited).
    int32_t limit = 0;

    /// @brief Number of results to skip before returning.
    int32_t offset = 0;

    /// @brief Search term for filtering Arc programs by name.
    std::string search_term;

    /// @brief Applies these options to a protobuf retrieve request.
    void apply(grpc::arc::RetrieveRequest &req) const {
        req.set_compile(compile);
        req.set_include_status(include_status);
        if (limit > 0) req.set_limit(limit);
        if (offset > 0) req.set_offset(offset);
        if (!search_term.empty()) req.set_search_term(search_term);
    }
};

/// @brief Represents an Arc automation program.
/// @details Arc is a domain-specific language for control systems. An Arc program
/// contains both a visual graph representation and text-based source code.
/// See https://docs.synnaxlabs.com/reference/concepts/arc for more information.
struct Arc {
    /// @brief Unique identifier for the Arc program (UUID).
    std::string key;

    /// @brief Human-readable name for the Arc program.
    std::string name;

    /// @brief Visual graph representation of the Arc program.
    arc::graph::Graph graph;

    /// @brief Text-based source code representation.
    arc::text::Text text;

    /// @brief Compiled module with IR and WASM bytecode.
    arc::module::Module module;

    /// @brief Whether the Arc program should be deployed and running.
    bool deploy = false;

    /// @brief Version string for the Arc program.
    std::string version;

    /// @brief Constructs an empty, invalid Arc program.
    Arc() = default;

    /// @brief Constructs a new Arc program with the given name.
    /// @param name Human-readable name for the Arc program.
    explicit Arc(std::string name);

    /// @brief Constructs an Arc program from its protobuf representation.
    /// @param pb Protobuf message representing the Arc program.
    explicit Arc(const grpc::arc::Arc &pb);

private:
    /// @brief Converts the Arc program to its protobuf representation.
    /// @param pb Pointer to protobuf message to populate.
    void to_proto(grpc::arc::Arc *pb) const;

    friend class ArcClient;
};

/// @brief Client for managing Arc automation programs in a Synnax cluster.
/// @details Provides methods to create, retrieve, and delete Arc programs.
/// Arc programs can contain visual graph representations and/or text-based source code.
class ArcClient {
public:
    /// @brief Constructs an empty Arc client (invalid).
    ArcClient() = default;

    /// @brief Constructs an Arc client with the given transport clients.
    /// @param retrieve_client Client for retrieving Arc programs.
    /// @param create_client Client for creating Arc programs.
    /// @param delete_client Client for deleting Arc programs.
    ArcClient(
        std::shared_ptr<ArcRetrieveClient> retrieve_client,
        std::shared_ptr<ArcCreateClient> create_client,
        std::shared_ptr<ArcDeleteClient> delete_client
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
    [[nodiscard]] std::pair<Arc, x::errors::Error> create(const std::string &name) const;

    /// @brief Retrieves an Arc program by its name.
    /// @param name The name of the Arc program to retrieve.
    /// @param options Optional retrieve options (compile, include_status, etc.).
    /// @returns A pair containing the retrieved Arc program and an error.
    /// If the Arc program does not exist or multiple programs have the same name,
    /// an error is returned.
    [[nodiscard]] std::pair<Arc, x::errors::Error> retrieve_by_name(
        const std::string &name,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves an Arc program by its key (UUID).
    /// @param key The key of the Arc program to retrieve.
    /// @param options Optional retrieve options (compile, include_status, etc.).
    /// @returns A pair containing the retrieved Arc program and an error.
    /// If the Arc program does not exist, an error is returned.
    [[nodiscard]] std::pair<Arc, x::errors::Error>
    retrieve_by_key(const std::string &key, const RetrieveOptions &options = {}) const;

    /// @brief Retrieves Arc programs by their names.
    /// @param names Vector of names of Arc programs to retrieve.
    /// @param options Optional retrieve options (compile, include_status, etc.).
    /// @returns A pair containing a vector of retrieved Arc programs and an error.
    /// If an Arc program with a given name does not exist, it will not be in the
    /// result.
    [[nodiscard]] std::pair<std::vector<Arc>, x::errors::Error> retrieve(
        const std::vector<std::string> &names,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Retrieves Arc programs by their keys (UUIDs).
    /// @param keys Vector of keys of Arc programs to retrieve.
    /// @param options Optional retrieve options (compile, include_status, etc.).
    /// @returns A pair containing a vector of retrieved Arc programs and an error.
    /// If an Arc program with a given key does not exist, it will not be in the result.
    [[nodiscard]] std::pair<std::vector<Arc>, x::errors::Error> retrieve_by_keys(
        const std::vector<std::string> &keys,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief Deletes an Arc program by its key.
    /// @param key The key of the Arc program to delete.
    /// @returns An error if the Arc program could not be deleted.
    [[nodiscard]] x::errors::Error delete_arc(const std::string &key) const;

    /// @brief Deletes multiple Arc programs by their keys.
    /// @param keys Vector of keys of Arc programs to delete.
    /// @returns An error if the Arc programs could not be deleted.
    [[nodiscard]] x::errors::Error delete_arc(const std::vector<std::string> &keys) const;

private:
    /// @brief Client for retrieving Arc programs.
    std::shared_ptr<ArcRetrieveClient> retrieve_client;

    /// @brief Client for creating Arc programs.
    std::shared_ptr<ArcCreateClient> create_client;

    /// @brief Client for deleting Arc programs.
    std::shared_ptr<ArcDeleteClient> delete_client;
};

}
