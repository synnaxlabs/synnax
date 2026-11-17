// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <functional>
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/api.h"
#include "driver/ethercat/igh/ecrt.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::igh {
/// @brief how the IgH master waits for its slaves.
struct Config {
    /// @brief how long the slaves get to reach OP after activation. Some slaves take up
    /// to 9 s.
    x::telem::TimeSpan op_timeout = x::telem::SECOND * 10;
    /// @brief the host clock that the master waits on.
    x::loop::HostClock clock;
};

/// @brief IgH EtherLab implementation of the Master interface.
class Master final : public master::Master {
    /// @brief API wrapper for dynamic library loading.
    std::shared_ptr<API> api;
    /// @brief IgH master index (typically 0, configured in /etc/ethercat.conf).
    unsigned int master_index;
    /// @brief how long the master waits for the slaves.
    Config cfg;

    // Non-owning pointers managed by IgH kernel module via ecrt_release_master().

    /// @brief IgH master handle from ecrt_request_master().
    ec_master_t *ec_master;

    /// @brief the process data of one enabled slave. Its own domain gives the working
    /// counter of the slave alone.
    struct Domain {
        uint16_t position = 0;
        ec_domain_t *handle = nullptr;
        /// @brief the domain memory, valid only after activation.
        uint8_t *data = nullptr;
        /// @brief where the domain starts in the inputs and outputs.
        size_t offset = 0;
        size_t size = 0;
        master::WorkingCounterShare share;

        /// @brief returns where the domain ends in the inputs and outputs.
        [[nodiscard]] size_t end() const { return this->offset + this->size; }
    };
    /// @brief the domain of each enabled slave with process data, in position order.
    std::vector<Domain> domains;
    /// @brief a copy of every domain after each receive(), as the image inputs.
    std::vector<uint8_t> inputs;
    /// @brief the image outputs, which send() copies to every domain.
    std::vector<uint8_t> outputs;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> cached_slaves;
    /// @brief the configuration of each enabled slave, in position order.
    std::vector<std::pair<uint16_t, ec_slave_config_t *>> slave_configs;
    /// @brief whether the master has been initialized.
    bool initialized;
    /// @brief whether the master has been activated.
    bool activated;
    /// @brief what shortfall() returns.
    std::optional<uint32_t> latest_shortfall;
    /// @brief what inputless() returns.
    std::vector<uint16_t> latest_inputless;
    /// @brief what the latest check() returned.
    std::vector<slave::Health> latest_health;

public:
    /// @brief constructs an IgH master with the given API and master index.
    explicit Master(
        std::shared_ptr<API> api,
        unsigned int master_index = 0,
        Config cfg = {}
    );

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    /// @brief returns the key of the master, such as "igh:0".
    [[nodiscard]] std::string name() const override {
        return "igh:" + std::to_string(this->master_index);
    }

    std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> scan() override;

    std::pair<master::Change, x::errors::Error> changed() override;

    [[nodiscard]] bool restarts_visible() const override { return false; }

    /// @brief does what master::Master::activate does, but ignores the period and runs
    /// no DC sync, so a slave that needs DC sync refuses SAFE-OP. Returns
    /// errors::SLAVE_CONFIG_ERROR, before it configures any slave, when the chosen mode
    /// of a slave is one that the IgH master cannot run. The IgH master moves the
    /// slaves to OP on its own, so a slave that has its AL error flag set at
    /// Config::op_timeout counts as a refusal.
    master::Activation activate(
        const master::Plan &plan,
        const std::function<void(const master::Image &)> &start_cycle
    ) override;

    void deactivate() override;

    x::errors::Error receive() override;

    x::errors::Error send() override;

    /// @brief reports each configured slave. The IgH master returns slaves that leave
    /// OP to it on its own, so this takes no recovery step. While IgH scans the bus
    /// again, which it does when a slave loses power, it reads no slave state. A slave
    /// whose domain still answers then keeps the health of the previous check().
    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override;

    /// @brief does nothing: the IgH master brings slaves back to OP by itself.
    void recover(const x::breaker::Breaker &) override {}

    /// @brief does what master::Master::write does, but on a running bus writes
    /// nothing, since IgH takes startup parameters only before activation. It does not
    /// write those parameters again when the slave restarts, so a slave that loses
    /// power goes back to the values it stores.
    /// @returns errors::SETTING_CONFLICT when the bus runs and the slave holds another
    /// value for a parameter. errors::STARTUP_PARAMETER_ERROR when the slave rejects or
    /// fails to read one.
    x::errors::Error write(
        uint16_t position,
        const std::vector<slave::StartupParameter> &parameters
    ) override;

    std::pair<uint32_t, x::errors::Error>
    read(uint16_t position, uint16_t index, uint8_t sub_index) override;

    [[nodiscard]] std::optional<uint32_t> shortfall() const override {
        return this->latest_shortfall;
    }

    /// @brief returns the slaves with inputs whose domain gave no working counter in
    /// the latest receive(), or none when no slave answered.
    [[nodiscard]] std::span<const uint16_t> inputless() const override {
        return this->latest_inputless;
    }

    /// @brief returns nullopt, since this master does not read the distributed clock.
    [[nodiscard]] std::optional<x::telem::TimeStamp> dc_time() const override {
        return std::nullopt;
    }

private:
    /// @brief opens the master and reads each slave on the bus.
    x::errors::Error initialize();

    /// @brief waits until every enabled slave reaches OP.
    /// @returns errors::STATE_CHANGE_REFUSED with the refusals when a slave still has
    /// its AL error flag set at Config::op_timeout, or errors::STATE_CHANGE_ERROR when
    /// none has.
    master::Activation wait_for_op();

    /// @brief builds the activation for the slaves at pending that are still below OP,
    /// or returns no error when all of them reached OP since the last poll.
    master::Activation op_error(const std::vector<uint16_t> &pending);

    /// @brief reads the PDO layout a slave holds after its startup parameters, since a
    /// parameter can change it, and makes that layout the slave's config and cached
    /// layout.
    x::errors::Error configure_assigned_pdos(uint16_t position, ec_slave_config_t *sc);

    /// @brief registers every PDO entry that a slave maps in a new domain of the slave,
    /// recording in image the offset of each entry or the reason the master rejected
    /// it.
    /// @returns errors::ACTIVATION_ERROR when IgH fails to create the domain.
    x::errors::Error register_pdo_entries(
        const slave::Properties &slave,
        ec_slave_config_t *sc,
        master::Image &image
    );

    /// @brief returns true when the slave at position has a domain that gave no working
    /// counter in the latest receive().
    [[nodiscard]] bool silent(uint16_t position) const;

    /// @brief writes a parameter to the slave at position with a direct SDO download.
    /// @returns errors::STARTUP_PARAMETER_ERROR with the abort reason when the slave
    /// rejects it.
    x::errors::Error
    download(const slave::Properties &slave, const slave::StartupParameter &param);

    /// @brief writes a slave's startup parameters and adds them to its config, so the
    /// master writes them again at each move to SAFE-OP.
    x::errors::Error add_startup_parameters(
        const slave::Properties &slave,
        ec_slave_config_t *sc,
        const std::vector<slave::StartupParameter> &parameters
    );

    /// @brief appends the PDO entries that IgH scanned for a sync manager of a slave.
    /// Returns PDO_DISCOVERY_ERROR when IgH fails to give a part of the layout.
    x::errors::Error append_scanned_pdos(slave::Properties &props, uint8_t sync_index);
};

/// @brief IgH-based implementation of master::Manager.
class Manager final : public master::Manager {
    std::shared_ptr<API> api;

public:
    /// @brief creates a manager whose masters make their IgH calls through api.
    explicit Manager(std::shared_ptr<API> api): api(std::move(api)) {}

    /// @brief opens the IgH manager.
    /// @returns errors::MASTER_INIT_ERROR when the IgH device is missing, or the error
    /// of loading the IgH library.
    static std::pair<std::unique_ptr<Manager>, x::errors::Error> open();

    [[nodiscard]] std::vector<master::Info> enumerate() override;

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override;
};

}
