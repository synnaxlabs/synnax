// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <format>

#include "x/cpp/binary/binary.h"

#include "driver/can/nixnet/nixnet.h"
#include "driver/can/nixnet/prod.h"

namespace driver::can::nixnet {
namespace {
constexpr std::size_t FLAGS_OFFSET = 13;
constexpr std::size_t LENGTH_OFFSET = 15;
constexpr std::int64_t TICK_NS = 100;

/// @returns NI-XNET's description of a status code.
std::string describe(API &api, const nxStatus_t status) {
    std::array<char, 2048> text{};
    api.StatusToString(status, static_cast<u32>(text.size()), text.data());
    return {text.data()};
}

/// @returns the error for a failed call on an interface.
x::errors::Error
hardware_error(API &api, const std::string &name, const nxStatus_t status) {
    return {TEMPORARY_HARDWARE_ERROR, "NI-XNET " + name + ": " + describe(api, status)};
}

/// @brief reads a string property of a system object.
std::string string_property(API &api, const nxSessionRef_t ref, const u32 id) {
    u32 size = 0;
    if (api.GetPropertySize(ref, id, &size) != nxSuccess || size == 0) return "";
    std::string value(size, '\0');
    if (api.GetProperty(ref, id, size, value.data()) != nxSuccess) return "";
    value.resize(std::strlen(value.c_str()));
    return value;
}
}

std::size_t raw_size(const std::uint8_t length) {
    const std::size_t padded = (static_cast<std::size_t>(length) + 7) & ~std::size_t{7};
    return HEADER_SIZE + std::max<std::size_t>(padded, 8);
}

std::vector<std::uint8_t> encode(const Frame &frame, const bool fd) {
    std::vector<std::uint8_t> out;
    x::binary::Writer writer(out, raw_size(frame.length));
    auto id = frame.id;
    if (frame.extended) id |= nxFrameId_CAN_IsExtended;
    u8 type = fd ? nxFrameType_CAN20_Data : nxFrameType_CAN_Data;
    if (frame.fd)
        type = frame.bitrate_switched ? nxFrameType_CANFDBRS_Data
                                      : nxFrameType_CANFD_Data;
    if (frame.type == Type::REMOTE) type = nxFrameType_CAN_Remote;
    writer.uint64(0);
    writer.uint32(id);
    writer.uint8(type);
    writer.uint8(0);
    writer.uint8(0);
    writer.uint8(frame.length);
    if (frame.type != Type::REMOTE) writer.write(frame.data.data(), frame.length);
    return out;
}

std::pair<std::size_t, x::errors::Error>
decode(const std::span<const std::uint8_t> bytes, Frame &frame) {
    const auto truncated = x::errors::Error(
        CRITICAL_HARDWARE_ERROR,
        "NI-XNET returned a truncated frame"
    );
    if (bytes.size() < HEADER_SIZE + 8) return {0, truncated};
    const auto length = std::min<std::uint8_t>(bytes[LENGTH_OFFSET], MAX_FD_LENGTH);
    const auto size = raw_size(length);
    if (bytes.size() < size) return {0, truncated};
    x::binary::Reader reader(bytes.data(), bytes.size());
    const auto ticks = reader.uint64();
    const auto id = reader.uint32();
    const auto type = reader.uint8();
    frame.extended = (id & nxFrameId_CAN_IsExtended) != 0;
    frame.id = id & ~nxFrameId_CAN_IsExtended;
    frame.fd = type == nxFrameType_CANFD_Data || type == nxFrameType_CANFDBRS_Data;
    frame.bitrate_switched = type == nxFrameType_CANFDBRS_Data;
    frame.error_passive = false;
    frame.type = Type::DATA;
    if (type == nxFrameType_CAN_Remote) frame.type = Type::REMOTE;
    if (type == nxFrameType_CAN_BusError) frame.type = Type::BUS_ERROR;
    frame.length = length;
    std::memcpy(frame.data.data(), bytes.data() + HEADER_SIZE, length);
    frame.time = x::telem::TimeStamp(
        (static_cast<std::int64_t>(ticks) -
         static_cast<std::int64_t>(UNIX_EPOCH_TICKS)) *
        TICK_NS
    );
    frame.clock = Clock::HARDWARE;
    return {size, x::errors::NIL};
}

Bus::Bus(
    std::shared_ptr<API> api,
    const nxSessionRef_t in,
    const std::optional<nxSessionRef_t> out,
    std::string name,
    const bool fd
):
    can::Bus(std::move(name), fd, !out.has_value()),
    api(std::move(api)),
    in(in),
    out(out) {}

Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    return poll_queue(timeout, POLL_INTERVAL, [&] { return this->take(frame); });
}

std::optional<std::pair<bool, x::errors::Error>> Bus::take(Frame &frame) {
    while (true) {
        while (this->offset < this->size) {
            const std::span<const std::uint8_t> pending(
                this->buffer.data() + this->offset,
                this->size - this->offset
            );
            const auto flags = pending.size() > FLAGS_OFFSET ? pending[FLAGS_OFFSET]
                                                             : 0;
            auto [consumed, err] = decode(pending, frame);
            if (err) {
                this->offset = this->size;
                return std::pair{false, err};
            }
            this->offset += consumed;
            if ((flags & nxFrameFlags_TransmitEcho) != 0) continue;
            return std::pair{true, x::errors::NIL};
        }
        u32 returned = 0;
        const auto status = this->api->ReadFrame(
            this->in,
            this->buffer.data(),
            static_cast<u32>(this->buffer.size()),
            0,
            &returned
        );
        if (status < nxSuccess)
            return std::pair{false, hardware_error(*this->api, this->name, status)};
        this->offset = 0;
        this->size = returned;
        if (returned == 0) return std::nullopt;
    }
}

x::errors::Error Bus::transmit(const Frame &frame) {
    auto raw = encode(frame, this->fd);
    const auto status = this->api->WriteFrame(
        *this->out,
        raw.data(),
        static_cast<u32>(raw.size()),
        0
    );
    if (status < nxSuccess) return hardware_error(*this->api, this->name, status);
    return x::errors::NIL;
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    auto status = this->api->Clear(this->in);
    if (this->out.has_value()) {
        const auto out_status = this->api->Clear(*this->out);
        if (status >= nxSuccess) status = out_status;
    }
    if (status >= nxSuccess) return x::errors::NIL;
    return {
        CRITICAL_HARDWARE_ERROR,
        "NI-XNET " + this->name + ": " + describe(*this->api, status)
    };
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    auto &api = *this->api;
    nxSessionRef_t system = 0;
    if (const auto status = api.SystemOpen(&system); status < nxSuccess)
        return {{}, hardware_error(api, "system", status)};
    std::vector<Channel> found;
    u32 size = 0;
    auto status = api.GetPropertySize(system, nxPropSys_IntfRefsCAN, &size);
    std::vector<nxSessionRef_t> interfaces(size / sizeof(nxSessionRef_t));
    if (status >= nxSuccess && !interfaces.empty())
        status = api.GetProperty(
            system,
            nxPropSys_IntfRefsCAN,
            size,
            interfaces.data()
        );
    if (status < nxSuccess) {
        api.SystemClose(system);
        return {{}, hardware_error(api, "system", status)};
    }
    for (const auto intf: interfaces) {
        const auto name = string_property(api, intf, nxPropIntf_Name);
        if (name.empty()) continue;
        nxSessionRef_t device = 0;
        std::string description = "NI-XNET interface";
        if (api.GetProperty(intf, nxPropIntf_DevRef, sizeof(device), &device) ==
            nxSuccess) {
            const auto product = string_property(api, device, nxPropDev_Name);
            u32 serial = 0;
            api.GetProperty(device, nxPropDev_SerNum, sizeof(serial), &serial);
            if (!product.empty())
                description = product + ", S/N " + std::format("{:X}", serial);
        }
        found.push_back({
            .backend = synnax::can::BACKEND_NIXNET,
            .name = name,
            .description = description,
        });
    }
    api.SystemClose(system);
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    if (props.channel.empty())
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "NI-XNET needs an interface name, such as CAN1"
            )
        };
    auto &api = *this->api;
    std::vector<nxSessionRef_t> sessions;
    const auto fail = [&](const nxStatus_t status) {
        for (const auto s: sessions)
            api.Clear(s);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{
            nullptr,
            hardware_error(api, props.channel, status)
        };
    };
    const auto create = [&](const u32 mode) -> nxStatus_t {
        nxSessionRef_t session = 0;
        auto status = api.CreateSession(
            ":memory:",
            "",
            "",
            props.channel.c_str(),
            mode,
            &session
        );
        if (status < nxSuccess) return status;
        sessions.push_back(session);
        u64 bitrate = props.bitrate;
        status = api.SetProperty(
            session,
            nxPropSession_IntfBaudRate64,
            sizeof(bitrate),
            &bitrate
        );
        if (status < nxSuccess) return status;
        if (props.fd) {
            u32 io_mode = nxCANioMode_CAN_FD_BRS;
            status = api.SetProperty(
                session,
                nxPropSession_IntfCanIoMode,
                sizeof(io_mode),
                &io_mode
            );
            if (status < nxSuccess) return status;
            u64 data_bitrate = props.data_bitrate;
            status = api.SetProperty(
                session,
                nxPropSession_IntfCanFdBaudRate64,
                sizeof(data_bitrate),
                &data_bitrate
            );
            if (status < nxSuccess) return status;
        }
        if (props.listen_only) {
            u8 on = 1;
            status = api.SetProperty(
                session,
                nxPropSession_IntfCANLstnOnly,
                sizeof(on),
                &on
            );
            if (status < nxSuccess) return status;
        }
        return api.Start(session, nxStartStop_Normal);
    };
    if (const auto status = create(nxMode_FrameInStream); status < nxSuccess)
        return fail(status);
    std::optional<nxSessionRef_t> out;
    if (!props.listen_only) {
        if (const auto status = create(nxMode_FrameOutStream); status < nxSuccess)
            return fail(status);
        out = sessions.back();
    }
    return {
        std::make_unique<Bus>(this->api, sessions[0], out, props.channel, props.fd),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    return or_unavailable<Backend>(ProdAPI::load());
}
}
