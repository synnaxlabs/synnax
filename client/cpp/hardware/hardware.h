// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/status/status.h"
#include "x/cpp/xjson/xjson.h"

#include "core/pkg/api/grpc/v1/hardware.pb.h"

namespace synnax {
/// @brief Type alias for the transport used to create a rack.
using HardwareCreateRackClient = freighter::UnaryClient<
    api::v1::HardwareCreateRackRequest,
    api::v1::HardwareCreateRackResponse>;

/// @brief Type alias for the transport used to retrieve a rack.
using HardwareRetrieveRackClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveRackRequest,
    api::v1::HardwareRetrieveRackResponse>;

/// @brief Type alias for the transport used to delete a rack.
using HardwareDeleteRackClient = freighter::
    UnaryClient<api::v1::HardwareDeleteRackRequest, google::protobuf::Empty>;

/// @brief Type alias for the transport used to create a task.
using HardwareCreateTaskClient = freighter::UnaryClient<
    api::v1::HardwareCreateTaskRequest,
    api::v1::HardwareCreateTaskResponse>;

/// @brief Type alias for the transport used to retrieve a task.
using HardwareRetrieveTaskClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveTaskRequest,
    api::v1::HardwareRetrieveTaskResponse>;

/// @brief Type alias for the transport used to delete a task.
using HardwareDeleteTaskClient = freighter::
    UnaryClient<api::v1::HardwareDeleteTaskRequest, google::protobuf::Empty>;

/// @brief Type alias for the transport used to create a device.
using HardwareCreateDeviceClient = freighter::UnaryClient<
    api::v1::HardwareCreateDeviceRequest,
    api::v1::HardwareCreateDeviceResponse>;

/// @brief Type alias for the transport used to retrieve a device.
using HardwareRetrieveDeviceClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveDeviceRequest,
    api::v1::HardwareRetrieveDeviceResponse>;

/// @brief Type alias for the transport used to delete a device.
using HardwareDeleteDeviceClient = freighter::
    UnaryClient<api::v1::HardwareDeleteDeviceRequest, google::protobuf::Empty>;

/// @brief An alias for the type of rack's key.
using RackKey = std::uint32_t;

/// @brief An alias for the type of task's key.
using TaskKey = std::uint64_t;

/// @brief the name of the channel used to propagate device state.
const std::string DEVICE_STATUS_CHANNEL_NAME = "sy_device_status";
/// @brief the name of the channel used to propagate task state.
const std::string TASK_STATUS_CHANNEL_NAME = "sy_task_status";
/// @brief the name of the channel used to propagate rack state.
const std::string RACK_STATUS_CHANNEL_NAME = "sy_rack_status";

/// @brief Creates a task key from a rack key and a local task key.
/// @param rack The rack key.
/// @param task The local task key.
/// @returns A combined task key.
inline TaskKey create_task_key(const RackKey rack, const TaskKey task) {
    return static_cast<TaskKey>(rack) << 32 | task;
}

/// @brief Extracts the rack key from a task key.
/// @param key The task key.
/// @returns The rack key portion of the task key.
inline RackKey rack_key_from_task_key(const TaskKey key) {
    return key >> 32;
}

/// @brief Extracts the local task key from a task key.
/// @param key The task key.
/// @returns The local task key portion of the task key.
inline std::uint32_t local_task_key(const TaskKey key) {
    return key & 0xFFFFFFFF;
}

/// @brief A Task is a data structure used to configure and execute operations on a
/// hardware device. Tasks are associated with a specific rack and can be created,
/// retrieved, and deleted.
class Task {
public:
    /// @brief The unique identifier for the task.
    TaskKey key = 0;
    /// @brief A human-readable name for the task.
    std::string name;
    /// @brief The type of the task, which determines its behavior.
    std::string type;
    /// @brief Configuration data for the task, typically in JSON format.
    std::string config;
    /// @brief Whether the task is internal to the system.
    bool internal = false;
    /// @brief Whether the task is a snapshot.
    bool snapshot = false;

    /// @brief Constructs a new task with the given properties.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    /// @brief Constructs a new task with the given properties and key.
    /// @param key The unique identifier for the task.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        TaskKey key,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    /// @brief Constructs a new task with the given properties and rack.
    /// @param rack The rack key that this task belongs to.
    /// @param name A human-readable name for the task.
    /// @param type The type of the task.
    /// @param config Configuration data for the task.
    /// @param internal Whether the task is internal to the system.
    /// @param snapshot Whether the task is a snapshot and cannot be modified.
    Task(
        RackKey rack,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false,
        bool snapshot = false
    );

    /// @brief Constructs a task from its protobuf representation.
    /// @param task The protobuf representation of the task.
    explicit Task(const api::v1::Task &task);

    /// @brief Default constructor for an empty task.
    Task() = default;

    friend std::ostream &operator<<(std::ostream &os, const Task &task) {
        return os << task.name << " (" << task.key << ")";
    }

private:
    /// @brief Converts the task to its protobuf representation.
    /// @param task The protobuf object to populate.
    void to_proto(api::v1::Task *task) const;

    friend class TaskClient;
};

/// @brief Client for managing tasks on a specific rack.
class TaskClient {
public:
    /// @brief Constructs a new task client for the given rack.
    /// @param rack The rack key that this client operates on.
    /// @param task_create_client Client for creating tasks.
    /// @param task_retrieve_client Client for retrieving tasks.
    /// @param task_delete_client Client for deleting tasks.
    TaskClient(
        const RackKey rack,
        std::shared_ptr<HardwareCreateTaskClient> task_create_client,
        std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client,
        std::shared_ptr<HardwareDeleteTaskClient> task_delete_client
    ):
        rack(rack),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)) {}

    /// @brief Creates a task on the rack.
    /// @param task The task to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create(Task &task) const;

    /// @brief Retrieves a task by its key.
    /// @param key The key of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(std::uint64_t key) const;

    /// @brief Retrieves a task by its type.
    /// @param type The type of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve_by_type(const std::string &type) const;

    /// @brief Retrieves a task by its name.
    /// @param name The name of the task to retrieve.
    /// @returns A pair containing the retrieved task and an error if one occurred.
    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(const std::string &name) const;

    /// @brief Retrieves multiple tasks by their names.
    /// @param names The names of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error>
    retrieve(const std::vector<std::string> &names) const;

    /// @brief Retrieves multiple tasks by their types.
    /// @param types The types of the tasks to retrieve.
    /// @returns A pair containing the retrieved tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error>
    retrieve_by_type(const std::vector<std::string> &types) const;

    /// @brief Deletes a task by its key.
    /// @param key The key of the task to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error del(std::uint64_t key) const;

    /// @brief Lists all tasks on the rack.
    /// @returns A pair containing the list of tasks and an error if one occurred.
    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error> list() const;

private:
    /// @brief Key of rack that this client belongs to.
    RackKey rack;
    /// @brief Task creation transport.
    std::shared_ptr<HardwareCreateTaskClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<HardwareDeleteTaskClient> task_delete_client;
};

/// @brief Extracts the node ID from a rack key.
/// @param key The rack key.
/// @returns The node ID portion of the rack key.
inline std::uint16_t rack_key_node(const RackKey key) {
    return key >> 12;
}

/// @brief A Rack represents a physical or logical grouping of hardware devices.
/// Racks contain tasks that can be used to interact with hardware.
class Rack {
public:
    /// @brief The unique identifier for the rack.
    RackKey key{};

    /// @brief A human-readable name for the rack.
    std::string name;

    /// @brief Client for managing tasks on this rack.
    TaskClient tasks = TaskClient(0, nullptr, nullptr, nullptr);

    /// @brief Constructs a new rack with the given key and name.
    /// @param key The unique identifier for the rack.
    /// @param name A human-readable name for the rack.
    Rack(RackKey key, std::string name);

    /// @brief Constructs a new rack with the given name.
    /// @param name A human-readable name for the rack.
    explicit Rack(std::string name);

    /// @brief Default constructor for an empty rack.
    Rack() = default;

    /// @brief Constructs a rack from its protobuf representation.
    /// @param rack The protobuf representation of the rack.
    explicit Rack(const api::v1::Rack &rack);

    /// @brief Equality operator for racks.
    /// @param rack The rack to compare with.
    /// @returns True if the racks have the same key.
    bool operator==(const Rack &rack) const { return rack.key == key; }

private:
    /// @brief Converts the rack to its protobuf representation.
    /// @param rack The protobuf object to populate.
    void to_proto(api::v1::Rack *rack) const;

    friend class HardwareClient;
};

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

/// @brief specific status details for devices.
struct RackStatusDetails {
    /// @brief the rack that this device is connected to.
    RackKey rack = 0;

    /// @brief parses the device status details from a JSON parser.
    static DeviceStatusDetails parse(xjson::Parser parser) {
        return DeviceStatusDetails{
            .rack = parser.field<RackKey>("rack"),
        };
    }

    /// @brief converts the device status details to JSON.
    [[nodiscard]] json to_json() const {
        json j;
        j["rack"] = this->rack;
        return j;
    }
};

/// @brief specific status details for tasks.
struct TaskStatusDetails {
    /// @brief The key of the task that this status is for.
    TaskKey task;
    /// @brief whether the task is currently running.
    bool running;
    /// @brief additional data associated with the task.
    json data;

    /// @brief parses the task status details from a JSON parser.
    static TaskStatusDetails parse(xjson::Parser parser) {
        return TaskStatusDetails{
            .task = parser.field<TaskKey>("task"),
            .running = parser.field<bool>("running"),
            .data = parser.field<json>("data"),
        };
    }

    /// @brief converts the task status details to JSON.
    [[nodiscard]] json to_json() const {
        json j;
        j["task"] = this->task;
        j["running"] = this->running;
        j["data"] = this->data;
        return j;
    }
};

/// @brief status information about a device.
using DeviceStatus = status::Status<DeviceStatusDetails>;
/// @brief status information for a device.
using RackStatus = status::Status<RackStatusDetails>;
/// @brief status information for a task.
using TaskStatus = status::Status<TaskStatusDetails>;

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
    /// @brief The state of the device.
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

    /// @brief Constructs a device from its protobuf representation.
    /// @param device The protobuf representation of the device.
    explicit Device(const api::v1::Device &device);

private:
    void to_proto(api::v1::Device *device) const;

    friend class HardwareClient;
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

struct HardwareDeviceRetrieveRequest {
    std::vector<std::string> keys;
    std::vector<std::string> names;
    std::vector<std::string> makes;
    std::vector<std::string> models;
    std::vector<std::string> locations;
    std::vector<RackKey> racks;
    std::string search;
    std::uint32_t limit;
    std::uint32_t offset;
    bool ignore_not_found = false;

    void to_proto(api::v1::HardwareRetrieveDeviceRequest &request) const {
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
    }
};

/// @brief Client for managing hardware components in a Synnax cluster.
/// Provides methods for creating, retrieving, and deleting racks, tasks, and
/// devices.
class HardwareClient {
public:
    /// @brief Constructs a new hardware client with the given transport clients.
    /// @param rack_create_client Client for creating racks.
    /// @param rack_retrieve_client Client for retrieving racks.
    /// @param rack_delete_client Client for deleting racks.
    /// @param task_create_client Client for creating tasks.
    /// @param task_retrieve_client Client for retrieving tasks.
    /// @param task_delete_client Client for deleting tasks.
    /// @param device_create_client Client for creating devices.
    /// @param device_retrieve_client Client for retrieving devices.
    /// @param device_delete_client Client for deleting devices.
    HardwareClient(
        std::unique_ptr<HardwareCreateRackClient> rack_create_client,
        std::unique_ptr<HardwareRetrieveRackClient> rack_retrieve_client,
        std::unique_ptr<HardwareDeleteRackClient> rack_delete_client,
        std::shared_ptr<HardwareCreateTaskClient> task_create_client,
        std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client,
        std::shared_ptr<HardwareDeleteTaskClient> task_delete_client,
        std::unique_ptr<HardwareCreateDeviceClient> device_create_client,
        std::unique_ptr<HardwareRetrieveDeviceClient> device_retrieve_client,
        std::unique_ptr<HardwareDeleteDeviceClient> device_delete_client
    ):
        rack_create_client(std::move(rack_create_client)),
        rack_retrieve_client(std::move(rack_retrieve_client)),
        rack_delete_client(std::move(rack_delete_client)),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)),
        device_create_client(std::move(device_create_client)),
        device_retrieve_client(std::move(device_retrieve_client)),
        device_delete_client(std::move(device_delete_client)) {}

    /// @brief Creates a rack in the cluster.
    /// @param rack The rack to create. Will be updated with the assigned key and
    /// task client.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create_rack(Rack &rack) const;

    /// @brief Creates a rack with the given name in the cluster.
    /// @param name The name of the rack to create.
    /// @returns A pair containing the created rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, xerrors::Error> create_rack(const std::string &name) const;

    /// @brief Retrieves a rack by its key.
    /// @param key The key of the rack to retrieve.
    /// @returns A pair containing the retrieved rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, xerrors::Error> retrieve_rack(std::uint32_t key) const;

    /// @brief Retrieves a rack by its name.
    /// @param name The name of the rack to retrieve.
    /// @returns A pair containing the retrieved rack and an error if one occurred.
    [[nodiscard]]
    std::pair<Rack, xerrors::Error> retrieve_rack(const std::string &name) const;

    /// @brief Retrieves a device by its key.
    /// @param key The key of the device to retrieve.
    /// @param ignore_not_found If true, returns an empty device without error when
    /// not found.
    /// @returns A pair containing the retrieved device and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<Device, xerrors::Error>
    retrieve_device(const std::string &key, bool ignore_not_found = false) const;

    /// @brief Retrieves multiple devices by their keys.
    /// @param keys The keys of the devices to retrieve.
    /// @param ignore_not_found If true, skips non-existent devices without error.
    /// @returns A pair containing the retrieved devices and an error if one
    /// occurred.
    [[nodiscard]]
    std::pair<std::vector<Device>, xerrors::Error> retrieve_devices(
        const std::vector<std::string> &keys,
        bool ignore_not_found = false
    ) const;

    std::pair<std::vector<Device>, xerrors::Error>
    retrieve_devices(HardwareDeviceRetrieveRequest &req) const;

    /// @brief Creates a device in the cluster.
    /// @param device The device to create. Will be updated with the assigned key.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create_device(Device &device) const;

    /// @brief Creates multiple devices in the cluster.
    /// @param devs The devices to create. Will be updated with the assigned keys.
    /// @returns An error if the creation failed.
    [[nodiscard]]
    xerrors::Error create_devices(const std::vector<Device> &devs) const;

    /// @brief Deletes a rack by its key.
    /// @param key The key of the rack to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error delete_rack(std::uint32_t key) const;

    /// @brief Deletes a device by its key.
    /// @param key The key of the device to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error delete_device(const std::string &key) const;

    /// @brief Deletes multiple devices by their keys.
    /// @param keys The keys of the devices to delete.
    /// @returns An error if the deletion failed.
    [[nodiscard]]
    xerrors::Error delete_devices(const std::vector<std::string> &keys) const;

private:
    /// @brief Rack creation transport.
    std::unique_ptr<HardwareCreateRackClient> rack_create_client;
    /// @brief Rack retrieval transport.
    std::unique_ptr<HardwareRetrieveRackClient> rack_retrieve_client;
    /// @brief Rack deletion transport.
    std::unique_ptr<HardwareDeleteRackClient> rack_delete_client;
    /// @brief Task creation transport.
    std::shared_ptr<HardwareCreateTaskClient> task_create_client;
    /// @brief Task retrieval transport.
    std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client;
    /// @brief Task deletion transport.
    std::shared_ptr<HardwareDeleteTaskClient> task_delete_client;
    /// @brief Device creation transport.
    std::shared_ptr<HardwareCreateDeviceClient> device_create_client;
    /// @brief Device retrieval transport.
    std::shared_ptr<HardwareRetrieveDeviceClient> device_retrieve_client;
    /// @brief Device deletion transport.
    std::shared_ptr<HardwareDeleteDeviceClient> device_delete_client;
};

}
