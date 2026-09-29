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

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/can/gs_usb/api.h"

namespace driver::can::gs_usb {
/// @brief the libusb shared library, loaded at runtime, and one context it owns.
class ProdAPI final : public API {
    std::unique_ptr<x::lib::Shared> lib;
    libusb_context *ctx = nullptr;
    decltype(&libusb_init) init = nullptr;
    decltype(&libusb_exit) exit = nullptr;
    decltype(&libusb_get_device_list) get_device_list = nullptr;
    decltype(&libusb_free_device_list) free_device_list = nullptr;
    decltype(&libusb_get_device_descriptor) get_device_descriptor = nullptr;
    decltype(&libusb_get_bus_number) get_bus_number = nullptr;
    decltype(&libusb_get_device_address) get_device_address = nullptr;
    decltype(&libusb_open) open = nullptr;
    decltype(&libusb_close) close = nullptr;
    decltype(&libusb_get_string_descriptor_ascii) get_string_descriptor_ascii = nullptr;
    decltype(&libusb_claim_interface) claim_interface = nullptr;
    decltype(&libusb_release_interface) release_interface = nullptr;
    decltype(&libusb_control_transfer) control_transfer = nullptr;
    decltype(&libusb_bulk_transfer) bulk_transfer = nullptr;
    decltype(&libusb_error_name) error_name = nullptr;

public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> lib): lib(std::move(lib)) {}
    ~ProdAPI() override;

    /// @brief loads libusb and creates a context.
    /// @returns UNSUPPORTED_ERROR on Linux, the missing library error when libusb is
    /// not installed, and CRITICAL_HARDWARE_ERROR when the library lacks a function the
    /// backend calls or cannot create a context.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    std::ptrdiff_t GetDeviceList(libusb_device ***list) override;
    void FreeDeviceList(libusb_device **list, int unref_devices) override;
    int
    GetDeviceDescriptor(libusb_device *dev, libusb_device_descriptor *desc) override;
    std::uint8_t GetBusNumber(libusb_device *dev) override;
    std::uint8_t GetDeviceAddress(libusb_device *dev) override;
    int Open(libusb_device *dev, libusb_device_handle **handle) override;
    void Close(libusb_device_handle *handle) override;
    int GetStringDescriptorAscii(
        libusb_device_handle *handle,
        std::uint8_t index,
        unsigned char *data,
        int length
    ) override;
    int ClaimInterface(libusb_device_handle *handle, int number) override;
    int ReleaseInterface(libusb_device_handle *handle, int number) override;
    int ControlTransfer(
        libusb_device_handle *handle,
        std::uint8_t request_type,
        std::uint8_t request,
        std::uint16_t value,
        std::uint16_t index,
        unsigned char *data,
        std::uint16_t length,
        unsigned int timeout
    ) override;
    int BulkTransfer(
        libusb_device_handle *handle,
        unsigned char endpoint,
        unsigned char *data,
        int length,
        int *transferred,
        unsigned int timeout
    ) override;
    const char *ErrorName(int code) override;
};
}
