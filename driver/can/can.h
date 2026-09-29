// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/can/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/errors/errors.h"

namespace driver::can {
/// @brief invalid device properties, an unknown backend or channel, or a frame the bus
/// cannot send.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub("can");
/// @brief a backend that this platform does not support.
const x::errors::Error UNSUPPORTED_ERROR = CONFIG_ERROR.sub("unsupported");
/// @brief a send on a listen only bus.
const x::errors::Error LISTEN_ONLY_ERROR = CONFIG_ERROR.sub("listen_only");
/// @brief a frame whose identifier, length, or flags the bus cannot send.
const x::errors::Error FRAME_ERROR = CONFIG_ERROR.sub("frame");
/// @brief a bus fault that can clear, such as bus off or a full transmit queue.
const x::errors::Error
    TEMPORARY_HARDWARE_ERROR = driver::errors::TEMPORARY_HARDWARE_ERROR.sub("can");
/// @brief an adapter fault that retrying does not clear.
const x::errors::Error CRITICAL_HARDWARE_ERROR = driver::errors::CRITICAL_HARDWARE_ERROR
                                                     .sub("can");

/// @brief the largest payload of a CAN FD frame.
constexpr std::size_t MAX_FD_LENGTH = 64;
/// @brief the largest payload of a classic CAN frame.
constexpr std::size_t MAX_CLASSIC_LENGTH = 8;
/// @brief the largest standard (11-bit) identifier.
constexpr std::uint32_t MAX_STANDARD_ID = 0x7FF;
/// @brief the largest extended (29-bit) identifier.
constexpr std::uint32_t MAX_EXTENDED_ID = 0x1FFFFFFF;

/// @brief what a frame carries.
enum class Type : std::uint8_t {
    /// @brief a data frame.
    DATA,
    /// @brief a remote transmission request. length holds the requested length.
    REMOTE,
    /// @brief a bus error the adapter reports. id and data hold the backend's error
    /// class and details.
    BUS_ERROR,
};

/// @brief the clock a frame's time comes from.
enum class Clock : std::uint8_t {
    /// @brief the host clock, in nanoseconds since the Unix epoch.
    HOST,
    /// @brief the adapter's clock, in nanoseconds from an epoch of the adapter's
    /// choice. Only differences between two hardware times of one bus are meaningful.
    HARDWARE,
};

/// @brief one CAN or CAN FD frame.
struct Frame {
    /// @brief the arbitration identifier.
    std::uint32_t id = 0;
    /// @brief true for a 29-bit identifier.
    bool extended = false;
    /// @brief true for a CAN FD frame.
    bool fd = false;
    /// @brief true when a CAN FD frame sends its data phase at the data bitrate.
    bool bitrate_switched = false;
    /// @brief true when the transmitter of a received CAN FD frame was error passive.
    bool error_passive = false;
    /// @brief what the frame carries.
    Type type = Type::DATA;
    /// @brief the number of valid bytes in data.
    std::uint8_t length = 0;
    /// @brief the payload. Bytes past length are unspecified.
    std::array<std::uint8_t, MAX_FD_LENGTH> data{};
    /// @brief when the frame was received. Unset on frames to send.
    x::telem::TimeStamp time;
    /// @brief the clock time comes from.
    Clock clock = Clock::HOST;

    /// @returns the valid bytes of data.
    [[nodiscard]] std::span<const std::uint8_t> payload() const {
        return {this->data.data(), this->length};
    }
};

/// @returns the payload length a data length code stands for. Codes above 8 stand for
/// 8 bytes on classic CAN and for the CAN FD lengths 12 to 64 on CAN FD.
/// @param dlc the data length code, 0 to 15.
/// @param fd true for a CAN FD frame.
[[nodiscard]] std::uint8_t dlc_to_length(std::uint8_t dlc, bool fd);

/// @returns the data length code of a payload length, or nullopt when no code stands
/// for exactly that length.
[[nodiscard]] std::optional<std::uint8_t> length_to_dlc(std::uint8_t length);

/// @brief checks that a bus can send a frame.
/// @param frame the frame to send.
/// @param fd true when the bus runs CAN FD.
/// @returns FRAME_ERROR when the identifier exceeds its width, when the length has no
/// data length code or exceeds the frame kind, when an FD frame goes to a classic bus,
/// when a remote frame is an FD frame, or when the frame is a bus error.
[[nodiscard]] x::errors::Error validate(const Frame &frame, bool fd);

/// @brief extends a wrapping 32-bit hardware counter into a 64-bit count that never
/// wraps, provided it is read at least once per wrap period.
class Counter {
    std::uint32_t last = 0;
    std::uint64_t high = 0;
    bool started = false;

public:
    /// @returns raw extended to 64 bits.
    std::uint64_t extend(std::uint32_t raw);
};

/// @brief a channel a backend found.
struct Channel {
    /// @brief the backend the channel belongs to, one of synnax::can::BACKEND_*.
    std::string backend;
    /// @brief the value of synnax::can::Properties::channel that opens this channel.
    std::string name;
    /// @brief a human readable description of the adapter.
    std::string description;
};

/// @brief an open CAN channel. receive and send may run at the same time on two
/// threads. close must not run at the same time as either, and neither may run after
/// close. A bus never receives the frames it sends.
class Bus {
public:
    virtual ~Bus() = default;

    /// @brief blocks until a frame arrives or the timeout elapses.
    /// @param frame receives the frame when one arrived.
    /// @param timeout the longest time to wait.
    /// @returns true when a frame arrived, and false when the timeout elapsed.
    /// TEMPORARY_HARDWARE_ERROR or CRITICAL_HARDWARE_ERROR on a fault.
    [[nodiscard]] virtual std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) = 0;

    /// @brief queues a frame for transmission without waiting for it to reach the bus.
    /// @returns LISTEN_ONLY_ERROR on a listen only bus, FRAME_ERROR when validate
    /// rejects the frame, and TEMPORARY_HARDWARE_ERROR when the transmit queue is full
    /// or the bus is off.
    [[nodiscard]] virtual x::errors::Error send(const Frame &frame) = 0;

    /// @brief takes the channel off the bus and releases it. Calling close again does
    /// nothing. The destructor closes the bus.
    virtual x::errors::Error close() = 0;
};

/// @brief a way of reaching CAN adapters, such as a kernel interface or a vendor
/// library.
class Backend {
public:
    virtual ~Backend() = default;

    /// @returns the channels the backend finds.
    [[nodiscard]] virtual std::pair<std::vector<Channel>, x::errors::Error> scan() = 0;

    /// @brief opens the channel the properties name and puts it on the bus.
    /// @returns CONFIG_ERROR when the channel does not exist or the adapter cannot run
    /// the bitrates or modes the properties ask for.
    [[nodiscard]] virtual std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) = 0;
};

/// @brief a backend that could not load. scan and open return the error that stopped
/// it from loading.
class Unavailable final : public Backend {
    x::errors::Error err;

public:
    explicit Unavailable(x::errors::Error err): err(std::move(err)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override {
        return {{}, this->err};
    }

    [[nodiscard]] std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &) override {
        return {nullptr, this->err};
    }
};

/// @brief maps each backend name, one of synnax::can::BACKEND_*, to its backend.
using Backends = std::unordered_map<std::string, std::shared_ptr<Backend>>;

/// @brief opens a bus on the backend the properties name.
/// @returns CONFIG_ERROR when no backend has that name, and otherwise what the
/// backend's open returns.
[[nodiscard]] std::pair<std::unique_ptr<Bus>, x::errors::Error>
open(const Backends &backends, const synnax::can::Properties &props);
}
