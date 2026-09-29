// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "driver/can/can.h"
#include "driver/can/gs_usb/api.h"
#include "driver/can/gs_usb/protocol.h"

/// @brief the gs_usb backend for candleLight and CANable adapters over libusb.
namespace driver::can::gs_usb {
/// @brief the longest a control request or a send waits for the adapter.
const auto USB_TIMEOUT = x::telem::SECOND;
/// @brief the number of echo ids the backend cycles through.
constexpr std::uint32_t ECHO_IDS = 64;

/// @brief one channel of a gs_usb adapter. The bus claims the adapter's USB interface,
/// so a second channel of the same adapter cannot open in the same process.
class Bus final : public can::Bus {
    std::shared_ptr<API> api;
    libusb_device_handle *handle;
    std::uint8_t channel;
    std::string name;
    bool fd;
    bool listen_only;
    bool timestamps;
    Counter counter;
    std::atomic<std::uint32_t> next_echo = 0;
    bool closed = false;

public:
    Bus(std::shared_ptr<API> api,
        libusb_device_handle *handle,
        std::uint8_t channel,
        std::string name,
        bool fd,
        bool listen_only,
        bool timestamps);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    [[nodiscard]] x::errors::Error send(const Frame &frame) override;

    x::errors::Error close() override;
};

/// @brief finds and opens gs_usb adapters. A channel's name is the adapter's USB serial
/// number and the channel index, such as 0039002C4E55:0.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns the backend over the installed libusb, or an Unavailable backend holding
/// the reason libusb did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
