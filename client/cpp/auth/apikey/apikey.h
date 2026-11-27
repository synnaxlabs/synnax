// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/auth/apikey/json.gen.h"
#include "client/cpp/auth/apikey/types.gen.h"
#include "client/cpp/auth/auth.h"
#include "client/cpp/resource/id.h"
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
/// @brief transport that rotates the API key of the caller's session.
using RotateClient = freighter::
    UnaryClient<google::protobuf::Empty, grpc::auth::APIKeyCreateResponse>;

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
            .expires = x::telem::TimeStamp(pb.expires()),
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

/// @brief returns the API key and its token in a gRPC create or rotate response.
inline std::pair<New, x::errors::Error>
new_from_pb(const grpc::auth::APIKeyCreateResponse &res) {
    auto [key, err] = from_pb(res.key());
    if (err) return {{}, err};
    return {New{res.token(), key}, x::errors::NIL};
}

/// @brief creates, lists, rotates, and deletes API keys.
class Client {
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DeleteClient> delete_client;
    std::unique_ptr<RotateClient> rotate_client;
    /// @brief receives the token of each rotated key, so later logins use it.
    std::function<void(const std::string &)> on_rotate;

public:
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DeleteClient> delete_client,
        std::unique_ptr<RotateClient> rotate_client,
        std::function<void(const std::string &)> on_rotate
    ):
        create_client(std::move(create_client)),
        retrieve_client(std::move(retrieve_client)),
        delete_client(std::move(delete_client)),
        rotate_client(std::move(rotate_client)),
        on_rotate(std::move(on_rotate)) {}

    /// @brief creates an API key for the subject, or for the caller when the subject
    /// is empty. The key never expires when expires is TIME_STAMP_MAX, the default.
    [[nodiscard]] std::pair<New, x::errors::Error> create(
        const std::string &name,
        const resource::ID &subject = {},
        const x::telem::TimeStamp &expires = x::telem::TIME_STAMP_MAX
    ) const {
        grpc::auth::APIKeyCreateRequest req;
        req.set_name(name);
        if (!subject.is_zero()) subject_to_pb(subject, req.mutable_subject());
        req.set_expires(expires.nanoseconds());
        auto [res, err] = this->create_client->send("/auth/api-key/create", req);
        if (err) return {{}, err};
        return new_from_pb(res);
    }

    /// @brief replaces the API key that the client logged in with by a new key for the
    /// same subject and name, and returns the new key. The client's session stays open
    /// and later logins use the new key. The old key cannot log in again. Returns a
    /// validation error when the client did not log in with an API key, and
    /// ERR_INVALID_CREDENTIALS when the key has expired.
    [[nodiscard]] std::pair<New, x::errors::Error> rotate() const {
        google::protobuf::Empty req;
        auto [res, err] = this->rotate_client->send("/auth/api-key/rotate", req);
        if (err) return {{}, err};
        auto [rotated, rotated_err] = new_from_pb(res);
        if (rotated_err) return {{}, rotated_err};
        this->on_rotate(rotated.token);
        return {rotated, x::errors::NIL};
    }

    /// @brief lists the caller's API keys, or those of the given subjects when the
    /// caller may retrieve them.
    [[nodiscard]] std::pair<std::vector<APIKey>, x::errors::Error>
    retrieve(const std::vector<resource::ID> &subjects = {}) const {
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
