// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/hardware/hardware.h"
#include "glog/logging.h"

namespace hardware::daqmx {
Base::Base(TaskHandle task_handle, std::shared_ptr<::daqmx::SugaredAPI> dmx)
    : task_handle(task_handle), dmx(std::move(dmx)) {
}

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
): Base(task_handle, dmx) {
}

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
): Base(task_handle, dmx) {
}

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
): Base(task_handle, dmx) {
}

std::pair<ReadDigest, xerrors::Error> DigitalReader::read(
    const size_t samples_per_channel,
    std::vector<unsigned char> &data
) {
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
    ReadDigest dig;
    dig.samps_per_chan_read = samples_read;
    return {dig, err};
}

AnalogReader::AnalogReader(
    const std::shared_ptr<::daqmx::SugaredAPI> &dmx,
    TaskHandle task_handle
): Base(task_handle, dmx) {
}

std::pair<ReadDigest, xerrors::Error> AnalogReader::read(
    const size_t samples_per_channel,
    std::vector<double> &data
) {
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
    ReadDigest dig;
    dig.samps_per_chan_read = samples_read;
    uInt64 next_high_water = 0;
    if (const auto err = this->dmx->GetReadTotalSampPerChanAcquired(this->task_handle, &next_high_water))
        return {dig, err};
    dig.samps_per_chan_acquired = next_high_water - this->total_samples_acquired_high_water;
    if (dig.samps_per_chan_acquired < dig.samps_per_chan_read) dig.samps_per_chan_acquired = dig.samps_per_chan_read;
    this->total_samples_acquired_high_water = next_high_water;
    this->requested_total_samples += samples_read;
    if ((this->total_samples_acquired_high_water - 30) > this->requested_total_samples) {
        auto skew = this->total_samples_acquired_high_water  -   this->requested_total_samples;
        VLOG(1) << "[driver.ni] application is trailing data acquisition loop by " << skew << " samples";
    }
    return {dig, err};
}

xerrors::Error AnalogReader::start() {
    this->total_samples_acquired_high_water = 0;
    this->requested_total_samples = 0;
    if (const auto err = this->dmx->SetReadOverWrite(this->task_handle, DAQmx_Val_OverwriteUnreadSamps)) return err;
    return Base::start();
}
}
