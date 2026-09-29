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
    libusb_init_t init = nullptr;
    libusb_exit_t exit = nullptr;
    libusb_get_device_list_t get_device_list = nullptr;
    libusb_free_device_list_t free_device_list = nullptr;
    libusb_get_device_descriptor_t get_device_descriptor = nullptr;
    libusb_get_bus_number_t get_bus_number = nullptr;
    libusb_get_device_address_t get_device_address = nullptr;
    libusb_open_t open = nullptr;
    libusb_close_t close = nullptr;
    libusb_get_string_descriptor_ascii_t get_string_descriptor_ascii = nullptr;
    libusb_set_auto_detach_kernel_driver_t set_auto_detach_kernel_driver = nullptr;
    libusb_claim_interface_t claim_interface = nullptr;
    libusb_release_interface_t release_interface = nullptr;
    libusb_control_transfer_t control_transfer = nullptr;
    libusb_bulk_transfer_t bulk_transfer = nullptr;
    libusb_error_name_t error_name = nullptr;

public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> lib): lib(std::move(lib)) {}
    ~ProdAPI() override;

    /// @brief loads libusb and creates a context.
    /// @returns the missing library error when libusb is not installed, and
    /// CRITICAL_HARDWARE_ERROR when the library lacks a function the backend calls or
    /// cannot create a context.
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
    int SetAutoDetachKernelDriver(libusb_device_handle *handle, int enable) override;
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
