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

namespace hardware {
struct Hardware {
    virtual ~Hardware() = default;

    /// @brief starts the task.
    [[nodiscard]] virtual xerrors::Error start() = 0;

    /// @brief stops the task.
    [[nodiscard]] virtual xerrors::Error stop() = 0;
};

/// @brief a thing shim on top of NI DAQMX that allows us to use different read
/// interfaces for analog and digital tasks. It also allows us to mock the hardware
/// during testing.
template<typename T>
struct Reader : virtual Hardware {
    /// @brief reads data from the hardware.
    /// @param samples_per_channel the number of samples to read per channel.
    /// @param data the buffer to read data into.
    /// @return a pair containing the number of samples read and an error if one
    /// occurred.
    [[nodiscard]] virtual std::pair<size_t, xerrors::Error> read(
        size_t samples_per_channel,
        std::vector<T> &data
    ) = 0;
};

template<typename T>
struct Writer : virtual Hardware {
    [[nodiscard]] virtual xerrors::Error write(const std::vector<T> &data) = 0;
};


namespace daqmx {
/// @brief a base implementation of the hardware interface that uses the NI DAQMX
/// in the background.
struct Base : virtual Hardware {
protected:
    /// @brief the handle for the task.
    TaskHandle task_handle;
    /// @brief the NI DAQmx API.
    std::shared_ptr<SugaredDAQmx> dmx;
    /// @brief a flag to indicate if the task is running.
    std::atomic<bool> running = false;

    Base(TaskHandle task_handle, std::shared_ptr<SugaredDAQmx> dmx);
    ~Base() override;

public:
    /// @brief implements HardwareInterface to start the DAQmx task.
    xerrors::Error start() override;

    /// @brief implements the HardwareInterface to stop the DAQmx task.
    xerrors::Error stop() override;
};

struct DigitalWriter final : Base, Writer<uint8_t> {
    DigitalWriter(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    );
    xerrors::Error write(const std::vector<uint8_t> &data) override;
};

struct AnalogWriter final : Base, Writer<double> {
    AnalogWriter(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    );
    xerrors::Error write(const std::vector<double> &data) override;
};

/// @brief a hardware interface for digital tasks.
struct DigitalReader final : Base, Reader<uint8_t> {
    DigitalReader(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    );
    std::pair<size_t, xerrors::Error> read(
        const size_t samples_per_channel,
        std::vector<unsigned char> &data
    ) override;
};

/// @brief a hardware interface for analog tasks.
struct AnalogReader final : Base, Reader<double> {
    AnalogReader(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    );
    std::pair<size_t, xerrors::Error> read(
        const size_t samples_per_channel,
        std::vector<double> &data
    ) override;
};
}

namespace mock {
struct Base : virtual hardware::Hardware {
    std::vector<xerrors::Error> start_errors;
    std::vector<xerrors::Error> stop_errors;
    size_t start_call_count;
    size_t stop_call_count;

protected:
    explicit Base(
        const std::vector<xerrors::Error>& start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error>& stop_errors = {xerrors::NIL}
    );

public:
    xerrors::Error start() override;
    xerrors::Error stop() override;
};

template<typename T>
class Reader final : public Base, public hardware::Reader<T> {
public:
    std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses;
    size_t read_call_count;
    explicit Reader(
        const std::vector<xerrors::Error>& start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error>& stop_errors = {xerrors::NIL},
        std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses = {{{0.5}, xerrors::NIL}}
    );

    std::pair<size_t, xerrors::Error> read(
        size_t samples_per_channel,
        std::vector<T>& data
    ) override;
};

template<typename T>
class Writer final : public Base, public hardware::Writer<T> {
public:
    std::vector<xerrors::Error> write_responses;
    size_t write_call_count;
    std::shared_ptr<std::vector<std::vector<T>>> written_data;
    explicit Writer(
        std::shared_ptr<std::vector<std::vector<T>>> written_data = std::make_shared<std::vector<std::vector<T>>>(),
        const std::vector<xerrors::Error>& start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error>& stop_errors = {xerrors::NIL},
        std::vector<xerrors::Error> write_responses = {xerrors::NIL}
    );

    xerrors::Error write(const std::vector<T> &data) override;
    
    std::shared_ptr<std::vector<std::vector<T>>> get_written_data() const { return written_data; }
};

}

}