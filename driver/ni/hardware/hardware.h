// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"

#include "driver/common/read_task.h"
#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::hardware {
struct Hardware {
    virtual ~Hardware() = default;

    /// @brief starts the task.
    [[nodiscard]] virtual x::errors::Error start() = 0;

    /// @brief stops the task.
    [[nodiscard]] virtual x::errors::Error stop() = 0;
};

struct ReadResult : driver::task::common::ReadResult {
    int64 skew = 0;
};

/// @brief a thin shim on top of NI-DAQmx that allows us to use different read
/// interfaces for analog and digital tasks. It also allows us to mock the hardware
/// during testing.
template<typename T>
struct Reader : virtual Hardware {
    /// @brief reads data from the hardware.
    /// @param samples_per_channel the number of samples to read per channel.
    /// @param data the buffer to read data into.
    /// @return a pair containing the number of samples read and an error if one
    /// occurred.
    [[nodiscard]] virtual ReadResult
    read(size_t samples_per_channel, std::vector<T> &data) = 0;
};

/// @brief Writer interface for hardware that supports writing data.
/// @tparam T The data type to write (uint8_t for digital, double for analog)
template<typename T>
struct Writer : virtual Hardware {
    /// @brief writes data to the hardware.
    /// @param data vector of values to write to the hardware
    /// @return error if the write operation failed
    [[nodiscard]] virtual x::errors::Error write(const std::vector<T> &data) = 0;
};

namespace daqmx {
/// @brief Base DAQmx hardware implementation that manages task lifecycle
struct Base : virtual Hardware {
protected:
    /// @brief the handle for the task.
    TaskHandle task_handle;
    /// @brief the NI-DAQmx API.
    std::shared_ptr<ni::daqmx::SugaredAPI> dmx;
    /// @brief a flag to indicate if the task is running.
    std::atomic<bool> running = false;

    Base(TaskHandle task_handle, std::shared_ptr<ni::daqmx::SugaredAPI> dmx);
    ~Base() override;

public:
    /// @brief implements HardwareInterface to start the DAQmx task.
    x::errors::Error start() override;

    /// @brief implements the HardwareInterface to stop the DAQmx task.
    x::errors::Error stop() override;
};

/// @brief Implementation of digital output writing using DAQmx
struct DigitalWriter final : Base, Writer<uint8_t> {
    /// @brief Constructs a new digital writer
    /// @param dmx The DAQmx API interface
    /// @param task_handle Handle to the DAQmx task
    DigitalWriter(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    x::errors::Error write(const std::vector<uint8_t> &data) override;
};

/// @brief Implementation of analog output writing using DAQmx
struct AnalogWriter final : Base, Writer<double> {
    /// @brief Constructs a new analog writer
    /// @param dmx The DAQmx API interface
    /// @param task_handle Handle to the DAQmx task
    AnalogWriter(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    x::errors::Error write(const std::vector<double> &data) override;
};

/// @brief a hardware interface for digital tasks.
struct DigitalReader final : Base, Reader<uint8_t> {
    DigitalReader(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult
    read(size_t samples_per_channel, std::vector<unsigned char> &data) override;
};

/// @brief Base class for readers that track sample acquisition skew
template<typename T>
struct SkewTrackingReader : Base, Reader<T> {
protected:
    /// @brief the total number of samples requested by calls to read() from
    /// the user.
    size_t total_samples_requested = 0;
    /// @brief the total number of samples actually acquired from the hardware
    /// by DAQmx.
    uInt64 total_samples_acquired = 0;

    SkewTrackingReader(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ):
        Base(task_handle, dmx) {}

    /// @brief Updates the skew between requested and acquired samples
    /// @param n_requested Number of samples requested in this read
    /// @return The current skew (acquired - requested)
    int64 update_skew(const size_t &n_requested);

public:
    x::errors::Error start() override;
};

/// @brief a hardware interface for analog tasks.
struct AnalogReader final : SkewTrackingReader<double> {
    AnalogReader(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult read(size_t samples_per_channel, std::vector<double> &data) override;
};

/// @brief a hardware interface for counter input tasks.
struct CounterReader final : SkewTrackingReader<double> {
    CounterReader(
        const std::shared_ptr<ni::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult read(size_t samples_per_channel, std::vector<double> &data) override;
};
}

namespace mock {
/// @brief Base mock implementation for testing hardware interfaces
struct Base : virtual Hardware {
    /// @brief Errors to return from start() calls in sequence
    std::vector<x::errors::Error> start_errors;
    /// @brief Errors to return from stop() calls in sequence
    std::vector<x::errors::Error> stop_errors;
    /// @brief Number of times start() was called
    size_t start_call_count;
    /// @brief Number of times stop() was called
    size_t stop_call_count;

protected:
    explicit Base(
        const std::vector<x::errors::Error> &start_errors = {x::errors::NIL},
        const std::vector<x::errors::Error> &stop_errors = {x::errors::NIL}
    );

public:
    x::errors::Error start() override;
    x::errors::Error stop() override;
};

/// @brief Mock implementation of Reader interface for testing
/// @tparam T The data type to read (uint8_t for digital, double for analog)
template<typename T>
class Reader final : public Base, public hardware::Reader<T> {
public:
    /// @brief Predefined responses for read() calls
    std::vector<std::pair<std::vector<T>, x::errors::Error>> read_responses;
    /// @brief Number of times read() was called
    size_t read_call_count;

    /// @brief Constructs a new mock reader
    /// @param start_errors Sequence of errors to return from start()
    /// @param stop_errors Sequence of errors to return from stop()
    /// @param read_responses Sequence of data and errors to return from read()
    explicit Reader(
        const std::vector<x::errors::Error> &start_errors = {x::errors::NIL},
        const std::vector<x::errors::Error> &stop_errors = {x::errors::NIL},
        std::vector<std::pair<std::vector<T>, x::errors::Error>> read_responses = {
            {{0.5}, x::errors::NIL}
        }
    );

    ReadResult read(size_t samples_per_channel, std::vector<T> &data) override;
};

/// @brief Mock implementation of Writer interface for testing
/// @tparam T The data type to write (uint8_t for digital, double for analog)
template<typename T>
class Writer final : public Base, public hardware::Writer<T> {
public:
    /// @brief Errors to return from write() calls in sequence
    std::vector<x::errors::Error> write_responses;
    /// @brief Number of times write() was called
    size_t write_call_count;
    /// @brief Storage for data written through this mock
    std::shared_ptr<std::vector<std::vector<T>>> written_data;

    /// @brief Constructs a new mock writer
    /// @param written_data Shared pointer to store written data
    /// @param start_errors Sequence of errors to return from start()
    /// @param stop_errors Sequence of errors to return from stop()
    /// @param write_responses Sequence of errors to return from write()
    explicit Writer(
        std::shared_ptr<std::vector<std::vector<T>>> written_data =
            std::make_shared<std::vector<std::vector<T>>>(),
        const std::vector<x::errors::Error> &start_errors = {x::errors::NIL},
        const std::vector<x::errors::Error> &stop_errors = {x::errors::NIL},
        std::vector<x::errors::Error> write_responses = {x::errors::NIL}
    );

    x::errors::Error write(const std::vector<T> &data) override;

    std::shared_ptr<std::vector<std::vector<T>>> get_written_data() const {
        return written_data;
    }
};
}
}
