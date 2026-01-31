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
#include <functional>
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xthread/rt.h"

#include "driver/ethercat/master/master.h"

namespace ethercat::engine {
/// @brief Configuration for the Loop.
struct Config {
    /// Cycle time for PDO exchange. Defaults to 1ms (1kHz).
    telem::TimeSpan cycle_time = telem::MILLISECOND;
    /// Maximum allowed cycle overrun before logging a warning.
    telem::TimeSpan max_overrun = telem::TimeSpan(0);
    /// Real-time thread configuration for the cycle thread.
    xthread::RTConfig rt;

    Config() = default;

    explicit Config(const telem::TimeSpan cycle_time):
        cycle_time(cycle_time), max_overrun(cycle_time * 0.1) {}
};

/// @brief Coordinates cyclic PDO exchange between EtherCAT master and tasks.
///
/// The Engine manages a dedicated thread that performs EtherCAT cyclic exchange at a
/// fixed rate.
///
/// Lifecycle:
/// 1. Create Engine with Master and config
/// 2. Open Readers/Writers via open_reader()/open_writer()
/// 3. Engine automatically starts when first Reader or Writer is opened
/// 4. Engine automatically stops when last Reader and Writer are closed
class Engine {
    Config config;

    struct Registration {
        size_t id;
        std::vector<PDOEntry> entries;
        std::vector<size_t> offsets;
    };

    mutable std::mutex read_mu;
    std::condition_variable read_cv;
    std::vector<uint8_t> read_data;
    std::atomic<uint64_t> read_epoch{0};
    std::atomic<bool> restarting{false};
    std::vector<Registration> read_registrations;
    size_t read_next_id{0};

    mutable std::mutex write_mu;
    std::vector<uint8_t> write_staging;
    std::vector<uint8_t> write_active;
    std::vector<Registration> write_registrations;
    size_t write_next_id{0};

    std::thread run_thread;
    breaker::Breaker breaker;

public:
    void run();
    [[nodiscard]] xerrors::Error start();
    void stop();
    [[nodiscard]] xerrors::Error reconfigure();
    [[nodiscard]] bool should_be_running() const;

    void publish_inputs(const uint8_t *src, size_t len);
    const uint8_t *consume_outputs(size_t &out_len);
    void update_read_offsets();
    void update_write_offsets(size_t total_size);
    void unregister_reader(size_t id);
    void unregister_writer(size_t id);
    [[nodiscard]] size_t reader_count() const;
    [[nodiscard]] size_t writer_count() const;
    [[nodiscard]] xerrors::Error request_read_reconfiguration();
    [[nodiscard]] xerrors::Error request_write_reconfiguration();

public:
    /// @brief Proxy for reading input data from the EtherCAT cycle engine.
    ///
    /// Each Reader receives its registered PDO data laid out contiguously in
    /// registration order. Multiple Readers can exist simultaneously. The Reader is
    /// automatically unregistered when destroyed.
    class Reader {
        Engine &engine;
        size_t id;
        size_t total_size;
        std::vector<size_t> offsets;
        std::vector<size_t> lengths;

    public:
        Reader(
            Engine &eng,
            size_t id,
            size_t total_size,
            std::vector<size_t> offsets,
            std::vector<size_t> lengths
        );

        Reader(const Reader &) = delete;
        Reader &operator=(const Reader &) = delete;

        ~Reader();

        /// @brief Blocks until new input data is available, then writes one sample
        /// to each series in the frame.
        /// @param brk Breaker for cancellation.
        /// @param frame Frame with series in registration order. Each series must
        ///              have the correct data type for its corresponding PDO entry.
        /// @return xerrors::NIL on success, or error if stopped or engine is not
        /// running.
        [[nodiscard]] xerrors::Error
        read(const breaker::Breaker &brk, const telem::Frame &frame) const;

        /// @brief Returns the total size in bytes of all registered PDO entries.
        [[nodiscard]] size_t size() const { return this->total_size; }
    };

    /// @brief Proxy for writing output data to the EtherCAT cycle engine.
    ///
    /// Each Writer writes to its registered PDO entries in registration order.
    /// Multiple Writers can exist simultaneously. Must call close() before
    /// destruction to unregister from the engine.
    class Writer {
        Engine &engine;
        size_t id;
        std::vector<size_t> offsets;
        std::vector<size_t> lengths;

    public:
        Writer(
            Engine &eng,
            size_t id,
            std::vector<size_t> offsets,
            std::vector<size_t> lengths
        );
        ~Writer();

        /// Prevent copies - would cause double-unregister on close().
        Writer(const Writer &) = delete;
        Writer &operator=(const Writer &) = delete;

        /// @brief Writes data to a specific PDO entry by index.
        /// @param pdo_index Index into the PDO entries registered with this Writer.
        /// @param data Pointer to the data to write.
        /// @param length Number of bytes to write.
        void write(size_t pdo_index, const void *data, size_t length) const;
    };

    const std::shared_ptr<Master> master;

    /// @brief Constructs an Engine with the given master and configuration.
    /// @param master The EtherCAT master for cyclic exchange.
    /// @param config Configuration for cycle timing and RT thread setup.
    explicit Engine(std::shared_ptr<Master> master, const Config &config = Config());

    ~Engine();

    Engine(const Engine &) = delete;
    Engine &operator=(const Engine &) = delete;

    /// @brief Opens a new Reader for the specified PDO entries.
    [[nodiscard]] std::pair<std::unique_ptr<Reader>, xerrors::Error>
    open_reader(const std::vector<PDOEntry> &entries);

    /// @brief Opens a new Writer for the specified PDO entries.
    [[nodiscard]] std::pair<std::unique_ptr<Writer>, xerrors::Error>
    open_writer(const std::vector<PDOEntry> &entries);

    /// @brief Returns whether the engine is currently running.
    [[nodiscard]] bool is_running() const { return this->breaker.running(); }
};
}
