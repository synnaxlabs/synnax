// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <cstdint>
#include <functional>
#include <limits>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <span>
#include <string>
#include <unordered_map>
#include <vector>

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/soem/api.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// @brief how long the master waits for the slaves.
struct Config {
    /// @brief how long the slaves get to reach PRE-OP after discovery. The default is
    /// the PreopTimeout of the ESI files.
    x::telem::TimeSpan pre_op_timeout = x::telem::SECOND * 2;
    /// @brief how long the slaves get to reach SAFE-OP, OP, or INIT.
    x::telem::TimeSpan state_change_timeout = x::telem::SECOND * 2;
    /// @brief how long the clocks of the slaves in a DC mode get to settle on the
    /// reference clock before SYNC0 starts.
    x::telem::TimeSpan dc_settle_timeout = x::telem::SECOND * 2;
};

/// @brief SOEM-based implementation of the Master interface.
class Master final : public master::Master {
    /// @brief the network interface name (e.g., "eth0", "enp3s0").
    std::string iface_name;
    /// @brief API abstraction for SOEM operations.
    std::unique_ptr<API> api;
    /// @brief how long the master waits for slaves to change state.
    Config cfg;
    /// @brief IOmap buffer for PDO exchange.
    std::vector<uint8_t> iomap;
    /// @brief input size in bytes (TxPDO, slave->master).
    size_t input_sz;
    /// @brief output size in bytes (RxPDO, master->slave).
    size_t output_sz;
    /// @brief cached PDO offsets computed at activation time.
    pdo::Offsets pdo_offsets;
    /// @brief why each slave whose discovered layout disagrees with what SOEM mapped
    /// has no offsets, keyed by position.
    std::unordered_map<uint16_t, x::errors::Error> layout_errors;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> slave_list;
    /// @brief slave positions that are disabled (excluded from cyclic exchange).
    std::set<uint16_t> disabled_slaves;
    /// @brief protects slave state queries.
    mutable std::mutex mu;
    /// @brief whether the master has been initialized.
    bool initialized;
    /// @brief whether the master has been activated.
    bool activated;
    /// @brief expected working counter value for cyclic exchange.
    int expected_wkc;
    /// @brief whether the enabled slaves' frames carry the reference clock time.
    bool has_dc = false;
    /// @brief the process data frames that came back to the master.
    std::atomic<uint64_t> frames{0};
    /// @brief last_dc_time and host_dc_offset before a frame carries a DC time.
    static constexpr int64_t NO_DC_FRAME = std::numeric_limits<int64_t>::min();
    /// @brief the latest DC time that moved forward, in nanoseconds.
    int64_t last_dc_time = NO_DC_FRAME;
    /// @brief the DC time minus the host steady clock time, in nanoseconds, at the
    /// frame that gave last_dc_time.
    std::atomic<int64_t> host_dc_offset{NO_DC_FRAME};

    /// @brief the DC registers of one slave that runs in a DC mode.
    struct DCSetup {
        uint16_t position = 0;
        dc::Registers registers;
    };
    /// @brief the DC setup of each enabled slave in a DC mode, set at activation.
    std::vector<DCSetup> dc_setups;
    /// @brief why the master could not restore the distributed clock of a slave that
    /// came back, keyed by position.
    std::unordered_map<uint16_t, std::string> dc_restore_errors;

public:
    /// @brief constructs a SOEM master with injected API, interface name, and config.
    Master(std::unique_ptr<API> api, std::string interface_name, Config cfg = {});

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    x::errors::Error initialize() override;

    std::pair<size_t, x::errors::Error> count_slaves() override;

    void set_slave_enabled(uint16_t position, bool enabled) override;

    x::errors::Error activate(
        x::telem::TimeSpan period,
        const std::function<void()> &start_cycle
    ) override;

    void deactivate() override;

    x::errors::Error receive() override;

    x::errors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    std::pair<pdo::Offset, x::errors::Error>
    pdo_offset(const pdo::Entry &entry) const override;

    std::vector<slave::DiscoveryResult> slaves() const override;

    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override;

    std::optional<x::telem::TimeStamp> dc_time() const override;

    std::string interface_name() const override;

private:
    /// @brief populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// @brief computes and caches PDO offsets for all slaves after activation.
    void cache_pdo_offsets();

    /// @brief discovers PDO entries for a slave and populates its PDO lists.
    void discover_slave_pdos(slave::DiscoveryResult &slave);

    /// @brief discovers PDOs using CoE SDO reads (primary method).
    bool discover_pdos_coe(slave::DiscoveryResult &slave);

    /// @brief discovers PDOs from SII EEPROM (fallback method).
    void discover_pdos_sii(slave::DiscoveryResult &slave);

    /// @brief reads the synchronization modes the slave at the given position lists in
    /// its SII. Returns an empty list when the SII has no DC category.
    std::vector<dc::OpMode> read_op_modes(uint16_t position);

    /// @brief reads PDO assignment object to get list of assigned PDOs.
    x::errors::Error read_pdo_assign(
        uint16_t slave_pos,
        uint16_t assign_index,
        bool is_input,
        slave::DiscoveryResult &slave
    );

    /// @brief reads the name of a CoE object, or nullopt when the slave has none.
    std::optional<std::string> read_object_name(uint16_t slave_pos, uint16_t index);

    /// @brief reads PDO mapping entries from a specific PDO.
    x::errors::Error read_pdo_mapping(
        uint16_t slave_pos,
        uint16_t pdo_index,
        bool is_input,
        slave::DiscoveryResult &slave
    );

    /// @brief reads the name and CoE data type of a PDO entry from the object
    /// dictionary.
    std::pair<std::string, telem::DataType>
    read_pdo_entry(uint16_t slave_pos, uint16_t index, uint8_t sub_index);

    /// @brief scans the CoE object dictionary to find PDO mapping indices.
    bool
    scan_object_dictionary_for_pdos(uint16_t slave_pos, slave::DiscoveryResult &slave);

    /// @brief writes data to a DC register of a slave.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take the write.
    x::errors::Error
    write_dc_register(uint16_t pos, uint16_t address, std::vector<uint8_t> data);

    /// @brief reads a DC register of a slave into data.
    /// @returns errors::ACTIVATION_ERROR when the slave does not answer.
    x::errors::Error
    read_dc_register(uint16_t pos, uint16_t address, std::span<uint8_t> data);

    /// @brief turns SYNC off on a slave: writes zero to 0x0980, then to 0x0981.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take a write.
    x::errors::Error clear_sync(uint16_t pos);

    /// @brief writes zero to the DC activation registers (0x0980 and 0x0981) of every
    /// enabled slave with a distributed clock, also after a slave fails a write.
    /// @returns errors::ACTIVATION_ERROR that names each slave that failed a write.
    x::errors::Error clear_dc_activation();

    /// @brief returns the DC registers of each enabled slave that dc::select puts in a
    /// DC mode, at a bus cycle of period.
    /// @returns errors::SLAVE_CONFIG_ERROR when a mode gives values the registers
    /// cannot hold.
    std::pair<std::vector<DCSetup>, x::errors::Error>
    resolve_dc(x::telem::TimeSpan period) const;

    /// @brief waits until a frame carries the reference clock to the slaves, then until
    /// the clock of each slave at positions is within 10 µs of the reference clock.
    /// @returns errors::ACTIVATION_ERROR when no frame comes back, a slave does not
    /// answer, or a clock does not settle within Config::dc_settle_timeout.
    x::errors::Error wait_for_dc_settle(const std::vector<uint16_t> &positions);

    /// @brief starts SYNC0 on a slave, with its first SYNC0 event on a whole SYNC0
    /// cycle of DC time at least 100 ms after its system time.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take a write.
    x::errors::Error start_sync(const DCSetup &setup);

    /// @brief starts SYNC0 on each slave in dc_setups once their clocks settle. Call it
    /// while the bus cycles and before SAFE-OP.
    /// @returns errors::ACTIVATION_ERROR when a slave does not take a write.
    x::errors::Error start_dc();

    /// @brief sets the DC system time offset (0x0920) of a slave so that its clock
    /// matches the clock of another enabled DC slave in SAFE-OP or OP. Without one, the
    /// clock continues the latest DC time that moved forward.
    /// @returns errors::ACTIVATION_ERROR when no frame has carried the DC time, or the
    /// slave does not answer or take the write.
    x::errors::Error align_clock(uint16_t pos);

    /// @brief sets up the distributed clock of a slave that came back to PRE-OP while
    /// the bus cycles: its propagation delay, its offset, and SYNC0 when it runs in a
    /// DC mode. Blocks until its clock settles.
    /// @returns errors::ACTIVATION_ERROR when a step fails.
    x::errors::Error restore_dc(uint16_t pos);

    /// @brief requests the state from all enabled slaves, then waits until they reach
    /// it, one of them refuses it, or the timeout in microseconds passes.
    x::errors::Error request_state(uint16_t state, int timeout);

    /// @brief waits until a frame that the cycle sent after the call comes back.
    /// @returns false when the deadline passes first.
    [[nodiscard]] bool
    wait_for_frame(std::chrono::steady_clock::time_point deadline) const;

    /// @brief whether the slave has its AL error flag set and gives an AL status code.
    [[nodiscard]] bool al_error(uint16_t pos) const;

    /// @brief whether the slave refused the state: it stays below the state with an AL
    /// error. A slave above the state can still show an error from before a request to
    /// go down.
    [[nodiscard]] bool refuses(uint16_t pos, uint16_t state) const;

    /// @brief takes one recovery step for a slave that is not in OP, following the SOEM
    /// ecatcheck sample: acknowledge an error, request OP, reconfigure, or restore a
    /// slave that lost power.
    void recover(uint16_t pos);

    /// @brief builds the error for enabled slaves that did not reach a state. It is a
    /// refusal when a slave refuses the state with an AL status code that a new start
    /// cannot clear.
    x::errors::Error state_error(const std::vector<uint16_t> &failed, uint16_t state);
};

/// @brief SOEM-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override;

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override;
};

}
