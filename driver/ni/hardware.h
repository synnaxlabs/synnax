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

struct Hardware {
    virtual ~Hardware() = default;

    /// @brief starts the task.
    [[nodiscard]] virtual xerrors::Error start() = 0;

    /// @brief stops the task.
    [[nodiscard]] virtual xerrors::Error stop() = 0;
};


/// @brief a base implementation of the hardware interface that uses the NI DAQMX
/// in the background.
struct DAQmxHardware : virtual Hardware {
protected:
    /// @brief the handle for the task.
    TaskHandle task_handle;
    /// @brief the NI DAQmx API.
    std::shared_ptr<SugaredDAQmx> dmx;

    DAQmxHardware(TaskHandle task_handle, std::shared_ptr<SugaredDAQmx> dmx):
        task_handle(task_handle), dmx(std::move(dmx)) {
    }

    ~DAQmxHardware() override {
        if (const auto err = this->dmx->ClearTask(this->task_handle))
            LOG(ERROR) << "[ni] unexpected failure to clear daqmx task: " << err;
    }

public:
    /// @brief implements HardwareInterface to start the DAQmx task.
    xerrors::Error start() override {
        return this->dmx->StartTask(this->task_handle);
    }

    /// @brief implements the HardwareInterface to stop the DAQmx task.
    xerrors::Error stop() override {
        return this->dmx->StopTask(this->task_handle);
    }
};

/// @brief a thing shim on top of NI DAQMX that allows us to use different read
/// interfaces for analog and digital tasks. It also allows us to mock the hardware
/// during testing.
template<typename T>
struct HardwareReader : virtual Hardware {
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
struct HardwareWriter : virtual Hardware {
    [[nodiscard]] virtual xerrors::Error write(const T *data) = 0;
};

struct DigitalHardwareWriter final : DAQmxHardware, HardwareWriter<uint8_t> {
    DigitalHardwareWriter(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardware(task_handle, dmx) {
    }

    xerrors::Error write(const uint8_t *data) override {
        return this->dmx->WriteDigitalU8(
            this->task_handle,
            1,
            1,
            10.0,
            DAQmx_Val_GroupByChannel,
            data,
            nullptr,
            nullptr
        );
    }
};

struct AnalogHardwareWriter final : DAQmxHardware, HardwareWriter<double> {
    AnalogHardwareWriter(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardware(task_handle, dmx) {
    }

    xerrors::Error write(const double *data) override {
        return this->dmx->WriteAnalogF64(
            this->task_handle,
            1,
            1,
            10.0,
            DAQmx_Val_GroupByChannel,
            data,
            nullptr,
            nullptr
        );
    }
};

/// @brief a hardware interface for digital tasks.
struct DigitalHardwareReader final : DAQmxHardware, HardwareReader<uint8_t> {
    DigitalHardwareReader(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardware(task_handle, dmx) {
    }

    std::pair<size_t, xerrors::Error> read(
        const size_t samples_per_channel,
        std::vector<unsigned char> &data
    ) override {
        int32 samples_read = 0;
        const auto err = this->dmx->ReadDigitalLines(
            this->task_handle,
            static_cast<int32>(samples_per_channel),
            DAQmx_Val_WaitInfinitely,
            DAQmx_Val_GroupByChannel,
            data.data(),
            data.size(),
            &samples_read,
            nullptr,
            nullptr
        );
        return {static_cast<size_t>(samples_read), err};
    }
};

/// @brief a hardware interface for analog tasks.
struct AnalogHardwareReader final : DAQmxHardware, HardwareReader<double> {
    AnalogHardwareReader(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardware(task_handle, dmx) {
    }

    std::pair<size_t, xerrors::Error> read(
        const size_t samples_per_channel,
        std::vector<double> &data
    ) override {
        int32 samples_read = 0;
        const auto err = this->dmx->ReadAnalogF64(
            this->task_handle,
            static_cast<int32>(samples_per_channel),
            DAQmx_Val_WaitInfinitely,
            DAQmx_Val_GroupByChannel,
            data.data(),
            data.size(),
            &samples_read,
            nullptr
        );
        return {static_cast<size_t>(samples_read), err};
    }
};
