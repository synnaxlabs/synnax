// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"

#include "driver/ni/daqmx/sugared.h"
#include "driver/task/common/read_task.h"

namespace hardware {
struct Hardware {
    virtual ~Hardware() = default;

    /// @brief starts the task.
    [[nodiscard]] virtual xerrors::Error start() = 0;

    /// @brief stops the task.
    [[nodiscard]] virtual xerrors::Error stop() = 0;
};

struct ReadResult : common::ReadResult {
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
    [[nodiscard]] virtual xerrors::Error write(const std::vector<T> &data) = 0;
};

namespace daqmx {
/// @brief Base DAQmx hardware implementation that manages task lifecycle
struct Base : virtual Hardware {
protected:
    /// @brief the handle for the task.
    TaskHandle task_handle;
    /// @brief the NI-DAQmx API.
    std::shared_ptr<::daqmx::SugaredAPI> dmx;
    /// @brief a flag to indicate if the task is running.
    std::atomic<bool> running = false;

    Base(TaskHandle task_handle, std::shared_ptr<::daqmx::SugaredAPI> dmx);
    ~Base() override;

public:
    /// @brief implements HardwareInterface to start the DAQmx task.
    xerrors::Error start() override;

    /// @brief implements the HardwareInterface to stop the DAQmx task.
    xerrors::Error stop() override;
};

/// @brief Implementation of digital output writing using DAQmx
struct DigitalWriter final : Base, Writer<uint8_t> {
    /// @brief Constructs a new digital writer
    /// @param dmx The DAQmx API interface
    /// @param task_handle Handle to the DAQmx task
    DigitalWriter(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    xerrors::Error write(const std::vector<uint8_t> &data) override;
};

/// @brief Implementation of analog output writing using DAQmx
struct AnalogWriter final : Base, Writer<double> {
    /// @brief Constructs a new analog writer
    /// @param dmx The DAQmx API interface
    /// @param task_handle Handle to the DAQmx task
    AnalogWriter(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    xerrors::Error write(const std::vector<double> &data) override;
};

/// @brief a hardware interface for digital tasks.
struct DigitalReader final : Base, Reader<uint8_t> {
    DigitalReader(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult
    read(size_t samples_per_channel, std::vector<unsigned char> &data) override;
};

/// @brief a hardware interface for analog tasks.
struct AnalogReader final : Base, Reader<double> {
    /// @brief the total number of samples requested by calls to read() from
    /// the user.
    size_t total_samples_requested = 0;
    /// @brief the total number of samples actually acquired from the hardware
    /// by DAQmx.
    uInt64 total_samples_acquired = 0;

    AnalogReader(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult read(size_t samples_per_channel, std::vector<double> &data) override;

    xerrors::Error start() override;

    int64 update_skew(const size_t &n_requested);
};

/// @brief a hardware interface for counter input tasks.
struct CounterReader final : Base, Reader<double> {
    /// @brief the total number of samples requested by calls to read() from
    /// the user.
    size_t total_samples_requested = 0;
    /// @brief the total number of samples actually acquired from the hardware
    /// by DAQmx.
    uInt64 total_samples_acquired = 0;

    CounterReader(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    ReadResult read(size_t samples_per_channel, std::vector<double> &data) override;

    xerrors::Error start() override;

    int64 update_skew(const size_t &n_requested);
};

/// @brief Implementation of counter output writing using DAQmx
/// Counter output tasks must clear the task on stop to release resources.
/// This is a known NI-DAQmx limitation - DAQmxTaskControl(Unreserve) does not
/// work for counter output tasks. After stopping, the task must be reconfigured
/// before it can be started again.
struct CounterWriter final : Base, Writer<double> {
    /// @brief Constructs a new counter writer
    /// @param dmx The DAQmx API interface
    /// @param task_handle Handle to the DAQmx task
    CounterWriter(
        const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    );
    xerrors::Error write(const std::vector<double> &data) override;

    /// @brief Override stop() to clear task and release counter resources
    /// After calling stop(), the task cannot be restarted - must reconfigure
    xerrors::Error stop() override;
};
}

namespace mock {
/// @brief Base mock implementation for testing hardware interfaces
struct Base : virtual hardware::Hardware {
    /// @brief Errors to return from start() calls in sequence
    std::vector<xerrors::Error> start_errors;
    /// @brief Errors to return from stop() calls in sequence
    std::vector<xerrors::Error> stop_errors;
    /// @brief Number of times start() was called
    size_t start_call_count;
    /// @brief Number of times stop() was called
    size_t stop_call_count;

protected:
    explicit Base(
        const std::vector<xerrors::Error> &start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error> &stop_errors = {xerrors::NIL}
    );

public:
    xerrors::Error start() override;
    xerrors::Error stop() override;
};

/// @brief Mock implementation of Reader interface for testing
/// @tparam T The data type to read (uint8_t for digital, double for analog)
template<typename T>
class Reader final : public Base, public hardware::Reader<T> {
public:
    /// @brief Predefined responses for read() calls
    std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses;
    /// @brief Number of times read() was called
    size_t read_call_count;

    /// @brief Constructs a new mock reader
    /// @param start_errors Sequence of errors to return from start()
    /// @param stop_errors Sequence of errors to return from stop()
    /// @param read_responses Sequence of data and errors to return from read()
    explicit Reader(
        const std::vector<xerrors::Error> &start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error> &stop_errors = {xerrors::NIL},
        std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses = {
            {{0.5}, xerrors::NIL}
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
    std::vector<xerrors::Error> write_responses;
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
        const std::vector<xerrors::Error> &start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error> &stop_errors = {xerrors::NIL},
        std::vector<xerrors::Error> write_responses = {xerrors::NIL}
    );

    xerrors::Error write(const std::vector<T> &data) override;

    std::shared_ptr<std::vector<std::vector<T>>> get_written_data() const {
        return written_data;
    }
};
}
}
