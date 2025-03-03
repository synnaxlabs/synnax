// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/read_task.h"
#include "driver/labjack/util.h"

labjack::ReaderSource::ReaderSource(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task,
    const ReadTaskConfig &reader_config,
    const std::shared_ptr<labjack::DeviceManager> &device_manager
) : reader_config(reader_config),
    ctx(ctx),
    breaker(breaker::default_config(task.name)),
    task(task),
    device_manager(device_manager) {
    this->handle = this->device_manager->get_device_handle(this->reader_config.serial_number);
}

    // if (err == LJME_RECONNECT_FAILED ||
    //     err == LJME_NO_RESPONSE_BYTES_RECEIVED ||
    //     err == LJME_INCORRECT_NUM_COMMAND_BYTES_SENT ||
    //     err == LJME_NO_COMMAND_BYTES_SENT ||
    //     err == LJME_INCORRECT_NUM_RESPONSE_BYTES_RECEIVED
    // ) {
    //     this->device_manager->close_device(this->reader_config.serial_number);
    // }
