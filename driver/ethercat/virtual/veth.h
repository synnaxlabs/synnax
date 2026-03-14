// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdlib>
#include <string>
#include <unistd.h>

#include "x/cpp/xerrors/errors.h"

namespace ethercat::virtual_esc {

/// @brief Checks if a network interface exists.
/// @param iface The interface name to check.
/// @return true if the interface exists, false otherwise.
inline bool interface_exists(const std::string& iface) {
    const std::string path = "/sys/class/net/" + iface;
    return access(path.c_str(), F_OK) == 0;
}

/// @brief RAII wrapper for creating and managing a veth (virtual Ethernet) pair.
///
/// A veth pair creates two virtual network interfaces that are connected to each other.
/// Packets sent on one interface are received on the other, making them ideal for
/// testing network protocols without real hardware.
///
/// Usage:
/// @code
///     VethPair veth("ectest");
///     auto err = veth.create();
///     if (err) { /* handle error */ }
///     // veth.master_interface() -> "ectest0"
///     // veth.slave_interface() -> "ectest1"
///     // Use for testing...
///     // Automatically destroyed when veth goes out of scope
/// @endcode
class VethPair {
public:
    /// @brief Constructs a VethPair with the given interface name prefix.
    /// @param prefix The prefix for the interface names (default: "ectest").
    ///        The interfaces will be named prefix0 and prefix1.
    explicit VethPair(std::string prefix = "ectest"):
        prefix(std::move(prefix)),
        created(false) {}

    ~VethPair() {
        if (this->created) this->destroy();
    }

    VethPair(const VethPair&) = delete;
    VethPair& operator=(const VethPair&) = delete;

    VethPair(VethPair&& other) noexcept:
        prefix(std::move(other.prefix)),
        created(other.created) {
        other.created = false;
    }

    VethPair& operator=(VethPair&& other) noexcept {
        if (this != &other) {
            if (this->created) this->destroy();
            this->prefix = std::move(other.prefix);
            this->created = other.created;
            other.created = false;
        }
        return *this;
    }

    /// @brief Creates the veth pair, or uses an existing one if available.
    /// @return An error if creation failed and no existing pair is available.
    [[nodiscard]] xerrors::Error create() {
        if (this->created) return xerrors::NIL;
        if (interface_exists(this->master_interface()) &&
            interface_exists(this->slave_interface())) {
            this->created = false;
            return xerrors::NIL;
        }
        const std::string cmd = "ip link add " + this->master_interface() +
                               " type veth peer name " + this->slave_interface();
        int ret = std::system(cmd.c_str());
        if (ret != 0) {
            return xerrors::Error(
                "virtual_esc.veth",
                "failed to create veth pair: " + cmd +
                " (exit code: " + std::to_string(ret) + ")"
            );
        }
        auto err = this->bring_up(this->master_interface());
        if (err) {
            this->destroy_quietly();
            return err;
        }
        err = this->bring_up(this->slave_interface());
        if (err) {
            this->destroy_quietly();
            return err;
        }
        this->created = true;
        return xerrors::NIL;
    }

    /// @brief Destroys the veth pair.
    void destroy() {
        if (!this->created) return;
        this->destroy_quietly();
        this->created = false;
    }

    /// @brief Returns the master-side interface name.
    [[nodiscard]] std::string master_interface() const {
        return this->prefix + "0";
    }

    /// @brief Returns the slave-side interface name.
    [[nodiscard]] std::string slave_interface() const {
        return this->prefix + "1";
    }

    /// @brief Returns true if the veth pair has been created.
    [[nodiscard]] bool is_created() const { return this->created; }

private:
    std::string prefix;
    bool created;

    [[nodiscard]] xerrors::Error bring_up(const std::string& iface) const {
        const std::string cmd = "ip link set " + iface + " up";
        int ret = std::system(cmd.c_str());
        if (ret != 0) {
            return xerrors::Error(
                "virtual_esc.veth",
                "failed to bring up interface: " + iface +
                " (exit code: " + std::to_string(ret) + ")"
            );
        }
        return xerrors::NIL;
    }

    void destroy_quietly() const {
        const std::string cmd = "ip link delete " + this->master_interface() +
                               " 2>/dev/null";
        std::system(cmd.c_str());
    }
};

/// @brief Checks if the current process has the required capabilities to create veth pairs,
///        or if a pre-existing veth pair with the given prefix is available.
/// @param prefix The veth pair prefix to check for (e.g., "ecsoem" checks for ecsoem0/ecsoem1).
/// @return true if veth pairs can be created or already exist, false otherwise.
inline bool can_create_veth(const std::string& prefix = "") {
    if (geteuid() == 0) return true;
    if (!prefix.empty()) {
        return interface_exists(prefix + "0") && interface_exists(prefix + "1");
    }
    return false;
}

}
