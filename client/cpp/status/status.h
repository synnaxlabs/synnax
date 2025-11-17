// Copyright 2025 Synnax Labs, Inc.
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

#include "freighter/cpp/freighter.h"
#include "x/cpp/status/status.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "core/pkg/api/grpc/v1/status.pb.h"

namespace synnax {

class StatusClient;

/// @brief Alias for status with default details (no custom details).
using Status = status::Status<status::DefaultDetails>;

/// @brief Freighter retrieve transport.
using StatusRetrieveClient = freighter::
    UnaryClient<api::v1::StatusRetrieveRequest, api::v1::StatusRetrieveResponse>;

/// @brief Freighter set transport.
using StatusSetClient = freighter::
    UnaryClient<api::v1::StatusSetRequest, api::v1::StatusSetResponse>;

/// @brief Freighter delete transport.
using StatusDeleteClient = freighter::
    UnaryClient<api::v1::StatusDeleteRequest, google::protobuf::Empty>;

/// @brief StatusClient for creating, retrieving, and deleting statuses in a Synnax
/// cluster.
class StatusClient {
public:
    StatusClient() = default;

    StatusClient(
        std::shared_ptr<StatusRetrieveClient> retrieve_client,
        std::shared_ptr<StatusSetClient> set_client,
        std::shared_ptr<StatusDeleteClient> delete_client
    );

    /// @brief Creates or updates the given status in the Synnax cluster.
    /// @param status The status to create or update.
    /// @modifies status May update the key if auto-generated.
    /// @returns An error where ok() is false if the status could not be created.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] xerrors::Error set(Status &status) const;

    /// @brief Creates or updates the given statuses in the Synnax cluster.
    /// @details More efficient than calling set on each status individually.
    /// @param statuses The statuses to create or update.
    /// @modifies statuses May update keys if auto-generated.
    /// @returns An error where ok() is false if the statuses could not be created.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] xerrors::Error set(std::vector<Status> &statuses) const;

    /// @brief Retrieves a status with the given key.
    /// @param key The key of the status to retrieve.
    /// @returns A pair containing the retrieved status and an error where ok() is
    /// false if the status could not be retrieved. In the case of an error, the
    /// returned status will be invalid. Use err.message() to get the error message
    /// or err.type to get the error type.
    [[nodiscard]] std::pair<Status, xerrors::Error>
    retrieve(const std::string &key) const;

    /// @brief Retrieves statuses with the given keys.
    /// @param keys The keys of the statuses to retrieve.
    /// @returns A pair containing all statuses matching the given keys and an error
    /// where ok() is false if the statuses could not be retrieved. Statuses that
    /// don't exist will not be in the returned vector.
    [[nodiscard]] std::pair<std::vector<Status>, xerrors::Error>
    retrieve(const std::vector<std::string> &keys) const;

    /// @brief Deletes a status with the given key.
    /// @details This operation is idempotent - deleting a non-existent status will
    /// not raise an error.
    /// @param key The key of the status to delete.
    /// @returns An error where ok() is false if the status could not be deleted.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] xerrors::Error del(const std::string &key) const;

    /// @brief Deletes statuses with the given keys.
    /// @details This operation is idempotent - deleting non-existent statuses will
    /// not raise an error.
    /// @param keys The keys of the statuses to delete.
    /// @returns An error where ok() is false if the statuses could not be deleted.
    /// Use err.message() to get the error message or err.type to get the error type.
    [[nodiscard]] xerrors::Error del(const std::vector<std::string> &keys) const;

private:
    std::shared_ptr<StatusRetrieveClient> retrieve_client;
    std::shared_ptr<StatusSetClient> set_client;
    std::shared_ptr<StatusDeleteClient> delete_client;
};

} // namespace synnax
