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
#include <utility>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "client/cpp/labjack/device/actions.gen.h"
#include "client/cpp/labjack/device/proto.gen.h"
#include "client/cpp/labjack/device/types.gen.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"

#include "core/pkg/api/labjack/device/pb/labjack_device.pb.h"
#include "core/pkg/transport/grpc/labjack/device/device.pb.h"

namespace synnax::labjack::device {
/// @brief the channel on which the Core publishes each LabJack device that a write
/// sets.
const std::string SET_CHANNEL = "sy_labjack_device_set";
/// @brief the channel on which the Core publishes the key of each deleted LabJack
/// device.
const std::string DELETE_CHANNEL = "sy_labjack_device_delete";

/// @brief the transport that creates LabJack devices.
using CreateClient = freighter::UnaryClient<
    grpc::labjack_device::CreateRequest,
    grpc::labjack_device::CreateResponse>;

/// @brief the transport that retrieves LabJack devices.
using RetrieveClient = freighter::UnaryClient<
    ::api::labjack_device::pb::RetrieveRequest,
    ::api::labjack_device::pb::RetrieveResponse>;

/// @brief the transport that applies actions to a LabJack device.
using DispatchClient = freighter::
    UnaryClient<grpc::labjack_device::DispatchRequest, google::protobuf::Empty>;

/// @brief the transport that deletes LabJack devices.
using DeleteClient = freighter::
    UnaryClient<grpc::labjack_device::DeleteRequest, google::protobuf::Empty>;

/// @brief creates, retrieves, dispatches to, and deletes LabJack devices in a Core.
class Client {
public:
    Client(
        std::unique_ptr<CreateClient> create_client,
        std::unique_ptr<RetrieveClient> retrieve_client,
        std::unique_ptr<DispatchClient> dispatch_client,
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

    /// @brief applies the actions to the device with the given key in order, as one
    /// atomic edit.
    /// @param key the key of the device.
    /// @param actions the actions to apply.
    /// @returns an error that matches NOT_FOUND when no device has the key, or
    /// VALIDATION when an action fails or the result is invalid.
    [[nodiscard]] x::errors::Error
    dispatch(const Key &key, const std::vector<Action> &actions) const;

    /// @brief deletes the devices with the given keys.
    /// @param keys the keys of the devices.
    /// @returns an error if the delete failed.
    [[nodiscard]] x::errors::Error del(const std::vector<Key> &keys) const;

private:
    std::unique_ptr<CreateClient> create_client;
    std::unique_ptr<RetrieveClient> retrieve_client;
    std::unique_ptr<DispatchClient> dispatch_client;
    std::unique_ptr<DeleteClient> delete_client;
};
}
