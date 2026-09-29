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

#include "libusb.h"

namespace driver::can::gs_usb {
/// @brief the libusb calls the backend makes, on one libusb context. ProdAPI loads them
/// from the libusb shared library, and MockAPI simulates gs_usb adapters in tests.
class API {
public:
    virtual ~API() = default;

    virtual std::ptrdiff_t GetDeviceList(libusb_device ***list) = 0;
    virtual void FreeDeviceList(libusb_device **list, int unref_devices) = 0;
    virtual int
    GetDeviceDescriptor(libusb_device *dev, libusb_device_descriptor *desc) = 0;
    virtual std::uint8_t GetBusNumber(libusb_device *dev) = 0;
    virtual std::uint8_t GetDeviceAddress(libusb_device *dev) = 0;
    virtual int Open(libusb_device *dev, libusb_device_handle **handle) = 0;
    virtual void Close(libusb_device_handle *handle) = 0;
    virtual int GetStringDescriptorAscii(
        libusb_device_handle *handle,
        std::uint8_t index,
        unsigned char *data,
        int length
    ) = 0;
    virtual int ClaimInterface(libusb_device_handle *handle, int number) = 0;
    virtual int ReleaseInterface(libusb_device_handle *handle, int number) = 0;
    virtual int ControlTransfer(
        libusb_device_handle *handle,
        std::uint8_t request_type,
        std::uint8_t request,
        std::uint16_t value,
        std::uint16_t index,
        unsigned char *data,
        std::uint16_t length,
        unsigned int timeout
    ) = 0;
    virtual int BulkTransfer(
        libusb_device_handle *handle,
        unsigned char endpoint,
        unsigned char *data,
        int length,
        int *transferred,
        unsigned int timeout
    ) = 0;
    virtual const char *ErrorName(int code) = 0;
};
}
