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

namespace ethercat {
/// @brief Configuration for the Loop.
struct LoopConfig {
    /// Cycle time for PDO exchange. Defaults to 1ms (1kHz).
    telem::TimeSpan cycle_time = telem::MILLISECOND;

    /// Maximum allowed cycle overrun before logging a warning.
    telem::TimeSpan max_overrun = telem::TimeSpan(0);

    /// Real-time thread configuration for the cycle thread.
    xthread::RTConfig rt;

    LoopConfig() = default;

    explicit LoopConfig(const telem::TimeSpan cycle_time):
        cycle_time(cycle_time), max_overrun(cycle_time * 0.1) {}
};

/// @brief Coordinates cyclic PDO exchange between EtherCAT master and tasks.
///
/// The Loop manages a dedicated thread that performs EtherCAT cyclic exchange at a
/// fixed rate. It owns ReadBuffer and WriteBuffer which provide access to I/O data
/// through Reader and Writer proxies.
///
/// Lifecycle:
/// 1. Create Loop with Master and config
/// 2. Open Readers/Writers via open_reader()/open_writer()
/// 3. Loop automatically starts when first Reader or Writer is opened
/// 4. Loop automatically stops when last Reader and Writer are closed
class Loop {
public:
    class ReadBuffer;
    class WriteBuffer;

    /// @brief Proxy for reading input data from the EtherCAT cycle loop.
    ///
    /// Each Reader receives its registered PDO data laid out contiguously in
    /// registration order. Multiple Readers can exist simultaneously. The Reader is
    /// automatically unregistered when destroyed.
    class Reader {
        friend class ReadBuffer;

        ReadBuffer *buffer;
        size_t id;
        size_t total_size;

        Reader(ReadBuffer *buf, size_t id, size_t total_size);

    public:
        ~Reader();

        Reader(const Reader &) = delete;
        Reader &operator=(const Reader &) = delete;
        Reader(Reader &&other) noexcept;
        Reader &operator=(Reader &&other) noexcept;

        /// @brief Blocks until new input data is available, then copies it to the
        /// buffer.
        /// @param dest Destination buffer. Must be at least size() bytes.
        /// @param stopped Atomic flag checked for early termination.
        /// @return xerrors::NIL on success, or error if stopped or loop is not running.
        [[nodiscard]] xerrors::Error
        read(std::vector<uint8_t> &dest, const std::atomic<bool> &stopped) const;

        /// @brief Returns the total size in bytes of all registered PDO entries.
        [[nodiscard]] size_t size() const { return this->total_size; }
    };

    /// @brief Proxy for writing output data to the EtherCAT cycle loop.
    ///
    /// Each Writer writes to its registered PDO entries in registration order.
    /// Multiple Writers can exist simultaneously. The Writer is automatically
    /// unregistered when destroyed.
    class Writer {
        friend class WriteBuffer;

        WriteBuffer *buffer;
        size_t id;
        std::vector<size_t> offsets;
        std::vector<size_t> lengths;

        Writer(
            WriteBuffer *buf,
            size_t id,
            std::vector<size_t> offsets,
            std::vector<size_t> lengths
        );

    public:
        ~Writer();

        Writer(const Writer &) = delete;
        Writer &operator=(const Writer &) = delete;
        Writer(Writer &&other) noexcept;
        Writer &operator=(Writer &&other) noexcept;

        /// @brief Writes data to a specific PDO entry by index.
        /// @param pdo_index Index into the PDO entries registered with this Writer.
        /// @param data Pointer to the data to write.
        /// @param length Number of bytes to write.
        void write(size_t pdo_index, const void *data, size_t length);
    };

    /// @brief Manages the input data buffer and Reader registration.
    class ReadBuffer {
        friend class Loop;
        friend class Reader;

        Loop &loop;
        mutable std::mutex mu;
        std::condition_variable cv;
        std::vector<uint8_t> data;
        std::atomic<uint64_t> epoch{0};
        std::atomic<bool> running{false};
        std::atomic<bool> restarting{false};

        struct Registration {
            size_t id;
            std::vector<PDOEntry> entries;
            std::vector<size_t> offsets;
        };
        std::vector<Registration> registrations;
        size_t next_id{0};

        explicit ReadBuffer(Loop &lp);

        void publish(const uint8_t *src, size_t len);
        void set_running(bool r);
        void set_restarting(bool r);
        void update_offsets();
        void unregister(size_t id);
        [[nodiscard]] xerrors::Error request_reconfiguration();
        [[nodiscard]] std::pair<Reader, xerrors::Error>
        open_reader(std::vector<PDOEntry> entries);
        [[nodiscard]] size_t reader_count() const;
        [[nodiscard]] std::vector<PDOEntry> all_entries() const;
        [[nodiscard]] uint64_t current_epoch() const { return this->epoch.load(); }
    };

    /// @brief Manages the output data buffer and Writer registration.
    class WriteBuffer {
        friend class Loop;
        friend class Writer;

        Loop &loop;
        mutable std::mutex mu;
        std::vector<uint8_t> staging;
        std::vector<uint8_t> active;

        struct Registration {
            size_t id;
            std::vector<PDOEntry> entries;
            std::vector<size_t> offsets;
        };
        std::vector<Registration> registrations;
        size_t next_id{0};

        explicit WriteBuffer(Loop &lp);

        const uint8_t *consume(size_t &out_len);
        void update_offsets(size_t total_size);
        void unregister(size_t id);
        [[nodiscard]] xerrors::Error request_reconfiguration();
        [[nodiscard]] std::pair<Writer, xerrors::Error>
        open_writer(std::vector<PDOEntry> entries);
        [[nodiscard]] size_t writer_count() const;
    };

public:
    std::shared_ptr<Master> master;
    LoopConfig config;
    ReadBuffer read_buf;
    WriteBuffer write_buf;
    std::thread cycle_thread;
    breaker::Breaker breaker;

    void run();
    [[nodiscard]] xerrors::Error start();
    void stop();
    [[nodiscard]] xerrors::Error reconfigure();
    [[nodiscard]] bool should_be_running() const;

public:
    /// @brief Constructs a Loop with the given master and configuration.
    /// @param master The EtherCAT master for cyclic exchange.
    /// @param config Configuration for cycle timing and RT thread setup.
    explicit Loop(
        std::shared_ptr<Master> master,
        const LoopConfig &config = LoopConfig()
    );

    ~Loop();

    Loop(const Loop &) = delete;
    Loop &operator=(const Loop &) = delete;

    /// @brief Opens a new Reader for the specified PDO entries.
    [[nodiscard]] std::pair<Reader, xerrors::Error>
    open_reader(std::vector<PDOEntry> entries);

    /// @brief Opens a new Writer for the specified PDO entries.
    [[nodiscard]] std::pair<Writer, xerrors::Error>
    open_writer(std::vector<PDOEntry> entries);

    /// @brief Returns whether the loop is currently running.
    [[nodiscard]] bool is_running() const { return this->breaker.running(); }
};
}
