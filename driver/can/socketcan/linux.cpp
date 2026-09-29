// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cerrno>
#include <cstring>
#include <ctime>
#include <filesystem>
#include <fstream>
#include <string>
#include <system_error>

#include <linux/can.h>
#include <linux/can/raw.h>
#include <linux/net_tstamp.h>
#include <net/if.h>
#include <poll.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <unistd.h>

#include "driver/can/socketcan/socketcan.h"

namespace driver::can::socketcan {
namespace {
/// @brief the ARPHRD_CAN hardware type that /sys/class/net/<name>/type holds for a CAN
/// interface.
constexpr int CAN_INTERFACE_TYPE = 280;
/// @brief the flags that ask the kernel for hardware and software receive times.
constexpr int TIMESTAMPING = SOF_TIMESTAMPING_RX_HARDWARE |
                             SOF_TIMESTAMPING_RAW_HARDWARE |
                             SOF_TIMESTAMPING_RX_SOFTWARE | SOF_TIMESTAMPING_SOFTWARE;

/// @brief the payload of an SO_TIMESTAMPING control message: the software time, an
/// unused slot, and the raw hardware time.
struct Timestamps {
    timespec ts[3];
};

/// @returns an error of the given type whose message is context followed by the
/// description of errno.
x::errors::Error os_error(const x::errors::Error &type, const std::string &context) {
    const int code = errno;
    return {type, context + ": " + std::generic_category().message(code)};
}

std::int64_t nanoseconds(const timespec &ts) {
    return static_cast<std::int64_t>(ts.tv_sec) * 1000000000 + ts.tv_nsec;
}

/// @brief sets the time of a received frame from the socket's control messages, and
/// to the host time now when the kernel attached none.
void stamp(msghdr &msg, Frame &frame) {
    frame.time = x::telem::TimeStamp::now();
    frame.clock = Clock::HOST;
    for (auto *cmsg = CMSG_FIRSTHDR(&msg); cmsg != nullptr;
         cmsg = CMSG_NXTHDR(&msg, cmsg)) {
        if (cmsg->cmsg_level != SOL_SOCKET) continue;
        if (cmsg->cmsg_type == SCM_TIMESTAMPING) {
            Timestamps stamps{};
            std::memcpy(&stamps, CMSG_DATA(cmsg), sizeof(stamps));
            if (nanoseconds(stamps.ts[2]) != 0) {
                frame.time = x::telem::TimeStamp(nanoseconds(stamps.ts[2]));
                frame.clock = Clock::HARDWARE;
            } else if (nanoseconds(stamps.ts[0]) != 0)
                frame.time = x::telem::TimeStamp(nanoseconds(stamps.ts[0]));
            return;
        }
        if (cmsg->cmsg_type == SCM_TIMESTAMP) {
            timeval tv{};
            std::memcpy(&tv, CMSG_DATA(cmsg), sizeof(tv));
            frame.time = x::telem::TimeStamp(
                static_cast<std::int64_t>(tv.tv_sec) * 1000000000 + tv.tv_usec * 1000
            );
            return;
        }
    }
}

void decode(const canfd_frame &raw, const bool fd, Frame &frame) {
    frame.extended = (raw.can_id & CAN_EFF_FLAG) != 0;
    frame.fd = fd;
    frame.type = Type::DATA;
    if ((raw.can_id & CAN_ERR_FLAG) != 0)
        frame.type = Type::BUS_ERROR;
    else if (!fd && (raw.can_id & CAN_RTR_FLAG) != 0)
        frame.type = Type::REMOTE;
    if (frame.type == Type::BUS_ERROR)
        frame.id = raw.can_id & CAN_ERR_MASK;
    else
        frame.id = raw.can_id & (frame.extended ? CAN_EFF_MASK : CAN_SFF_MASK);
    frame.bitrate_switched = fd && (raw.flags & CANFD_BRS) != 0;
    frame.error_passive = fd && (raw.flags & CANFD_ESI) != 0;
    const auto max_length = fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    frame.length = std::min<std::uint8_t>(raw.len, max_length);
    std::memcpy(frame.data.data(), raw.data, frame.length);
}

class Bus final : public can::Bus {
    int socket;
    bool closed = false;

public:
    Bus(const int socket, std::string name, const bool fd, const bool listen_only):
        can::Bus(std::move(name), fd, listen_only), socket(socket) {}

    ~Bus() override { this->close(); }

    std::pair<bool, x::errors::Error>
    receive(Frame &frame, const x::telem::TimeSpan timeout) override {
        const auto ns = std::max<std::int64_t>(timeout.nanoseconds(), 0);
        const timespec wait{
            .tv_sec = static_cast<time_t>(ns / 1000000000),
            .tv_nsec = static_cast<long>(ns % 1000000000),
        };
        pollfd pfd{.fd = this->socket, .events = POLLIN, .revents = 0};
        const int ready = ::ppoll(&pfd, 1, &wait, nullptr);
        if (ready < 0 && errno != EINTR)
            return {
                false,
                os_error(
                    TEMPORARY_HARDWARE_ERROR,
                    this->name + ": failed to wait for a frame"
                )
            };
        if (ready <= 0) return {false, x::errors::NIL};
        canfd_frame raw{};
        iovec iov{.iov_base = &raw, .iov_len = sizeof(raw)};
        alignas(
            cmsghdr
        ) char control[CMSG_SPACE(sizeof(Timestamps)) + CMSG_SPACE(sizeof(timeval))];
        msghdr msg{};
        msg.msg_iov = &iov;
        msg.msg_iovlen = 1;
        msg.msg_control = control;
        msg.msg_controllen = sizeof(control);
        const auto size = ::recvmsg(this->socket, &msg, MSG_DONTWAIT);
        if (size < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR)
                return {false, x::errors::NIL};
            return {
                false,
                os_error(
                    TEMPORARY_HARDWARE_ERROR,
                    this->name + ": failed to receive a frame"
                )
            };
        }
        if (size != CAN_MTU && size != CANFD_MTU)
            return {
                false,
                {TEMPORARY_HARDWARE_ERROR,
                 this->name + ": received " + std::to_string(size) +
                     " bytes, which is not a CAN frame"}
            };
        decode(raw, size == CANFD_MTU, frame);
        stamp(msg, frame);
        return {true, x::errors::NIL};
    }

private:
    x::errors::Error transmit(const Frame &frame) override {
        // A classic frame shares the first CAN_MTU bytes of a CAN FD frame's layout.
        canfd_frame raw{};
        raw.can_id = frame.id;
        if (frame.extended) raw.can_id |= CAN_EFF_FLAG;
        if (frame.type == Type::REMOTE) raw.can_id |= CAN_RTR_FLAG;
        raw.len = frame.length;
        if (frame.fd) {
#ifdef CANFD_FDF
            raw.flags |= CANFD_FDF;
#endif
            if (frame.bitrate_switched) raw.flags |= CANFD_BRS;
        }
        std::memcpy(raw.data, frame.data.data(), frame.length);
        const std::size_t size = frame.fd ? CANFD_MTU : CAN_MTU;
        if (::write(this->socket, &raw, size) == static_cast<ssize_t>(size))
            return x::errors::NIL;
        if (errno == ENOBUFS || errno == EAGAIN || errno == EWOULDBLOCK)
            return {TEMPORARY_HARDWARE_ERROR, this->name + ": transmit queue is full"};
        return os_error(
            TEMPORARY_HARDWARE_ERROR,
            this->name + ": failed to send a frame"
        );
    }

public:
    x::errors::Error close() override {
        if (this->closed) return x::errors::NIL;
        this->closed = true;
        if (::close(this->socket) != 0)
            return os_error(
                TEMPORARY_HARDWARE_ERROR,
                this->name + ": failed to close the socket"
            );
        return x::errors::NIL;
    }
};

/// @returns the first line of a file under /sys/class/net/<name>, or an empty string
/// when the file cannot be read.
std::string attribute(const std::filesystem::path &interface, const std::string &file) {
    std::ifstream in(interface / file);
    std::string line;
    std::getline(in, line);
    return line;
}

class Backend final : public can::Backend {
public:
    std::pair<std::vector<Channel>, x::errors::Error> scan() override {
        std::vector<Channel> channels;
        std::error_code ec;
        for (const auto &entry:
             std::filesystem::directory_iterator("/sys/class/net", ec)) {
            if (attribute(entry.path(), "type") != std::to_string(CAN_INTERFACE_TYPE))
                continue;
            const auto name = entry.path().filename().string();
            std::string description = name;
            if (attribute(entry.path(), "mtu") == std::to_string(CANFD_MTU))
                description += ", CAN FD";
            if (attribute(entry.path(), "operstate") == "down") description += ", down";
            channels.push_back({
                .backend = synnax::can::BACKEND_SOCKETCAN,
                .name = name,
                .description = description,
            });
        }
        if (ec)
            return {
                {},
                {TEMPORARY_HARDWARE_ERROR,
                 "failed to list the network interfaces: " + ec.message()}
            };
        std::ranges::sort(channels, {}, &Channel::name);
        return {channels, x::errors::NIL};
    }

    std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        const auto &name = props.channel;
        if (name.empty() || name.size() >= IFNAMSIZ)
            return {
                nullptr,
                x::errors::Error(
                    CONFIG_ERROR,
                    "'" + name + "' is not an interface name"
                )
            };
        const int sock = ::socket(
            PF_CAN,
            SOCK_RAW | SOCK_NONBLOCK | SOCK_CLOEXEC,
            CAN_RAW
        );
        if (sock < 0)
            return {
                nullptr,
                os_error(CRITICAL_HARDWARE_ERROR, "failed to open a SocketCAN socket")
            };
        auto bus = std::make_unique<Bus>(sock, name, props.fd, props.listen_only);
        ifreq ifr{};
        std::memcpy(ifr.ifr_name, name.data(), name.size());
        if (::ioctl(sock, SIOCGIFINDEX, &ifr) < 0)
            return {
                nullptr,
                x::errors::Error(CONFIG_ERROR, "no CAN interface is named " + name)
            };
        const int index = ifr.ifr_ifindex;
        if (::ioctl(sock, SIOCGIFFLAGS, &ifr) < 0)
            return {
                nullptr,
                os_error(
                    TEMPORARY_HARDWARE_ERROR,
                    "failed to read the state of " + name
                )
            };
        if ((ifr.ifr_flags & IFF_UP) == 0)
            return {
                nullptr,
                x::errors::Error(
                    TEMPORARY_HARDWARE_ERROR,
                    "the interface " + name +
                        " is down. Bring it up with: ip link set " + name +
                        " up type can bitrate " + std::to_string(props.bitrate)
                )
            };
        if (props.fd) {
            if (::ioctl(sock, SIOCGIFMTU, &ifr) < 0 || ifr.ifr_mtu != CANFD_MTU)
                return {
                    nullptr,
                    x::errors::Error(
                        CONFIG_ERROR,
                        "the interface " + name +
                            " does not run CAN FD. Bring it up with: ip link set " +
                            name + " up type can bitrate " +
                            std::to_string(props.bitrate) + " dbitrate " +
                            std::to_string(props.data_bitrate) + " fd on"
                    )
                };
            const int enabled = 1;
            if (::setsockopt(
                    sock,
                    SOL_CAN_RAW,
                    CAN_RAW_FD_FRAMES,
                    &enabled,
                    sizeof(enabled)
                ) < 0)
                return {
                    nullptr,
                    os_error(
                        CONFIG_ERROR,
                        "the kernel cannot send CAN FD frames on " + name
                    )
                };
        }
        const can_err_mask_t errors = CAN_ERR_MASK;
        if (::setsockopt(
                sock,
                SOL_CAN_RAW,
                CAN_RAW_ERR_FILTER,
                &errors,
                sizeof(errors)
            ) < 0)
            return {
                nullptr,
                os_error(
                    CRITICAL_HARDWARE_ERROR,
                    "failed to subscribe to the bus errors of " + name
                )
            };
        // Kernels without SO_TIMESTAMPING still offer SO_TIMESTAMP. A kernel with
        // neither leaves frames at the host time of the read.
        if (::setsockopt(
                sock,
                SOL_SOCKET,
                SO_TIMESTAMPING,
                &TIMESTAMPING,
                sizeof(TIMESTAMPING)
            ) < 0) {
            const int enabled = 1;
            ::setsockopt(sock, SOL_SOCKET, SO_TIMESTAMP, &enabled, sizeof(enabled));
        }
        sockaddr_can addr{};
        addr.can_family = AF_CAN;
        addr.can_ifindex = index;
        if (::bind(sock, reinterpret_cast<sockaddr *>(&addr), sizeof(addr)) < 0)
            return {
                nullptr,
                os_error(TEMPORARY_HARDWARE_ERROR, "failed to bind to " + name)
            };
        return {std::move(bus), x::errors::NIL};
    }
};
}

std::shared_ptr<can::Backend> load() {
    return std::make_shared<Backend>();
}
}
