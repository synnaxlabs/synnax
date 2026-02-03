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
#include <span>
#include <thread>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xthread/rt.h"

#include "driver/ethercat/master/master.h"

namespace ethercat::engine {
/// @brief Configuration for the Loop.
struct Config {
    /// @brief maximum allowed cycle overrun before logging a warning.
    telem::TimeSpan max_overrun = telem::TimeSpan(0);
    /// @brief real-time thread configuration for the cycle thread.
    xthread::RTConfig rt;

    Config() = default;
};

/// @brief coordinates cyclic PDO exchange between EtherCAT master and tasks.
class Engine {
    Config config;

    struct Registration {
        size_t id;
        std::vector<pdo::Entry> entries;
        std::vector<pdo::Offset> offsets;
        telem::Rate rate;
    };

    std::atomic<size_t> next_id = 0;

    std::atomic<int64_t> cycle_time_ns{telem::MILLISECOND.nanoseconds()};

    struct alignas(64) {
        std::atomic<uint64_t> seq = 0;
    } read_seq;

    struct alignas(64) {
        std::atomic<uint64_t> epoch = 0;
    } read_epoch;

    alignas(64) std::vector<uint8_t> shared_input_buffer;

    mutable std::mutex notify_mu;
    std::condition_variable read_cv;

    mutable std::mutex registration_mu;
    std::vector<Registration> read_registrations;

    mutable std::mutex write_mu;
    std::vector<uint8_t> write_staging;
    std::vector<uint8_t> write_active;
    std::vector<Registration> write_registrations;

    std::thread run_thread;
    std::atomic<bool> restarting{false};
    breaker::Breaker breaker;

    void run();
    void stop();
    [[nodiscard]] xerrors::Error reconfigure();
    [[nodiscard]] bool should_be_running() const;
    void update_cycle_time();

    void publish_inputs(std::span<const uint8_t> src);
    const uint8_t *consume_outputs(size_t &out_len);
    void update_read_offsets_locked();
    void update_read_offsets();
    void update_write_offsets_locked(size_t total_size);
    void update_write_offsets(size_t total_size);
    void unregister_reader(size_t id);
    void unregister_writer(size_t id);

    std::shared_ptr<master::Master> master;
    mutable std::mutex master_init_mu;

public:
    /// @brief resolved PDO entry with offset and type information.
    struct ResolvedPDO {
        pdo::Offset offset;
        telem::DataType data_type;
        uint8_t bit_length;
    };

    /// @brief proxy for reading input data from the EtherCAT cycle engine.
    class Reader {
        Engine &engine;
        size_t id;
        size_t total_size;
        std::vector<ResolvedPDO> pdos;
        mutable std::vector<uint8_t> private_buffer;
        mutable uint64_t last_seen_epoch = 0;

    public:
        Reader(
            Engine &eng,
            size_t id,
            size_t total_size,
            std::vector<ResolvedPDO> pdos,
            size_t input_frame_size
        );

        Reader(const Reader &) = delete;
        Reader &operator=(const Reader &) = delete;

        ~Reader();

        /// @brief blocks until new input data is available, then writes to the frame.
        [[nodiscard]] xerrors::Error
        read(const breaker::Breaker &brk, const telem::Frame &frame) const;

        /// @brief blocks until the next PDO exchange epoch without extracting data.
        [[nodiscard]] xerrors::Error wait(const breaker::Breaker &brk) const;

        /// @brief returns the total size in bytes of all registered PDO entries.
        [[nodiscard]] size_t size() const { return this->total_size; }
    };

    /// @brief proxy for writing output data to the EtherCAT cycle engine.
    class Writer {
        Engine &engine;
        size_t id;
        std::vector<ResolvedPDO> pdos;

    public:
        /// @brief RAII batch writer that holds the write lock for multiple writes.
        class Transaction {
            Engine &engine;
            const std::vector<ResolvedPDO> &pdos;
            std::unique_lock<std::mutex> lock;

        public:
            Transaction(Engine &eng, const std::vector<ResolvedPDO> &pdos);
            Transaction(const Transaction &) = delete;
            Transaction &operator=(const Transaction &) = delete;
            Transaction(Transaction &&) = delete;
            Transaction &operator=(Transaction &&) = delete;

            /// @brief writes a value to a specific PDO entry by index.
            void write(size_t pdo_index, const telem::SampleValue &value) const;
        };

        Writer(Engine &eng, size_t id, std::vector<ResolvedPDO> pdos);
        ~Writer();

        Writer(const Writer &) = delete;
        Writer &operator=(const Writer &) = delete;

        /// @brief creates a transaction for writing multiple PDO entries under a lock.
        [[nodiscard]] Transaction open_tx() const;

        /// @brief writes a value to a specific PDO entry by index.
        void write(size_t pdo_index, const telem::SampleValue &value) const;
    };

    /// @brief constructs an Engine with the given master and configuration.
    explicit Engine(std::shared_ptr<master::Master> master, const Config &config);

    /// @brief constructs an Engine with the given master using default configuration.
    explicit Engine(std::shared_ptr<master::Master> master);

    ~Engine();

    Engine(const Engine &) = delete;
    Engine &operator=(const Engine &) = delete;

    /// @brief opens a new Reader for the specified PDO entries.
    [[nodiscard]] std::pair<std::unique_ptr<Reader>, xerrors::Error>
    open_reader(const std::vector<pdo::Entry> &entries, telem::Rate sample_rate);

    /// @brief opens a new Writer for the specified PDO entries.
    [[nodiscard]] std::pair<std::unique_ptr<Writer>, xerrors::Error>
    open_writer(const std::vector<pdo::Entry> &entries, telem::Rate execution_rate);

    /// @brief returns true if the engine is running.
    bool running() const { return this->breaker.running(); }

    /// @brief returns the engine configuration.
    [[nodiscard]] const Config &cfg() const { return this->config; }

    /// @brief returns the current engine cycle rate (thread-safe).
    [[nodiscard]] telem::Rate cycle_rate() const;

    /// @brief initializes the master (thread-safe, idempotent).
    [[nodiscard]] xerrors::Error ensure_initialized() const;

    /// @brief returns discovered slaves.
    [[nodiscard]] std::vector<slave::Properties> slaves() const;

    /// @brief returns the interface name.
    [[nodiscard]] std::string interface_name() const;

    /// @brief sets whether a slave is enabled for cyclic exchange.
    void set_slave_enabled(uint16_t position, bool enabled);
};
}
