// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <cstring>
#include <deque>
#include <mutex>
#include <optional>
#include <stdexcept>
#include <string>
#include <vector>

#include "driver/can/gs_usb/api.h"
#include "driver/can/gs_usb/protocol.h"

namespace driver::can::gs_usb {
/// @brief a model of libusb with candleLight adapters attached, written from the
/// candleLight firmware (candle-usb/candleLight_fw) and libusb (libusb/libusb). Each
/// adapter holds a finite pool of frame objects, echoes every frame it sends, and loses
/// bus frames while the pool is empty. Bulk transfers block until their timeouts, an
/// unplugged adapter fails every call, and a second program is refused the way the
/// host's libusb backend refuses it. Safe for concurrent use.
class MockAPI final : public API {
public:
    /// @brief how the host refuses a second program on an adapter.
    enum class Host : std::uint8_t {
        /// @brief usbfs lets a second program open the adapter and refuses its claim of
        /// the interface with LIBUSB_ERROR_BUSY (libusb os/linux_usbfs.c).
        LINUX,
        /// @brief WinUSB takes one open handle, so a second open fails with
        /// LIBUSB_ERROR_ACCESS (libusb os/windows_winusb.c, winusbx_open).
        WINDOWS,
    };

    /// @brief the frame objects candleLight holds per channel. Every frame to or from
    /// the host takes one until the host reads it (include/config.h, CAN_QUEUE_SIZE).
    static constexpr std::size_t POOL_PER_CHANNEL = 64;

    /// @brief an adapter to attach.
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
    };

    /// @brief a control request the backend made.
    struct Control {
        std::uint8_t request_type = 0;
        std::uint8_t request = 0;
        std::uint16_t value = 0;
        std::vector<std::uint8_t> body;
    };

    explicit MockAPI(const Host host = Host::LINUX): host(host) {}

    /// @brief plugs an adapter in.
    void attach(Device device) {
        std::lock_guard lock(this->mu);
        this->instances.push_back(this->fresh(std::move(device)));
    }

    /// @brief unplugs an adapter. The device list omits it, and every call on a handle
    /// to it fails with LIBUSB_ERROR_NO_DEVICE, including transfers already blocked.
    void unplug(const std::string &serial) {
        {
            std::lock_guard lock(this->mu);
            this->find(serial).attached = false;
        }
        this->cv.notify_all();
    }

    /// @brief plugs an unplugged adapter back in at a new address, with its firmware
    /// restarted. Handles opened before the unplug stay failed.
    void plug(const std::string &serial) {
        std::lock_guard lock(this->mu);
        auto &old = this->find_any(serial);
        auto device = old.device;
        const bool held = old.held;
        this->instances.push_back(this->fresh(std::move(device)));
        this->instances.back().held = held;
    }

    /// @brief makes another program hold the adapter open with its interface claimed,
    /// or let it go.
    void hold(const std::string &serial, const bool held) {
        std::lock_guard lock(this->mu);
        this->find(serial).held = held;
    }

    /// @brief makes every later control request to the adapter fail with rc.
    void fail_controls(const std::string &serial, const int rc) {
        std::lock_guard lock(this->mu);
        this->find(serial).control_status = rc;
    }

    /// @brief a frame arrives from the bus on a channel of the adapter. The firmware
    /// queues it for the host while the channel runs and a frame object is free, and
    /// loses it otherwise (src/can_common.c, CAN_ReceiveFrame).
    /// @param timestamp_us the adapter's microsecond timer when the frame arrived.
    void deliver(
        const std::string &serial,
        const std::uint8_t channel,
        const Frame &frame,
        const std::uint32_t timestamp_us = 0
    ) {
        {
            std::lock_guard lock(this->mu);
            auto &d = this->find(serial);
            auto &ch = d.state.at(channel);
            if (!ch.started || d.pool == 0) {
                d.lost++;
                return;
            }
            d.pool--;
            auto bytes = host_frame(frame, ECHO_ID_RX, channel);
            if (ch.timestamped) append_timestamp(bytes, timestamp_us);
            d.to_host.push_back({.channel = channel, .bytes = std::move(bytes)});
        }
        this->cv.notify_all();
    }

    /// @returns the control requests made to any adapter with a request code.
    std::vector<Control> controls_of(const Request request) {
        std::lock_guard lock(this->mu);
        std::vector<Control> matched;
        for (const auto &c: this->controls)
            if (c.request == static_cast<std::uint8_t>(request)) matched.push_back(c);
        return matched;
    }

    /// @returns the host frames the adapter put on the bus, in order.
    std::vector<std::vector<std::uint8_t>> sent(const std::string &serial) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).sent;
    }

    /// @returns the number of handles the backend holds open to the adapter.
    std::size_t handles(const std::string &serial) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).handles;
    }

    /// @returns the number of times the backend opened the adapter.
    std::size_t opens(const std::string &serial) {
        std::lock_guard lock(this->mu);
        std::size_t n = 0;
        for (const auto &d: this->instances)
            if (d.device.serial == serial) n += d.opens;
        return n;
    }

    /// @returns true while the backend holds the adapter's interface claimed.
    bool claimed(const std::string &serial) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).claimed;
    }

    /// @returns the bus frames the adapter lost.
    std::size_t lost(const std::string &serial) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).lost;
    }

    /// @returns the frames the adapter holds for the host to read.
    std::size_t queued(const std::string &serial) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).to_host.size();
    }

    /// @returns true while a channel of the adapter runs.
    bool started(const std::string &serial, const std::uint8_t channel) {
        std::lock_guard lock(this->mu);
        return this->find_any(serial).state.at(channel).started;
    }

    std::ptrdiff_t GetDeviceList(libusb_device ***list) override {
        std::lock_guard lock(this->mu);
        this->list.clear();
        for (std::size_t i = 0; i < this->instances.size(); i++)
            if (this->instances[i].attached)
                this->list.push_back(reinterpret_cast<libusb_device *>(i + 1));
        const auto count = static_cast<std::ptrdiff_t>(this->list.size());
        this->list.push_back(nullptr);
        *list = this->list.data();
        return count;
    }

    void FreeDeviceList(libusb_device **, int) override {}

    int
    GetDeviceDescriptor(libusb_device *dev, libusb_device_descriptor *desc) override {
        std::lock_guard lock(this->mu);
        const auto &d = this->instance(dev);
        *desc = {};
        desc->idVendor = d.device.vendor;
        desc->idProduct = d.device.product;
        desc->iProduct = PRODUCT_INDEX;
        desc->iSerialNumber = SERIAL_INDEX;
        return LIBUSB_SUCCESS;
    }

    std::uint8_t GetBusNumber(libusb_device *) override { return 1; }

    std::uint8_t GetDeviceAddress(libusb_device *dev) override {
        std::lock_guard lock(this->mu);
        return this->instance(dev).address;
    }

    int Open(libusb_device *dev, libusb_device_handle **handle) override {
        std::lock_guard lock(this->mu);
        auto &d = this->instance(dev);
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
        if (this->host == Host::WINDOWS && (d.held || d.handles > 0))
            return LIBUSB_ERROR_ACCESS;
        d.handles++;
        d.opens++;
        *handle = reinterpret_cast<libusb_device_handle *>(dev);
        return LIBUSB_SUCCESS;
    }

    void Close(libusb_device_handle *handle) override {
        std::lock_guard lock(this->mu);
        auto &d = this->instance(handle);
        d.handles--;
        if (d.handles == 0) d.claimed = false;
    }

    int GetStringDescriptorAscii(
        libusb_device_handle *handle,
        const std::uint8_t index,
        unsigned char *data,
        const int length
    ) override {
        std::lock_guard lock(this->mu);
        const auto &d = this->instance(handle);
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
        const auto &text = index == SERIAL_INDEX ? d.device.serial
                                                 : d.device.product_name;
        const auto n = std::min<int>(static_cast<int>(text.size()), length);
        std::memcpy(data, text.data(), static_cast<std::size_t>(n));
        return n;
    }

    int ClaimInterface(libusb_device_handle *handle, int) override {
        std::lock_guard lock(this->mu);
        auto &d = this->instance(handle);
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
        if (d.held || d.claimed) return LIBUSB_ERROR_BUSY;
        d.claimed = true;
        return LIBUSB_SUCCESS;
    }

    int ReleaseInterface(libusb_device_handle *handle, int) override {
        std::lock_guard lock(this->mu);
        auto &d = this->instance(handle);
        d.claimed = false;
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
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
        auto &d = this->instance(handle);
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
        if (d.control_status.has_value()) return *d.control_status;
        if (request_type == REQUEST_OUT) {
            std::vector<std::uint8_t> body(data, data + length);
            if (request == static_cast<std::uint8_t>(Request::MODE))
                this->mode(d, static_cast<std::uint8_t>(value), body);
            this->controls.push_back({
                .request_type = request_type,
                .request = request,
                .value = value,
                .body = std::move(body),
            });
            return length;
        }
        this->controls.push_back({.request_type = request_type, .request = request});
        std::vector<std::uint8_t> reply;
        if (request == static_cast<std::uint8_t>(Request::DEVICE_CONFIG)) {
            reply.resize(12);
            reply[3] = static_cast<std::uint8_t>(d.device.channels - 1);
        }
        if (request == static_cast<std::uint8_t>(Request::BT_CONST))
            reply = encode_bt_const(d.device.bt_const, false);
        if (request == static_cast<std::uint8_t>(Request::BT_CONST_EXT))
            reply = encode_bt_const(d.device.bt_const, true);
        const auto n = std::min<std::size_t>(reply.size(), length);
        std::memcpy(data, reply.data(), n);
        return static_cast<int>(n);
    }

    int BulkTransfer(
        libusb_device_handle *handle,
        const unsigned char endpoint,
        unsigned char *data,
        const int length,
        int *transferred,
        const unsigned int timeout
    ) override {
        *transferred = 0;
        std::unique_lock lock(this->mu);
        const auto index = reinterpret_cast<std::uintptr_t>(handle) - 1;
        const auto deadline = std::chrono::steady_clock::now() +
                              std::chrono::milliseconds(timeout);
        if (endpoint == ENDPOINT_OUT) {
            // The firmware NAKs OUT transfers while its pool is empty
            // (src/usbd_gs_can.c, USBD_GS_CAN_DataOut).
            this->cv.wait_until(lock, deadline, [&] {
                const auto &d = this->instances[index];
                return !d.attached || d.pool > 0;
            });
            auto &d = this->instances[index];
            if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
            if (d.pool == 0) return LIBUSB_ERROR_TIMEOUT;
            this->send(d, std::vector<std::uint8_t>(data, data + length));
            *transferred = length;
            lock.unlock();
            this->cv.notify_all();
            return LIBUSB_SUCCESS;
        }
        this->cv.wait_until(lock, deadline, [&] {
            const auto &d = this->instances[index];
            return !d.attached || !d.to_host.empty();
        });
        auto &d = this->instances[index];
        if (!d.attached) return LIBUSB_ERROR_NO_DEVICE;
        if (d.to_host.empty()) return LIBUSB_ERROR_TIMEOUT;
        const auto chunk = std::move(d.to_host.front().bytes);
        d.to_host.pop_front();
        d.pool++;
        const auto n = std::min<std::size_t>(
            chunk.size(),
            static_cast<std::size_t>(length)
        );
        std::memcpy(data, chunk.data(), n);
        *transferred = static_cast<int>(n);
        lock.unlock();
        this->cv.notify_all();
        return LIBUSB_SUCCESS;
    }

    const char *ErrorName(const int code) override {
        switch (code) {
            case LIBUSB_ERROR_IO:
                return "LIBUSB_ERROR_IO";
            case LIBUSB_ERROR_ACCESS:
                return "LIBUSB_ERROR_ACCESS";
            case LIBUSB_ERROR_NO_DEVICE:
                return "LIBUSB_ERROR_NO_DEVICE";
            case LIBUSB_ERROR_BUSY:
                return "LIBUSB_ERROR_BUSY";
            case LIBUSB_ERROR_TIMEOUT:
                return "LIBUSB_ERROR_TIMEOUT";
            case LIBUSB_ERROR_PIPE:
                return "LIBUSB_ERROR_PIPE";
            case LIBUSB_ERROR_NOT_SUPPORTED:
                return "LIBUSB_ERROR_NOT_SUPPORTED";
            default:
                return "LIBUSB_ERROR_OTHER";
        }
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

    /// @returns a frame as the adapter sends it to the host, without a timestamp.
    static std::vector<std::uint8_t> host_frame(
        const Frame &frame,
        const std::uint32_t echo_id,
        const std::uint8_t channel
    ) {
        auto bytes = encode_frame(frame, echo_id, channel);
        if (frame.type == Type::BUS_ERROR) bytes[7] |= ID_ERROR >> 24;
        return bytes;
    }

private:
    static constexpr std::uint8_t PRODUCT_INDEX = 2;
    static constexpr std::uint8_t SERIAL_INDEX = 3;

    /// @brief the state of one channel in the firmware.
    struct Channel {
        bool started = false;
        bool timestamped = false;
        /// @brief frames from the host that wait for the channel to start.
        std::size_t waiting = 0;
    };

    /// @brief a frame waiting for the host to read it.
    struct Pending {
        std::uint8_t channel = 0;
        std::vector<std::uint8_t> bytes;
    };

    /// @brief one attachment of an adapter, from plug to unplug.
    struct Instance {
        Device device;
        std::uint8_t address = 0;
        bool attached = true;
        bool held = false;
        std::size_t handles = 0;
        std::size_t opens = 0;
        bool claimed = false;
        std::optional<int> control_status;
        std::vector<Channel> state;
        /// @brief the free frame objects.
        std::size_t pool = 0;
        std::deque<Pending> to_host;
        std::vector<std::vector<std::uint8_t>> sent;
        std::size_t lost = 0;
    };

    const Host host;
    std::mutex mu;
    std::condition_variable cv;
    std::vector<Instance> instances;
    std::vector<Control> controls;
    std::vector<libusb_device *> list;
    std::uint8_t next_address = 1;

    Instance fresh(Device device) {
        Instance d;
        d.address = this->next_address++;
        d.state.resize(device.channels);
        d.pool = POOL_PER_CHANNEL * device.channels;
        d.device = std::move(device);
        return d;
    }

    static void append_timestamp(std::vector<std::uint8_t> &bytes, std::uint32_t us) {
        for (int i = 0; i < 4; i++)
            bytes.push_back(static_cast<std::uint8_t>(us >> (8 * i)));
    }

    /// @brief applies a MODE request. A reset purges the channel's frames waiting for
    /// the host (src/usbd_gs_can.c, usbd_gs_can_purge_to_host_list_by_channel).
    void mode(
        Instance &d,
        const std::uint8_t channel,
        const std::vector<std::uint8_t> &body
    ) {
        auto &ch = d.state.at(channel);
        const auto mode = body[0];
        const auto flags = body[4] | body[5] << 8;
        if (mode == MODE_START) {
            ch.started = true;
            ch.timestamped = (flags & FEATURE_HW_TIMESTAMP) != 0;
            return;
        }
        ch.started = false;
        const auto purged = std::erase_if(d.to_host, [&](const Pending &p) {
            return p.channel == channel;
        });
        d.pool += purged + ch.waiting;
        ch.waiting = 0;
    }

    /// @brief takes a frame from the host, puts it on the bus, and queues its echo
    /// (src/can_common.c, CAN_SendFrame). The firmware drops a short frame or one for a
    /// channel it lacks, and holds a frame for a stopped channel until the channel
    /// resets (src/usbd_gs_can.c, USBD_GS_CAN_DataOut).
    void send(Instance &d, std::vector<std::uint8_t> bytes) {
        if (bytes.size() < HEADER_SIZE + MAX_CLASSIC_LENGTH) return;
        const auto channel = bytes[9];
        if (channel >= d.state.size()) return;
        d.pool--;
        auto &ch = d.state[channel];
        if (!ch.started) {
            ch.waiting++;
            return;
        }
        d.sent.push_back(bytes);
        if (ch.timestamped) append_timestamp(bytes, 0);
        d.to_host.push_back({.channel = channel, .bytes = std::move(bytes)});
    }

    Instance &find(const std::string &serial) {
        for (auto &d: this->instances)
            if (d.device.serial == serial && d.attached) return d;
        throw std::out_of_range("no attached adapter has serial number " + serial);
    }

    /// @returns the latest attachment of an adapter, attached or not.
    Instance &find_any(const std::string &serial) {
        for (auto it = this->instances.rbegin(); it != this->instances.rend(); ++it)
            if (it->device.serial == serial) return *it;
        throw std::out_of_range("no adapter has serial number " + serial);
    }

    Instance &instance(libusb_device *dev) {
        return this->instances.at(reinterpret_cast<std::uintptr_t>(dev) - 1);
    }

    Instance &instance(libusb_device_handle *handle) {
        return this->instances.at(reinterpret_cast<std::uintptr_t>(handle) - 1);
    }
};
}
