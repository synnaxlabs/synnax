// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "driver/ni/hardware/hardware.h"

namespace hardware::daqmx {
Base::Base(TaskHandle task_handle, std::shared_ptr<::daqmx::SugaredAPI> dmx):
    task_handle(task_handle), dmx(std::move(dmx)) {}

Base::~Base() {
    if (const auto err = this->dmx->ClearTask(this->task_handle))
        LOG(ERROR) << "[ni] unexpected failure to clear daqmx task: " << err;
}

xerrors::Error Base::start() {
    if (this->running.exchange(true)) return xerrors::NIL;
    return this->dmx->StartTask(this->task_handle);
}

xerrors::Error Base::stop() {
    if (!this->running.exchange(false)) return xerrors::NIL;
    return this->dmx->StopTask(this->task_handle);
}

DigitalWriter::DigitalWriter(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
):
    Base(task_handle, dmx) {}

xerrors::Error DigitalWriter::write(const std::vector<uint8_t> &data) {
    return this->dmx->WriteDigitalLines(
        this->task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data.data(),
        nullptr,
        nullptr
    );
}

AnalogWriter::AnalogWriter(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
):
    Base(task_handle, dmx) {}

xerrors::Error AnalogWriter::write(const std::vector<double> &data) {
    return this->dmx->WriteAnalogF64(
        this->task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data.data(),
        nullptr,
        nullptr
    );
}

DigitalReader::DigitalReader(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
):
    Base(task_handle, dmx) {}

ReadResult DigitalReader::read(
    const size_t samples_per_channel,
    std::vector<unsigned char> &data
) {
    ReadResult res;
    int32 samples_read = 0;
    res.error = this->dmx->ReadDigitalLines(
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
    return res;
}

AnalogReader::AnalogReader(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
):
    Base(task_handle, dmx) {}

ReadResult
AnalogReader::read(const size_t samples_per_channel, std::vector<double> &data) {
    ReadResult res;
    int32 samples_read = 0;
    if (res.error = this->dmx->ReadAnalogF64(
            this->task_handle,
            static_cast<int32>(samples_per_channel),
            DAQmx_Val_WaitInfinitely,
            DAQmx_Val_GroupByChannel,
            data.data(),
            data.size(),
            &samples_read,
            nullptr
        );
        res.error)
        return res;
    res.skew = this->update_skew(samples_read);
    return res;
}

xerrors::Error AnalogReader::start() {
    this->total_samples_acquired = 0;
    this->total_samples_requested = 0;
    if (const auto err = this->dmx->SetReadOverWrite(
            this->task_handle,
            DAQmx_Val_OverwriteUnreadSamps
        ))
        return err;
    return Base::start();
}

int64 AnalogReader::update_skew(const size_t &n_requested) {
    uInt64 next_total_samples_acquired;
    if (const auto err = this->dmx->GetReadTotalSampPerChanAcquired(
            this->task_handle,
            &next_total_samples_acquired
        ))
        LOG(WARNING) << "[ni] failed to get total samples acquired: " << err;
    if (next_total_samples_acquired < this->total_samples_acquired) {
        LOG(WARNING) << "[ni] hardware reader detected recovery from failure.";
        this->total_samples_requested = 0;
    }
    this->total_samples_acquired = next_total_samples_acquired;
    this->total_samples_requested += n_requested;
    return static_cast<int64>(this->total_samples_acquired) -
           static_cast<int64>(this->total_samples_requested);
}

CounterReader::CounterReader(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
):
    Base(task_handle, dmx) {}

ReadResult
CounterReader::read(const size_t samples_per_channel, std::vector<double> &data) {
    ReadResult res;
    int32 samples_read = 0;
    if (res.error = this->dmx->ReadCounterScalarF64(
            this->task_handle,
            static_cast<int32>(samples_per_channel),
            DAQmx_Val_WaitInfinitely,
            data.data(),
            data.size(),
            &samples_read,
            nullptr
        );
        res.error)
        return res;
    res.skew = this->update_skew(samples_read);
    return res;
}

xerrors::Error CounterReader::start() {
    this->total_samples_acquired = 0;
    this->total_samples_requested = 0;
    if (const auto err = this->dmx->SetReadOverWrite(
            this->task_handle,
            DAQmx_Val_OverwriteUnreadSamps
        ))
        return err;
    return Base::start();
}

int64 CounterReader::update_skew(const size_t &n_requested) {
    uInt64 next_total_samples_acquired;
    if (const auto err = this->dmx->GetReadTotalSampPerChanAcquired(
            this->task_handle,
            &next_total_samples_acquired
        ))
        LOG(WARNING) << "[ni] failed to get total samples acquired: " << err;
    if (next_total_samples_acquired < this->total_samples_acquired) {
        LOG(WARNING) << "[ni] hardware reader detected recovery from failure.";
        this->total_samples_requested = 0;
    }
    this->total_samples_acquired = next_total_samples_acquired;
    this->total_samples_requested += n_requested;
    return static_cast<int64>(this->total_samples_acquired) -
           static_cast<int64>(this->total_samples_requested);
}
}
