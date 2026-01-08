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

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/status/status.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/api/grpc/status/status.pb.h"

namespace synnax {

const std::string STATUS_SET_CHANNEL_NAME = "sy_status_set";

/// @brief Freighter retrieve transport.
using StatusRetrieveClient = freighter::
    UnaryClient<grpc::status::RetrieveRequest, grpc::status::RetrieveResponse>;

/// @brief Freighter set transport.
using StatusSetClient = freighter::
    UnaryClient<grpc::status::SetRequest, grpc::status::SetResponse>;

/// @brief Freighter delete transport.
using StatusDeleteClient = freighter::
    UnaryClient<grpc::status::DeleteRequest, google::protobuf::Empty>;

/// @brief StatusClient for creating, retrieving, and deleting statuses in a Synnax
/// cluster.
class StatusClient {
public:
    StatusClient() = default;

    StatusClient(
        std::shared_ptr<StatusRetrieveClient> retrieve_client,
        std::shared_ptr<StatusSetClient> set_client,
        std::shared_ptr<StatusDeleteClient> delete_client
    ):
        retrieve_client(std::move(retrieve_client)),
        set_client(std::move(set_client)),
        delete_client(std::move(delete_client)) {}

    /// @brief Creates or updates the given status in the Synnax cluster.
    /// @tparam Details The type of custom details for the status.
    /// @param status The status to create or update.
    /// @modifies status May update the key if auto-generated.
    /// @returns An error where ok() is false if the status could not be created.
    /// Use err.message() to get the error message or err.type to get the error type.
    template<typename Details = json>
    [[nodiscard]] x::errors::Error set(status::Status<Details> &status) const {
        grpc::status::SetRequest req;
        *req.add_statuses() = status.to_proto();
        auto [res, err] = this->set_client->send("/status/set", req);
        if (err) return err;
        if (res.statuses_size() == 0) return unexpected_missing_error("status");
        auto [decoded, decode_err] = status::Status<Details>::from_proto(
            res.statuses(0)
        );
        if (decode_err) return decode_err;
        status = decoded;
        return x::errors::NIL;
    }

    /// @brief Creates or updates the given statuses in the Synnax cluster.
    /// @tparam Details The type of custom details for the statuses.
    /// @details More efficient than calling set on each status individually.
    /// @param statuses The statuses to create or update.
    /// @modifies statuses May update keys if auto-generated.
    /// @returns An error where ok() is false if the statuses could not be created.
    /// Use err.message() to get the error message or err.type to get the error type.
    template<typename Details = json>
    [[nodiscard]] x::errors::Error
    set(std::vector<status::Status<Details>> &statuses) const {
        grpc::status::SetRequest req;
        req.mutable_statuses()->Reserve(static_cast<int>(statuses.size()));
        for (const auto &s: statuses)
            *req.add_statuses() = s.to_proto();
        auto [res, err] = this->set_client->send("/status/set", req);
        if (err) return err;
        for (int i = 0; i < res.statuses_size(); i++) {
            auto [decoded, decode_err] = status::Status<Details>::from_proto(
                res.statuses(i)
            );
            if (decode_err) return decode_err;
            statuses[i] = decoded;
        }
        return x::errors::NIL;
    }

    /// @brief Retrieves a status with the given key.
    /// @tparam Details The type of custom details expected in the retrieved status.
    /// @param key The key of the status to retrieve.
    /// @returns A pair containing the retrieved status and an error where ok() is
    /// false if the status could not be retrieved. In the case of an error, the
    /// returned status will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    template<typename Details = json>
    [[nodiscard]] std::pair<status::Status<Details>, x::errors::Error>
    retrieve(const std::string &key) const {
        auto [statuses, err] = this->retrieve<Details>(std::vector{key});
        if (err) return {status::Status<Details>{}, err};
        if (statuses.empty()) {
            return {status::Status<Details>(), not_found_error("status", "key " + key)};
        }
        return {statuses[0], x::errors::NIL};
    }

    /// @brief Retrieves statuses with the given keys.
    /// @tparam Details The type of custom details expected in the retrieved statuses.
    /// @param keys The keys of the statuses to retrieve.
    /// @returns A pair containing all statuses matching the given keys and an error
    /// where ok() is false if the statuses could not be retrieved. Statuses that
    /// don't exist will not be in the returned vector.
    template<typename Details = json>
    [[nodiscard]] std::pair<std::vector<status::Status<Details>>, x::errors::Error>
    retrieve(const std::vector<std::string> &keys) const {
        grpc::status::RetrieveRequest req;
        req.mutable_keys()->Add(keys.begin(), keys.end());
        auto [res, err] = this->retrieve_client->send("/status/retrieve", req);
        if (err) return {std::vector<status::Status<Details>>(), err};
        std::vector<status::Status<Details>> statuses;
        statuses.reserve(res.statuses_size());
        for (const auto &pb_status: res.statuses()) {
            auto [decoded, decode_err] = status::Status<Details>::from_proto(pb_status);
            if (decode_err) return {std::vector<status::Status<Details>>(), decode_err};
            statuses.push_back(decoded);
        }
        return {statuses, x::errors::NIL};
    }

    /// @brief Deletes a status with the given key.
    /// @details This operation is idempotent - deleting a non-existent status will
    /// not raise an error.
    /// @param key The key of the status to delete.
    /// @returns An error where ok() is false if the status could not be deleted.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] x::errors::Error del(const std::string &key) const {
        return this->del(std::vector{key});
    }

    /// @brief Deletes statuses with the given keys.
    /// @details This operation is idempotent - deleting non-existent statuses will
    /// not raise an error.
    /// @param keys The keys of the statuses to delete.
    /// @returns An error where ok() is false if the statuses could not be deleted.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] x::errors::Error del(const std::vector<std::string> &keys) const {
        grpc::status::DeleteRequest req;
        req.mutable_keys()->Add(keys.begin(), keys.end());
        return this->delete_client->send("/status/delete", req).second;
    }

private:
    std::shared_ptr<StatusRetrieveClient> retrieve_client;
    std::shared_ptr<StatusSetClient> set_client;
    std::shared_ptr<StatusDeleteClient> delete_client;
};

}
