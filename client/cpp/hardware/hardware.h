// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "client/cpp/telem/telem.h"
#include "freighter/cpp/freighter.h"
#include "google/protobuf/empty.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/hardware.pb.h"

using namespace synnax;

namespace synnax {
/// @brief type alias for the transport used to create a rack.
    typedef freighter::UnaryClient<
        api::v1::HardwareCreateRackRequest,
        api::v1::HardwareCreateRackResponse
    > HardwareCreateRackClient;

/// @brief type alias for the transport used to retrieve a rack.
    typedef freighter::UnaryClient<
        api::v1::HardwareRetrieveRackRequest,
        api::v1::HardwareRetrieveRackResponse
    > HardwareRetrieveRackClient;

/// @brief type alias for the transport used to delete a rack.
    typedef freighter::UnaryClient<
        api::v1::HardwareDeleteRackRequest,
        google::protobuf::Empty
    > HardwareDeleteRackClient;

/// @brief type alias for the transport used to create a task.
    typedef freighter::UnaryClient<
        api::v1::HardwareCreateTaskRequest,
        api::v1::HardwareCreateTaskResponse
    > HardwareCreateTaskClient;

/// @brief type alias for the transport used to retrieve a task.
    typedef freighter::UnaryClient<
        api::v1::HardwareRetrieveTaskRequest,
        api::v1::HardwareRetrieveTaskResponse
    > HardwareRetrieveTaskClient;

/// @brief type alias for the transport used to delete a task.
    typedef freighter::UnaryClient<
        api::v1::HardwareDeleteTaskRequest,
        google::protobuf::Empty
    > HardwareDeleteTaskClient;

/// @brief type alias for the transport used to create a device.
    typedef freighter::UnaryClient<
        api::v1::HardwareCreateDeviceRequest,
        api::v1::HardwareCreateDeviceResponse
    > HardwareCreateDeviceClient;

/// @brief type alias for the transport used to retrieve a device.
    typedef freighter::UnaryClient<
        api::v1::HardwareRetrieveDeviceRequest,
        api::v1::HardwareRetrieveDeviceResponse
    > HardwareRetrieveDeviceClient;

/// @brief type alias for the transport used to delete a device.
    typedef freighter::UnaryClient<
        api::v1::HardwareDeleteDeviceRequest,
        google::protobuf::Empty
    > HardwareDeleteDeviceClient;


    typedef std::uint32_t RackKey;

    typedef std::uint64_t TaskKey;

    inline TaskKey createTaskKey(RackKey rack, std::uint32_t task) {
        return static_cast<TaskKey>(rack) << 32 | task;
    }

    inline RackKey taskKeyRack(TaskKey key) { return key >> 32; }

    inline std::uint32_t taskKeyLocal(TaskKey key) { return key & 0xFFFFFFFF; }


/// @brief a Task is a data structure used to configure and execute operations on a hardware device.
    class Task {
    public:
        TaskKey key;
        std::string name;
        std::string type;
        std::string config;
        bool internal;

        Task(std::string name, std::string type, std::string config,
             bool internal = false);

        Task(TaskKey key, std::string name, std::string type, std::string config,
             bool internal = false);

        Task(RackKey rack, std::string name, std::string type, std::string config,
             bool internal = false);

        explicit Task(const api::v1::Task &task);


        Task() = default;

    private:
        void to_proto(api::v1::Task *task) const;


        friend class TaskClient;
    };

    class TaskClient {
    public:
        TaskClient(
            RackKey rack,
            std::shared_ptr<HardwareCreateTaskClient> task_create_client,
            std::shared_ptr<HardwareRetrieveTaskClient> task_retrieve_client,
            std::shared_ptr<HardwareDeleteTaskClient> task_delete_client
        ) : rack(rack),
            task_create_client(std::move(task_create_client)),
            task_retrieve_client(std::move(task_retrieve_client)),
            task_delete_client(std::move(task_delete_client)) {
        }

        [[nodiscard]]
        freighter::Error create(Task &task) const;

        [[nodiscard]]
        std::pair<Task, freighter::Error> retrieve(std::uint64_t key) const;

        [[nodiscard]]
        std::pair<Task, freighter::Error> retrieveByType(const std::string &type) const;

        [[nodiscard]]
        std::pair<Task, freighter::Error> retrieve(const std::string &name) const;

        [[nodiscard]]
        std::pair<std::vector<Task>, freighter::Error> retrieve(
            const std::vector<std::string> &names
        ) const;

        [[nodiscard]]
        std::pair<std::vector<Task>, freighter::Error> retrieveByType(
            const std::vector<std::string> &types
        ) const;

        [[nodiscard]]
        freighter::Error del(std::uint64_t key) const;

        [[nodiscard]]
        std::pair<std::vector<Task>, freighter::Error> list() const;

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

    inline std::uint16_t rackKeyNode(RackKey key) { return key >> 12; }

    class Rack {
    public:
        RackKey key{};
        std::string name;
        TaskClient tasks = TaskClient(0, nullptr, nullptr, nullptr);

        Rack(RackKey key, std::string name);

        Rack(std::string name);

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
        RackKey rack;
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
        ) : key(key), name(name), rack(rack), location(location),
            identifier(identifier),
            make(make), model(model), properties(properties) {
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
        freighter::Error createRack(Rack &rack) const;

        [[nodiscard]]
        std::pair<Rack, freighter::Error> createRack(const std::string &name) const;

        [[nodiscard]]
        std::pair<Rack, freighter::Error> retrieveRack(std::uint32_t key) const;

        [[nodiscard]]
        std::pair<Rack, freighter::Error> retrieveRack(const std::string &name) const;

        [[nodiscard]]
        std::pair<Device, freighter::Error>
        retrieveDevice(const std::string &key) const;

        [[nodiscard]]
        freighter::Error createDevice(Device &device) const;

        [[nodiscard]]
        freighter::Error deleteRack(std::uint32_t key) const;

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
