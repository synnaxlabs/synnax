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

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/master/domain.h"
#include "driver/ethercat/master/master.h"

namespace ethercat {
/// Configuration for the CyclicEngine.
struct CyclicEngineConfig {
    /// The cycle time for PDO exchange. Defaults to 1ms (1kHz).
    telem::TimeSpan cycle_time = telem::MILLISECOND;

    /// Maximum allowed cycle overrun before warning. Defaults to 10% of cycle_time.
    telem::TimeSpan max_overrun = telem::TimeSpan(0);

    CyclicEngineConfig() = default;

    explicit CyclicEngineConfig(const telem::TimeSpan cycle_time):
        cycle_time(cycle_time),
        max_overrun(cycle_time * 0.1) {}
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
    std::shared_ptr<Master> master_;
    std::unique_ptr<Domain> domain_;
    CyclicEngineConfig config_;

    std::thread cycle_thread_;
    std::atomic<bool> running_;
    std::atomic<int> task_count_;

    mutable std::mutex input_mutex_;
    std::condition_variable input_cv_;
    std::vector<uint8_t> input_snapshot_;
    uint64_t input_cycle_count_;

    mutable std::mutex output_mutex_;
    std::vector<uint8_t> output_buffer_;

    struct PDORegistration {
        PDOEntry entry;
        size_t relative_offset;  // Offset within the slave's input/output region
        size_t actual_offset;    // Actual offset in IOmap (set after activation)
    };

    mutable std::mutex registration_mutex_;
    std::vector<PDORegistration> input_pdos_;
    std::vector<PDORegistration> output_pdos_;

    // Track cumulative offsets per slave during registration
    std::unordered_map<uint16_t, size_t> slave_input_offsets_;
    std::unordered_map<uint16_t, size_t> slave_output_offsets_;

    xerrors::Error last_error_;

    void cycle_loop();
    void resolve_pdo_offsets();

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
    /// Must be called before any task is added. The returned offset indicates
    /// where in the input snapshot this PDO's data will be located.
    ///
    /// @param entry PDO entry describing the slave position, index, subindex,
    ///              and bit length.
    /// @returns A pair containing:
    ///          - size_t: Byte offset into the input snapshot buffer.
    ///          - xerrors::Error: PDO_MAPPING_ERROR if registration fails.
    [[nodiscard]] std::pair<size_t, xerrors::Error> register_input_pdo(
        const PDOEntry &entry
    );

    /// Registers an output PDO (RxPDO, master→slave) for writing.
    ///
    /// Must be called before any task is added. The returned offset indicates
    /// where in the output buffer this PDO's data should be written.
    ///
    /// @param entry PDO entry describing the slave position, index, subindex,
    ///              and bit length.
    /// @returns A pair containing:
    ///          - size_t: Byte offset into the output buffer.
    ///          - xerrors::Error: PDO_MAPPING_ERROR if registration fails.
    [[nodiscard]] std::pair<size_t, xerrors::Error> register_output_pdo(
        const PDOEntry &entry
    );

    /// Adds a task to the engine, starting cyclic exchange if this is the first task.
    ///
    /// The engine uses reference counting: the first add_task() initializes the
    /// master and starts the cycle thread, subsequent calls increment the count.
    ///
    /// @returns xerrors::NIL on success, or an error if master activation fails.
    [[nodiscard]] xerrors::Error add_task();

    /// Removes a task from the engine, stopping cyclic exchange when the last task exits.
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

    /// Returns the current cycle count for diagnostics.
    [[nodiscard]] uint64_t cycle_count() const;

    /// Returns the last error that occurred during cyclic exchange.
    [[nodiscard]] xerrors::Error last_error() const;

    /// Returns whether the cyclic engine is currently running.
    [[nodiscard]] bool running() const { return running_.load(); }

    /// Returns the configured cycle time.
    [[nodiscard]] telem::TimeSpan cycle_time() const { return config_.cycle_time; }

    /// Returns the number of currently registered tasks.
    [[nodiscard]] int task_count() const { return task_count_.load(); }

    /// Returns information about all slaves on the network.
    [[nodiscard]] std::vector<SlaveInfo> slaves() const { return master_->slaves(); }

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
