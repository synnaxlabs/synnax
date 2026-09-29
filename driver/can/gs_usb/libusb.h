// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>

/// @brief the subset of the libusb 1.0 ABI the backend calls, transcribed from the
/// libusb API reference. libusb is LGPL-2.1, so the Driver only loads it at runtime.
namespace driver::can::gs_usb {
#ifdef _WIN32
#define LIBUSB_CALL __stdcall
#else
#define LIBUSB_CALL
#endif

struct libusb_context;
struct libusb_device;
struct libusb_device_handle;

struct libusb_device_descriptor {
    std::uint8_t bLength;
    std::uint8_t bDescriptorType;
    std::uint16_t bcdUSB;
    std::uint8_t bDeviceClass;
    std::uint8_t bDeviceSubClass;
    std::uint8_t bDeviceProtocol;
    std::uint8_t bMaxPacketSize0;
    std::uint16_t idVendor;
    std::uint16_t idProduct;
    std::uint16_t bcdDevice;
    std::uint8_t iManufacturer;
    std::uint8_t iProduct;
    std::uint8_t iSerialNumber;
    std::uint8_t bNumConfigurations;
};

constexpr int LIBUSB_SUCCESS = 0;
constexpr int LIBUSB_ERROR_NO_DEVICE = -4;
constexpr int LIBUSB_ERROR_BUSY = -6;
constexpr int LIBUSB_ERROR_TIMEOUT = -7;
constexpr int LIBUSB_ERROR_NOT_SUPPORTED = -12;

using libusb_init_t = int(LIBUSB_CALL *)(libusb_context **ctx);
using libusb_exit_t = void(LIBUSB_CALL *)(libusb_context *ctx);
using libusb_get_device_list_t =
    std::ptrdiff_t(LIBUSB_CALL *)(libusb_context *ctx, libusb_device ***list);
using libusb_free_device_list_t =
    void(LIBUSB_CALL *)(libusb_device **list, int unref_devices);
using libusb_get_device_descriptor_t =
    int(LIBUSB_CALL *)(libusb_device *dev, libusb_device_descriptor *desc);
using libusb_get_bus_number_t = std::uint8_t(LIBUSB_CALL *)(libusb_device *dev);
using libusb_get_device_address_t = std::uint8_t(LIBUSB_CALL *)(libusb_device *dev);
using libusb_open_t =
    int(LIBUSB_CALL *)(libusb_device *dev, libusb_device_handle **dev_handle);
using libusb_close_t = void(LIBUSB_CALL *)(libusb_device_handle *dev_handle);
using libusb_get_string_descriptor_ascii_t = int(LIBUSB_CALL *)(
    libusb_device_handle *dev_handle,
    std::uint8_t desc_index,
    unsigned char *data,
    int length
);
using libusb_set_auto_detach_kernel_driver_t =
    int(LIBUSB_CALL *)(libusb_device_handle *dev_handle, int enable);
using libusb_claim_interface_t =
    int(LIBUSB_CALL *)(libusb_device_handle *dev_handle, int interface_number);
using libusb_release_interface_t =
    int(LIBUSB_CALL *)(libusb_device_handle *dev_handle, int interface_number);
using libusb_control_transfer_t = int(LIBUSB_CALL *)(
    libusb_device_handle *dev_handle,
    std::uint8_t request_type,
    std::uint8_t bRequest,
    std::uint16_t wValue,
    std::uint16_t wIndex,
    unsigned char *data,
    std::uint16_t wLength,
    unsigned int timeout
);
using libusb_bulk_transfer_t = int(LIBUSB_CALL *)(
    libusb_device_handle *dev_handle,
    unsigned char endpoint,
    unsigned char *data,
    int length,
    int *actual_length,
    unsigned int timeout
);
using libusb_error_name_t = const char *(LIBUSB_CALL *) (int error_code);
}
