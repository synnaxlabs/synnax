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
#include <condition_variable>
#include <memory>
#include <mutex>
#include <thread>
#include <unordered_map>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/master/domain.h"
#include "driver/ethercat/master/master.h"

namespace ethercat {
/// Stable handle for PDO access (index never changes across restarts).
struct PDOHandle {
    size_t index;
    bool is_input;

    [[nodiscard]] bool valid() const { return index != SIZE_MAX; }
    static PDOHandle invalid() { return {SIZE_MAX, false}; }
};
/// Configuration for the CyclicEngine.
struct CyclicEngineConfig {
    /// The cycle time for PDO exchange. Defaults to 1ms (1kHz).
    telem::TimeSpan cycle_time = telem::MILLISECOND;

    /// Maximum allowed cycle overrun before warning. Defaults to 10% of cycle_time.
    telem::TimeSpan max_overrun = telem::TimeSpan(0);

    /// Enable real-time thread scheduling (SCHED_FIFO on Linux).
    /// Requires CAP_SYS_NICE capability or root privileges.
    bool enable_realtime = false;

    /// Real-time thread priority (1-99, higher = more priority).
    /// Only used if enable_realtime is true.
    int realtime_priority = 80;

    CyclicEngineConfig() = default;

    explicit CyclicEngineConfig(const telem::TimeSpan cycle_time):
        cycle_time(cycle_time), max_overrun(cycle_time * 0.1) {}
};

/// Coordinates cyclic PDO exchange between EtherCAT master and Synnax tasks.
///
/// The CyclicEngine manages a dedicated thread that performs the EtherCAT cyclic
/// exchange at a fixed rate. It provides thread-safe access to input and output
/// data for read and write tasks.
///
/// Key features:
/// - Reference counting for automatic activation/deactivation
/// - Thread-safe input snapshot for readers
/// - Thread-safe output staging for writers
/// - Condition variable signaling for synchronized reads
///
/// Thread model:
/// ```
/// Cycle Thread (runs at cycle_time):
///   master->receive()
///   master->process(domain)
///   {lock} copy domain inputs to input_snapshot_
///   input_cv_.notify_all()  // Wake waiting readers
///   {lock} copy output_buffer_ to domain outputs
///   master->queue(domain)
///   master->send()
/// ```
class CyclicEngine {
    std::shared_ptr<Master> master;
    std::unique_ptr<Domain> domain;
    CyclicEngineConfig config;

    std::thread cycle_thread;
    std::atomic<bool> running;
    std::atomic<int> task_count;

    mutable std::mutex input_mu;
    std::condition_variable input_cv;
    std::vector<uint8_t> input_snapshot;
    uint64_t input_cycle_count;

    mutable std::mutex output_mu;
    std::vector<uint8_t> output_buffer;

    struct PDORegistration {
        PDOEntry entry;
        size_t relative_offset; // Offset within the slave's input/output region
        size_t actual_offset; // Actual offset in IOmap (set after activation)
    };

    mutable std::mutex registration_mu;
    std::vector<PDORegistration> input_pdos;
    std::vector<PDORegistration> output_pdos;

    // Track cumulative offsets per slave during registration
    std::unordered_map<uint16_t, size_t> slave_input_offsets;
    std::unordered_map<uint16_t, size_t> slave_output_offsets;

    xerrors::Error last_err;

    /// Flag indicating engine is restarting for reconfiguration.
    std::atomic<bool> restarting{false};

    /// Breaker for retry logic during restart.
    breaker::Breaker restart_breaker;

    void cycle_loop();
    void resolve_pdo_offsets();

    /// Internal restart when PDO registered while running.
    xerrors::Error restart_for_reconfiguration();

public:
    /// Constructs a CyclicEngine with the given master and configuration.
    /// @param master The EtherCAT master to use for cyclic exchange.
    /// @param config Configuration for cycle timing and error handling.
    explicit CyclicEngine(
        std::shared_ptr<Master> master,
        CyclicEngineConfig config = CyclicEngineConfig()
    );

    ~CyclicEngine();

    CyclicEngine(const CyclicEngine &) = delete;
    CyclicEngine &operator=(const CyclicEngine &) = delete;

    /// Registers an input PDO (TxPDO, slave→master) for reading.
    ///
    /// Can be called while running - engine will automatically restart to
    /// reconfigure the IOmap.
    ///
    /// @param entry PDO entry describing the slave position, index, subindex,
    ///              and bit length.
    /// @returns A pair containing:
    ///          - PDOHandle: Stable handle for accessing the PDO data.
    ///          - xerrors::Error: PDO_MAPPING_ERROR if registration fails.
    [[nodiscard]] std::pair<PDOHandle, xerrors::Error>
    register_input_pdo(const PDOEntry &entry);

    /// Registers an output PDO (RxPDO, master→slave) for writing.
    ///
    /// Can be called while running - engine will automatically restart to
    /// reconfigure the IOmap.
    ///
    /// @param entry PDO entry describing the slave position, index, subindex,
    ///              and bit length.
    /// @returns A pair containing:
    ///          - PDOHandle: Stable handle for accessing the PDO data.
    ///          - xerrors::Error: PDO_MAPPING_ERROR if registration fails.
    [[nodiscard]] std::pair<PDOHandle, xerrors::Error>
    register_output_pdo(const PDOEntry &entry);

    /// Adds a task to the engine, starting cyclic exchange if this is the first task.
    ///
    /// The engine uses reference counting: the first add_task() initializes the
    /// master and starts the cycle thread, subsequent calls increment the count.
    ///
    /// @returns xerrors::NIL on success, or an error if master activation fails.
    [[nodiscard]] xerrors::Error add_task();

    /// Removes a task from the engine, stopping cyclic exchange when the last task
    /// exits.
    ///
    /// When the task count reaches zero, the cycle thread is stopped and the
    /// master is deactivated.
    void remove_task();

    /// Blocks until new input data is available from the cyclic exchange.
    ///
    /// This method waits for the next cycle to complete, then copies the input
    /// snapshot to the provided buffer. The buffer must be sized to hold all
    /// registered input PDOs.
    ///
    /// @param buffer Destination buffer for input data.
    /// @param breaker Breaker to check for cancellation while waiting.
    /// @returns xerrors::NIL on success, or an error if the engine is stopped.
    [[nodiscard]] xerrors::Error
    wait_for_inputs(std::vector<uint8_t> &buffer, std::atomic<bool> &breaker);

    /// Writes output data to be sent in the next cyclic exchange.
    ///
    /// The data is staged in an internal buffer and will be copied to the
    /// domain buffer during the next cycle.
    ///
    /// @param offset Byte offset into the output buffer.
    /// @param data Pointer to the data to write.
    /// @param length Number of bytes to write.
    void write_output(size_t offset, const void *data, size_t length);

    /// Reads input data via handle (translates to actual offset internally).
    ///
    /// @param handle The PDOHandle returned from register_input_pdo().
    /// @param buffer Destination buffer for the data.
    /// @param length Number of bytes to read.
    /// @returns xerrors::NIL on success, or PDO_MAPPING_ERROR if handle is invalid.
    [[nodiscard]] xerrors::Error
    read_input(PDOHandle handle, void *buffer, size_t length);

    /// Writes output data via handle (translates to actual offset internally).
    ///
    /// @param handle The PDOHandle returned from register_output_pdo().
    /// @param data Pointer to the data to write.
    /// @param length Number of bytes to write.
    void write_output(PDOHandle handle, const void *data, size_t length);

    /// Returns the current cycle count for diagnostics.
    [[nodiscard]] uint64_t cycle_count() const;

    /// Returns the last error that occurred during cyclic exchange.
    [[nodiscard]] xerrors::Error last_error() const;

    /// Returns whether the cyclic engine is currently running.
    [[nodiscard]] bool is_running() const { return this->running.load(); }

    /// Returns the configured cycle time.
    [[nodiscard]] telem::TimeSpan cycle_time() const { return this->config.cycle_time; }

    /// Returns the number of currently registered tasks.
    [[nodiscard]] int get_task_count() const { return this->task_count.load(); }

    /// Returns information about all slaves on the network.
    [[nodiscard]] std::vector<SlaveInfo> slaves() const {
        return this->master->slaves();
    }

    /// Returns the actual input offset for a registration index.
    ///
    /// Must be called after add_task() succeeds. The registration index is
    /// the order in which register_input_pdo() was called (0-based).
    ///
    /// @param registration_index The index of the registered PDO.
    /// @returns The actual byte offset in the input buffer.
    [[nodiscard]] size_t get_actual_input_offset(size_t registration_index) const;

    /// Returns the actual output offset for a registration index.
    ///
    /// Must be called after add_task() succeeds.
    ///
    /// @param registration_index The index of the registered PDO.
    /// @returns The actual byte offset in the output buffer.
    [[nodiscard]] size_t get_actual_output_offset(size_t registration_index) const;
};
}
