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
#include <bitset>
#include <string>

#include "driver/errors/errors.h"
#include "driver/mil1553/simulated/backend.h"

namespace driver::mil1553::simulated {
namespace {
using Words = std::array<std::uint16_t, codec::mil1553::MAX_WORDS>;
using Terminals = std::bitset<codec::mil1553::MAX_RT + 1>;
using Receiver = bus::Queue<Transfer>;

/// @brief the number of subaddresses a terminal has, counting the mode code ones.
constexpr std::size_t SUBADDRESSES = 32;

/// @brief Terminal is the state of one remote terminal address.
struct Terminal {
    /// @brief owned is true while a remote terminal channel owns the terminal.
    bool owned = false;
    /// @brief received holds the words last received on each subaddress.
    std::array<Words, SUBADDRESSES> received{};
    /// @brief responses holds the words an owned terminal answers each subaddress
    /// with.
    std::array<Words, SUBADDRESSES> responses{};
    /// @brief responding is true for each subaddress with a response set.
    std::bitset<SUBADDRESSES> responding;
};

/// @brief Listener is a channel that reads transfers.
struct Listener {
    /// @brief queue holds the transfers the channel has not read.
    std::shared_ptr<Receiver> queue;
    /// @brief terminals are the terminals whose transfers the channel reads. Unused
    /// for a monitor.
    Terminals terminals;
    /// @brief monitor is true when the channel reads every transfer.
    bool monitor = false;
};

x::errors::Error misuse(const std::string &message) {
    return x::errors::Error(errors::CRITICAL_HARDWARE_ERROR, message);
}
}

struct Backend::Bus {
    /// @brief mu guards terminals and listeners.
    std::mutex mu;
    /// @brief terminals holds each remote terminal address.
    std::array<Terminal, codec::mil1553::MAX_RT + 1> terminals;
    /// @brief listeners holds each monitor and remote terminal channel.
    std::vector<Listener> listeners;
};

namespace {
/// @brief Channel is one open channel on a simulated bus.
class Channel final : public mil1553::Channel {
    std::shared_ptr<Backend::Bus> bus;
    std::string role;
    Terminals terminals;
    std::shared_ptr<Receiver> queue;

public:
    /// @brief opens a channel in role. The caller marks the terminals as owned
    /// before it opens a remote terminal.
    Channel(std::shared_ptr<Backend::Bus> bus, std::string role, Terminals terminals):
        bus(std::move(bus)), role(std::move(role)), terminals(terminals) {
        if (this->role == synnax::mil1553::ROLE_BUS_CONTROLLER) return;
        this->queue = std::make_shared<Receiver>(RECEIVE_CAPACITY);
        std::lock_guard lock(this->bus->mu);
        this->bus->listeners.push_back({
            .queue = this->queue,
            .terminals = this->terminals,
            .monitor = this->role == synnax::mil1553::ROLE_MONITOR,
        });
    }

    ~Channel() override {
        std::lock_guard lock(this->bus->mu);
        for (std::size_t rt = 0; rt < this->terminals.size(); rt++) {
            if (!this->terminals.test(rt)) continue;
            this->bus->terminals[rt] = Terminal{};
        }
        auto &l = this->bus->listeners;
        l.erase(
            std::remove_if(
                l.begin(),
                l.end(),
                [this](const Listener &x) { return x.queue == this->queue; }
            ),
            l.end()
        );
    }

    std::pair<Transfer, x::errors::Error> transact(
        const codec::mil1553::Command &command,
        const std::span<const std::uint16_t> words
    ) override {
        if (this->role != synnax::mil1553::ROLE_BUS_CONTROLLER)
            return {{}, misuse("only a bus controller channel makes transfers")};
        if (command.mode_code() || command.rt > codec::mil1553::MAX_RT)
            return {
                {},
                misuse("the simulated bus does not carry mode codes or broadcasts"),
            };
        if (command.count == 0 || command.count > codec::mil1553::MAX_WORDS)
            return {{}, misuse("a transfer carries from 1 to 32 words")};
        if (!command.transmit && words.size() != command.count)
            return {
                {},
                misuse(
                    "a receive command with " + std::to_string(command.count) +
                    " words got " + std::to_string(words.size())
                ),
            };
        Transfer t{
            .command = command,
            .answered = true,
            .status = {.rt = command.rt},
        };
        std::lock_guard lock(this->bus->mu);
        auto &term = this->bus->terminals[command.rt];
        const auto sa = command.subaddress;
        if (!command.transmit) {
            std::ranges::copy(words, t.words.begin());
            std::ranges::copy(words, term.received[sa].begin());
        } else if (!term.owned)
            t.words = term.received[sa];
        else if (term.responding.test(sa))
            t.words = term.responses[sa];
        else
            t.status.busy = true;
        std::fill(t.words.begin() + t.count(), t.words.end(), 0);
        t.time = x::telem::TimeStamp::now();
        for (const auto &l: this->bus->listeners)
            if (l.monitor || l.terminals.test(command.rt)) l.queue->push(t);
        return {t, x::errors::NIL};
    }

    std::pair<bus::Batch, x::errors::Error>
    read(const std::span<Transfer> out, const x::telem::TimeSpan timeout) override {
        if (this->queue == nullptr)
            return {{}, misuse("a bus controller channel does not read transfers")};
        return {this->queue->pop(out, timeout), x::errors::NIL};
    }

    x::errors::Error respond(
        const std::uint8_t rt,
        const std::uint8_t subaddress,
        const std::span<const std::uint16_t> words
    ) override {
        if (rt >= this->terminals.size() || !this->terminals.test(rt))
            return misuse("the channel does not own terminal " + std::to_string(rt));
        if (subaddress == 0 || subaddress > codec::mil1553::MAX_RT)
            return misuse(
                "subaddress " + std::to_string(subaddress) + " is not from 1 to 30"
            );
        if (words.empty() || words.size() > codec::mil1553::MAX_WORDS)
            return misuse("a response carries from 1 to 32 words");
        std::lock_guard lock(this->bus->mu);
        auto &term = this->bus->terminals[rt];
        term.responses[subaddress] = {};
        std::ranges::copy(words, term.responses[subaddress].begin());
        term.responding.set(subaddress);
        return x::errors::NIL;
    }
};
}

std::pair<std::unique_ptr<mil1553::Channel>, x::errors::Error>
Backend::open(const synnax::mil1553::Properties &props) {
    if (auto err = validate(props)) return {nullptr, err};
    std::shared_ptr<Bus> bus;
    {
        std::lock_guard lock(this->mu);
        auto &b = this->buses[{props.card, props.channel}];
        if (b == nullptr) b = std::make_shared<Bus>();
        bus = b;
    }
    Terminals terminals;
    for (const auto rt: props.terminals)
        terminals.set(rt);
    if (terminals.any()) {
        std::lock_guard lock(bus->mu);
        for (const auto rt: props.terminals)
            if (bus->terminals[rt].owned)
                return {
                    nullptr,
                    x::errors::Error(
                        errors::CONFIGURATION_ERROR,
                        "terminal " + std::to_string(rt) +
                            " is already owned by another channel"
                    ),
                };
        for (const auto rt: props.terminals)
            bus->terminals[rt] = Terminal{.owned = true};
    }
    return {
        std::make_unique<Channel>(std::move(bus), props.role, terminals),
        x::errors::NIL,
    };
}

std::pair<std::vector<Info>, x::errors::Error> Backend::list() {
    return {
        {Info{.card = 0, .channel = 0, .name = "Simulated MIL-STD-1553"}},
        x::errors::NIL,
    };
}
}
