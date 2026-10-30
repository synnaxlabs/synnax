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

#include "client/cpp/auth/auth.h"
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

/// @brief an API key as the Core returns it, without its secret.
struct APIKey {
    /// @brief the unique key of the API key.
    std::string key;
    /// @brief the user or rack the key logs in as.
    Subject subject;
    /// @brief a label that tells the subject's keys apart.
    std::string name;
    /// @brief when the key was created.
    x::telem::TimeStamp created = x::telem::TimeStamp(0);

    APIKey() = default;

    explicit APIKey(const grpc::auth::APIKey &k):
        key(k.key()),
        subject{k.subject().type(), k.subject().key()},
        name(k.name()),
        created(k.created()) {}
};

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
    create(const std::string &name, const Subject &subject = {}) const {
        grpc::auth::APIKeyCreateRequest req;
        req.set_name(name);
        if (!subject.type.empty()) {
            req.mutable_subject()->set_type(subject.type);
            req.mutable_subject()->set_key(subject.key);
        }
        auto [res, err] = this->create_client->send("/auth/api-key/create", req);
        if (err) return {{}, err};
        return {New{res.token(), APIKey(res.key())}, x::errors::NIL};
    }

    /// @brief lists the caller's API keys, or those of the given subjects when the
    /// caller may retrieve them.
    [[nodiscard]] std::pair<std::vector<APIKey>, x::errors::Error>
    retrieve(const std::vector<Subject> &subjects = {}) const {
        grpc::auth::APIKeyRetrieveRequest req;
        for (const auto &s: subjects) {
            auto *id = req.add_subjects();
            id->set_type(s.type);
            id->set_key(s.key);
        }
        auto [res, err] = this->retrieve_client->send("/auth/api-key/retrieve", req);
        if (err) return {{}, err};
        std::vector<APIKey> keys;
        keys.reserve(res.keys_size());
        for (const auto &k: res.keys())
            keys.emplace_back(k);
        return {keys, x::errors::NIL};
    }

    /// @brief deletes API keys and ends their sessions. Unknown keys are ignored.
    [[nodiscard]] x::errors::Error del(const std::vector<std::string> &keys) const {
        grpc::auth::APIKeyDeleteRequest req;
        for (const auto &k: keys)
            req.add_keys(k);
        return this->delete_client->send("/auth/api-key/delete", req).second;
    }
};
}
