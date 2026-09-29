// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "driver/can/slcan/slcan.h"

namespace driver::can::slcan {
namespace {
/// @brief the longest a queued frame waits for the thread that owns the port.
const x::telem::TimeSpan READ_SLICE = 5 * x::telem::MILLISECOND;
const x::telem::TimeSpan WRITE_TIMEOUT = 100 * x::telem::MILLISECOND;

std::span<const std::uint8_t> bytes(const std::string_view data) {
    return {reinterpret_cast<const std::uint8_t *>(data.data()), data.size()};
}

/// @returns the command without its carriage return, for messages.
std::string printable(const std::string_view command) {
    return std::string(command.substr(0, command.size() - 1));
}
}

Bus::Bus(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props):
    port(std::move(port)),
    name(props.channel),
    fd(props.fd),
    listen_only(props.listen_only) {}

Bus::~Bus() {
    this->close();
}

std::pair<std::optional<Reply>, x::errors::Error>
Bus::read(const x::telem::TimeSpan timeout) {
    auto [chunk, err] = this->port->read(timeout);
    if (err) return {std::nullopt, err};
    std::optional<Reply> result;
    std::vector<Frame> frames;
    for (const auto byte: chunk.data) {
        Frame frame;
        const auto reply = this->decoder.push(static_cast<char>(byte), frame);
        if (!reply.has_value()) continue;
        switch (*reply) {
            case Reply::FRAME:
                frame.time = chunk.time;
                frame.clock = Clock::HOST;
                frames.push_back(frame);
                break;
            case Reply::MALFORMED:
                LOG(WARNING) << this->name << ": the adapter sent a malformed line";
                break;
            case Reply::ACCEPTED:
                if (!result.has_value()) result = Reply::ACCEPTED;
                break;
            case Reply::REJECTED:
                result = Reply::REJECTED;
                break;
            case Reply::OTHER:
                break;
        }
    }
    if (!frames.empty()) {
        {
            std::lock_guard lock(this->mu);
            this->inbox.insert(this->inbox.end(), frames.begin(), frames.end());
        }
        this->cv.notify_all();
    }
    return {result, x::errors::NIL};
}

std::pair<Reply, x::errors::Error> Bus::command(const std::string_view command) {
    if (auto err = this->port->write(bytes(command), WRITE_TIMEOUT))
        return {Reply::REJECTED, err};
    const auto deadline = x::telem::TimeStamp::now() + REPLY_TIMEOUT;
    for (auto now = x::telem::TimeStamp::now(); now < deadline;
         now = x::telem::TimeStamp::now()) {
        auto [reply, err] = this->read(deadline - now);
        if (err) return {Reply::REJECTED, err};
        if (reply.has_value()) return {*reply, x::errors::NIL};
    }
    return {
        Reply::REJECTED,
        {TEMPORARY_HARDWARE_ERROR,
         "the device at " + this->name + " did not answer the slcan command '" +
             printable(command) + "'. Check that it is an slcan adapter"}
    };
}

std::pair<std::unique_ptr<Bus>, x::errors::Error>
Bus::open(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props) {
    auto [bitrate, bitrate_err] = bitrate_command(props.bitrate);
    if (bitrate_err) return {nullptr, bitrate_err};
    std::string data_bitrate;
    if (props.fd) {
        auto [command, err] = data_bitrate_command(props.data_bitrate);
        if (err) return {nullptr, err};
        data_bitrate = command;
    }
    std::unique_ptr<Bus> bus(new Bus(std::move(port), props));
    // An adapter that is already closed rejects C, and C also clears a partial command
    // that an earlier session left in the adapter.
    if (auto [_, err] = bus->command(CLOSE_COMMAND); err) return {nullptr, err};
    bus->inbox.clear();
    for (const auto &command:
         {bitrate, data_bitrate, open_command(props.listen_only)}) {
        if (command.empty()) continue;
        auto [reply, err] = bus->command(command);
        if (err) return {nullptr, err};
        if (reply == Reply::REJECTED)
            return {
                nullptr,
                x::errors::Error(
                    CONFIG_ERROR,
                    "the slcan adapter at " + props.channel +
                        " rejected the command '" + printable(command) + "'"
                )
            };
    }
    bus->running = true;
    bus->thread = std::thread([raw = bus.get()] { raw->run(); });
    return {std::move(bus), x::errors::NIL};
}

x::errors::Error Bus::exchange() {
    std::vector<std::string> lines;
    {
        std::lock_guard lock(this->mu);
        lines.swap(this->outbox);
    }
    for (const auto &line: lines)
        if (auto err = this->port->write(bytes(line), WRITE_TIMEOUT)) return err;
    auto [reply, err] = this->read(READ_SLICE);
    if (err) return err;
    if (reply == Reply::REJECTED)
        LOG(WARNING) << this->name << ": the adapter rejected a frame";
    return x::errors::NIL;
}

void Bus::run() {
    while (this->running) {
        auto err = this->exchange();
        if (!err) continue;
        {
            std::lock_guard lock(this->mu);
            this->failure = err;
        }
        this->cv.notify_all();
        return;
    }
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    std::unique_lock lock(this->mu);
    if (!this->cv.wait_for(lock, timeout.chrono(), [this] {
            return !this->inbox.empty() || this->failure;
        }))
        return {false, x::errors::NIL};
    if (this->inbox.empty()) return {false, this->failure};
    frame = this->inbox.front();
    this->inbox.pop_front();
    return {true, x::errors::NIL};
}

x::errors::Error Bus::send(const Frame &frame) {
    if (this->listen_only)
        return {LISTEN_ONLY_ERROR, "channel " + this->name + " is listen only"};
    if (auto err = validate(frame, this->fd)) return err;
    std::lock_guard lock(this->mu);
    if (this->failure) return this->failure;
    if (this->outbox.size() >= MAX_QUEUED)
        return {TEMPORARY_HARDWARE_ERROR, this->name + ": transmit queue is full"};
    this->outbox.push_back(encode(frame));
    return x::errors::NIL;
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    this->running = false;
    if (this->thread.joinable()) this->thread.join();
    auto lines = std::move(this->outbox);
    lines.emplace_back(CLOSE_COMMAND);
    x::errors::Error err = this->failure;
    for (const auto &line: lines) {
        if (err) break;
        err = this->port->write(bytes(line), WRITE_TIMEOUT);
    }
    this->port->close(WRITE_TIMEOUT);
    return err;
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    return {{}, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    synnax::serial::Properties serial;
    serial.port = props.channel;
    serial.baud_rate = BAUD_RATE;
    auto [port, err] = serial::Port::open(serial);
    if (err) return {nullptr, err};
    auto [bus, open_err] = Bus::open(std::move(port), props);
    return {std::move(bus), open_err};
}
}
