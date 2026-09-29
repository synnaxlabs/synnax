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
#include <charconv>
#include <cstring>

#include "absl/log/log.h"

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

std::pair<long, x::errors::Error> lookup(
    const std::vector<std::pair<std::uint32_t, long>> &table,
    const std::uint32_t bitrate,
    const std::string &kind
) {
    std::string supported;
    for (const auto &[rate, constant]: table) {
        if (rate == bitrate) return {constant, x::errors::NIL};
        supported += (supported.empty() ? "" : ", ") + std::to_string(rate);
    }
    return {
        0,
        {CONFIG_ERROR,
         "CANlib cannot run a " + kind + " at " + std::to_string(bitrate) +
             " bit/s. Supported bitrates: " + supported}
    };
}

/// @returns CANlib's description of a status code.
std::string describe(API &api, const canStatus status) {
    std::array<char, 256> text{};
    if (api.GetErrorText(status, text.data(), text.size()) == canOK)
        return {text.data()};
    return "CANlib status " + std::to_string(status);
}

/// @returns the error for a failed call while opening a channel.
x::errors::Error open_error(API &api, const std::string &name, const canStatus status) {
    const auto &type = status == canERR_PARAM ? CONFIG_ERROR : TEMPORARY_HARDWARE_ERROR;
    return {type, "CANlib channel " + name + ": " + describe(api, status)};
}

/// @brief takes a handle off the bus and closes it.
canStatus release(API &api, const canHandle hnd) {
    const auto off = api.BusOff(hnd);
    const auto closed = api.Close(hnd);
    return off != canOK ? off : closed;
}
}

std::pair<long, x::errors::Error> bitrate_constant(const std::uint32_t bitrate) {
    return lookup(BITRATES, bitrate, "classic CAN bus");
}

std::pair<long, x::errors::Error>
fd_bitrate_constant(const std::uint32_t bitrate, const bool data) {
    if (data) return lookup(FD_DATA_BITRATES, bitrate, "CAN FD data phase");
    return lookup(FD_ARBITRATION_BITRATES, bitrate, "CAN FD arbitration phase");
}

Bus::Bus(
    std::shared_ptr<API> api,
    const canHandle rx,
    const std::optional<canHandle> tx,
    std::string name,
    const bool fd
):
    api(std::move(api)), rx(rx), tx(tx), name(std::move(name)), fd(fd) {}

Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
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
        return {
            false,
            {TEMPORARY_HARDWARE_ERROR,
             "CANlib channel " + this->name + ": " + describe(*this->api, status)}
        };
    if ((flags & canMSGERR_OVERRUN) != 0)
        LOG(WARNING) << "[can] CANlib channel " << this->name
                     << ": receive buffer overran and frames were lost";
    frame.id = static_cast<std::uint32_t>(id);
    frame.extended = (flags & canMSG_EXT) != 0;
    frame.fd = (flags & canFDMSG_FDF) != 0;
    frame.bitrate_switched = (flags & canFDMSG_BRS) != 0;
    frame.error_passive = (flags & canFDMSG_ESI) != 0;
    frame.type = Type::DATA;
    if ((flags & canMSG_RTR) != 0) frame.type = Type::REMOTE;
    if ((flags & canMSG_ERROR_FRAME) != 0) frame.type = Type::BUS_ERROR;
    const auto max = frame.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    frame.length = static_cast<std::uint8_t>(std::min<std::size_t>(dlc, max));
    std::memcpy(frame.data.data(), data.data(), frame.length);
    const auto ticks = this->counter.extend(static_cast<std::uint32_t>(time));
    frame.time = x::telem::TimeStamp(
        static_cast<std::int64_t>(ticks) * TIMER_SCALE.nanoseconds()
    );
    frame.clock = Clock::HARDWARE;
    return {true, x::errors::NIL};
}

x::errors::Error Bus::send(const Frame &frame) {
    if (!this->tx.has_value())
        return {LISTEN_ONLY_ERROR, "channel " + this->name + " is listen only"};
    if (auto err = validate(frame, this->fd)) return err;
    unsigned int flags = frame.extended ? canMSG_EXT : canMSG_STD;
    if (frame.type == Type::REMOTE) flags |= canMSG_RTR;
    if (frame.fd) flags |= canFDMSG_FDF;
    if (frame.bitrate_switched) flags |= canFDMSG_BRS;
    auto data = frame.data;
    const auto status = this->api->Write(
        *this->tx,
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
    return {
        TEMPORARY_HARDWARE_ERROR,
        "CANlib channel " + this->name + ": " + describe(*this->api, status)
    };
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    auto status = release(*this->api, this->rx);
    if (this->tx.has_value()) {
        const auto tx_status = release(*this->api, *this->tx);
        if (status == canOK) status = tx_status;
    }
    if (status == canOK) return x::errors::NIL;
    return {
        CRITICAL_HARDWARE_ERROR,
        "CANlib channel " + this->name + ": " + describe(*this->api, status)
    };
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    int count = 0;
    if (const auto status = this->api->GetNumberOfChannels(&count); status != canOK)
        return {
            {},
            {TEMPORARY_HARDWARE_ERROR, "CANlib: " + describe(*this->api, status)}
        };
    std::vector<Channel> found;
    for (int channel = 0; channel < count; channel++) {
        std::array<char, 256> description{};
        std::uint64_t serial = 0;
        unsigned int number = 0;
        this->api->GetChannelData(
            channel,
            canCHANNELDATA_DEVDESCR_ASCII,
            description.data(),
            description.size()
        );
        this->api->GetChannelData(
            channel,
            canCHANNELDATA_CARD_SERIAL_NO,
            &serial,
            sizeof(serial)
        );
        this->api->GetChannelData(
            channel,
            canCHANNELDATA_CHAN_NO_ON_CARD,
            &number,
            sizeof(number)
        );
        found.push_back({
            .backend = synnax::can::BACKEND_CANLIB,
            .name = std::to_string(channel),
            .description = std::string(description.data()) + ", S/N " +
                           std::to_string(serial) + ", channel " +
                           std::to_string(number + 1),
        });
    }
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    int number = -1;
    const auto *end = props.channel.data() + props.channel.size();
    const auto [ptr, ec] = std::from_chars(props.channel.data(), end, number);
    if (ec != std::errc() || ptr != end || number < 0)
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "CANlib channel '" + props.channel +
                    "' must be a channel number, such as 0"
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
    const int flags = canOPEN_ACCEPT_VIRTUAL | (props.fd ? canOPEN_CAN_FD : 0);
    auto &api = *this->api;
    const auto rx = api.OpenChannel(number, flags);
    if (rx < 0) {
        const auto &type = rx == canERR_NOTFOUND ? CONFIG_ERROR
                                                 : TEMPORARY_HARDWARE_ERROR;
        return {
            nullptr,
            x::errors::Error(
                type,
                "CANlib channel " + props.channel + ": " + describe(api, rx)
            )
        };
    }
    const auto fail = [&](const canStatus status) {
        api.Close(rx);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{
            nullptr,
            open_error(api, props.channel, status)
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
    std::optional<canHandle> tx;
    if (!props.listen_only) {
        const auto h = api.OpenChannel(number, flags);
        if (h < 0) return fail(h);
        tx = h;
    }
    const auto release_tx = [&] {
        if (tx.has_value()) api.Close(*tx);
    };
    if (const auto s = api.BusOn(rx); s != canOK) {
        release_tx();
        return fail(s);
    }
    if (tx.has_value())
        if (const auto s = api.BusOn(*tx); s != canOK) {
            release_tx();
            api.BusOff(rx);
            return fail(s);
        }
    return {
        std::make_unique<Bus>(this->api, rx, tx, props.channel, props.fd),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    auto [api, err] = ProdAPI::load();
    if (err) return std::make_shared<Unavailable>(err);
    return std::make_shared<Backend>(api);
}
}
