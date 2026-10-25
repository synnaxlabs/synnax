// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cctype>
#include <cstring>
#include <format>

#include "driver/can/canlib/canlib.h"
#include "driver/can/canlib/prod.h"

namespace driver::can::canlib {
namespace {
const std::vector<std::pair<std::uint32_t, long>> BITRATES = {
    {1000000, canBITRATE_1M},
    {500000, canBITRATE_500K},
    {250000, canBITRATE_250K},
    {125000, canBITRATE_125K},
    {100000, canBITRATE_100K},
    {83333, canBITRATE_83K},
    {62500, canBITRATE_62K},
    {50000, canBITRATE_50K},
    {10000, canBITRATE_10K},
};

const std::vector<std::pair<std::uint32_t, long>> FD_ARBITRATION_BITRATES = {
    {500000, canFD_BITRATE_500K_80P},
    {1000000, canFD_BITRATE_1M_80P},
};

const std::vector<std::pair<std::uint32_t, long>> FD_DATA_BITRATES = {
    {500000, canFD_BITRATE_500K_80P},
    {1000000, canFD_BITRATE_1M_80P},
    {2000000, canFD_BITRATE_2M_80P},
    {4000000, canFD_BITRATE_4M_80P},
    {8000000, canFD_BITRATE_8M_60P},
};

/// @returns CANlib's description of a status code.
std::string describe(API &api, const canStatus status) {
    std::array<char, 256> text{};
    if (api.GetErrorText(status, text.data(), text.size()) == canOK)
        return {text.data()};
    return "CANlib status " + std::to_string(status);
}

/// @returns the EAN of a card from canCHANNELDATA_CARD_UPC_NO, which holds it as BCD
/// digits with the least significant first.
std::string ean(const std::uint64_t bcd) {
    const auto digits = std::format("{:013x}", bcd);
    return digits.substr(0, 2) + "-" + digits.substr(2, 5) + "-" + digits.substr(7, 5) +
           "-" + digits.substr(12, 1);
}

/// @returns true when every character of text in [from, to) is a digit.
bool digits(const std::string &text, const std::size_t from, const std::size_t to) {
    if (from >= to || to > text.size()) return false;
    return std::all_of(text.begin() + from, text.begin() + to, [](const char c) {
        return std::isdigit(static_cast<unsigned char>(c)) != 0;
    });
}

/// @returns true when name has the form <EAN>:<serial>:<channel>.
bool well_formed(const std::string &name) {
    constexpr std::size_t ean_length = 16;
    if (name.size() <= ean_length || name[ean_length] != ':') return false;
    if (name[2] != '-' || name[8] != '-' || name[14] != '-') return false;
    if (!digits(name, 0, 2) || !digits(name, 3, 8) || !digits(name, 9, 14) ||
        !digits(name, 15, 16))
        return false;
    const auto colon = name.find(':', ean_length + 1);
    return colon != std::string::npos && digits(name, ean_length + 1, colon) &&
           digits(name, colon + 1, name.size());
}

/// @returns the bus error frame of a CANlib error frame, whose flags hold its protocol
/// errors.
Frame error_frame(const unsigned int flags) {
    std::uint8_t kinds = 0;
    if ((flags & canMSGERR_STUFF) != 0) kinds |= bus_error::PROTOCOL_STUFF;
    if ((flags & canMSGERR_FORM) != 0) kinds |= bus_error::PROTOCOL_FORM;
    if ((flags & canMSGERR_BIT0) != 0) kinds |= bus_error::PROTOCOL_BIT0;
    if ((flags & canMSGERR_BIT1) != 0) kinds |= bus_error::PROTOCOL_BIT1;
    auto frame = bus_error::make(
        bus_error::BUS | bus_error::PROTOCOL,
        0,
        x::telem::TimeStamp::now()
    );
    frame.data[2] = kinds;
    return frame;
}
}

x::errors::Error
translate(API &api, const canStatus status, const std::string &context) {
    const auto message = context + ": " + describe(api, status);
    switch (status) {
        // linuxcan returns canERR_HARDWARE for a removed card and canERR_NOCHANNELS for
        // a channel another handle holds exclusively.
        case canERR_NOTFOUND:
        case canERR_NOCHANNELS:
        case canERR_NO_ACCESS:
        case canERR_HARDWARE:
        case canERR_NOCARD:
        case canERR_TIMEOUT:
        case canERR_INTERRUPTED:
            return {UNREACHABLE_ERROR, message};
        case canERR_PARAM:
        case canERR_DRIVER:
        case canERR_NOT_SUPPORTED:
        case canERR_NOT_IMPLEMENTED:
        case canERR_DYNALOAD:
        case canERR_DYNALIB:
        case canERR_DYNAINIT:
        case canERR_DRIVERLOAD:
        case canERR_REGISTRY:
        case canERR_LICENSE:
            return {CONFIG_ERROR, message};
        case canERR_TXBUFOFL:
            return {TEMPORARY_HARDWARE_ERROR, message};
        default:
            return {CRITICAL_HARDWARE_ERROR, message};
    }
}

std::pair<long, x::errors::Error> bitrate_constant(const std::uint32_t bitrate) {
    return find_bitrate(BITRATES, bitrate, "CANlib cannot run a classic CAN bus");
}

std::pair<long, x::errors::Error>
fd_bitrate_constant(const std::uint32_t bitrate, const bool data) {
    if (data)
        return find_bitrate(
            FD_DATA_BITRATES,
            bitrate,
            "CANlib cannot run a CAN FD data phase"
        );
    return find_bitrate(
        FD_ARBITRATION_BITRATES,
        bitrate,
        "CANlib cannot run a CAN FD arbitration phase"
    );
}

Bus::Bus(
    std::shared_ptr<API> api,
    const canHandle rx,
    const std::optional<canHandle> tx,
    std::string name,
    const bool fd
):
    can::Bus(std::move(name), fd, !tx.has_value()),
    api(std::move(api)),
    rx(rx),
    tx(tx) {}

// NOLINTNEXTLINE(bugprone-exception-escape): close only builds error strings.
Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    if (this->pending.has_value()) {
        frame = *this->pending;
        this->pending.reset();
        return {true, x::errors::NIL};
    }
    long id = 0;
    unsigned int dlc = 0;
    unsigned int flags = 0;
    unsigned long time = 0;
    std::array<std::uint8_t, MAX_FD_LENGTH> data{};
    const auto timeout_ms = static_cast<unsigned long>(
        (timeout.nanoseconds() + x::telem::MILLISECOND.nanoseconds() - 1) /
        x::telem::MILLISECOND.nanoseconds()
    );
    const auto status = this->api->ReadWait(
        this->rx,
        &id,
        data.data(),
        &dlc,
        &flags,
        &time,
        timeout_ms
    );
    if (status == canERR_NOMSG) return {false, x::errors::NIL};
    if (status != canOK)
        return {false, translate(*this->api, status, "CANlib channel " + this->name)};
    if ((flags & canMSG_ERROR_FRAME) != 0) {
        frame = error_frame(flags);
        return {true, x::errors::NIL};
    }
    frame.id = static_cast<std::uint32_t>(id);
    frame.extended = (flags & canMSG_EXT) != 0;
    frame.fd = (flags & canFDMSG_FDF) != 0;
    frame.bitrate_switched = (flags & canFDMSG_BRS) != 0;
    frame.error_passive = (flags & canFDMSG_ESI) != 0;
    frame.type = (flags & canMSG_RTR) != 0 ? Type::REMOTE : Type::DATA;
    const auto max = frame.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    frame.length = static_cast<std::uint8_t>(std::min<std::size_t>(dlc, max));
    std::memcpy(frame.data.data(), data.data(), frame.length);
    const auto ticks = this->counter.extend(
        static_cast<std::uint32_t>(time),
        steady_now()
    );
    frame.time = x::telem::TimeStamp(
        static_cast<std::int64_t>(ticks) * TIMER_SCALE.nanoseconds()
    );
    frame.clock = Clock::HARDWARE;
    if ((flags & canMSGERR_OVERRUN) != 0) {
        this->pending = frame;
        frame = bus_error::make(
            bus_error::CONTROLLER,
            bus_error::CONTROLLER_RX_OVERFLOW,
            x::telem::TimeStamp::now()
        );
    }
    return {true, x::errors::NIL};
}

x::errors::Error Bus::transmit(const Frame &frame) {
    unsigned int flags = frame.extended ? canMSG_EXT : canMSG_STD;
    if (frame.type == Type::REMOTE) flags |= canMSG_RTR;
    if (frame.fd) flags |= canFDMSG_FDF;
    if (frame.bitrate_switched) flags |= canFDMSG_BRS;
    auto data = frame.data;
    const auto status = this->api->Write(
        this->tx.value(),
        static_cast<long>(frame.id),
        data.data(),
        frame.length,
        flags
    );
    if (status == canOK) return x::errors::NIL;
    if (status == canERR_TXBUFOFL)
        return {
            TEMPORARY_HARDWARE_ERROR,
            "CANlib channel " + this->name + ": transmit queue is full"
        };
    return translate(*this->api, status, "CANlib channel " + this->name);
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    // canClose takes the handle off the bus, and finishes even when the card is gone.
    auto status = this->api->Close(this->rx);
    if (this->tx.has_value()) {
        const auto tx_status = this->api->Close(*this->tx);
        if (status == canOK) status = tx_status;
    }
    if (status == canOK) return x::errors::NIL;
    return translate(*this->api, status, "CANlib channel " + this->name);
}

std::pair<std::vector<Backend::Found>, x::errors::Error> Backend::enumerate() {
    int count = 0;
    if (const auto status = this->api->EnumHardwareEx(&count); status != canOK)
        return {{}, translate(*this->api, status, "CANlib")};
    std::vector<Found> found;
    for (int channel = 0; channel < count; channel++) {
        std::array<char, 256> description{};
        std::uint64_t upc = 0;
        std::uint64_t serial = 0;
        unsigned int number = 0;
        std::uint32_t cap = 0;
        const auto get = [&](const int item, void *buffer, const std::size_t size) {
            return this->api->GetChannelData(channel, item, buffer, size);
        };
        auto status = get(
            canCHANNELDATA_DEVDESCR_ASCII,
            description.data(),
            description.size()
        );
        if (status == canOK)
            status = get(canCHANNELDATA_CARD_UPC_NO, &upc, sizeof(upc));
        if (status == canOK)
            status = get(canCHANNELDATA_CARD_SERIAL_NO, &serial, sizeof(serial));
        if (status == canOK)
            status = get(canCHANNELDATA_CHAN_NO_ON_CARD, &number, sizeof(number));
        if (status == canOK)
            status = get(canCHANNELDATA_CHANNEL_CAP, &cap, sizeof(cap));
        if (status != canOK)
            return {
                {},
                translate(
                    *this->api,
                    status,
                    "CANlib channel number " + std::to_string(channel)
                )
            };
        found.push_back({
            .number = channel,
            .name = std::format("{}:{}:{}", ean(upc), serial, number + 1),
            .description = std::format(
                "{}, S/N {}, channel {}",
                description.data(),
                serial,
                number + 1
            ),
            .is_virtual = (cap & canCHANNEL_CAP_VIRTUAL) != 0,
        });
    }
    return {found, x::errors::NIL};
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    std::lock_guard lock(this->mu);
    auto [found, err] = this->enumerate();
    if (err) return {{}, err};
    std::vector<Channel> channels;
    for (auto &f: found)
        channels.push_back({
            .backend = synnax::can::BACKEND_CANLIB,
            .name = std::move(f.name),
            .description = std::move(f.description),
        });
    return {channels, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    if (!well_formed(props.channel))
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "CANlib channel '" + props.channel +
                    "' must name a card and a channel on it, such as "
                    "73-30130-00685-0:12345:1"
            )
        };
    long nominal = 0;
    long data = 0;
    if (props.fd) {
        auto [n, n_err] = fd_bitrate_constant(props.bitrate, false);
        if (n_err) return {nullptr, n_err};
        auto [d, d_err] = fd_bitrate_constant(props.data_bitrate, true);
        if (d_err) return {nullptr, d_err};
        nominal = n;
        data = d;
    } else {
        auto [n, err] = bitrate_constant(props.bitrate);
        if (err) return {nullptr, err};
        nominal = n;
    }
    const auto context = "CANlib channel " + props.channel;
    std::lock_guard lock(this->mu);
    auto [found, enum_err] = this->enumerate();
    if (enum_err) return {nullptr, enum_err};
    const auto match = std::ranges::find(found, props.channel, &Found::name);
    if (match == found.end())
        return {
            nullptr,
            x::errors::Error(UNREACHABLE_ERROR, context + ": no adapter is attached")
        };
    const auto number = match->number;
    const int flags = (match->is_virtual ? canOPEN_ACCEPT_VIRTUAL : 0) |
                      (props.fd ? canOPEN_CAN_FD : 0);
    auto &api = *this->api;
    const auto rx = api.OpenChannel(number, flags);
    if (rx < 0) return {nullptr, translate(api, rx, context)};
    std::optional<canHandle> tx;
    const auto fail = [&](const canStatus status) {
        if (tx.has_value()) api.Close(*tx);
        api.Close(rx);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{
            nullptr,
            translate(api, status, context)
        };
    };
    auto scale = static_cast<unsigned long>(
        (TIMER_SCALE / x::telem::MICROSECOND).nanoseconds()
    );
    if (const auto s = api.IoCtl(rx, canIOCTL_SET_TIMER_SCALE, &scale, sizeof(scale));
        s != canOK)
        return fail(s);
    unsigned char echo = 0;
    if (const auto s = api.IoCtl(rx, canIOCTL_SET_LOCAL_TXECHO, &echo, sizeof(echo));
        s != canOK)
        return fail(s);
    if (const auto s = api.SetBusParams(rx, nominal, 0, 0, 0, 0, 0); s != canOK)
        return fail(s);
    if (props.fd)
        if (const auto s = api.SetBusParamsFd(rx, data, 0, 0, 0); s != canOK)
            return fail(s);
    const auto driver = props.listen_only ? canDRIVER_SILENT : canDRIVER_NORMAL;
    if (const auto s = api.SetBusOutputControl(rx, driver); s != canOK) return fail(s);
    if (!props.listen_only) {
        const auto h = api.OpenChannel(number, flags);
        if (h < 0) return fail(h);
        tx = h;
    }
    if (const auto s = api.BusOn(rx); s != canOK) return fail(s);
    if (tx.has_value())
        if (const auto s = api.BusOn(*tx); s != canOK) return fail(s);
    return {
        std::make_unique<Bus>(this->api, rx, tx, props.channel, props.fd),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    return or_unavailable<Backend>(ProdAPI::load());
}
}
