// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <cstring>
#include <deque>
#include <mutex>
#include <string>
#include <vector>

#include "driver/can/gs_usb/api.h"
#include "driver/can/gs_usb/protocol.h"

namespace driver::can::gs_usb {
/// @brief simulated libusb with attached gs_usb adapters. Tests queue the host frames
/// bulk reads return and inspect the control requests and bulk writes the backend made.
class MockAPI final : public API {
public:
    /// @brief an attached adapter.
    struct Device {
        std::uint16_t vendor = 0x1D50;
        std::uint16_t product = 0x606F;
        std::string serial;
        std::string product_name = "candleLight USB to CAN adapter";
        std::uint32_t channels = 1;
        BtConst bt_const{
            .features = FEATURE_LISTEN_ONLY | FEATURE_HW_TIMESTAMP,
            .clock_hz = 48000000,
            .nominal = {
                .tseg1_max = 16,
                .tseg2_max = 8,
                .sjw_max = 4,
                .brp_max = 1024,
            },
        };
        int open_status = LIBUSB_SUCCESS;
        int claim_status = LIBUSB_SUCCESS;
        bool claimed = false;
        bool open = false;
    };

    /// @brief a control request the backend made.
    struct Control {
        std::uint8_t request_type = 0;
        std::uint8_t request = 0;
        std::uint16_t value = 0;
        std::vector<std::uint8_t> body;
    };

    std::mutex mu;
    std::vector<Device> devices;
    std::vector<Control> controls;
    /// @brief the host frames bulk reads return in order. An empty queue times out.
    std::deque<std::vector<std::uint8_t>> in;
    /// @brief every host frame the backend sent.
    std::vector<std::vector<std::uint8_t>> out;
    int bulk_out_status = LIBUSB_SUCCESS;
    /// @brief the timeout of the last bulk read in milliseconds.
    unsigned int last_timeout = 0;

    /// @returns the control requests with a request code.
    std::vector<Control> controls_of(const Request request) {
        std::vector<Control> matched;
        for (const auto &c: this->controls)
            if (c.request == static_cast<std::uint8_t>(request)) matched.push_back(c);
        return matched;
    }

    std::ptrdiff_t GetDeviceList(libusb_device ***list) override {
        std::lock_guard lock(this->mu);
        this->list.clear();
        for (std::size_t i = 0; i < this->devices.size(); i++)
            this->list.push_back(reinterpret_cast<libusb_device *>(i + 1));
        this->list.push_back(nullptr);
        *list = this->list.data();
        return static_cast<std::ptrdiff_t>(this->devices.size());
    }

    void FreeDeviceList(libusb_device **, int) override {}

    int
    GetDeviceDescriptor(libusb_device *dev, libusb_device_descriptor *desc) override {
        std::lock_guard lock(this->mu);
        const auto &d = this->device(dev);
        *desc = {};
        desc->idVendor = d.vendor;
        desc->idProduct = d.product;
        desc->iProduct = PRODUCT_INDEX;
        desc->iSerialNumber = SERIAL_INDEX;
        return LIBUSB_SUCCESS;
    }

    std::uint8_t GetBusNumber(libusb_device *) override { return 1; }

    std::uint8_t GetDeviceAddress(libusb_device *dev) override {
        return static_cast<std::uint8_t>(reinterpret_cast<std::uintptr_t>(dev));
    }

    int Open(libusb_device *dev, libusb_device_handle **handle) override {
        std::lock_guard lock(this->mu);
        auto &d = this->device(dev);
        if (d.open_status != LIBUSB_SUCCESS) return d.open_status;
        d.open = true;
        *handle = reinterpret_cast<libusb_device_handle *>(dev);
        return LIBUSB_SUCCESS;
    }

    void Close(libusb_device_handle *handle) override {
        std::lock_guard lock(this->mu);
        this->device(handle).open = false;
    }

    int GetStringDescriptorAscii(
        libusb_device_handle *handle,
        const std::uint8_t index,
        unsigned char *data,
        const int length
    ) override {
        std::lock_guard lock(this->mu);
        const auto &d = this->device(handle);
        const auto &text = index == SERIAL_INDEX ? d.serial : d.product_name;
        const auto n = std::min<int>(static_cast<int>(text.size()), length);
        std::memcpy(data, text.data(), static_cast<std::size_t>(n));
        return n;
    }

    int SetAutoDetachKernelDriver(libusb_device_handle *, int) override {
        return LIBUSB_ERROR_NOT_SUPPORTED;
    }

    int ClaimInterface(libusb_device_handle *handle, int) override {
        std::lock_guard lock(this->mu);
        auto &d = this->device(handle);
        if (d.claim_status != LIBUSB_SUCCESS) return d.claim_status;
        d.claimed = true;
        return LIBUSB_SUCCESS;
    }

    int ReleaseInterface(libusb_device_handle *handle, int) override {
        std::lock_guard lock(this->mu);
        this->device(handle).claimed = false;
        return LIBUSB_SUCCESS;
    }

    int ControlTransfer(
        libusb_device_handle *handle,
        const std::uint8_t request_type,
        const std::uint8_t request,
        const std::uint16_t value,
        std::uint16_t,
        unsigned char *data,
        const std::uint16_t length,
        unsigned int
    ) override {
        std::lock_guard lock(this->mu);
        const auto &d = this->device(handle);
        if (request_type == REQUEST_OUT) {
            this->controls.push_back({
                .request_type = request_type,
                .request = request,
                .value = value,
                .body = {data, data + length},
            });
            return length;
        }
        this->controls.push_back({.request_type = request_type, .request = request});
        std::vector<std::uint8_t> reply;
        if (request == static_cast<std::uint8_t>(Request::DEVICE_CONFIG)) {
            reply.resize(12);
            reply[3] = static_cast<std::uint8_t>(d.channels - 1);
        }
        if (request == static_cast<std::uint8_t>(Request::BT_CONST))
            reply = encode_bt_const(d.bt_const, false);
        if (request == static_cast<std::uint8_t>(Request::BT_CONST_EXT))
            reply = encode_bt_const(d.bt_const, true);
        const auto n = std::min<std::size_t>(reply.size(), length);
        std::memcpy(data, reply.data(), n);
        return static_cast<int>(n);
    }

    int BulkTransfer(
        libusb_device_handle *,
        const unsigned char endpoint,
        unsigned char *data,
        const int length,
        int *transferred,
        const unsigned int timeout
    ) override {
        std::lock_guard lock(this->mu);
        *transferred = 0;
        if (endpoint == ENDPOINT_OUT) {
            if (this->bulk_out_status != LIBUSB_SUCCESS) return this->bulk_out_status;
            this->out.emplace_back(data, data + length);
            *transferred = length;
            return LIBUSB_SUCCESS;
        }
        this->last_timeout = timeout;
        if (this->in.empty()) return LIBUSB_ERROR_TIMEOUT;
        const auto chunk = this->in.front();
        this->in.pop_front();
        const auto n = std::min<std::size_t>(
            chunk.size(),
            static_cast<std::size_t>(length)
        );
        std::memcpy(data, chunk.data(), n);
        *transferred = static_cast<int>(n);
        return LIBUSB_SUCCESS;
    }

    const char *ErrorName(const int code) override {
        if (code == LIBUSB_ERROR_BUSY) return "LIBUSB_ERROR_BUSY";
        if (code == LIBUSB_ERROR_NO_DEVICE) return "LIBUSB_ERROR_NO_DEVICE";
        return "LIBUSB_ERROR_OTHER";
    }

    /// @returns the BT_CONST or BT_CONST_EXT reply for a set of capabilities.
    static std::vector<std::uint8_t>
    encode_bt_const(const BtConst &c, const bool extended) {
        std::vector<std::uint8_t> out;
        const auto put = [&](const std::uint32_t v) {
            for (int i = 0; i < 4; i++)
                out.push_back(static_cast<std::uint8_t>(v >> (8 * i)));
        };
        const auto put_limits = [&](const TimingLimits &l) {
            for (const auto v:
                 {l.tseg1_min,
                  l.tseg1_max,
                  l.tseg2_min,
                  l.tseg2_max,
                  l.sjw_max,
                  l.brp_min,
                  l.brp_max,
                  l.brp_inc})
                put(v);
        };
        put(c.features);
        put(c.clock_hz);
        put_limits(c.nominal);
        if (extended) put_limits(c.data);
        return out;
    }

private:
    static constexpr std::uint8_t PRODUCT_INDEX = 2;
    static constexpr std::uint8_t SERIAL_INDEX = 3;
    std::vector<libusb_device *> list;

    Device &device(libusb_device *dev) {
        return this->devices.at(reinterpret_cast<std::uintptr_t>(dev) - 1);
    }

    Device &device(libusb_device_handle *handle) {
        return this->devices.at(reinterpret_cast<std::uintptr_t>(handle) - 1);
    }
};
}
