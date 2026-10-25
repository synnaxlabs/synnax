// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <cstring>
#include <format>

#include "driver/can/pcan/pcan.h"
#include "driver/can/pcan/prod.h"
#include "driver/can/timing.h"

namespace driver::can::pcan {
namespace {
const TimingLimits NOMINAL_LIMITS{
    .tseg1_max = 256,
    .tseg2_max = 128,
    .sjw_max = 128,
    .brp_max = 1024,
};

const TimingLimits DATA_LIMITS{
    .tseg1_max = 32,
    .tseg2_max = 16,
    .sjw_max = 16,
    .brp_max = 1024,
};

const std::vector<std::pair<std::uint32_t, TPCANBaudrate>> BAUD_CODES = {
    {1000000, PCAN_BAUD_1M},
    {800000, PCAN_BAUD_800K},
    {500000, PCAN_BAUD_500K},
    {250000, PCAN_BAUD_250K},
    {125000, PCAN_BAUD_125K},
    {100000, PCAN_BAUD_100K},
    {95238, PCAN_BAUD_95K},
    {83333, PCAN_BAUD_83K},
    {50000, PCAN_BAUD_50K},
    {47619, PCAN_BAUD_47K},
    {33333, PCAN_BAUD_33K},
    {20000, PCAN_BAUD_20K},
    {10000, PCAN_BAUD_10K},
    {5000, PCAN_BAUD_5K},
};

constexpr std::uint16_t CHANNELS_PER_KIND = 16;

/// @returns every USB, PCI, and LAN channel name with its handle.
std::vector<std::pair<std::string, TPCANHandle>> channels() {
    std::vector<std::pair<std::string, TPCANHandle>> out;
    for (std::uint16_t i = 1; i <= CHANNELS_PER_KIND; i++) {
        const auto usb = static_cast<TPCANHandle>(
            i <= 8 ? PCAN_USBBUS1 + i - 1 : PCAN_USBBUS9 + i - 9
        );
        out.emplace_back("PCAN_USBBUS" + std::to_string(i), usb);
    }
    for (std::uint16_t i = 1; i <= CHANNELS_PER_KIND; i++) {
        const auto pci = static_cast<TPCANHandle>(
            i <= 8 ? PCAN_PCIBUS1 + i - 1 : PCAN_PCIBUS9 + i - 9
        );
        out.emplace_back("PCAN_PCIBUS" + std::to_string(i), pci);
    }
    for (std::uint16_t i = 1; i <= CHANNELS_PER_KIND; i++)
        out.emplace_back(
            "PCAN_LANBUS" + std::to_string(i),
            static_cast<TPCANHandle>(PCAN_LANBUS1 + i - 1)
        );
    return out;
}

/// @brief copies a PCAN-Basic message into a frame.
void fill(
    Frame &frame,
    const std::uint32_t id,
    const TPCANMessageType type,
    const std::uint8_t length,
    const std::uint8_t *data
) {
    frame.id = id;
    frame.extended = (type & PCAN_MESSAGE_EXTENDED) != 0;
    frame.fd = (type & PCAN_MESSAGE_FD) != 0;
    frame.bitrate_switched = (type & PCAN_MESSAGE_BRS) != 0;
    frame.error_passive = (type & PCAN_MESSAGE_ESI) != 0;
    frame.type = Type::DATA;
    if ((type & PCAN_MESSAGE_RTR) != 0) frame.type = Type::REMOTE;
    if ((type & PCAN_MESSAGE_ERRFRAME) != 0) frame.type = Type::BUS_ERROR;
    frame.length = length;
    std::memcpy(frame.data.data(), data, length);
    frame.clock = Clock::HARDWARE;
}

/// @returns the PCAN-Basic message type bits of a frame.
TPCANMessageType message_type(const Frame &frame) {
    TPCANMessageType type = frame.extended ? PCAN_MESSAGE_EXTENDED
                                           : PCAN_MESSAGE_STANDARD;
    if (frame.type == Type::REMOTE) type |= PCAN_MESSAGE_RTR;
    if (frame.fd) type |= PCAN_MESSAGE_FD;
    if (frame.bitrate_switched) type |= PCAN_MESSAGE_BRS;
    return type;
}

constexpr TPCANStatus BUS_STATES = PCAN_ERROR_BUSLIGHT | PCAN_ERROR_BUSHEAVY |
                                   PCAN_ERROR_BUSPASSIVE;

/// @returns the BUS_ERROR frame of a bus state. PCAN-Basic does not say which error
/// counter reached a limit, so the frame sets the bits of both, as Linux does when the
/// counters are equal.
Frame state_frame(const TPCANStatus state) {
    std::uint8_t controller = bus_error::CONTROLLER_ACTIVE;
    if ((state & PCAN_ERROR_BUSPASSIVE) != 0)
        controller = bus_error::CONTROLLER_RX_PASSIVE |
                     bus_error::CONTROLLER_TX_PASSIVE;
    else if (state != PCAN_ERROR_OK)
        controller = bus_error::CONTROLLER_RX_WARNING |
                     bus_error::CONTROLLER_TX_WARNING;
    return bus_error::make(
        bus_error::CONTROLLER,
        controller,
        x::telem::TimeStamp::now()
    );
}

/// @returns the error of an Initialize that refused a channel.
x::errors::Error
refusal(API &api, const TPCANStatus status, const synnax::can::Properties &props) {
    const auto handle = status & PCAN_ERROR_ILLHANDLE;
    if (handle == PCAN_ERROR_ILLHW || handle == PCAN_ERROR_ILLCLIENT)
        return {UNREACHABLE_ERROR, props.channel + ": no adapter is attached"};
    if ((status & PCAN_ERROR_INITIALIZE) != 0 || handle == PCAN_ERROR_HWINUSE ||
        handle == PCAN_ERROR_NETINUSE)
        return {
            UNREACHABLE_ERROR,
            props.channel + ": another task or application holds the channel"
        };
    return translate(api, status, props.channel);
}
}

std::string describe(API &api, const TPCANStatus status) {
    std::array<char, 256> text{};
    if (api.GetErrorText(status, LANGUAGE_ENGLISH, text.data()) == PCAN_ERROR_OK)
        return {text.data()};
    return std::format("PCAN-Basic status 0x{:X}", status);
}

x::errors::Error
translate(API &api, const TPCANStatus status, const std::string &context) {
    const auto message = context + ": " + describe(api, status);
    // PCAN-Basic reports a USB channel with no adapter as ILLHW or ILLCLIENT
    // (forum.peak-system.com/viewtopic.php?t=2025) and resets a bus off controller only
    // on a new Initialize (PCAN-Basic documentation, CAN_GetStatus).
    const auto handle = status & PCAN_ERROR_ILLHANDLE;
    constexpr TPCANStatus absent = PCAN_ERROR_INITIALIZE | PCAN_ERROR_REGTEST |
                                   PCAN_ERROR_BUSOFF | PCAN_ERROR_CAUTION;
    if (handle == PCAN_ERROR_ILLHW || handle == PCAN_ERROR_ILLCLIENT ||
        handle == PCAN_ERROR_HWINUSE || handle == PCAN_ERROR_NETINUSE ||
        (status & absent) != 0)
        return {UNREACHABLE_ERROR, message};
    constexpr TPCANStatus config = PCAN_ERROR_NODRIVER | PCAN_ERROR_ILLPARAMTYPE |
                                   PCAN_ERROR_ILLPARAMVAL;
    if ((status & config) != 0) return {CONFIG_ERROR, message};
    if ((status & (PCAN_ERROR_XMTFULL | PCAN_ERROR_QXMTFULL)) != 0)
        return {TEMPORARY_HARDWARE_ERROR, message};
    return {CRITICAL_HARDWARE_ERROR, message};
}

std::optional<TPCANHandle> parse_channel(const std::string &name) {
    for (const auto &[candidate, handle]: channels())
        if (candidate == name) return handle;
    return std::nullopt;
}

std::pair<TPCANBaudrate, x::errors::Error> baud_code(const std::uint32_t bitrate) {
    return find_bitrate(BAUD_CODES, bitrate, "PCAN-Basic cannot run a classic CAN bus");
}

std::pair<std::string, x::errors::Error>
fd_bitrate(const std::uint32_t nominal, const std::uint32_t data) {
    const auto [n, n_err] = compute_timing(FD_CLOCK_HZ, nominal, NOMINAL_LIMITS);
    if (n_err) return {"", n_err};
    const auto [d, d_err] = compute_timing(FD_CLOCK_HZ, data, DATA_LIMITS);
    if (d_err) return {"", d_err};
    return {
        std::format(
            "f_clock={},nom_brp={},nom_tseg1={},nom_tseg2={},nom_sjw={},"
            "data_brp={},data_tseg1={},data_tseg2={},data_sjw={}",
            FD_CLOCK_HZ,
            n.brp,
            n.tseg1,
            n.tseg2,
            n.sjw,
            d.brp,
            d.tseg1,
            d.tseg2,
            d.sjw
        ),
        x::errors::NIL
    };
}

Bus::Bus(
    std::shared_ptr<API> api,
    const TPCANHandle handle,
    std::unique_ptr<ReceiveEvent> event,
    std::string name,
    const bool fd,
    const bool listen_only
):
    can::Bus(std::move(name), fd, listen_only),
    api(std::move(api)),
    handle(handle),
    event(std::move(event)) {}

// NOLINTNEXTLINE(bugprone-exception-escape): close only builds error strings.
Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    while (true) {
        if (auto res = this->take(frame)) return *res;
        if (auto res = this->apply(this->api->GetStatus(this->handle), frame))
            return *res;
        const auto now = std::chrono::steady_clock::now();
        if (now >= deadline) return {false, x::errors::NIL};
        const auto remaining = std::chrono::duration_cast<std::chrono::nanoseconds>(
            deadline - now
        );
        if (auto err = this->event->wait(x::telem::TimeSpan(remaining)))
            return {false, x::errors::Error(err, this->name + ": " + err.data)};
    }
}

std::optional<std::pair<bool, x::errors::Error>> Bus::take(Frame &frame) {
    while (true) {
        TPCANStatus status = 0;
        TPCANMessageType type = 0;
        if (this->fd) {
            TPCANMsgFD msg{};
            TPCANTimestampFD us = 0;
            status = this->api->ReadFD(this->handle, &msg, &us);
            type = msg.MSGTYPE;
            if (status == PCAN_ERROR_OK) {
                const bool fd_frame = (type & PCAN_MESSAGE_FD) != 0;
                fill(frame, msg.ID, type, dlc_to_length(msg.DLC, fd_frame), msg.DATA);
                frame.time = x::telem::TimeStamp(static_cast<std::int64_t>(us) * 1000);
            }
        } else {
            TPCANMsg msg{};
            TPCANTimestamp ts{};
            status = this->api->Read(this->handle, &msg, &ts);
            type = msg.MSGTYPE;
            if (status == PCAN_ERROR_OK) {
                const auto length = std::min<std::uint8_t>(msg.LEN, MAX_CLASSIC_LENGTH);
                fill(frame, msg.ID, type, length, msg.DATA);
                const std::uint64_t us = ts.micros + 1000ULL * ts.millis +
                                         0x100000000ULL * 1000ULL * ts.millis_overflow;
                frame.time = x::telem::TimeStamp(static_cast<std::int64_t>(us) * 1000);
            }
        }
        if (status == PCAN_ERROR_OK) {
            if ((type & PCAN_MESSAGE_STATUS) == 0)
                return std::pair{true, x::errors::NIL};
            if (auto res = this->apply(this->api->GetStatus(this->handle), frame))
                return res;
            continue;
        }
        if ((status & (PCAN_ERROR_OVERRUN | PCAN_ERROR_QOVERRUN)) != 0) {
            frame = bus_error::make(
                bus_error::CONTROLLER,
                bus_error::CONTROLLER_RX_OVERFLOW,
                x::telem::TimeStamp::now()
            );
            return std::pair{true, x::errors::NIL};
        }
        if ((status & PCAN_ERROR_QRCVEMPTY) != 0) return std::nullopt;
        return this->apply(status, frame);
    }
}

std::optional<std::pair<bool, x::errors::Error>>
Bus::apply(const TPCANStatus status, Frame &frame) {
    if (auto err = this->fault(status)) return std::pair{false, err};
    const auto state = status & BUS_STATES;
    if (state == this->state) return std::nullopt;
    this->state = state;
    frame = state_frame(state);
    return std::pair{true, x::errors::NIL};
}

x::errors::Error Bus::fault(TPCANStatus status) {
    status &= ~BUS_STATES;
    if (status == PCAN_ERROR_OK) return x::errors::NIL;
    // An unplugged channel reads empty and takes writes. Only CAN_GetStatus reports the
    // unplug (forum.peak-system.com/viewtopic.php?p=20202).
    if ((status & PCAN_ERROR_ILLHANDLE) == PCAN_ERROR_ILLHW)
        return {UNREACHABLE_ERROR, this->name + ": the adapter was unplugged"};
    if ((status & PCAN_ERROR_BUSOFF) != 0)
        return {UNREACHABLE_ERROR, this->name + ": bus off"};
    return translate(*this->api, status, this->name);
}

x::errors::Error Bus::transmit(const Frame &frame) {
    if (auto err = this->fault(this->api->GetStatus(this->handle))) return err;
    TPCANStatus status = 0;
    if (this->fd) {
        TPCANMsgFD msg{};
        msg.ID = frame.id;
        msg.MSGTYPE = message_type(frame);
        msg.DLC = length_to_dlc(frame.length).value();
        std::memcpy(msg.DATA, frame.data.data(), frame.length);
        status = this->api->WriteFD(this->handle, &msg);
    } else {
        TPCANMsg msg{};
        msg.ID = frame.id;
        msg.MSGTYPE = message_type(frame);
        msg.LEN = frame.length;
        std::memcpy(msg.DATA, frame.data.data(), frame.length);
        status = this->api->Write(this->handle, &msg);
    }
    if (status == PCAN_ERROR_OK) return x::errors::NIL;
    if ((status & (PCAN_ERROR_XMTFULL | PCAN_ERROR_QXMTFULL)) != 0)
        return {TEMPORARY_HARDWARE_ERROR, this->name + ": transmit queue is full"};
    return this->fault(status);
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    const auto status = this->api->Uninitialize(this->handle);
    this->event.reset();
    if (status == PCAN_ERROR_OK) return x::errors::NIL;
    return translate(*this->api, status, this->name);
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    std::vector<Channel> found;
    for (const auto &[name, handle]: channels()) {
        std::uint32_t condition = PCAN_CHANNEL_UNAVAILABLE;
        const auto status = this->api->GetValue(
            handle,
            PCAN_CHANNEL_CONDITION,
            &condition,
            sizeof(condition)
        );
        if (status != PCAN_ERROR_OK || condition == PCAN_CHANNEL_UNAVAILABLE) continue;
        std::array<char, MAX_LENGTH_HARDWARE_NAME> hardware{};
        this->api->GetValue(
            handle,
            PCAN_HARDWARE_NAME,
            hardware.data(),
            static_cast<std::uint32_t>(hardware.size())
        );
        std::uint32_t features = 0;
        this->api->GetValue(handle, PCAN_CHANNEL_FEATURES, &features, sizeof(features));
        std::string description = hardware[0] != '\0' ? hardware.data()
                                                      : "PEAK adapter";
        if ((features & FEATURE_FD_CAPABLE) != 0) description += ", CAN FD";
        if ((condition & PCAN_CHANNEL_OCCUPIED) != 0) description += ", in use";
        found.push_back({
            .backend = synnax::can::BACKEND_PCAN,
            .name = name,
            .description = description,
        });
    }
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    const auto handle = parse_channel(props.channel);
    if (!handle.has_value())
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "unknown PCAN-Basic channel '" + props.channel +
                    "'. Expected a name such as PCAN_USBBUS1"
            )
        };
    if (props.listen_only) {
        std::uint32_t on = PCAN_PARAMETER_ON;
        const auto status = this->api
                                ->SetValue(*handle, PCAN_LISTEN_ONLY, &on, sizeof(on));
        if (status != PCAN_ERROR_OK)
            return {
                nullptr,
                translate(*this->api, status, props.channel + " cannot listen only")
            };
    }
    TPCANStatus status = 0;
    if (props.fd) {
        auto [bitrate, err] = fd_bitrate(props.bitrate, props.data_bitrate);
        if (err) return {nullptr, err};
        status = this->api->InitializeFD(*handle, bitrate.data());
    } else {
        const auto [code, err] = baud_code(props.bitrate);
        if (err) return {nullptr, err};
        status = this->api->Initialize(*handle, code, 0, 0, 0);
    }
    if (status == PCAN_ERROR_CAUTION) {
        this->api->Uninitialize(*handle);
        return {
            nullptr,
            x::errors::Error(
                UNREACHABLE_ERROR,
                props.channel +
                    ": another application runs the channel at a different bitrate"
            )
        };
    }
    if (status != PCAN_ERROR_OK) return {nullptr, refusal(*this->api, status, props)};
    auto [event, err] = this->api->OpenReceiveEvent(*handle);
    if (err) {
        this->api->Uninitialize(*handle);
        return {nullptr, x::errors::Error(err, props.channel + ": " + err.data)};
    }
    return {
        std::make_unique<Bus>(
            this->api,
            *handle,
            std::move(event),
            props.channel,
            props.fd,
            props.listen_only
        ),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    return or_unavailable<Backend>(ProdAPI::load());
}
}
