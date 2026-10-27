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
#include <unordered_map>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/device/builder.gen.h"
#include "client/cpp/device/json.gen.h"
#include "client/cpp/device/proto.gen.h"
#include "client/cpp/device/types.gen.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/status/status.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/json/struct.h"

#include "core/pkg/api/device/pb/device.pb.h"
#include "core/pkg/transport/grpc/device/device.pb.h"

namespace synnax::device {
const std::string SET_CHANNEL = "sy_device_set";
const std::string DELETE_CHANNEL = "sy_device_delete";

/// @brief Type alias for the transport used to create a device.
using CreateClient = freighter::
    UnaryClient<grpc::device::CreateRequest, grpc::device::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a device.
using RetrieveClient = freighter::
    UnaryClient<grpc::device::RetrieveRequest, ::api::device::pb::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a device.
using DeleteClient = freighter::
    UnaryClient<grpc::device::DeleteRequest, google::protobuf::Empty>;

/// @brief Creates a map of device keys to devices.
/// @param devices The devices to map.
/// @returns A map from device keys to devices.
inline std::unordered_map<std::string, Device>
map_device_keys(const std::vector<Device> &devices) {
    std::unordered_map<std::string, Device> map;
    map.reserve(devices.size());
    for (const auto &device: devices)
        map[device.key] = device;
    return map;
}

/// @brief Returns a unique status key for a device.
/// @param device The device.
/// @returns A unique key for status updates, derived from the device key.
inline std::string status_key(const Device &device) {
    return ontology_id(device.key).string();
}

/// @brief Client for managing devices in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new device client with the given transport clients.
    /// @param device_create_client Client for creating devices.
    /// @param device_retrieve_client Client for retrieving devices.
    /// @param device_delete_client Client for deleting devices.
    Client(
        std::unique_ptr<CreateClient> device_create_client,
        std::unique_ptr<RetrieveClient> device_retrieve_client,
        std::unique_ptr<DeleteClient> device_delete_client
    );

    /// @brief Retrieves the device with the given key, with each include option in
    /// includes applied.
    /// @returns the device, or a NOT_FOUND error when no device has the key.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<Device, x::errors::Error>
    retrieve(const std::string &key, Includes... includes) const {
        return this->retrieve_one(
            x::filter::request<Where, RetrieveRequest>(
                device::key.eq(key),
                includes...
            ),
            "key " + key
        );
    }

    /// @brief Retrieves the devices with the given keys, with each include option in
    /// includes applied. A key that no device has is left out.
    template<typename... Includes>
        requires x::filter::includes<RetrieveRequest, Includes...>
    [[nodiscard]] std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const std::vector<std::string> &keys, Includes... includes) const {
        if (keys.empty()) return {std::vector<Device>(), x::errors::NIL};
        return this->retrieve(
            x::filter::request<Where, RetrieveRequest>(
                device::key.in(keys),
                includes...
            )
        );
    }

    /// @brief Retrieves the devices that req selects.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const RetrieveRequest &req) const;

    /// @brief Retrieves the devices that match every condition in parts, with each
    /// option in parts applied.
    /// @returns the devices, or one device when parts end with device::one(). One
    /// device is an error when no device or more than one device matches.
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

    /// @brief Creates a device in the cluster.
    /// @param device The device to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    x::errors::Error create(Device &device) const;

    /// @brief Creates multiple devices in the cluster.
    /// @param devs The devices to create. Will be updated with the assigned keys.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    x::errors::Error create(const std::vector<Device> &devs) const;

    /// @brief Deletes a device by its key.
    /// @param key The key of the device to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(const std::string &key) const;

    /// @brief Deletes multiple devices by their keys.
    /// @param keys The keys of the devices to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    x::errors::Error del(const std::vector<std::string> &keys) const;

private:
    [[nodiscard]]
    std::pair<Device, x::errors::Error>
    retrieve_one(const RetrieveRequest &req, const std::string &query) const;

    /// @brief Device creation transport.
    std::unique_ptr<CreateClient> device_create_client;
    /// @brief Device retrieval transport.
    std::unique_ptr<RetrieveClient> device_retrieve_client;
    /// @brief Device deletion transport.
    std::unique_ptr<DeleteClient> device_delete_client;
};

}
