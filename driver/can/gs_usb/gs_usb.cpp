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
#include <condition_variable>
#include <deque>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "driver/can/gs_usb/gs_usb.h"
#include "driver/can/gs_usb/prod.h"

namespace driver::can::gs_usb {
namespace {
const auto USB_TIMEOUT_MS = static_cast<unsigned int>(
    (USB_TIMEOUT / x::telem::MILLISECOND).nanoseconds()
);
const auto READ_SLICE_MS = static_cast<unsigned int>(
    (READ_SLICE / x::telem::MILLISECOND).nanoseconds()
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

/// @brief where an adapter sits on the host's USB buses.
struct Location {
    std::uint8_t bus = 0;
    std::uint8_t address = 0;

    bool operator==(const Location &) const = default;
};

Location locate(API &api, libusb_device *dev) {
    return {.bus = api.GetBusNumber(dev), .address = api.GetDeviceAddress(dev)};
}

/// @brief what the backend reads from an adapter to name its channels.
struct Identity {
    std::string serial;
    std::string product;
    std::uint32_t channels = 1;
};

/// @returns the identity of an adapter on an open handle.
Identity
identify(API &api, libusb_device_handle *handle, const libusb_device_descriptor &desc) {
    Identity id{
        .serial = read_string(api, handle, desc.iSerialNumber),
        .product = read_string(api, handle, desc.iProduct),
    };
    if (id.product.empty()) id.product = "gs_usb adapter";
    std::array<std::uint8_t, 12> reply{};
    const int n = control_in(api, handle, Request::DEVICE_CONFIG, 0, reply);
    if (n > 0) {
        const auto [config, err] = decode_device_config(
            std::span(reply.data(), static_cast<std::size_t>(n))
        );
        if (!err) id.channels = config.channels;
    }
    return id;
}

/// @brief an open adapter found by serial number.
struct Found {
    libusb_device_handle *handle = nullptr;
    std::string product;
    Location location;
};

/// @brief opens the gs_usb adapter with a serial number.
/// @returns a null handle when no attached adapter has the serial number, and the error
/// of an adapter that would not open when no other adapter matched.
std::pair<Found, x::errors::Error> find(API &api, const std::string &serial) {
    libusb_device **list = nullptr;
    const auto count = api.GetDeviceList(&list);
    if (count < 0)
        return {
            {},
            translate(api, static_cast<int>(count), "libusb cannot list devices")
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
            err = translate(api, rc, "cannot open a gs_usb adapter");
            continue;
        }
        if (read_string(api, handle, desc.iSerialNumber) != serial) {
            api.Close(handle);
            continue;
        }
        found = {
            .handle = handle,
            .product = read_string(api, handle, desc.iProduct),
            .location = locate(api, list[i]),
        };
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
        return {{}, translate(api, n, "gs_usb adapter did not report its bit timing")};
    return decode_bt_const(
        std::span(reply.data(), static_cast<std::size_t>(n)),
        extended
    );
}
}

x::errors::Error translate(API &api, const int rc, const std::string &context) {
    const auto message = context + ": " + describe(api, rc);
    switch (rc) {
        // libusb reports an adapter that another program holds as BUSY on Linux and
        // ACCESS on Windows, and an unplug mid-transfer as IO or NO_DEVICE.
        case LIBUSB_ERROR_NO_DEVICE:
        case LIBUSB_ERROR_NOT_FOUND:
        case LIBUSB_ERROR_BUSY:
        case LIBUSB_ERROR_ACCESS:
        case LIBUSB_ERROR_IO:
        case LIBUSB_ERROR_TIMEOUT:
        case LIBUSB_ERROR_INTERRUPTED:
            return {UNREACHABLE_ERROR, message};
        case LIBUSB_ERROR_NOT_SUPPORTED:
            return {CONFIG_ERROR, message};
        default:
            return {CRITICAL_HARDWARE_ERROR, message};
    }
}

/// @brief an open gs_usb adapter with its USB interface claimed, shared by the buses of
/// its channels. The adapter sends the frames of every channel on one endpoint, which a
/// thread reads from construction to destruction. The thread drops echoes and queues
/// each received frame for the bus of its channel. Releases the interface and closes
/// the handle on destruction.
class Adapter {
public:
    /// @brief a frame the adapter sent to the host.
    struct Received {
        std::vector<std::uint8_t> bytes;
        /// @brief the host time at which the frame arrived.
        x::telem::TimeStamp time;
        /// @brief the steady host time at which the frame arrived.
        x::telem::TimeStamp steady;
        /// @brief the frames of the channel dropped just before this one.
        std::size_t dropped = 0;
    };

    /// @brief the libusb calls to make.
    const std::shared_ptr<API> api;
    /// @brief the open handle, with the interface claimed.
    libusb_device_handle *const handle;
    /// @brief the adapter's USB serial number and product name, and its number of CAN
    /// channels.
    const Identity identity;
    /// @brief where the adapter sits on the host's USB buses.
    const Location location;

    Adapter(
        std::shared_ptr<API> api,
        libusb_device_handle *handle,
        Identity identity,
        const Location location
    ):
        api(std::move(api)),
        handle(handle),
        identity(std::move(identity)),
        location(location),
        thread([this] { this->run(); }) {}

    Adapter(const Adapter &) = delete;
    Adapter &operator=(const Adapter &) = delete;

    ~Adapter() {
        this->running = false;
        this->thread.join();
        this->api->ReleaseInterface(this->handle, INTERFACE);
        this->api->Close(this->handle);
    }

    /// @brief starts queueing the frames of a channel.
    /// @returns false when a bus already holds the channel.
    bool attach(const std::uint8_t channel) {
        std::lock_guard lock(this->mu);
        return this->queues.try_emplace(channel).second;
    }

    /// @brief stops queueing the frames of a channel and drops the queued ones.
    void detach(const std::uint8_t channel) {
        std::lock_guard lock(this->mu);
        this->queues.erase(channel);
    }

    /// @returns true once a read of the adapter failed.
    bool failed() {
        std::lock_guard lock(this->mu);
        return !this->failure.ok();
    }

    /// @brief waits for a frame of an attached channel.
    /// @returns nullopt once the deadline passes, and the error of the failed read once
    /// the channel's queued frames run out.
    std::pair<std::optional<Received>, x::errors::Error> receive(
        const std::uint8_t channel,
        const std::chrono::steady_clock::time_point deadline
    ) {
        std::unique_lock lock(this->mu);
        auto &queue = this->queues.at(channel);
        this->cv.wait_until(lock, deadline, [&] {
            return !queue.empty() || !this->failure.ok();
        });
        if (queue.empty()) return {std::nullopt, this->failure};
        auto received = std::move(queue.front());
        queue.pop_front();
        return {std::move(received), x::errors::NIL};
    }

private:
    std::mutex mu;
    std::condition_variable cv;
    /// @brief the frames the adapter received, by channel.
    std::unordered_map<std::uint8_t, std::deque<Received>> queues;
    x::errors::Error failure = x::errors::NIL;
    std::atomic<bool> running = true;
    std::thread thread;

    void run() {
        std::vector<std::uint8_t> bytes(MAX_FRAME_SIZE);
        while (this->running) {
            int transferred = 0;
            const int rc = this->api->BulkTransfer(
                this->handle,
                ENDPOINT_IN,
                bytes.data(),
                static_cast<int>(bytes.size()),
                &transferred,
                READ_SLICE_MS
            );
            if (rc == LIBUSB_ERROR_TIMEOUT) continue;
            x::errors::Error err = x::errors::NIL;
            if (rc != LIBUSB_SUCCESS)
                err = translate(
                    *this->api,
                    rc,
                    "gs_usb adapter " + this->identity.serial
                );
            {
                std::lock_guard lock(this->mu);
                if (!err)
                    err = this->route({
                        .bytes = {bytes.begin(), bytes.begin() + transferred},
                        .time = x::telem::TimeStamp::now(),
                        .steady = steady_now(),
                    });
                if (err) this->failure = err;
            }
            this->cv.notify_all();
            if (err) return;
        }
    }

    /// @brief queues a frame for the bus of its channel, and drops it when no bus holds
    /// the channel or the frame echoes a sent one. Requires mu.
    /// @returns CRITICAL_HARDWARE_ERROR when the frame is shorter than a header.
    x::errors::Error route(Received received) {
        const auto [channel, err] = decode_channel(received.bytes);
        if (err) return err;
        if (!channel.has_value()) return x::errors::NIL;
        const auto it = this->queues.find(*channel);
        if (it == this->queues.end()) return x::errors::NIL;
        auto &queue = it->second;
        std::size_t dropped = 0;
        if (queue.size() >= MAX_QUEUED) {
            dropped = 1 + queue.front().dropped;
            queue.pop_front();
        }
        if (queue.empty())
            received.dropped = dropped;
        else
            queue.front().dropped += dropped;
        queue.push_back(std::move(received));
        return x::errors::NIL;
    }
};

Bus::Bus(
    std::shared_ptr<Adapter> adapter,
    const std::uint8_t channel,
    std::string name,
    const bool fd,
    const bool listen_only,
    const bool timestamped
):
    can::Bus(std::move(name), fd, listen_only),
    adapter(std::move(adapter)),
    channel(channel),
    timestamped(timestamped) {}

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
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    auto [received, err] = this->adapter->receive(this->channel, deadline);
    if (!received.has_value()) return {false, err};
    auto [hf, decode_err] = decode_frame(received->bytes, this->timestamped);
    if (decode_err) return {false, decode_err};
    auto next = hf.frame;
    if (hf.timestamp_us.has_value()) {
        const auto us = this->counter.extend(*hf.timestamp_us, received->steady);
        next.time = x::telem::TimeStamp(static_cast<std::int64_t>(us) * 1000);
        next.clock = Clock::HARDWARE;
    } else {
        next.time = received->time;
        next.clock = Clock::HOST;
    }
    if (received->dropped == 0 && !hf.overflowed) {
        frame = next;
        return {true, x::errors::NIL};
    }
    this->pending = next;
    frame = bus_error::make(
        bus_error::CONTROLLER,
        bus_error::CONTROLLER_RX_OVERFLOW,
        received->time
    );
    return {true, x::errors::NIL};
}

x::errors::Error Bus::transmit(const Frame &frame) {
    const auto echo = this->next_echo.fetch_add(1) % ECHO_IDS;
    auto bytes = encode_frame(frame, echo, this->channel);
    int transferred = 0;
    const int rc = this->adapter->api->BulkTransfer(
        this->adapter->handle,
        ENDPOINT_OUT,
        bytes.data(),
        static_cast<int>(bytes.size()),
        &transferred,
        USB_TIMEOUT_MS
    );
    if (rc == LIBUSB_SUCCESS) return x::errors::NIL;
    // candleLight refuses OUT transfers while every frame object waits for the bus or
    // the host (src/usbd_gs_can.c, USBD_GS_CAN_DataOut).
    if (rc == LIBUSB_ERROR_TIMEOUT)
        return {
            TEMPORARY_HARDWARE_ERROR,
            "gs_usb adapter " + this->name + ": transmit queue is full"
        };
    return translate(*this->adapter->api, rc, "gs_usb adapter " + this->name);
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    auto mode = encode_mode(MODE_RESET, 0);
    const int rc = control_out(
        *this->adapter->api,
        this->adapter->handle,
        Request::MODE,
        this->channel,
        mode
    );
    x::errors::Error err = x::errors::NIL;
    if (rc < 0 && rc != LIBUSB_ERROR_NO_DEVICE)
        err = translate(*this->adapter->api, rc, "gs_usb adapter " + this->name);
    this->adapter->detach(this->channel);
    this->adapter.reset();
    return err;
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    auto &api = *this->api;
    std::lock_guard lock(this->mu);
    std::vector<std::shared_ptr<Adapter>> open;
    for (const auto &[_, weak]: this->adapters)
        if (auto adapter = weak.lock()) open.push_back(std::move(adapter));
    libusb_device **list = nullptr;
    const auto count = api.GetDeviceList(&list);
    if (count < 0)
        return {
            {},
            translate(api, static_cast<int>(count), "libusb cannot list devices")
        };
    std::vector<Channel> found;
    for (std::ptrdiff_t i = 0; i < count; i++) {
        libusb_device_descriptor desc{};
        if (api.GetDeviceDescriptor(list[i], &desc) != LIBUSB_SUCCESS ||
            !is_gs_usb(desc))
            continue;
        const auto location = locate(api, list[i]);
        Identity id;
        const auto held = std::ranges::find_if(open, [&](const auto &adapter) {
            return adapter->location == location;
        });
        if (held != open.end())
            id = (*held)->identity;
        else {
            libusb_device_handle *handle = nullptr;
            if (const int rc = api.Open(list[i], &handle); rc != LIBUSB_SUCCESS) {
                LOG(WARNING) << "[can] cannot open the gs_usb adapter at USB bus "
                             << static_cast<int>(location.bus) << " address "
                             << static_cast<int>(location.address) << ": "
                             << describe(api, rc);
                continue;
            }
            id = identify(api, handle, desc);
            api.Close(handle);
        }
        if (id.serial.empty()) {
            LOG(WARNING) << "[can] skipping a gs_usb adapter with no serial number";
            continue;
        }
        for (std::uint32_t c = 0; c < id.channels; c++)
            found.push_back({
                .backend = synnax::can::BACKEND_GS_USB,
                .name = id.serial + ":" + std::to_string(c),
                .description = id.product + " channel " + std::to_string(c),
            });
    }
    api.FreeDeviceList(list, 1);
    return {found, x::errors::NIL};
}

std::pair<std::shared_ptr<Adapter>, x::errors::Error>
Backend::adapter(const std::string &serial, const std::string &channel) {
    if (auto open = this->adapters[serial].lock(); open != nullptr && !open->failed())
        return {open, x::errors::NIL};
    auto &api = *this->api;
    auto [found, find_err] = find(api, serial);
    if (find_err) return {nullptr, find_err};
    if (found.handle == nullptr)
        return {
            nullptr,
            x::errors::Error(
                UNREACHABLE_ERROR,
                "no gs_usb adapter with serial number " + serial + " is attached"
            )
        };
    auto *handle = found.handle;
    if (const int rc = api.ClaimInterface(handle, INTERFACE); rc != LIBUSB_SUCCESS) {
        api.Close(handle);
        const auto what = rc == LIBUSB_ERROR_BUSY ? "the adapter is in use"
                                                  : "cannot claim the adapter";
        return {nullptr, translate(api, rc, channel + ": " + what)};
    }
    const auto release = [&](x::errors::Error err) {
        api.ReleaseInterface(handle, INTERFACE);
        api.Close(handle);
        return std::pair<std::shared_ptr<Adapter>, x::errors::Error>{nullptr, err};
    };
    std::array<std::uint8_t, 12> reply{};
    const int n = control_in(api, handle, Request::DEVICE_CONFIG, 0, reply);
    if (n < 0)
        return release(
            translate(api, n, channel + ": cannot read the adapter configuration")
        );
    auto [config, config_err] = decode_device_config(
        std::span(reply.data(), static_cast<std::size_t>(n))
    );
    if (config_err) return release(config_err);
    auto adapter = std::make_shared<Adapter>(
        this->api,
        handle,
        Identity{
            .serial = serial,
            .product = found.product.empty() ? "gs_usb adapter" : found.product,
            .channels = config.channels,
        },
        found.location
    );
    this->adapters[serial] = adapter;
    return {adapter, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
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
    std::lock_guard lock(this->mu);
    auto [adapter, adapter_err] = this->adapter(serial, props.channel);
    if (adapter_err) return {nullptr, adapter_err};
    auto &api = *this->api;
    auto *handle = adapter->handle;
    if (index >= adapter->identity.channels)
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                props.channel + ": the adapter has " +
                    std::to_string(adapter->identity.channels) + " channel(s)"
            )
        };
    const auto channel = static_cast<std::uint8_t>(index);
    if (!adapter->attach(channel))
        return {
            nullptr,
            x::errors::Error(
                UNREACHABLE_ERROR,
                props.channel + ": the channel is in use"
            )
        };
    const auto fail = [&](x::errors::Error err) {
        adapter->detach(channel);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{nullptr, err};
    };
    const auto usb_error = [&](const std::string &what, const int rc) {
        return fail(translate(api, rc, props.channel + ": " + what));
    };
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
            std::move(adapter),
            channel,
            props.channel,
            props.fd,
            props.listen_only,
            timestamped
        ),
        x::errors::NIL
    };
}

std::shared_ptr<can::Backend> load() {
    return or_unavailable<Backend>(ProdAPI::load());
}
}
