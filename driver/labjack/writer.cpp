// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
labjack::StateSource::StateSource(
    synnax::Rate state_rate, // TODO: make this synnax::Rate?
    synnax::ChannelKey &state_index_key,
    std::map<synnax::ChannelKey, out_state> state_map
) : state_rate(state_rate),
    state_index_key(state_index_key),
    state_map(state_map){
    this->timer = loop::Timer(this->state_rate); // check if ic an move this to member initializer list
}

std::pair<synnax::Frame, freighter::Error> labjack::StateSource::read(
    breaker::Breaker &breaker){
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // sleep for state period
    this->timer.wait(breaker);
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), freighter::NIL);
}

synnax::Frame ni::StateSource::get_state(){
    // frame size = # monitored states + 1 index channel for all those states
    auto state_Frame = synnax::Frame(this->state_map.size() + 1);
    state_Frame.add(
        this->state_index_key,
        synnax::Series(
            synnax::Timestamp::now().value,
            synnax::TIMESTAMP
        )
    );
    for(auto &[key, value]: this->state_map) {
        state_Frame.add(
                key,
                synnax::Series(value.state, value.data_type)
        );
    }

    return state_Frame;
}

void ni::StateSource::update_state(synnax::Frame frame){
    std::unique_lock<std::mutex> lock(this->state_mutex);
    //TODO: come back to and implement this
    for (auto key: *(frame.channels)){
        if (key == this->state_index_key) continue;
        auto value = frame.get(key).value;
        this->state_map[key].state = value;
    }

    waiting_reader.notify_one();
}

///////////////////////////////////////////////////////////////////////////////////
//                                   WriteSink                                   //
///////////////////////////////////////////////////////////////////////////////////

labjack::WriteSink::WriteSink(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const labjack::WriterConfig &writer_config
    ) : ctx(ctx),
        task(task),
        writer_config(writer_config){

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    }

    this->breaker = breaker::Breaker(breaker_config);

    this->state_source = std::make_shared<labjack::StateSource(
        writer_config.state_rate,
        writer_config.state_index_key,
        writer_config.initial_state_map
    );
}

~labjack::WriteSink::WriteSink(){
    this->stop("");
    CloseOrDie(this->handle);
}

void labjack::WriteSink::init(){
    auto err = LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle);
    ErrorCheck(err, "[labjack.writer] LJM_Open error on serial num: %s ", this->reader_config.serial_number);
}

freighter::Error labjack::WriteSink::write(synnax::Frame frame){
    for(auto key: *(frame.channels)){
        double value = 0;
        std::string loc = this->writer_config.initial_state_map[key].location;
        auto err = LJM_eWriteName(this->handle, loc.c_str(), value);
    }
    this-state_source->update_state(frame);
    return freighter::NIL;
}

freighter::Error labjack::WriteSink::stop(const std::string &cmd_key){
    PrintErrorIfError(err, "LJM_CleanInterval");
    CloseOrDie(this->handle);
    ctx->setState({
                          .task = task.key,
                          .key = cmd_key,
                          .variant = "success",
                          .details = {
                                  {"running", false},
                                  {"message", "Task stopped successfully"}
                          }
                  });
    return freighter::NIL;
}

freighter::Error labjack::WriteSink::start(const std::string &cmd_key){
    this->init();
    ctx->setState({
                          .task = task.key,
                          .key = cmd_key,
                          .variant = "success",
                          .details = {
                                  {"running", true},
                                  {"message", "Task started successfully"}
                          }
                  });
    return freighter::NIL;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_cmd_channel_keys(){ // TODO: rename to get_cmd_keys
    std::vector<synnax::ChannelKey> keys
    for (auto &channel: this->writer_config.channels){
        if(channel.enabled) keys.push_back(channel.cmd_key);
    }
    // Don't need index key as we're only using this for streaming cmds
    return keys;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_state_channel_keys(){ // TODO: rename to get_state_keys
    std::vector<synnax::ChannelKey> keys;
    for(auto &channel: this->writer_config.channels){
        if(channel.enabled) keys.push_back(channel.state_key);
    }
    keys.push_back(this->writer_config.state_index_key);
    return keys;
}