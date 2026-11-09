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

#include "client/cpp/device/actions.gen.h"
#include "client/cpp/device/json.gen.h"
#include "client/cpp/device/proto.gen.h"
#include "client/cpp/device/types.gen.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/status/status.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"
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

using DispatchClient = freighter::
    UnaryClient<grpc::device::DispatchRequest, google::protobuf::Empty>;

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

/// @brief Client for managing devices in a Synnax cluster.
class Client {
public:
    /// @brief Constructs a new device client with the given transport clients.
    /// @param device_create_client Client for creating devices.
    /// @param device_retrieve_client Client for retrieving devices.
    /// @param device_delete_client Client for deleting devices.
    /// @param device_dispatch_client Client for dispatching actions to devices.
    Client(
        std::unique_ptr<CreateClient> device_create_client,
        std::unique_ptr<RetrieveClient> device_retrieve_client,
        std::unique_ptr<DeleteClient> device_delete_client,
        std::unique_ptr<DispatchClient> device_dispatch_client
    );

    /// @brief retrieves the device with the given key.
    /// @param key the key of the device.
    /// @param options the options of the retrieve.
    /// @returns the device, and an error that matches NOT_FOUND when no device has the
    /// key.
    [[nodiscard]]
    std::pair<Device, x::errors::Error>
    retrieve(const std::string &key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the devices with the given keys.
    /// @param keys the keys of the devices.
    /// @param options the options of the retrieve.
    /// @returns the devices for the keys that exist, and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error> retrieve(
        const std::vector<std::string> &keys,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief retrieves the devices that where selects.
    /// @param where the filter that the devices match. An empty filter matches all
    /// devices.
    /// @param options the options of the retrieve.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one device that where selects.
    /// @param where the filter that the device matches.
    /// @param options the options of the retrieve.
    /// @returns the device, and an error that matches NOT_FOUND when no device
    /// matches, MULTIPLE_RESULTS when more than one does, or VALIDATION when options
    /// set a limit or a cursor.
    [[nodiscard]]
    std::pair<Device, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

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

    /// @brief applies the actions to the device with the given key in order, as one
    /// atomic edit.
    /// @param key the key of the device.
    /// @param actions the actions to apply.
    /// @returns an error that matches NOT_FOUND when no device has the key, or
    /// VALIDATION when the device cannot hold an action.
    [[nodiscard]]
    x::errors::Error dispatch(const Key &key, const std::vector<Action> &actions) const;

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
    /// @brief Device creation transport.
    std::unique_ptr<CreateClient> device_create_client;
    /// @brief Device retrieval transport.
    std::unique_ptr<RetrieveClient> device_retrieve_client;
    /// @brief Device deletion transport.
    std::unique_ptr<DeleteClient> device_delete_client;
    /// @brief Device dispatch transport.
    std::unique_ptr<DispatchClient> device_dispatch_client;
};

}
