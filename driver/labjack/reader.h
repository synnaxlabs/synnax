// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <string>
#include <vector>
#include <map>
#include <thread>
#include <stdio.h>

#include "LJM_Utilities.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/errors/errors.h"
#include "driver/task/task.h"

namespace labjack{

    struct ReaderChannelConfig{
        ///@brief name that labjack references physical channel by
        std::string location;
        bool enabled = true;
        synnax::DataType data_type;
        uint32_t channel_key;
        float32 range = 10.0;
        std::string channel_types = ""; // TODO: make this enumerated
        //TODO: negative channels for T7
    };

    struct ReaderConfig{
        std::string device_type;
        std::string device_key;
        std::vector<ReaderChannelConfig> channels;
        synnax::Rate sample_rate = synnax::Rate(1); // TODO change default?
        synnax::Rate stream_rate = synnax::Rate(1); // TODO change default?
        std::string task_name;
        synnax::ChannelKey task_key;
        std::set<uint32_t> index_keys;
        std::string serial_number; // used to open devices
        std::string connection_type; // used to open devices
        std::map<std::string, uint32_t> channel_map;
        int num_index = 0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                   Source                                      //
///////////////////////////////////////////////////////////////////////////////////
class Source : public pipeline::Source{
public:
    explicit Source (
            int handle,
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task task,
            const ReaderConfig &reader_config
        ) : handle(handle),
            ctx(ctx),
            task(task),
            reader_config(reader_config) {
        this->init();
    }

    // destructor
    ~Source();

    std::vector<synnax::ChannelKey> get_channel_keys();

    void stopped_with_err(const freighter::Error &err);

    std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker);

    void init();
private:

    int handle;
    ReaderConfig reader_config
    std::shared_ptr<task::Context> ctx;
    breaker::Breaker breaker;
    synnax::Task task;
};
}