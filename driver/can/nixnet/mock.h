// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <deque>
#include <map>
#include <mutex>
#include <string>
#include <vector>

#include "driver/can/nixnet/api.h"

namespace driver::can::nixnet {
/// @brief a simulated NI-XNET. Tests queue the raw frame bytes reads return and
/// inspect the sessions the backend created and the raw frames it wrote.
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

        /// @returns the value of a property the backend set.
        template<typename T>
        T property(const u32 id) const {
            T value{};
            std::memcpy(&value, this->properties.at(id).data(), sizeof(T));
            return value;
        }
    };

    /// @brief an interface the mock reports to scans.
    struct Interface {
        std::string name;
        std::string device;
        u32 serial = 0;
    };

    std::mutex mu;
    std::map<nxSessionRef_t, Session> sessions;
    std::vector<Interface> interfaces;
    /// @brief the byte chunks reads return in order.
    std::deque<std::vector<std::uint8_t>> reads;
    /// @brief every raw frame buffer WriteFrame sent.
    std::vector<std::vector<std::uint8_t>> written;
    nxStatus_t create_status = nxSuccess;
    nxStatus_t write_status = nxSuccess;

    nxStatus_t CreateSession(
        const char *database,
        const char *,
        const char *,
        const char *interface,
        const u32 mode,
        nxSessionRef_t *session
    ) override {
        std::lock_guard lock(this->mu);
        if (this->create_status < nxSuccess) return this->create_status;
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
        this->sessions.at(session).cleared = true;
        return nxSuccess;
    }

    nxStatus_t SetProperty(
        const nxSessionRef_t session,
        const u32 id,
        const u32 size,
        void *value
    ) override {
        std::lock_guard lock(this->mu);
        const auto *bytes = static_cast<std::uint8_t *>(value);
        this->sessions.at(session).properties[id] = {bytes, bytes + size};
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
            for (u32 i = 0; i < size / sizeof(nxSessionRef_t); i++) {
                const nxSessionRef_t intf = 100 + i;
                std::memcpy(
                    static_cast<std::uint8_t *>(value) + i * sizeof(intf),
                    &intf,
                    sizeof(intf)
                );
            }
            return nxSuccess;
        }
        if (id == nxPropIntf_DevRef) {
            const nxSessionRef_t device = ref + 100;
            std::memcpy(value, &device, sizeof(device));
            return nxSuccess;
        }
        if (id == nxPropDev_SerNum) {
            std::memcpy(value, &this->interfaces.at(ref - 200).serial, sizeof(u32));
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
            *size = static_cast<u32>(this->interfaces.size() * sizeof(nxSessionRef_t));
            return nxSuccess;
        }
        *size = static_cast<u32>(this->string(ref, id).size() + 1);
        return nxSuccess;
    }

    nxStatus_t ReadFrame(
        nxSessionRef_t,
        void *buffer,
        const u32 size,
        f64,
        u32 *returned
    ) override {
        std::lock_guard lock(this->mu);
        *returned = 0;
        if (this->reads.empty()) return nxSuccess;
        const auto chunk = this->reads.front();
        this->reads.pop_front();
        const auto n = std::min<std::size_t>(chunk.size(), size);
        std::memcpy(buffer, chunk.data(), n);
        *returned = static_cast<u32>(n);
        return nxSuccess;
    }

    nxStatus_t WriteFrame(nxSessionRef_t, void *buffer, const u32 size, f64) override {
        std::lock_guard lock(this->mu);
        if (this->write_status < nxSuccess) return this->write_status;
        const auto *bytes = static_cast<std::uint8_t *>(buffer);
        this->written.emplace_back(bytes, bytes + size);
        return nxSuccess;
    }

    nxStatus_t Start(const nxSessionRef_t session, u32) override {
        std::lock_guard lock(this->mu);
        this->sessions.at(session).started = true;
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
    /// @returns the string property of an interface (refs from 100) or device (refs
    /// from 200).
    std::string string(const nxSessionRef_t ref, const u32 id) const {
        if (id == nxPropIntf_Name) return this->interfaces.at(ref - 100).name;
        if (id == nxPropDev_Name) return this->interfaces.at(ref - 200).device;
        return "";
    }
};
}
