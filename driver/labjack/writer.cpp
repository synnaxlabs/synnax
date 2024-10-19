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

    waiting_reader.notify_one();
}

///////////////////////////////////////////////////////////////////////////////////
//                                   WriteSink                                   //
///////////////////////////////////////////////////////////////////////////////////