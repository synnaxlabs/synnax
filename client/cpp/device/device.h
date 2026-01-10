// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/device/json.gen.h"
#include "client/cpp/device/proto.gen.h"
#include "client/cpp/device/types.gen.h"
#include "client/cpp/ontology/id.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/status/status.h"

#include "core/pkg/api/grpc/device/device.pb.h"

namespace synnax::device {
const std::string DEVICE_SET_CHANNEL = "sy_device_set";
const std::string DEVICE_DELETE_CHANNEL = "sy_device_delete";

/// @brief Type alias for rack keys used in device context.
using RackKey = ::synnax::rack::Key;

/// @brief Type alias for the transport used to create a device.
using CreateClient = freighter::
    UnaryClient<grpc::device::CreateRequest, grpc::device::CreateResponse>;

/// @brief Type alias for the transport used to retrieve a device.
using RetrieveClient = freighter::
    UnaryClient<grpc::device::RetrieveRequest, grpc::device::RetrieveResponse>;

/// @brief Type alias for the transport used to delete a device.
using DeleteClient = freighter::
    UnaryClient<grpc::device::DeleteRequest, google::protobuf::Empty>;

/// @brief Converts a device key to an ontology ID.
/// @param key The device key.
/// @returns An ontology ID with type "device" and the given key.
inline ontology::ID ontology_id(const std::string &key) {
    return ontology::ID("device", key);
}

/// @brief Converts a vector of device keys to a vector of ontology IDs.
/// @param keys The device keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID> ontology_ids(const std::vector<std::string> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(ontology_id(key));
    return ids;
}

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

/// @brief Options for retrieving devices.
struct RetrieveOptions {
    /// @brief Whether to include status information in the retrieved devices.
    bool include_status = false;
};

/// @brief Request structure for retrieving devices with various filter options.
struct RetrieveRequest {
    std::vector<std::string> keys;
    std::vector<std::string> names;
    std::vector<std::string> makes;
    std::vector<std::string> models;
    std::vector<std::string> locations;
    std::vector<RackKey> racks;
    std::string search;
    std::uint32_t limit = 0;
    std::uint32_t offset = 0;
    bool ignore_not_found = false;
    bool include_status = false;

    void to_proto(grpc::device::RetrieveRequest &request) const {
        request.set_ignore_not_found(ignore_not_found);
        request.set_limit(limit);
        request.set_offset(offset);
        request.mutable_keys()->Add(keys.begin(), keys.end());
        request.mutable_names()->Add(names.begin(), names.end());
        request.mutable_makes()->Add(makes.begin(), makes.end());
        request.mutable_models()->Add(models.begin(), models.end());
        request.mutable_locations()->Add(locations.begin(), locations.end());
        request.mutable_racks()->Add(racks.begin(), racks.end());
        request.set_search(search);
        request.set_include_status(include_status);
    }
};

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

    /// @brief Retrieves a device by its key.
    /// @param key The key of the device to retrieve.
    /// @returns A pair containing the retrieved device and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<Device, x::errors::Error> retrieve(const std::string &key) const;

    /// @brief Retrieves a device by its key with options.
    /// @param key The key of the device to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved device and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<Device, x::errors::Error>
    retrieve(const std::string &key, const RetrieveOptions &options) const;

    /// @brief Retrieves multiple devices by their keys.
    /// @param keys The keys of the devices to retrieve.
    /// @returns A pair containing the retrieved devices and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const std::vector<std::string> &keys) const;

    /// @brief Retrieves multiple devices by their keys with options.
    /// @param keys The keys of the devices to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved devices and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error> retrieve(
        const std::vector<std::string> &keys,
        const RetrieveOptions &options
    ) const;

    /// @brief Retrieves devices using a custom retrieve request.
    /// @param req The retrieve request with filter criteria.
    /// @returns A pair containing the retrieved devices and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, x::errors::Error>
    retrieve(RetrieveRequest &req) const;

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
    /// @brief Device creation transport.
    std::unique_ptr<CreateClient> device_create_client;
    /// @brief Device retrieval transport.
    std::unique_ptr<RetrieveClient> device_retrieve_client;
    /// @brief Device deletion transport.
    std::unique_ptr<DeleteClient> device_delete_client;
};

}
