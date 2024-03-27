// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <string>
#include <vector>

/// freighter
#include "freighter/cpp/freighter/freighter.h"

/// api protos
#include "google/protobuf/empty.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/hardware.pb.h"


/// internal
#include "client/cpp/synnax/telem/telem.h"

using namespace synnax;

namespace synnax {


/// @brief type alias for the transport used to create a rack.
typedef freighter::UnaryClient<
        api::v1::HardwareCreateRackResponse,
        api::v1::HardwareCreateRackRequest
> HardwareCreateRackClient;

/// @brief type alias for the transport used to retrieve a rack.
typedef freighter::UnaryClient<
        api::v1::HardwareRetrieveRackResponse,
        api::v1::HardwareRetrieveRackRequest
> HardwareRetrieveRackClient;

/// @brief type alias for the transport used to delete a rack.
typedef freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::HardwareDeleteRackRequest
> HardwareDeleteRackClient;

/// @brief type alias for the transport used to create a task.
typedef freighter::UnaryClient<
        api::v1::HardwareCreateTaskResponse,
        api::v1::HardwareCreateTaskRequest
> HardwareCreateTaskClient;

/// @brief type alias for the transport used to retrieve a task.
typedef freighter::UnaryClient<
        api::v1::HardwareRetrieveTaskResponse,
        api::v1::HardwareRetrieveTaskRequest
> HardwareRetrieveTaskClient;

/// @brief type alias for the transport used to delete a task.
typedef freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::HardwareDeleteTaskRequest
> HardwareDeleteTaskClient;

class RackKey {
public:
    std::uint32_t value = 0;

    RackKey(std::uint32_t value) : value(value) {}

    RackKey(std::uint16_t node_key, std::uint16_t local_key) :
            value((node_key << 16) | local_key) {}

    std::uint16_t node_key() const { return value >> 16; }

    std::uint16_t local_key() const { return value & 0xFFFF; }

    RackKey() = default;
};

struct TaskKey {
    std::uint64_t value = 0;

    TaskKey(std::uint64_t value) : value(value) {}

    TaskKey(RackKey rack_key, std::uint32_t local_key) :
            value((static_cast<std::uint64_t>(rack_key.value) << 32) | local_key) {}

    operator std::uint64_t() const {
        return value;
    }

    RackKey rack_key() const {
        return RackKey(value >> 32);
    }

    std::uint32_t local_key() const {
        return value & 0xFFFFFFFF;
    }

    TaskKey() = default;
};

class Task {
public:
    TaskKey key;
    std::string name;
    std::string type;
    std::string config;

    Task(TaskKey key, std::string name, std::string type, std::string config);

    Task(RackKey rack, std::string name, std::string type, std::string config);

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
    ) :
            rack(rack),
            task_create_client(std::move(task_create_client)),
            task_retrieve_client(std::move(task_retrieve_client)),
            task_delete_client(std::move(task_delete_client)) {}

    [[nodiscard]]
    freighter::Error create(Task &task) const;

    [[nodiscard]]
    std::pair<Task, freighter::Error> retrieve(std::uint64_t key) const;

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

class Rack {
public:
    RackKey key;
    std::string name;
    TaskClient tasks = TaskClient(0, nullptr, nullptr, nullptr);

    Rack(RackKey key, std::string name);

    Rack(std:: string name);

    explicit Rack(const api::v1::Rack &rack);

    bool operator==(const Rack &rack) const { return rack.key.value == key.value; }

private:
    void to_proto(api::v1::Rack *rack) const;

    Rack() = default;

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
            std::shared_ptr<HardwareDeleteTaskClient> task_delete_client
    ) :
            rack_create_client(std::move(rack_create_client)),
            rack_retrieve_client(std::move(rack_retrieve_client)),
            rack_delete_client(std::move(rack_delete_client)),
            task_create_client(std::move(task_create_client)),
            task_retrieve_client(std::move(task_retrieve_client)),
            task_delete_client(std::move(task_delete_client)) {}


    [[nodiscard]]
    freighter::Error createRack(Rack &rack) const;

    [[nodiscard]]
    std::pair<Rack, freighter::Error> createRack(const std::string &name) const;

    [[nodiscard]]
    std::pair<Rack, freighter::Error> retrieveRack(std::uint32_t key) const;

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
};
}
