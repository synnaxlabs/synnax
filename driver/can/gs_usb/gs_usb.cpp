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
#include <chrono>
#include <string>

#include "absl/log/log.h"

#include "driver/can/gs_usb/gs_usb.h"
#include "driver/can/gs_usb/prod.h"

namespace driver::can::gs_usb {
namespace {
const auto USB_TIMEOUT_MS = static_cast<unsigned int>(
    (USB_TIMEOUT / x::telem::MILLISECOND).nanoseconds()
);

bool is_gs_usb(const libusb_device_descriptor &desc) {
    return std::ranges::any_of(USB_IDS, [&](const UsbId &id) {
        return id.vendor == desc.idVendor && id.product == desc.idProduct;
    });
}

std::string describe(API &api, const int rc) {
    const char *name = api.ErrorName(rc);
    return name == nullptr ? "libusb error " + std::to_string(rc) : std::string(name);
}

/// @brief reads a string descriptor, or returns an empty string when the adapter has
/// none.
std::string
read_string(API &api, libusb_device_handle *handle, const std::uint8_t index) {
    if (index == 0) return "";
    std::array<unsigned char, 256> text{};
    const int n = api.GetStringDescriptorAscii(
        handle,
        index,
        text.data(),
        static_cast<int>(text.size())
    );
    if (n <= 0) return "";
    return {reinterpret_cast<const char *>(text.data()), static_cast<std::size_t>(n)};
}

/// @brief sends a vendor request with a body to the adapter.
int control_out(
    API &api,
    libusb_device_handle *handle,
    const Request request,
    const std::uint16_t channel,
    std::span<std::uint8_t> body
) {
    return api.ControlTransfer(
        handle,
        REQUEST_OUT,
        static_cast<std::uint8_t>(request),
        channel,
        INTERFACE,
        body.data(),
        static_cast<std::uint16_t>(body.size()),
        USB_TIMEOUT_MS
    );
}

/// @brief reads the answer to a vendor request into out.
/// @returns the number of bytes read, or a negative libusb error.
int control_in(
    API &api,
    libusb_device_handle *handle,
    const Request request,
    const std::uint16_t channel,
    std::span<std::uint8_t> out
) {
    return api.ControlTransfer(
        handle,
        REQUEST_IN,
        static_cast<std::uint8_t>(request),
        channel,
        INTERFACE,
        out.data(),
        static_cast<std::uint16_t>(out.size()),
        USB_TIMEOUT_MS
    );
}

/// @brief an open adapter found by serial number.
struct Found {
    libusb_device_handle *handle = nullptr;
    std::string product;
};

/// @brief opens the gs_usb adapter with a serial number.
/// @returns a null handle when no attached adapter has the serial number.
std::pair<Found, x::errors::Error> find(API &api, const std::string &serial) {
    libusb_device **list = nullptr;
    const auto count = api.GetDeviceList(&list);
    if (count < 0)
        return {
            {},
            {TEMPORARY_HARDWARE_ERROR,
             "libusb cannot list devices: " + describe(api, static_cast<int>(count))}
        };
    Found found;
    x::errors::Error err = x::errors::NIL;
    for (std::ptrdiff_t i = 0; i < count && found.handle == nullptr; i++) {
        libusb_device_descriptor desc{};
        if (api.GetDeviceDescriptor(list[i], &desc) != LIBUSB_SUCCESS ||
            !is_gs_usb(desc))
            continue;
        libusb_device_handle *handle = nullptr;
        if (const int rc = api.Open(list[i], &handle); rc != LIBUSB_SUCCESS) {
            err = {
                TEMPORARY_HARDWARE_ERROR,
                "cannot open a gs_usb adapter: " + describe(api, rc)
            };
            continue;
        }
        if (read_string(api, handle, desc.iSerialNumber) != serial) {
            api.Close(handle);
            continue;
        }
        found = {.handle = handle, .product = read_string(api, handle, desc.iProduct)};
    }
    api.FreeDeviceList(list, 1);
    if (found.handle == nullptr && err) return {found, err};
    return {found, x::errors::NIL};
}

/// @brief reads the capabilities of an adapter channel.
std::pair<BtConst, x::errors::Error> read_bt_const(
    API &api,
    libusb_device_handle *handle,
    const std::uint16_t channel,
    const bool extended
) {
    std::array<std::uint8_t, 72> reply{};
    const auto size = extended ? 72 : 40;
    const int n = control_in(
        api,
        handle,
        extended ? Request::BT_CONST_EXT : Request::BT_CONST,
        channel,
        std::span(reply.data(), size)
    );
    if (n < 0)
        return {
            {},
            {TEMPORARY_HARDWARE_ERROR,
             "gs_usb adapter did not report its bit timing: " + describe(api, n)}
        };
    return decode_bt_const(
        std::span(reply.data(), static_cast<std::size_t>(n)),
        extended
    );
}
}

Bus::Bus(
    std::shared_ptr<API> api,
    libusb_device_handle *handle,
    const std::uint8_t channel,
    std::string name,
    const bool fd,
    const bool listen_only,
    const bool timestamped
):
    api(std::move(api)),
    handle(handle),
    channel(channel),
    name(std::move(name)),
    fd(fd),
    listen_only(listen_only),
    timestamped(timestamped) {}

Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    std::array<std::uint8_t, MAX_FRAME_SIZE> buffer{};
    while (true) {
        const auto remaining = std::chrono::ceil<std::chrono::milliseconds>(
            deadline - std::chrono::steady_clock::now()
        );
        const auto ms = static_cast<unsigned int>(
            std::max<std::int64_t>(remaining.count(), 1)
        );
        int transferred = 0;
        const int rc = this->api->BulkTransfer(
            this->handle,
            ENDPOINT_IN,
            buffer.data(),
            static_cast<int>(buffer.size()),
            &transferred,
            ms
        );
        if (rc == LIBUSB_ERROR_TIMEOUT && transferred == 0)
            return {false, x::errors::NIL};
        if (rc == LIBUSB_ERROR_NO_DEVICE)
            return {
                false,
                {TEMPORARY_HARDWARE_ERROR,
                 "gs_usb adapter " + this->name + " unplugged"}
            };
        if (rc != LIBUSB_SUCCESS && rc != LIBUSB_ERROR_TIMEOUT)
            return {
                false,
                {TEMPORARY_HARDWARE_ERROR,
                 "gs_usb adapter " + this->name + ": " + describe(*this->api, rc)}
            };
        auto [hf, err] = decode_frame(
            std::span(buffer.data(), static_cast<std::size_t>(transferred)),
            this->timestamped
        );
        if (err) return {false, err};
        if (hf.overflowed)
            LOG(WARNING) << "[can] gs_usb adapter " << this->name
                         << " overran and frames were lost";
        if (hf.echo_id == ECHO_ID_RX && hf.channel == this->channel) {
            frame = hf.frame;
            if (hf.timestamp_us.has_value()) {
                const auto us = this->counter.extend(*hf.timestamp_us);
                frame.time = x::telem::TimeStamp(static_cast<std::int64_t>(us) * 1000);
                frame.clock = Clock::HARDWARE;
            } else {
                frame.time = x::telem::TimeStamp::now();
                frame.clock = Clock::HOST;
            }
            return {true, x::errors::NIL};
        }
        if (std::chrono::steady_clock::now() >= deadline)
            return {false, x::errors::NIL};
    }
}

x::errors::Error Bus::send(const Frame &frame) {
    if (this->listen_only)
        return {LISTEN_ONLY_ERROR, "channel " + this->name + " is listen only"};
    if (auto err = validate(frame, this->fd)) return err;
    std::array<std::uint8_t, MAX_FRAME_SIZE> buffer{};
    const auto echo = this->next_echo.fetch_add(1) % ECHO_IDS;
    const auto size = encode_frame(frame, echo, this->channel, buffer);
    int transferred = 0;
    const int rc = this->api->BulkTransfer(
        this->handle,
        ENDPOINT_OUT,
        buffer.data(),
        static_cast<int>(size),
        &transferred,
        USB_TIMEOUT_MS
    );
    if (rc == LIBUSB_SUCCESS) return x::errors::NIL;
    if (rc == LIBUSB_ERROR_TIMEOUT)
        return {
            TEMPORARY_HARDWARE_ERROR,
            "gs_usb adapter " + this->name + ": transmit queue is full"
        };
    return {
        TEMPORARY_HARDWARE_ERROR,
        "gs_usb adapter " + this->name + ": " + describe(*this->api, rc)
    };
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    auto mode = encode_mode(MODE_RESET, 0);
    const int
        rc = control_out(*this->api, this->handle, Request::MODE, this->channel, mode);
    this->api->ReleaseInterface(this->handle, INTERFACE);
    this->api->Close(this->handle);
    if (rc >= 0 || rc == LIBUSB_ERROR_NO_DEVICE) return x::errors::NIL;
    return {
        CRITICAL_HARDWARE_ERROR,
        "gs_usb adapter " + this->name + ": " + describe(*this->api, rc)
    };
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    auto &api = *this->api;
    libusb_device **list = nullptr;
    const auto count = api.GetDeviceList(&list);
    if (count < 0)
        return {
            {},
            {TEMPORARY_HARDWARE_ERROR,
             "libusb cannot list devices: " + describe(api, static_cast<int>(count))}
        };
    std::vector<Channel> found;
    for (std::ptrdiff_t i = 0; i < count; i++) {
        libusb_device_descriptor desc{};
        if (api.GetDeviceDescriptor(list[i], &desc) != LIBUSB_SUCCESS ||
            !is_gs_usb(desc))
            continue;
        libusb_device_handle *handle = nullptr;
        if (const int rc = api.Open(list[i], &handle); rc != LIBUSB_SUCCESS) {
            LOG(WARNING) << "[can] cannot open the gs_usb adapter at USB bus "
                         << static_cast<int>(api.GetBusNumber(list[i])) << " address "
                         << static_cast<int>(api.GetDeviceAddress(list[i])) << ": "
                         << describe(api, rc);
            continue;
        }
        const auto serial = read_string(api, handle, desc.iSerialNumber);
        auto product = read_string(api, handle, desc.iProduct);
        if (product.empty()) product = "gs_usb adapter";
        std::uint32_t channels = 1;
        std::array<std::uint8_t, 12> reply{};
        const int n = control_in(api, handle, Request::DEVICE_CONFIG, 0, reply);
        if (n > 0) {
            const auto [config, err] = decode_device_config(
                std::span(reply.data(), static_cast<std::size_t>(n))
            );
            if (!err) channels = config.channels;
        }
        api.Close(handle);
        if (serial.empty()) {
            LOG(WARNING) << "[can] skipping a gs_usb adapter with no serial number";
            continue;
        }
        for (std::uint32_t c = 0; c < channels; c++)
            found.push_back({
                .backend = synnax::can::BACKEND_GS_USB,
                .name = serial + ":" + std::to_string(c),
                .description = product + " channel " + std::to_string(c),
            });
    }
    api.FreeDeviceList(list, 1);
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    auto &api = *this->api;
    std::string serial = props.channel;
    std::uint32_t index = 0;
    if (const auto colon = props.channel.rfind(':'); colon != std::string::npos) {
        serial = props.channel.substr(0, colon);
        const auto digits = props.channel.substr(colon + 1);
        if (digits.empty() || digits.size() > 2 ||
            !std::ranges::all_of(digits, [](const char c) {
                return c >= '0' && c <= '9';
            }))
            return {
                nullptr,
                x::errors::Error(
                    CONFIG_ERROR,
                    "gs_usb channel '" + props.channel +
                        "' must be a serial number and a channel index, such as "
                        "0039002C4E55:0"
                )
            };
        index = static_cast<std::uint32_t>(std::stoul(digits));
    }
    auto [found, find_err] = find(api, serial);
    if (find_err) return {nullptr, find_err};
    if (found.handle == nullptr)
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "no gs_usb adapter with serial number " + serial + " is attached"
            )
        };
    auto *handle = found.handle;
    const auto fail = [&](x::errors::Error err) {
        api.ReleaseInterface(handle, INTERFACE);
        api.Close(handle);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{nullptr, err};
    };
    const auto usb_error = [&](const std::string &what, const int rc) {
        return fail(
            {TEMPORARY_HARDWARE_ERROR,
             props.channel + ": " + what + ": " + describe(api, rc)}
        );
    };
    if (const int rc = api.SetAutoDetachKernelDriver(handle, 1);
        rc != LIBUSB_SUCCESS && rc != LIBUSB_ERROR_NOT_SUPPORTED)
        return usb_error("cannot detach the kernel driver", rc);
    if (const int rc = api.ClaimInterface(handle, INTERFACE); rc != LIBUSB_SUCCESS) {
        api.Close(handle);
        const auto what = rc == LIBUSB_ERROR_BUSY ? "the adapter is in use"
                                                  : "cannot claim the adapter";
        return {
            nullptr,
            x::errors::Error(
                TEMPORARY_HARDWARE_ERROR,
                props.channel + ": " + what + ": " + describe(api, rc)
            )
        };
    }
    std::array<std::uint8_t, 12> config_reply{};
    const int n = control_in(api, handle, Request::DEVICE_CONFIG, 0, config_reply);
    if (n < 0) return usb_error("cannot read the adapter configuration", n);
    auto [config, config_err] = decode_device_config(
        std::span(config_reply.data(), static_cast<std::size_t>(n))
    );
    if (config_err) return fail(config_err);
    if (index >= config.channels)
        return fail(
            {CONFIG_ERROR,
             props.channel + ": the adapter has " + std::to_string(config.channels) +
                 " channel(s)"}
        );
    const auto channel = static_cast<std::uint16_t>(index);
    auto [bt, bt_err] = read_bt_const(api, handle, channel, false);
    if (bt_err) return fail(bt_err);
    if (props.fd) {
        if ((bt.features & FEATURE_FD) == 0 ||
            (bt.features & FEATURE_BT_CONST_EXT) == 0)
            return fail(
                {CONFIG_ERROR, props.channel + ": the adapter does not support CAN FD"}
            );
        auto [ext, ext_err] = read_bt_const(api, handle, channel, true);
        if (ext_err) return fail(ext_err);
        bt = ext;
    }
    if (props.listen_only && (bt.features & FEATURE_LISTEN_ONLY) == 0)
        return fail({CONFIG_ERROR, props.channel + ": the adapter cannot listen only"});
    auto reset = encode_mode(MODE_RESET, 0);
    if (const int rc = control_out(api, handle, Request::MODE, channel, reset); rc < 0)
        return usb_error("cannot reset the channel", rc);
    auto [nominal, nominal_err] = compute_timing(
        bt.clock_hz,
        props.bitrate,
        bt.nominal
    );
    if (nominal_err) return fail(nominal_err);
    auto nominal_body = encode_bittiming(nominal);
    if (const int
            rc = control_out(api, handle, Request::BITTIMING, channel, nominal_body);
        rc < 0)
        return usb_error("cannot set the bitrate", rc);
    std::uint32_t flags = 0;
    if (props.fd) {
        auto [data, data_err] = compute_timing(
            bt.clock_hz,
            props.data_bitrate,
            bt.data
        );
        if (data_err) return fail(data_err);
        auto data_body = encode_bittiming(data);
        if (const int rc = control_out(
                api,
                handle,
                Request::DATA_BITTIMING,
                channel,
                data_body
            );
            rc < 0)
            return usb_error("cannot set the data bitrate", rc);
        flags |= FEATURE_FD;
    }
    if (props.listen_only) flags |= FEATURE_LISTEN_ONLY;
    const bool timestamped = (bt.features & FEATURE_HW_TIMESTAMP) != 0;
    if (timestamped) flags |= FEATURE_HW_TIMESTAMP;
    auto start = encode_mode(MODE_START, flags);
    if (const int rc = control_out(api, handle, Request::MODE, channel, start); rc < 0)
        return usb_error("cannot start the channel", rc);
    return {
        std::make_unique<Bus>(
            this->api,
            handle,
            static_cast<std::uint8_t>(index),
            props.channel,
            props.fd,
            props.listen_only,
            timestamped
        ),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    auto [api, err] = ProdAPI::load();
    if (err) return std::make_shared<Unavailable>(err);
    return std::make_shared<Backend>(api);
}
}
