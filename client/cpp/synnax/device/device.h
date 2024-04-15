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
#include <memory>

/// freighter
#include "freighter/cpp/freighter/freighter.h"


/// api protos
#include "google/protobuf/empty.pb.h"
#include "synnax/pkg/api/grpc/v1/device.pb.h"


/// internal
#include "client/cpp/synnax/telem/telem.h"

using namespace synnax;

namespace synnax {


/// @brief type alias for the transport used to create a rack.
typedef freighter::UnaryClient<
        api::v1::DeviceCreateRackResponse,
        api::v1::DeviceCreateRackRequest
> DeviceCreateRackClient;

/// @brief type alias for the transport used to retrieve a rack.
typedef freighter::UnaryClient<
        api::v1::DeviceRetrieveRackResponse,
        api::v1::DeviceRetrieveRackRequest
> DeviceRetrieveRackClient;

/// @brief type alias for the transport used to delete a rack.
typedef freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::DeviceDeleteRackRequest
> DeviceDeleteRackClient;

/// @brief type alias for the transport used to create a module.
typedef freighter::UnaryClient<
        api::v1::DeviceCreateModuleResponse,
        api::v1::DeviceCreateModuleRequest
> DeviceCreateModuleClient;

/// @brief type alias for the transport used to retrieve a module.
typedef freighter::UnaryClient<
        api::v1::DeviceRetrieveModuleResponse,
        api::v1::DeviceRetrieveModuleRequest
> DeviceRetrieveModuleClient;

/// @brief type alias for the transport used to delete a module.
typedef freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::DeviceDeleteModuleRequest
> DeviceDeleteModuleClient;

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

struct ModuleKey {
    std::uint64_t value = 0;

    ModuleKey(std::uint64_t value) : value(value) {}

    ModuleKey(RackKey rack_key, std::uint32_t local_key) :
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

    ModuleKey() = default;
};

class Module {
public:
    ModuleKey key;
    std::string name;
    std::string type;
    std::string config;

    Module(ModuleKey key, std::string name, std::string type, std::string config);

    Module(RackKey rack, std::string name, std::string type, std::string config);

    explicit Module(const api::v1::Module &module);


    Module() = default;

private:
    void to_proto(api::v1::Module *module) const;


    friend class ModuleClient;
};

class ModuleClient {
public:
    ModuleClient(
            RackKey rack,
            std::shared_ptr<DeviceCreateModuleClient> module_create_client,
            std::shared_ptr<DeviceRetrieveModuleClient> module_retrieve_client,
            std::shared_ptr<DeviceDeleteModuleClient> module_delete_client
    ) :
            rack(rack),
            module_create_client(std::move(module_create_client)),
            module_retrieve_client(std::move(module_retrieve_client)),
            module_delete_client(std::move(module_delete_client)) {}

    [[nodiscard]]
    freighter::Error create(Module &module) const;

    [[nodiscard]]
    std::pair<Module, freighter::Error> retrieve(std::uint64_t key) const;

    [[nodiscard]]
    freighter::Error del(std::uint64_t key) const;

    [[nodiscard]]
    std::pair<std::vector<Module>, freighter::Error> list() const;

private:
    /// @brief key of rack that this client belongs to.
    RackKey rack;
    /// @brief module creation transport.
    std::shared_ptr<DeviceCreateModuleClient> module_create_client;
    /// @brief module retrieval transport.
    std::shared_ptr<DeviceRetrieveModuleClient> module_retrieve_client;
    /// @brief module deletion transport.
    std::shared_ptr<DeviceDeleteModuleClient> module_delete_client;
};

class Rack {
public:
    RackKey key;
    std::string name;
    ModuleClient modules = ModuleClient(0, nullptr, nullptr, nullptr);

    Rack(RackKey key, std::string name);

    Rack(std:: string name);

    explicit Rack(const api::v1::Rack &rack);

    bool operator==(const Rack &rack) const { return rack.key.value == key.value; }

private:
    void to_proto(api::v1::Rack *rack) const;

    Rack() = default;

    friend class DeviceClient;
};


class DeviceClient {
public:
    DeviceClient(
            std::unique_ptr<DeviceCreateRackClient> rack_create_client,
            std::unique_ptr<DeviceRetrieveRackClient> rack_retrieve_client,
            std::unique_ptr<DeviceDeleteRackClient> rack_delete_client,
            std::shared_ptr<DeviceCreateModuleClient> module_create_client,
            std::shared_ptr<DeviceRetrieveModuleClient> module_retrieve_client,
            std::shared_ptr<DeviceDeleteModuleClient> module_delete_client
    ) :
            rack_create_client(std::move(rack_create_client)),
            rack_retrieve_client(std::move(rack_retrieve_client)),
            rack_delete_client(std::move(rack_delete_client)),
            module_create_client(std::move(module_create_client)),
            module_retrieve_client(std::move(module_retrieve_client)),
            module_delete_client(std::move(module_delete_client)) {}


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
    std::unique_ptr<DeviceCreateRackClient> rack_create_client;
    /// @brief rack retrieval transport.
    std::unique_ptr<DeviceRetrieveRackClient> rack_retrieve_client;
    /// @brief rack deletion transport.
    std::unique_ptr<DeviceDeleteRackClient> rack_delete_client;
    /// @brief module creation transport.
    std::shared_ptr<DeviceCreateModuleClient> module_create_client;
    /// @brief module retrieval transport.
    std::shared_ptr<DeviceRetrieveModuleClient> module_retrieve_client;
    /// @brief module deletion transport.
    std::shared_ptr<DeviceDeleteModuleClient> module_delete_client;
};
}
