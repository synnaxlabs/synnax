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
#include "driver/transport/errors.h"

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

bool is_answer(const Message &msg) {
    return msg.reply == Reply::ACCEPTED || msg.reply == Reply::REJECTED;
}

bool is_line(const Message &msg) {
    return msg.reply == Reply::OTHER || msg.reply == Reply::MALFORMED;
}
}

x::errors::Error translate(const x::errors::Error &err) {
    if (!err) return err;
    if (err.matches(transport::UNREACHABLE_ERROR)) return {UNREACHABLE_ERROR, err.data};
    if (err.matches(transport::CONFIG_ERROR)) return {CONFIG_ERROR, err.data};
    return {CRITICAL_HARDWARE_ERROR, err.data};
}

Bus::Bus(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props):
    can::Bus(props.channel, props.fd, props.listen_only), port(std::move(port)) {}

Bus::~Bus() {
    this->close();
}

std::pair<std::vector<Message>, x::errors::Error>
Bus::read(const x::telem::TimeSpan timeout) {
    auto [chunk, err] = this->port->read(timeout);
    if (err) return {{}, translate(err)};
    std::vector<Message> messages;
    std::vector<Frame> frames;
    for (const auto byte: chunk.data) {
        auto msg = this->decoder.push(static_cast<char>(byte));
        if (!msg.has_value()) continue;
        if (msg->reply != Reply::FRAME) {
            messages.push_back(std::move(*msg));
            continue;
        }
        msg->frame.time = chunk.time;
        msg->frame.clock = Clock::HOST;
        frames.push_back(msg->frame);
    }
    if (!frames.empty()) {
        {
            std::lock_guard lock(this->mu);
            if (this->inbox.empty()) this->overflowed = false;
            for (const auto &frame: frames) {
                if (this->inbox.size() >= MAX_QUEUED) {
                    this->inbox.pop_front();
                    if (!this->overflowed)
                        LOG(WARNING) << "[can] " << this->name
                                     << ": frames arrive faster than they are read, "
                                        "dropping the oldest";
                    this->overflowed = true;
                }
                this->inbox.push_back(frame);
            }
        }
        this->cv.notify_all();
    }
    return {std::move(messages), x::errors::NIL};
}

x::errors::Error Bus::write(const std::string_view line) {
    return translate(this->port->write(bytes(line), WRITE_TIMEOUT));
}

std::pair<std::optional<Message>, x::errors::Error>
Bus::await(bool (*matches)(const Message &)) {
    const auto deadline = x::telem::TimeStamp::now() + REPLY_TIMEOUT;
    for (auto now = x::telem::TimeStamp::now(); now < deadline;
         now = x::telem::TimeStamp::now()) {
        auto [messages, err] = this->read(deadline - now);
        if (err) return {std::nullopt, err};
        for (auto &msg: messages)
            if (matches(msg)) return {std::move(msg), x::errors::NIL};
    }
    return {std::nullopt, x::errors::NIL};
}

std::pair<std::string, x::errors::Error> Bus::version() {
    if (auto err = this->write(VERSION_COMMAND)) return {"", err};
    // LAWICEL firmware answers C before V, so the version is the first other line.
    auto [msg, err] = this->await(is_line);
    if (err) return {"", err};
    if (!msg.has_value())
        return {
            "",
            {UNREACHABLE_ERROR,
             "the device at " + this->name + " did not answer the slcan command '" +
                 printable(VERSION_COMMAND) + "'. Check that it is an slcan adapter"}
        };
    return {msg->text, x::errors::NIL};
}

x::errors::Error Bus::configure(const std::string_view command) {
    if (auto err = this->write(command)) return err;
    if (!answers(this->firmware)) return x::errors::NIL;
    auto [msg, err] = this->await(is_answer);
    if (err) return err;
    if (!msg.has_value())
        return {
            UNREACHABLE_ERROR,
            "the slcan adapter at " + this->name + " did not answer the command '" +
                printable(command) + "'"
        };
    if (msg->reply == Reply::REJECTED)
        return {
            CONFIG_ERROR,
            "the slcan adapter at " + this->name + " rejected the command '" +
                printable(command) + "'"
        };
    return x::errors::NIL;
}

std::pair<std::unique_ptr<Bus>, x::errors::Error>
Bus::open(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props) {
    std::unique_ptr<Bus> bus(new Bus(std::move(port), props));
    // CANable firmware ignores settings while on the bus, so the adapter leaves it
    // first.
    if (auto err = bus->write(CLOSE_COMMAND)) return {nullptr, err};
    auto [version, version_err] = bus->version();
    if (version_err) return {nullptr, version_err};
    bus->firmware = identify(version);
    bus->inbox.clear();
    auto [bitrate, bitrate_err] = bitrate_command(bus->firmware, props.bitrate);
    if (bitrate_err) return {nullptr, bitrate_err};
    std::vector<std::string> commands{bitrate};
    if (props.fd) {
        auto [command, err] = data_bitrate_command(props.data_bitrate);
        if (err) return {nullptr, err};
        commands.push_back(command);
    }
    auto [opening, open_err] = open_commands(bus->firmware, props.listen_only);
    if (open_err) return {nullptr, open_err};
    commands.insert(commands.end(), opening.begin(), opening.end());
    for (const auto &command: commands)
        if (auto err = bus->configure(command)) return {nullptr, err};
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
        if (auto err = this->write(line)) return err;
    auto [messages, err] = this->read(READ_SLICE);
    if (err) return err;
    for (const auto &msg: messages) {
        if (msg.reply == Reply::REJECTED)
            LOG(WARNING) << "[can] " << this->name << ": the adapter rejected a frame";
        if (msg.reply == Reply::MALFORMED)
            LOG(WARNING) << "[can] " << this->name
                         << ": the adapter sent a malformed line";
    }
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

x::errors::Error Bus::transmit(const Frame &frame) {
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
        err = this->write(line);
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
    if (err) return {nullptr, translate(err)};
    auto [bus, open_err] = Bus::open(std::move(port), props);
    return {std::move(bus), open_err};
}
}
