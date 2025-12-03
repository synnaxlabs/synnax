// Copyright 2025 Synnax Labs, Inc.
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

#include "client/cpp/ontology/id.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/status/status.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/device.pb.h"

namespace synnax {
const std::string DEVICE_SET_CHANNEL = "sy_device_set";
const std::string DEVICE_DELETE_CHANNEL = "sy_device_delete";

// Forward declaration for RackKey (needed for Device struct)
using RackKey = std::uint32_t;

/// @brief Type alias for the transport used to create a device.
using DeviceCreateClient = freighter::
    UnaryClient<api::v1::DeviceCreateRequest, api::v1::DeviceCreateResponse>;

/// @brief Type alias for the transport used to retrieve a device.
using DeviceRetrieveClient = freighter::
    UnaryClient<api::v1::DeviceRetrieveRequest, api::v1::DeviceRetrieveResponse>;

/// @brief Type alias for the transport used to delete a device.
using DeviceDeleteClient = freighter::
    UnaryClient<api::v1::DeviceDeleteRequest, google::protobuf::Empty>;

/// @brief Converts a device key to an ontology ID.
/// @param key The device key.
/// @returns An ontology ID with type "device" and the given key.
inline ontology::ID device_ontology_id(const std::string &key) {
    return ontology::ID("device", key);
}

/// @brief Converts a vector of device keys to a vector of ontology IDs.
/// @param keys The device keys.
/// @returns A vector of ontology IDs.
inline std::vector<ontology::ID>
device_ontology_ids(const std::vector<std::string> &keys) {
    std::vector<ontology::ID> ids;
    ids.reserve(keys.size());
    for (const auto &key: keys)
        ids.push_back(device_ontology_id(key));
    return ids;
}

/// @brief specific status details for devices.
struct DeviceStatusDetails {
    /// @brief the rack that this device is connected to.
    RackKey rack = 0;
    /// @brief the device that this status is for.
    std::string device;

    /// @brief parses the device status details from a JSON parser.
    static DeviceStatusDetails parse(xjson::Parser parser) {
        return DeviceStatusDetails{
            .rack = parser.field<RackKey>("rack"),
            .device = parser.field<std::string>("device"),
        };
    }

    /// @brief converts the device status details to JSON.
    [[nodiscard]] json to_json() const {
        json j;
        j["rack"] = this->rack;
        j["device"] = this->device;
        return j;
    }
};

/// @brief status information about a device.
using DeviceStatus = status::Status<DeviceStatusDetails>;

/// @brief A Device represents a physical hardware device connected to a rack.
struct Device {
    /// @brief The unique identifier for the device.
    std::string key;
    /// @brief A human-readable name for the device.
    std::string name;
    /// @brief The rack that this device is connected to.
    RackKey rack = 0;
    /// @brief The physical location of the device.
    std::string location;
    /// @brief The manufacturer of the device.
    std::string make;
    /// @brief The model of the device.
    std::string model;
    /// @brief Additional properties of the device, typically in JSON format.
    std::string properties;
    /// @brief whether the device has been configured.
    bool configured = false;
    /// @brief Status information about the device.
    DeviceStatus status;

    /// @brief Constructs a new device with the given properties.
    /// @param key The unique identifier for the device.
    /// @param name A human-readable name for the device.
    /// @param rack The rack that this device is connected to.
    /// @param location The physical location of the device.
    /// @param make The manufacturer of the device.
    /// @param model The model of the device.
    /// @param properties Additional properties of the device.
    Device(
        std::string key,
        std::string name,
        RackKey rack,
        std::string location,
        std::string make,
        std::string model,
        std::string properties
    );

    /// @brief Default constructor for an empty device.
    Device() = default;

    /// @brief returns the key used for creating statuses associated with the task.
    [[nodiscard]] std::string status_key() const {
        return device_ontology_id(this->key).string();
    }

    /// @brief Constructs a device from its protobuf representation.
    /// @param device The protobuf representation of the device.
    /// @returns A pair containing the device and an error if one occurred.
    static std::pair<Device, xerrors::Error> from_proto(const api::v1::Device &device);

    /// @brief Parses a device from a JSON parser.
    /// @param parser The JSON parser containing device data.
    /// @returns The parsed device.
    static Device parse(xjson::Parser &parser);

private:
    void to_proto(api::v1::Device *device) const;

    friend class DeviceClient;
};

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
struct DeviceRetrieveOptions {
    /// @brief Whether to include status information in the retrieved devices.
    bool include_status = false;
};

/// @brief Request structure for retrieving devices with various filter options.
struct DeviceRetrieveRequest {
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

    void to_proto(api::v1::DeviceRetrieveRequest &request) const {
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
class DeviceClient {
public:
    /// @brief Constructs a new device client with the given transport clients.
    /// @param device_create_client Client for creating devices.
    /// @param device_retrieve_client Client for retrieving devices.
    /// @param device_delete_client Client for deleting devices.
    DeviceClient(
        std::unique_ptr<DeviceCreateClient> device_create_client,
        std::unique_ptr<DeviceRetrieveClient> device_retrieve_client,
        std::unique_ptr<DeviceDeleteClient> device_delete_client
    );

    /// @brief Retrieves a device by its key.
    /// @param key The key of the device to retrieve.
    /// @returns A pair containing the retrieved device and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<Device, xerrors::Error> retrieve(const std::string &key) const;

    /// @brief Retrieves a device by its key with options.
    /// @param key The key of the device to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved device and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<Device, xerrors::Error>
    retrieve(const std::string &key, const DeviceRetrieveOptions &options) const;

    /// @brief Retrieves multiple devices by their keys.
    /// @param keys The keys of the devices to retrieve.
    /// @returns A pair containing the retrieved devices and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, xerrors::Error>
    retrieve(const std::vector<std::string> &keys) const;

    /// @brief Retrieves multiple devices by their keys with options.
    /// @param keys The keys of the devices to retrieve.
    /// @param options Options for the retrieval.
    /// @returns A pair containing the retrieved devices and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, xerrors::Error> retrieve(
        const std::vector<std::string> &keys,
        const DeviceRetrieveOptions &options
    ) const;

    /// @brief Retrieves devices using a custom retrieve request.
    /// @param req The retrieve request with filter criteria.
    /// @returns A pair containing the retrieved devices and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, xerrors::Error>
    retrieve(DeviceRetrieveRequest &req) const;

    /// @brief Creates a device in the cluster.
    /// @param device The device to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create(Device &device) const;

    /// @brief Creates multiple devices in the cluster.
    /// @param devs The devices to create. Will be updated with the assigned keys.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create(const std::vector<Device> &devs) const;

    /// @brief Deletes a device by its key.
    /// @param key The key of the device to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error del(const std::string &key) const;

    /// @brief Deletes multiple devices by their keys.
    /// @param keys The keys of the devices to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error del(const std::vector<std::string> &keys) const;

private:
    /// @brief Device creation transport.
    std::unique_ptr<DeviceCreateClient> device_create_client;
    /// @brief Device retrieval transport.
    std::unique_ptr<DeviceRetrieveClient> device_retrieve_client;
    /// @brief Device deletion transport.
    std::unique_ptr<DeviceDeleteClient> device_delete_client;
};

}
