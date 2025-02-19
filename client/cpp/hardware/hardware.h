// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <utility>

/// external
#include "google/protobuf/empty.pb.h"

/// module
#include "freighter/cpp/freighter.h"

/// protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/hardware.pb.h"

namespace synnax {
/// @brief type alias for the transport used to create a rack.
using HardwareCreateRackClient = freighter::UnaryClient<
    api::v1::HardwareCreateRackRequest,
    api::v1::HardwareCreateRackResponse>;

/// @brief type alias for the transport used to retrieve a rack.
using HardwareRetrieveRackClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveRackRequest,
    api::v1::HardwareRetrieveRackResponse>;

/// @brief type alias for the transport used to delete a rack.
using HardwareDeleteRackClient = freighter::UnaryClient<
    api::v1::HardwareDeleteRackRequest,
    google::protobuf::Empty>;

/// @brief type alias for the transport used to create a task.
using HardwareCreateTaskClient = freighter::UnaryClient<
    api::v1::HardwareCreateTaskRequest,
    api::v1::HardwareCreateTaskResponse>;

/// @brief type alias for the transport used to retrieve a task.
using HardwareRetrieveTaskClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveTaskRequest,
    api::v1::HardwareRetrieveTaskResponse>;

/// @brief type alias for the transport used to delete a task.
using HardwareDeleteTaskClient = freighter::UnaryClient<
    api::v1::HardwareDeleteTaskRequest,
    google::protobuf::Empty>;

/// @brief type alias for the transport used to create a device.
using HardwareCreateDeviceClient = freighter::UnaryClient<
    api::v1::HardwareCreateDeviceRequest,
    api::v1::HardwareCreateDeviceResponse>;

/// @brief type alias for the transport used to retrieve a device.
using HardwareRetrieveDeviceClient = freighter::UnaryClient<
    api::v1::HardwareRetrieveDeviceRequest,
    api::v1::HardwareRetrieveDeviceResponse>;

/// @brief type alias for the transport used to delete a device.
using HardwareDeleteDeviceClient = freighter::UnaryClient<
    api::v1::HardwareDeleteDeviceRequest,
    google::protobuf::Empty>;

using RackKey = std::uint32_t;
using TaskKey = std::uint64_t;

inline TaskKey createTaskKey(const RackKey rack, const TaskKey task) {
    return static_cast<TaskKey>(rack) << 32U | task;
}

inline RackKey task_key_rack(const TaskKey key) { return key >> 32U; }

inline std::uint32_t task_key_local(const TaskKey key) { return key & 0xFFFFFFFFU; }


/// @brief a Task is a data structure used to configure and execute operations on a hardware device.
class Task {
public:
    TaskKey key = 0;
    std::string name;
    std::string type;
    std::string config;
    bool internal = false;

    Task(
        std::string name,
        std::string type,
        std::string config,
        bool internal = false
    );

    Task(
        TaskKey key,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false
    );

    Task(
        RackKey rack,
        std::string name,
        std::string type,
        std::string config,
        bool internal = false
    );

    explicit Task(const api::v1::Task &task);

    Task() = default;

private:
    void to_proto(api::v1::Task *task) const;

    friend class TaskClient;
};

class TaskClient {
public:
    TaskClient(
        const RackKey rack,
        std::shared_ptr<HardwareCreateTaskClient> task_create_client,
        std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client,
        std::shared_ptr<HardwareDeleteTaskClient> task_delete_client
    ) : rack(rack),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)) {
    }

    [[nodiscard]]
    xerrors::Error create(Task &task) const;

    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(std::uint64_t key) const;

    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieveByType(const std::string &type) const;

    [[nodiscard]]
    std::pair<Task, xerrors::Error> retrieve(const std::string &name) const;

    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error> retrieve(
        const std::vector<std::string> &names
    ) const;

    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error> retrieveByType(
        const std::vector<std::string> &types
    ) const;

    [[nodiscard]]
    xerrors::Error del(std::uint64_t key) const;

    [[nodiscard]]
    std::pair<std::vector<Task>, xerrors::Error> list() const;

private:
    /// @brief key of rack that this client belongs to.
    RackKey rack;
    /// @brief task creation transport.
    std::shared_ptr<HardwareCreateTaskClient> task_create_client;
    /// @brief task retrieval transport.
    std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client;
    /// @brief task deletion transport.
    std::shared_ptr<HardwareDeleteTaskClient> task_delete_client;
};

inline std::uint16_t rack_key_node(const RackKey key) { return key >> 12U; }

class Rack {
public:
    RackKey key{};
    std::string name;
    TaskClient tasks = TaskClient(0, nullptr, nullptr, nullptr);

    Rack(RackKey key, std::string name);

    explicit Rack(std::string name);

    Rack() = default;

    explicit Rack(const api::v1::Rack &rack);

    bool operator==(const Rack &rack) const { return rack.key == key; }

private:
    void to_proto(api::v1::Rack *rack) const;

    friend class HardwareClient;
};

struct Device {
    std::string key;
    std::string name;
    RackKey rack = 0;
    std::string location;
    std::string identifier;
    std::string make;
    std::string model;
    std::string properties;

    Device(
        std::string key,
        std::string name,
        RackKey rack,
        std::string location,
        std::string identifier,
        std::string make,
        std::string model,
        std::string properties
    ) : key(std::move(key)),
        name(std::move(name)),
        rack(rack),
        location(std::move(std::move(location))),
        identifier(std::move(identifier)),
        make(std::move(make)),
        model(std::move(model)),
        properties(std::move(properties)) {
    }

    Device() = default;

    explicit Device(const api::v1::Device &device);

private:
    void to_proto(api::v1::Device *device) const;

    friend class HardwareClient;
};


class HardwareClient {
public:
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
    ) : rack_create_client(std::move(rack_create_client)),
        rack_retrieve_client(std::move(rack_retrieve_client)),
        rack_delete_client(std::move(rack_delete_client)),
        task_create_client(std::move(task_create_client)),
        task_retrieve_client(std::move(task_retrieve_client)),
        task_delete_client(std::move(task_delete_client)),
        device_create_client(std::move(device_create_client)),
        device_retrieve_client(std::move(device_retrieve_client)),
        device_delete_client(std::move(device_delete_client)) {
    }


    [[nodiscard]]
    xerrors::Error create_rack(Rack &rack) const;

    [[nodiscard]]
    std::pair<Rack, xerrors::Error> create_rack(const std::string &name) const;

    [[nodiscard]]
    std::pair<Rack, xerrors::Error> retrieve_rack(std::uint32_t key) const;

    [[nodiscard]]
    std::pair<Rack, xerrors::Error> retrieve_rack(const std::string &name) const;

    [[nodiscard]]
    std::pair<Device, xerrors::Error> retrieve_device(const std::string &key) const;

    [[nodiscard]]
    xerrors::Error create_device(Device &device) const;

    [[nodiscard]]
    xerrors::Error delete_rack(std::uint32_t key) const;

private:
    /// @brief rack creation transport.
    std::unique_ptr<HardwareCreateRackClient> rack_create_client;
    /// @brief rack retrieval transport.
    std::unique_ptr<HardwareRetrieveRackClient> rack_retrieve_client;
    /// @brief rack deletion transport.
    std::unique_ptr<HardwareDeleteRackClient> rack_delete_client;
    /// @brief task creation transport.
    std::shared_ptr<HardwareCreateTaskClient> task_create_client;
    /// @brief task retrieval transport.
    std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client;
    /// @brief task deletion transport.
    std::shared_ptr<HardwareDeleteTaskClient> task_delete_client;
    /// @brief device creation transport.
    std::shared_ptr<HardwareCreateDeviceClient> device_create_client;
    /// @brief device retrieval transport.
    std::shared_ptr<HardwareRetrieveDeviceClient> device_retrieve_client;
    /// @brief device deletion transport.
    std::shared_ptr<HardwareDeleteDeviceClient> device_delete_client;
};
}
