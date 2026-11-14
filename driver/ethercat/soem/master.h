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
#include <concepts>
#include <cstdint>
#include <functional>
#include <limits>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <span>
#include <string>
#include <tuple>
#include <unordered_map>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/soem/api.h"
#include "driver/ethercat/soem/sii.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// @brief how long, and on which clock, the master waits for the slaves.
struct Config {
    /// @brief how long the slaves get to reach PRE-OP after discovery. The default is
    /// the PreopTimeout of the ESI files.
    x::telem::TimeSpan pre_op_timeout = x::telem::SECOND * 2;
    /// @brief how long the slaves get to reach SAFE-OP or OP. Some slaves take up to
    /// 9 s.
    x::telem::TimeSpan state_change_timeout = x::telem::SECOND * 10;
    /// @brief how long the slaves get to return to INIT when the bus stops.
    x::telem::TimeSpan init_timeout = x::telem::SECOND * 2;
    /// @brief how long the clocks of the slaves in a DC mode get to settle on the
    /// reference clock before SYNC0 starts.
    x::telem::TimeSpan dc_settle_timeout = x::telem::SECOND * 2;
    /// @brief the host clock that the master waits on and times process data sends
    /// with.
    x::loop::HostClock clock;
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
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> slave_list;
    /// @brief whether scan() opened the interface and found slaves.
    bool initialized = false;

    /// @brief what the object dictionary and SII of one slave model describe. They do
    /// not change for a model, so the master keeps them across scans. A modular slave's
    /// model includes the modules it detects.
    struct Description {
        /// @brief the names of the CoE objects, keyed by index.
        std::map<uint16_t, std::string> object_names;
        /// @brief the names and CoE data types of object entries, keyed by index and
        /// sub-index.
        std::map<std::pair<uint16_t, uint8_t>, std::pair<std::string, telem::DataType>>
            entries;
        /// @brief the synchronization modes in the SII, or none before a read finds
        /// them.
        std::vector<dc::OpMode> op_modes;
    };
    /// @brief a slave model: its vendor ID, product code, revision, and the identities
    /// of the modules that it detects.
    using Model = std::tuple<uint32_t, uint32_t, uint32_t, std::vector<uint32_t>>;
    /// @brief the description of each slave model.
    std::map<Model, Description> descriptions;
    /// @brief the frames that came back to the master.
    std::atomic<uint64_t> frames{0};
    /// @brief last_dc_time and host_dc_offset before a frame carries a DC time.
    static constexpr int64_t NO_DC_FRAME = std::numeric_limits<int64_t>::min();

    /// @brief the DC registers of one slave that runs in a DC mode.
    struct DCSetup {
        uint16_t position = 0;
        dc::Registers registers;
    };

    /// @brief the state of one bus run, from activate() to deactivate().
    struct Session {
        /// @brief the plan that activate() got.
        master::Plan plan;
        /// @brief input size in bytes (TxPDO, slave->master).
        size_t input_sz = 0;
        /// @brief output size in bytes (RxPDO, master->slave).
        size_t output_sz = 0;
        /// @brief the working counter of a frame that every enabled slave handled.
        int expected_wkc = 0;
        /// @brief whether the enabled slaves' frames carry the reference clock time.
        bool has_dc = false;
        /// @brief the DC setup of each enabled slave in a DC mode.
        std::vector<DCSetup> dc_setups;
        /// @brief why each slave rejected a startup parameter at its latest move to
        /// SAFE-OP, keyed by position.
        std::map<uint16_t, std::string> startup_failures;
        /// @brief why the master could not restore the distributed clock of a slave
        /// that came back, keyed by position.
        std::unordered_map<uint16_t, std::string> dc_restore_errors;
        /// @brief the latest DC time that moved forward, in nanoseconds.
        int64_t last_dc_time = NO_DC_FRAME;
        /// @brief the Config::clock time of the last process data send.
        std::chrono::steady_clock::time_point sent_at;
        /// @brief the DC time minus the Config::clock time, in nanoseconds, at the send
        /// of the frame that gave last_dc_time.
        std::atomic<int64_t> host_dc_offset{NO_DC_FRAME};
        /// @brief whether the cycle holds process data back. A frame then carries only
        /// the reference clock time.
        std::atomic<bool> process_data_held{false};
        /// @brief what receive returns after a send that held process data back, or NIL
        /// after a send that did not. Only the cycle uses it.
        x::errors::Error held_err;
        /// @brief what shortfall() returns. Only the cycle uses it.
        std::optional<uint32_t> shortfall;
    };
    /// @brief the current bus run, or nullopt between deactivate() and activate().
    std::optional<Session> session;
    /// @brief keeps the mailbox exchanges of check(), write(), and read() apart, and
    /// guards the startup parameters in the session plan while the bus runs.
    std::mutex mailbox_mu;

public:
    /// @brief constructs a SOEM master with injected API, interface name, and config.
    Master(std::unique_ptr<API> api, std::string interface_name, Config cfg = {});

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    /// @brief returns the name that Windows shows for the adapter, or the interface
    /// name when there is none.
    [[nodiscard]] std::string name() const override;

    std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> scan() override;

    std::pair<master::Change, x::errors::Error> changed() override;

    [[nodiscard]] bool restarts_visible() const override { return true; }

    master::Activation activate(
        const master::Plan &plan,
        const std::function<void(const master::Image &)> &start_cycle
    ) override;

    void deactivate() override;

    x::errors::Error receive() override;

    x::errors::Error send() override;

    /// @brief shares only atomics with receive() and send().
    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override;

    x::errors::Error write(
        uint16_t position,
        const std::vector<slave::StartupParameter> &parameters
    ) override;

    std::pair<uint32_t, x::errors::Error>
    read(uint16_t position, uint16_t index, uint8_t sub_index) override;

    std::optional<x::telem::TimeStamp> dc_time() const override;

    [[nodiscard]] std::optional<uint32_t> shortfall() const override;

private:
    /// @brief returns the record of the slave at pos in the SOEM slave table. Record 0
    /// describes the whole bus.
    [[nodiscard]] ec_slavet &record(const uint16_t pos) {
        return this->api->context.slavelist[pos];
    }

    [[nodiscard]] const ec_slavet &record(const uint16_t pos) const {
        return this->api->context.slavelist[pos];
    }

    /// @brief returns the name of the slave at pos as a NUL-terminated string.
    [[nodiscard]] const char *slave_name(const uint16_t pos) const {
        return this->record(pos).name;
    }

    /// @brief writes parameters to the slave at pos in order, returning
    /// STARTUP_PARAMETER_ERROR with why the slave rejected the first one that fails.
    x::errors::Error write_parameters(
        uint16_t pos,
        const std::vector<slave::StartupParameter> &parameters
    );

    /// @brief writes a slave's startup parameters in order, recording why the slave
    /// rejected the first one that fails. Returns true when the slave took them all.
    bool write_startup_parameters(
        uint16_t pos,
        const std::vector<slave::StartupParameter> &parameters
    );

    /// @brief populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// @brief reads the settings that the ESI catalog describes for the slave from it,
    /// or the reason the read failed into its status.
    void read_settings(slave::DiscoveryResult &slave);

    /// @brief reads the identity and PDO layout of the slave at a position.
    slave::DiscoveryResult discover_slave(uint16_t pos);

    /// @brief returns the settings that the session plan gives the slave at pos.
    [[nodiscard]] slave::Config config(uint16_t pos) const;

    /// @brief runs activate() in the new session.
    master::Activation
    start(const std::function<void(const master::Image &)> &start_cycle);

    /// @brief returns the process data of the mapped enabled slaves.
    master::Image image();

    /// @brief discovers PDO entries for a slave and populates its PDO lists, reading
    /// names and data types that description does not hold yet.
    void discover_slave_pdos(slave::DiscoveryResult &slave, Description &description);

    /// @brief reads an unsigned value of at most four bytes from the object dictionary
    /// of the slave at pos.
    std::pair<uint32_t, x::errors::Error>
    read_sdo(uint16_t pos, uint16_t index, uint8_t sub_index);

    /// @brief reads a whole object from the object dictionary of the slave at pos with
    /// complete access. Returns its bytes from sub-index 0, or nullopt when the read
    /// fails.
    std::optional<std::vector<uint8_t>> read_object(uint16_t pos, uint16_t index);

    /// @brief returns the model of the slave at pos, or nullopt when the slave does not
    /// give the modules it detects.
    std::optional<Model> read_model(uint16_t pos);

    /// @brief reads an SII category of the slave at pos, without its length word.
    /// Returns no bytes when the SII has no such category.
    std::vector<uint8_t> read_sii_category(uint16_t pos, uint16_t category);

    /// @brief returns the strings of the SII of the slave at pos.
    sii::Strings sii_strings(uint16_t pos);

    /// @brief discovers PDOs from the SII categories of the slave.
    void discover_pdos_sii(slave::DiscoveryResult &slave);

    /// @brief returns the input or output PDOs that the SII of the slave at pos lists
    /// on the sync managers that keep accepts, then gives the EEPROM back to the slave.
    std::vector<pdo::PDO> read_sii_pdos(
        uint16_t pos,
        bool is_input,
        const std::function<bool(uint8_t sync)> &keep
    );

    /// @brief returns the synchronization modes that the SII of the slave at the given
    /// position lists, from description when it holds them. Returns an empty list when
    /// the SII has no DC category.
    std::vector<dc::OpMode> read_op_modes(uint16_t position, Description &description);

    /// @brief returns the name of a CoE object, from description when it holds it, or
    /// nullopt when the slave has none.
    std::optional<std::string>
    read_object_name(uint16_t slave_pos, uint16_t index, Description &description);

    /// @brief returns the name and CoE data type of a PDO entry from the object
    /// dictionary, from description when it holds them.
    std::pair<std::string, telem::DataType> read_object_entry(
        uint16_t slave_pos,
        uint16_t index,
        uint8_t sub_index,
        Description &description
    );

    /// @brief writes value to the DC register of a slave at address. The register is
    /// as wide as T.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take the write.
    template<std::integral T>
    x::errors::Error write_dc_register(uint16_t pos, uint16_t address, T value);

    /// @brief reads the DC register of a slave at address. The register is as wide as
    /// T.
    /// @returns errors::ACTIVATION_ERROR when the slave does not answer.
    template<std::integral T>
    std::pair<T, x::errors::Error> read_dc_register(uint16_t pos, uint16_t address);

    /// @brief turns SYNC off on a slave: writes zero to 0x0980, then to 0x0981.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take a write.
    x::errors::Error clear_sync(uint16_t pos);

    /// @brief writes zero to the DC activation registers (0x0980 and 0x0981) of every
    /// enabled slave with a distributed clock, also after a slave fails a write.
    /// @returns errors::ACTIVATION_ERROR that names each slave that failed a write.
    x::errors::Error clear_dc_activation();

    /// @brief returns the DC registers of each enabled slave that runs a DC mode, the
    /// chosen mode or the one dc::select picks, at a bus cycle of period.
    /// @returns errors::SLAVE_CONFIG_ERROR when a slave cannot use its chosen mode,
    /// when a mode gives values the registers cannot hold, or when the SYNC0 cycles
    /// that mark process data share no dc::MIN_EVENT_CYCLE grid or longer with the
    /// period.
    std::pair<std::vector<DCSetup>, x::errors::Error> resolve_dc() const;

    /// @brief waits until a frame carries the reference clock to the slaves, then until
    /// the clock of each slave at positions is within 10 µs of the reference clock.
    /// @param stopped returns true when the wait must end early.
    /// @returns errors::ACTIVATION_ERROR when no frame comes back, a slave does not
    /// answer, a clock does not settle within Config::dc_settle_timeout, or stopped
    /// returns true first.
    x::errors::Error wait_for_dc_settle(
        const std::vector<uint16_t> &positions,
        const std::function<bool()> &stopped
    );

    /// @brief starts SYNC0 on a slave, with its first SYNC0 event on a whole SYNC0
    /// cycle of DC time at least 100 ms after its system time.
    /// @returns errors::ACTIVATION_ERROR when the slave does not take a write.
    x::errors::Error start_sync(const DCSetup &setup);

    /// @brief starts SYNC0 on each slave in dc_setups once their clocks settle. Call it
    /// while the bus cycles and before SAFE-OP.
    /// @returns errors::ACTIVATION_ERROR when a slave does not take a write.
    x::errors::Error start_dc();

    /// @brief sets the DC system time offset (0x0920) of a slave so that its clock
    /// matches the clock of another enabled DC slave in SAFE-OP or OP whose latest
    /// restore did not fail. Without one, the clock continues the latest DC time that
    /// moved forward.
    /// @returns errors::ACTIVATION_ERROR when no frame has carried the DC time, a slave
    /// does not answer a read, or the slave does not take the write.
    x::errors::Error align_clock(uint16_t pos);

    /// @brief sets up the distributed clock of a slave that came back to PRE-OP while
    /// the bus cycles: its propagation delay, its offset, and SYNC0 when it runs in a
    /// DC mode. Blocks until its clock settles or breaker stops.
    /// @returns errors::ACTIVATION_ERROR when a step fails or breaker stops first.
    x::errors::Error restore_dc(uint16_t pos, const x::breaker::Breaker &breaker);

    /// @brief requests the state from all enabled slaves, then waits until they reach
    /// it, one of them refuses it, or the timeout in microseconds passes.
    master::Activation request_state(uint16_t state, int timeout);

    /// @brief waits until a frame that the cycle sent after the call comes back.
    /// @returns false when the deadline passes or stopped returns true first.
    [[nodiscard]] bool wait_for_frame(
        std::chrono::steady_clock::time_point deadline,
        const std::function<bool()> &stopped
    ) const;

    /// @brief whether the latest clock restore of the slave failed.
    [[nodiscard]] bool restore_failed(uint16_t pos) const;

    /// @brief whether the slave has its AL error flag set and gives an AL status code.
    [[nodiscard]] bool al_error(uint16_t pos) const;

    /// @brief whether the slave refused the state: it stays below the state with an AL
    /// error. A slave above the state can still show an error from before a request to
    /// go down.
    [[nodiscard]] bool refuses(uint16_t pos, uint16_t state) const;

    /// @brief marks a slave lost when it is in EC_STATE_NONE after a second read, and
    /// clears the mark when it answers. A state read that gets no reply sets every
    /// slave to EC_STATE_NONE.
    void update_lost(uint16_t pos);

    /// @brief takes one recovery step for a slave that is not in OP, following the SOEM
    /// ecatcheck sample: acknowledge an error, request OP, reconfigure, or restore a
    /// slave that lost power. A slave in SAFE-OP whose clock restore failed goes back
    /// to PRE-OP, not to OP. Takes no step for a slave that rejected a startup
    /// parameter, so it stays in PRE-OP until the next activation. A wait for the
    /// clock of a DC slave ends once breaker stops.
    void recover(uint16_t pos, const x::breaker::Breaker &breaker);

    /// @brief builds the outcome for enabled slaves that did not reach a state. It is a
    /// refusal when a slave refuses the state with an AL status code that a new start
    /// cannot clear.
    master::Activation state_error(const std::vector<uint16_t> &failed, uint16_t state);
};

/// @brief SOEM-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override;

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override;
};

}
