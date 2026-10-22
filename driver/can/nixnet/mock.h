// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cstring>
#include <deque>
#include <map>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

#include "driver/can/nixnet/api.h"

namespace driver::can::nixnet {
/// @brief a model of NI-XNET with CAN interfaces, written from the status codes and
/// their solutions in nixnet.h and NI's knowledge base. Each session keeps its own
/// state. A session cannot be created on an interface that runs, and sessions that
/// start an interface at different bitrates conflict. A removed interface fails the
/// calls of its sessions with nxErrCommunicationLost and refuses new sessions with
/// nxErrInvalidInterface, as for a removed module or chassis. Safe for concurrent use.
class MockAPI final : public API {
public:
    /// @brief a session the backend created.
    struct Session {
        std::string database;
        std::string interface;
        u32 mode = 0;
        std::map<u32, std::vector<std::uint8_t>> properties;
        bool started = false;
        bool cleared = false;
        /// @brief true when the interface was removed after the session was created.
        bool lost = false;

        /// @returns the value of a property the backend set.
        template<typename T>
        T property(const u32 id) const {
            T value{};
            std::memcpy(&value, this->properties.at(id).data(), sizeof(T));
            return value;
        }
    };

    /// @brief a CAN interface of the system.
    struct Interface {
        std::string name;
        std::string device;
        u32 serial = 0;
    };

    /// @brief one result of ReadFrame on an input session.
    struct Read {
        nxStatus_t status = nxSuccess;
        std::vector<std::uint8_t> bytes;
    };

    /// @brief the status WriteFrame returns while the interface is present.
    nxStatus_t write_status = nxSuccess;

    /// @brief adds a CAN interface to the system.
    void add(const Interface &intf) {
        std::lock_guard lock(this->mu);
        this->interfaces.push_back({.intf = intf});
    }

    /// @brief removes an interface, as an unplugged device or chassis does.
    void unplug(const std::string &name) {
        std::lock_guard lock(this->mu);
        this->find(name)->present = false;
        for (auto &[ref, s]: this->sessions)
            if (s.interface == name && !s.cleared) s.lost = true;
    }

    /// @brief returns a removed interface to the system.
    void plug(const std::string &name) {
        std::lock_guard lock(this->mu);
        this->find(name)->present = true;
    }

    /// @brief makes another application run an interface at a bitrate, or stop.
    void hold(const std::string &name, const std::optional<u64> bitrate) {
        std::lock_guard lock(this->mu);
        this->find(name)->other = bitrate;
    }

    /// @brief queues a result for ReadFrame on every input session of an interface.
    void deliver(const std::string &name, const Read &read) {
        std::lock_guard lock(this->mu);
        for (auto &[ref, s]: this->sessions)
            if (s.interface == name && s.mode == nxMode_FrameInStream && !s.cleared)
                this->reads[ref].push_back(read);
    }

    /// @returns a copy of a session's state.
    Session session(const nxSessionRef_t ref) {
        std::lock_guard lock(this->mu);
        return this->sessions.at(ref);
    }

    /// @returns the number of sessions the backend created.
    std::size_t created() {
        std::lock_guard lock(this->mu);
        return this->sessions.size();
    }

    /// @returns every raw frame buffer WriteFrame sent.
    std::vector<std::vector<std::uint8_t>> written() {
        std::lock_guard lock(this->mu);
        return this->sent;
    }

    nxStatus_t CreateSession(
        const char *database,
        const char *,
        const char *,
        const char *interface,
        const u32 mode,
        nxSessionRef_t *session
    ) override {
        std::lock_guard lock(this->mu);
        const auto *intf = this->find(interface);
        if (intf == nullptr || !intf->present) return nxErrInvalidInterface;
        if (this->running(interface)) return nxErrInterfaceRunning;
        *session = static_cast<nxSessionRef_t>(this->sessions.size() + 1);
        this->sessions[*session] = {
            .database = database,
            .interface = interface,
            .mode = mode,
        };
        return nxSuccess;
    }

    nxStatus_t Clear(const nxSessionRef_t session) override {
        std::lock_guard lock(this->mu);
        auto &s = this->sessions.at(session);
        s.cleared = true;
        s.started = false;
        this->reads.erase(session);
        return nxSuccess;
    }

    nxStatus_t SetProperty(
        const nxSessionRef_t session,
        const u32 id,
        const u32 size,
        void *value
    ) override {
        std::lock_guard lock(this->mu);
        auto &s = this->sessions.at(session);
        if (s.lost) return nxErrCommunicationLost;
        const auto *bytes = static_cast<std::uint8_t *>(value);
        s.properties[id] = {bytes, bytes + size};
        return nxSuccess;
    }

    nxStatus_t GetProperty(
        const nxSessionRef_t ref,
        const u32 id,
        const u32 size,
        void *value
    ) override {
        std::lock_guard lock(this->mu);
        if (id == nxPropSys_IntfRefsCAN) {
            const auto present = this->present();
            for (u32 i = 0; i < size / sizeof(nxSessionRef_t); i++)
                std::memcpy(
                    static_cast<std::uint8_t *>(value) + i * sizeof(nxSessionRef_t),
                    &present.at(i),
                    sizeof(nxSessionRef_t)
                );
            return nxSuccess;
        }
        if (id == nxPropIntf_DevRef) {
            const nxSessionRef_t device = ref + 100;
            std::memcpy(value, &device, sizeof(device));
            return nxSuccess;
        }
        if (id == nxPropDev_SerNum) {
            std::memcpy(
                value,
                &this->interfaces.at(ref - 200).intf.serial,
                sizeof(u32)
            );
            return nxSuccess;
        }
        const auto text = this->string(ref, id);
        std::strncpy(static_cast<char *>(value), text.c_str(), size);
        return nxSuccess;
    }

    nxStatus_t
    GetPropertySize(const nxSessionRef_t ref, const u32 id, u32 *size) override {
        std::lock_guard lock(this->mu);
        if (id == nxPropSys_IntfRefsCAN) {
            *size = static_cast<u32>(this->present().size() * sizeof(nxSessionRef_t));
            return nxSuccess;
        }
        *size = static_cast<u32>(this->string(ref, id).size() + 1);
        return nxSuccess;
    }

    nxStatus_t ReadFrame(
        const nxSessionRef_t session,
        void *buffer,
        const u32 size,
        f64,
        u32 *returned
    ) override {
        std::lock_guard lock(this->mu);
        *returned = 0;
        if (this->sessions.at(session).lost) return nxErrCommunicationLost;
        auto &queue = this->reads[session];
        if (queue.empty()) return nxSuccess;
        const auto read = queue.front();
        queue.pop_front();
        if (read.status < nxSuccess) return read.status;
        const auto n = std::min<std::size_t>(read.bytes.size(), size);
        std::memcpy(buffer, read.bytes.data(), n);
        *returned = static_cast<u32>(n);
        return read.status;
    }

    nxStatus_t WriteFrame(
        const nxSessionRef_t session,
        void *buffer,
        const u32 size,
        f64
    ) override {
        std::lock_guard lock(this->mu);
        if (this->sessions.at(session).lost) return nxErrCommunicationLost;
        if (this->write_status < nxSuccess) return this->write_status;
        const auto *bytes = static_cast<std::uint8_t *>(buffer);
        this->sent.emplace_back(bytes, bytes + size);
        return nxSuccess;
    }

    nxStatus_t Start(const nxSessionRef_t session, u32) override {
        std::lock_guard lock(this->mu);
        auto &s = this->sessions.at(session);
        if (s.lost) return nxErrCommunicationLost;
        const auto bitrate = s.property<u64>(nxPropSession_IntfBaudRate64);
        const auto *intf = this->find(s.interface);
        if (intf->other.has_value() && *intf->other != bitrate)
            return nxErrDifferentClusterOpen;
        for (const auto &[ref, other]: this->sessions)
            if (other.started && other.interface == s.interface &&
                other.property<u64>(nxPropSession_IntfBaudRate64) != bitrate)
                return nxErrDifferentClusterOpen;
        s.started = true;
        return nxSuccess;
    }

    void StatusToString(
        const nxStatus_t status,
        const u32 size,
        char *description
    ) override {
        const auto text = "mock status " + std::to_string(status);
        std::strncpy(description, text.c_str(), size - 1);
    }

    nxStatus_t SystemOpen(nxSessionRef_t *system) override {
        *system = 1;
        return nxSuccess;
    }

    nxStatus_t SystemClose(nxSessionRef_t) override { return nxSuccess; }

private:
    struct State {
        Interface intf;
        bool present = true;
        std::optional<u64> other;
    };

    std::mutex mu;
    /// @brief the interfaces, whose refs start at 100 and whose device refs start at
    /// 200.
    std::vector<State> interfaces;
    std::map<nxSessionRef_t, Session> sessions;
    std::map<nxSessionRef_t, std::deque<Read>> reads;
    std::vector<std::vector<std::uint8_t>> sent;

    State *find(const std::string &name) {
        for (auto &s: this->interfaces)
            if (s.intf.name == name) return &s;
        return nullptr;
    }

    /// @returns true when a session of this process runs the interface.
    bool running(const std::string &name) {
        return std::ranges::any_of(this->sessions, [&](const auto &entry) {
            return entry.second.started && entry.second.interface == name;
        });
    }

    /// @returns the refs of the interfaces present.
    std::vector<nxSessionRef_t> present() const {
        std::vector<nxSessionRef_t> refs;
        for (std::size_t i = 0; i < this->interfaces.size(); i++)
            if (this->interfaces[i].present)
                refs.push_back(static_cast<nxSessionRef_t>(100 + i));
        return refs;
    }

    /// @returns the string property of an interface or a device.
    std::string string(const nxSessionRef_t ref, const u32 id) const {
        if (id == nxPropIntf_Name) return this->interfaces.at(ref - 100).intf.name;
        if (id == nxPropDev_Name) return this->interfaces.at(ref - 200).intf.device;
        return "";
    }
};
}
