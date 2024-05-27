// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "opc.h"
#include "util.h"
#include "driver/config/config.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"

namespace opc {

typedef struct {
} WriterChannelConfig;

typedef struct {

} WriterConfig;


class Writer final : public task::Task {

}


class WriterSink final : public pipeline::Sink {
public:
    WriterConfig cfg;
    std::shared_ptr<opc::Client> client;
    std::set>ChannelKey> keys;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;

    UA_WriteRequest request;
    std::vector<UA_WriteValue



}

} // namespace opc