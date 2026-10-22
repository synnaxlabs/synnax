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
#include <cstddef>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
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

/// @brief the most received frames an adapter holds for a channel that is not
/// receiving. It drops the oldest frame to make room, and the bus reports the loss as a
/// receive overflow bus error before the next frame.
constexpr std::size_t MAX_QUEUED = 1024;
/// @brief the longest one read of the adapter waits. Closing the last bus of an adapter
/// waits for the read in progress.
const auto READ_SLICE = 50 * x::telem::MILLISECOND;

class Adapter;

/// @brief one channel of a gs_usb adapter. The buses of an adapter's channels share its
/// USB handle, which closes with the last of them. A thread per adapter reads the
/// adapter from open to close, so the adapter's frame pool never fills with frames for
/// the host while no bus receives.
class Bus final : public can::Bus {
    std::shared_ptr<Adapter> adapter;
    std::uint8_t channel;
    bool timestamped;
    Counter counter{x::telem::MICROSECOND};
    std::atomic<std::uint32_t> next_echo = 0;
    /// @brief a received frame that waits behind the overflow error it revealed.
    std::optional<Frame> pending;
    bool closed = false;

    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    Bus(std::shared_ptr<Adapter> adapter,
        std::uint8_t channel,
        std::string name,
        bool fd,
        bool listen_only,
        bool timestamped);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    x::errors::Error close() override;
};

/// @brief finds and opens gs_usb adapters. A channel's name is the adapter's USB serial
/// number and the channel index, such as 0039002C4E55:0.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;
    std::mutex mu;
    /// @brief the open adapters by serial number.
    std::unordered_map<std::string, std::weak_ptr<Adapter>> adapters;

    /// @returns the open adapter with a serial number, opening it first when no bus
    /// holds it or the one held failed. Requires mu.
    std::pair<std::shared_ptr<Adapter>, x::errors::Error>
    adapter(const std::string &serial, const std::string &channel);

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    /// @returns the channels of every attached adapter. An adapter that a bus holds is
    /// listed from what the backend read when it opened, since Windows refuses a second
    /// open.
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    /// @returns UNREACHABLE_ERROR when no adapter with the serial number is attached,
    /// another program holds it, or a bus holds the channel.
    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns an error whose message is context followed by the name of a libusb error
/// code. Its type is UNREACHABLE_ERROR when the adapter is gone, busy, refused, or
/// silent, CONFIG_ERROR when the host cannot do the operation, and
/// CRITICAL_HARDWARE_ERROR otherwise.
[[nodiscard]] x::errors::Error translate(API &api, int rc, const std::string &context);

/// @returns the backend over the installed libusb, or an Unavailable backend holding
/// the reason libusb did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
