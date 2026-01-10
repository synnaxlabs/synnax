// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/hardware/hardware.h"

namespace hardware::mock {

Base::Base(
    const std::vector<x::errors::Error> &start_errors,
    const std::vector<x::errors::Error> &stop_errors
):
    start_errors(start_errors),
    stop_errors(stop_errors),
    start_call_count(0),
    stop_call_count(0) {}

x::errors::Error Base::start() {
    auto err = start_errors[std::min(start_call_count, start_errors.size() - 1)];
    start_call_count++;
    return err;
}

x::errors::Error Base::stop() {
    auto err = stop_errors[std::min(stop_call_count, stop_errors.size() - 1)];
    stop_call_count++;
    return err;
}

template<typename T>
Reader<T>::Reader(
    const std::vector<x::errors::Error> &start_errors,
    const std::vector<x::errors::Error> &stop_errors,
    std::vector<std::pair<std::vector<T>, x::errors::Error>> read_responses
):
    Base(start_errors, stop_errors),
    read_responses(std::move(read_responses)),
    read_call_count(0) {}

template<typename T>
ReadResult Reader<T>::read(size_t samples_per_channel, std::vector<T> &data) {
    auto [res_data, err] = read_responses
        [std::min(read_call_count, read_responses.size() - 1)];
    read_call_count++;
    if (!res_data.empty()) std::copy(res_data.begin(), res_data.end(), data.begin());
    ReadResult res;
    res.error = err;
    return res;
}

template<typename T>
Writer<T>::Writer(
    std::shared_ptr<std::vector<std::vector<T>>> written_data,
    const std::vector<x::errors::Error> &start_errors,
    const std::vector<x::errors::Error> &stop_errors,
    std::vector<x::errors::Error> write_responses
):
    Base(start_errors, stop_errors),
    write_responses(std::move(write_responses)),
    write_call_count(0),
    written_data(written_data) {}

template<typename T>
x::errors::Error Writer<T>::write(const std::vector<T> &data) {
    written_data->push_back(data);
    auto err = write_responses[std::min(write_call_count, write_responses.size() - 1)];
    write_call_count++;
    return err;
}

template class Reader<uint8_t>;
template class Reader<double>;
template class Writer<uint8_t>;
template class Writer<double>;

}
