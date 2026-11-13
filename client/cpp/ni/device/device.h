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
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/ni/device/proto.gen.h"
#include "client/cpp/ni/device/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"

#include "core/pkg/api/ni/device/pb/ni_device.pb.h"
#include "core/pkg/transport/grpc/ni/device/device.pb.h"

namespace synnax::ni::device {
/// @brief the transport that creates NI devices.
using CreateClient = freighter::
    UnaryClient<grpc::ni_device::CreateRequest, grpc::ni_device::CreateResponse>;

/// @brief the transport that retrieves NI devices.
using RetrieveClient = freighter::UnaryClient<
    ::api::ni_device::pb::RetrieveRequest,
    ::api::ni_device::pb::RetrieveResponse>;

/// @brief the transport that writes one field group to an NI device.
using UpdateClient = freighter::
    UnaryClient<grpc::ni_device::UpdateRequest, google::protobuf::Empty>;

/// @brief the transport that deletes NI devices.
using DeleteClient = freighter::
    UnaryClient<grpc::ni_device::DeleteRequest, google::protobuf::Empty>;

/// @brief creates, retrieves, updates, and deletes NI devices in a Core.
class Client {
public:
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<UpdateClient> update_client,
        std::unique_ptr<DeleteClient> delete_client
    );

    /// @brief creates the devices, and sets the key the Core assigns to each one.
    /// @param devices the devices to create.
    /// @returns an error if the create failed.
    [[nodiscard]] x::errors::Error create(std::vector<Device> &devices) const;

    /// @brief creates the device, and sets the key the Core assigns to it.
    /// @param device the device to create.
    /// @returns an error if the create failed.
    [[nodiscard]] x::errors::Error create(Device &device) const;

    /// @brief retrieves the device with the given key.
    /// @param key the key of the device.
    /// @param options the options of the retrieve.
    /// @returns the device, and an error that matches NOT_FOUND when no device has
    /// the key.
    [[nodiscard]] std::pair<Device, x::errors::Error>
    retrieve(const Key &key, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the devices that where selects.
    /// @param where the filter that the devices match. An empty filter matches all
    /// devices.
    /// @param options the options of the retrieve.
    [[nodiscard]] std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const Where &where, const RetrieveOptions &options = {}) const;

    /// @brief retrieves the one device that where selects.
    /// @param where the filter that the device matches.
    /// @param options the options of the retrieve.
    /// @returns the device, and an error that matches NOT_FOUND when no device
    /// matches, MULTIPLE_RESULTS when more than one does, or VALIDATION when options
    /// set a limit or a cursor.
    [[nodiscard]] std::pair<Device, x::errors::Error> retrieve(
        const Where &where,
        x::filter::One,
        const RetrieveOptions &options = {}
    ) const;

    /// @brief writes one field group to the device with the given key, and keeps the
    /// other group as stored.
    /// @param key the key of the device.
    /// @param update the field group to write.
    /// @returns an error that matches NOT_FOUND when no device has the key.
    [[nodiscard]] x::errors::Error update(const Key &key, const Update &update) const;

    /// @brief deletes the devices with the given keys.
    /// @param keys the keys of the devices.
    /// @returns an error if the delete failed.
    [[nodiscard]] x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<UpdateClient> update_client;
    std::unique_ptr<DeleteClient> delete_client;
};
}
