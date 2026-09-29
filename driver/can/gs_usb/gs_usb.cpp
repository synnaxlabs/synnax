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
#include <unordered_map>
#include <vector>

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

/// @brief an open gs_usb adapter with its USB interface claimed, shared by the buses of
/// its channels. The adapter sends the frames of every channel on one endpoint, so one
/// receiving bus at a time reads it and queues each frame for the bus of its channel.
/// Releases the interface and closes the handle on destruction.
class Adapter {
public:
    /// @brief the libusb calls to make.
    const std::shared_ptr<API> api;
    /// @brief the open handle, with the interface claimed.
    libusb_device_handle *const handle;
    /// @brief the adapter's USB serial number.
    const std::string serial;
    /// @brief the number of CAN channels the adapter has.
    const std::uint32_t channels;

    Adapter(
        std::shared_ptr<API> api,
        libusb_device_handle *handle,
        std::string serial,
        const std::uint32_t channels
    ):
        api(std::move(api)),
        handle(handle),
        serial(std::move(serial)),
        channels(channels) {}

    Adapter(const Adapter &) = delete;
    Adapter &operator=(const Adapter &) = delete;

    ~Adapter() {
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

    /// @brief waits for a host frame of an attached channel.
    /// @returns false once the deadline passes, and TEMPORARY_HARDWARE_ERROR when the
    /// adapter is unplugged or the read fails.
    std::pair<bool, x::errors::Error> receive(
        const std::uint8_t channel,
        std::vector<std::uint8_t> &bytes,
        const std::chrono::steady_clock::time_point deadline
    ) {
        std::unique_lock lock(this->mu);
        while (true) {
            auto &queue = this->queues.at(channel);
            if (!queue.frames.empty()) {
                bytes = std::move(queue.frames.front());
                queue.frames.pop_front();
                return {true, x::errors::NIL};
            }
            if (std::chrono::steady_clock::now() >= deadline)
                return {false, x::errors::NIL};
            if (this->reading) {
                this->cv.wait_until(lock, deadline);
                continue;
            }
            this->reading = true;
            lock.unlock();
            auto [read, err] = this->read(deadline);
            lock.lock();
            this->reading = false;
            this->cv.notify_all();
            if (err) return {false, err};
            if (read.empty()) continue;
            if (auto route_err = this->route(std::move(read)))
                return {false, route_err};
        }
    }

private:
    /// @brief the frames the adapter received for one channel.
    struct Queue {
        std::deque<std::vector<std::uint8_t>> frames;
        /// @brief true from the first frame the queue drops until it empties.
        bool overflowed = false;
    };

    std::mutex mu;
    std::condition_variable cv;
    /// @brief true while a bus reads the adapter's endpoint.
    bool reading = false;
    std::unordered_map<std::uint8_t, Queue> queues;

    /// @brief reads one host frame from the adapter.
    /// @returns no bytes when the deadline passes first.
    std::pair<std::vector<std::uint8_t>, x::errors::Error>
    read(const std::chrono::steady_clock::time_point deadline) {
        const auto remaining = std::chrono::ceil<std::chrono::milliseconds>(
            deadline - std::chrono::steady_clock::now()
        );
        const auto ms = static_cast<unsigned int>(
            std::max<std::int64_t>(remaining.count(), 1)
        );
        std::vector<std::uint8_t> bytes(MAX_FRAME_SIZE);
        int transferred = 0;
        const int rc = this->api->BulkTransfer(
            this->handle,
            ENDPOINT_IN,
            bytes.data(),
            static_cast<int>(bytes.size()),
            &transferred,
            ms
        );
        bytes.resize(static_cast<std::size_t>(transferred));
        if (rc == LIBUSB_SUCCESS || rc == LIBUSB_ERROR_TIMEOUT)
            return {std::move(bytes), x::errors::NIL};
        if (rc == LIBUSB_ERROR_NO_DEVICE)
            return {
                {},
                {TEMPORARY_HARDWARE_ERROR,
                 "gs_usb adapter " + this->serial + " unplugged"}
            };
        return {
            {},
            {TEMPORARY_HARDWARE_ERROR,
             "gs_usb adapter " + this->serial + ": " + describe(*this->api, rc)}
        };
    }

    /// @brief queues a host frame for the bus of its channel, and drops it when no bus
    /// holds the channel or the frame echoes a sent one. Requires mu.
    /// @returns CRITICAL_HARDWARE_ERROR when the frame is shorter than a header.
    x::errors::Error route(std::vector<std::uint8_t> bytes) {
        const auto [channel, err] = decode_channel(bytes);
        if (err) return err;
        if (!channel.has_value()) return x::errors::NIL;
        const auto it = this->queues.find(*channel);
        if (it == this->queues.end()) return x::errors::NIL;
        auto &queue = it->second;
        if (queue.frames.empty()) queue.overflowed = false;
        if (queue.frames.size() >= MAX_QUEUED) {
            queue.frames.pop_front();
            if (!queue.overflowed)
                LOG(WARNING) << "[can] gs_usb adapter " << this->serial << " channel "
                             << static_cast<int>(*channel)
                             << " is not being read, dropping its oldest frames";
            queue.overflowed = true;
        }
        queue.frames.push_back(std::move(bytes));
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
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    std::vector<std::uint8_t> bytes;
    auto [got, err] = this->adapter->receive(this->channel, bytes, deadline);
    if (!got) return {false, err};
    auto [hf, decode_err] = decode_frame(bytes, this->timestamped);
    if (decode_err) return {false, decode_err};
    if (hf.overflowed)
        LOG(WARNING) << "[can] gs_usb adapter " << this->name
                     << " overran and frames were lost";
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
    if (rc == LIBUSB_ERROR_TIMEOUT)
        return {
            TEMPORARY_HARDWARE_ERROR,
            "gs_usb adapter " + this->name + ": transmit queue is full"
        };
    return {
        TEMPORARY_HARDWARE_ERROR,
        "gs_usb adapter " + this->name + ": " + describe(*this->adapter->api, rc)
    };
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
        err = {
            CRITICAL_HARDWARE_ERROR,
            "gs_usb adapter " + this->name + ": " + describe(*this->adapter->api, rc)
        };
    this->adapter->detach(this->channel);
    this->adapter.reset();
    return err;
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

std::pair<std::shared_ptr<Adapter>, x::errors::Error>
Backend::adapter(const std::string &serial, const std::string &channel) {
    if (auto open = this->adapters[serial].lock()) return {open, x::errors::NIL};
    auto &api = *this->api;
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
    if (const int rc = api.ClaimInterface(handle, INTERFACE); rc != LIBUSB_SUCCESS) {
        api.Close(handle);
        const auto what = rc == LIBUSB_ERROR_BUSY ? "the adapter is in use"
                                                  : "cannot claim the adapter";
        return {
            nullptr,
            x::errors::Error(
                TEMPORARY_HARDWARE_ERROR,
                channel + ": " + what + ": " + describe(api, rc)
            )
        };
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
            {TEMPORARY_HARDWARE_ERROR,
             channel + ": cannot read the adapter configuration: " + describe(api, n)}
        );
    auto [config, config_err] = decode_device_config(
        std::span(reply.data(), static_cast<std::size_t>(n))
    );
    if (config_err) return release(config_err);
    auto
        adapter = std::make_shared<Adapter>(this->api, handle, serial, config.channels);
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
    if (index >= adapter->channels)
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                props.channel + ": the adapter has " +
                    std::to_string(adapter->channels) + " channel(s)"
            )
        };
    const auto channel = static_cast<std::uint8_t>(index);
    if (!adapter->attach(channel))
        return {
            nullptr,
            x::errors::Error(
                TEMPORARY_HARDWARE_ERROR,
                props.channel + ": the channel is in use"
            )
        };
    const auto fail = [&](x::errors::Error err) {
        adapter->detach(channel);
        return std::pair<std::unique_ptr<can::Bus>, x::errors::Error>{nullptr, err};
    };
    const auto usb_error = [&](const std::string &what, const int rc) {
        return fail(
            {TEMPORARY_HARDWARE_ERROR,
             props.channel + ": " + what + ": " + describe(api, rc)}
        );
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
