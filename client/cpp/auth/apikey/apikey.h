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

#include "client/cpp/auth/apikey/json.gen.h"
#include "client/cpp/auth/apikey/types.gen.h"
#include "client/cpp/auth/auth.h"
#include "client/cpp/ontology/id.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/transport/grpc/auth/auth.pb.h"

namespace synnax::auth::apikey {
/// @brief transport that creates API keys.
using CreateClient = freighter::
    UnaryClient<grpc::auth::APIKeyCreateRequest, grpc::auth::APIKeyCreateResponse>;
/// @brief transport that lists API keys.
using RetrieveClient = freighter::
    UnaryClient<grpc::auth::APIKeyRetrieveRequest, grpc::auth::APIKeyRetrieveResponse>;
/// @brief transport that deletes API keys.
using DeleteClient = freighter::
    UnaryClient<grpc::auth::APIKeyDeleteRequest, google::protobuf::Empty>;

/// @brief returns the API key in a gRPC message. Returns an error when its key is not
/// a UUID.
inline std::pair<APIKey, x::errors::Error> from_pb(const grpc::auth::APIKey &pb) {
    auto [key, err] = Key::parse(pb.key());
    if (err) return {{}, err};
    return {
        APIKey{
            .key = key,
            .subject = subject_from_pb(pb.subject()),
            .name = pb.name(),
            .created = x::telem::TimeStamp(pb.created()),
        },
        x::errors::NIL,
    };
}

/// @brief an API key just created, with the token that holds its secret.
struct New {
    /// @brief the token to log in with. It is the only copy of the key's secret.
    std::string token;
    /// @brief the stored key.
    APIKey key;
};

/// @brief creates, lists, and deletes API keys.
class Client {
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DeleteClient> delete_client;

public:
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DeleteClient> delete_client
    ):
        create_client(std::move(create_client)),
        retrieve_client(std::move(retrieve_client)),
        delete_client(std::move(delete_client)) {}

    /// @brief creates an API key for the subject, or for the caller when the subject
    /// is empty.
    [[nodiscard]] std::pair<New, x::errors::Error>
    create(const std::string &name, const ontology::ID &subject = {}) const {
        grpc::auth::APIKeyCreateRequest req;
        req.set_name(name);
        if (!subject.is_zero()) subject_to_pb(subject, req.mutable_subject());
        auto [res, err] = this->create_client->send("/auth/api-key/create", req);
        if (err) return {{}, err};
        auto [key, key_err] = from_pb(res.key());
        if (key_err) return {{}, key_err};
        return {New{res.token(), key}, x::errors::NIL};
    }

    /// @brief lists the caller's API keys, or those of the given subjects when the
    /// caller may retrieve them.
    [[nodiscard]] std::pair<std::vector<APIKey>, x::errors::Error>
    retrieve(const std::vector<ontology::ID> &subjects = {}) const {
        grpc::auth::APIKeyRetrieveRequest req;
        for (const auto &s: subjects)
            subject_to_pb(s, req.add_subjects());
        auto [res, err] = this->retrieve_client->send("/auth/api-key/retrieve", req);
        if (err) return {{}, err};
        std::vector<APIKey> keys;
        keys.reserve(res.keys_size());
        for (const auto &k: res.keys()) {
            auto [key, key_err] = from_pb(k);
            if (key_err) return {{}, key_err};
            keys.push_back(key);
        }
        return {keys, x::errors::NIL};
    }

    /// @brief deletes API keys and ends their sessions. Unknown keys are ignored.
    [[nodiscard]] x::errors::Error del(const std::vector<Key> &keys) const {
        grpc::auth::APIKeyDeleteRequest req;
        for (const auto &k: keys)
            req.add_keys(k.to_string());
        return this->delete_client->send("/auth/api-key/delete", req).second;
    }
};
}
