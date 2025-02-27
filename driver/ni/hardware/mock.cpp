// Copyright 2025 Synnax Labs, Inc.
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
    const std::vector<xerrors::Error>& start_errors,
    const std::vector<xerrors::Error>& stop_errors
) : start_errors(start_errors),
    stop_errors(stop_errors),
    start_call_count(0),
    stop_call_count(0) {}

xerrors::Error Base::start() {
    auto err = start_errors[std::min(start_call_count, start_errors.size() - 1)];
    start_call_count++;
    return err;
}

xerrors::Error Base::stop() {
    auto err = stop_errors[std::min(stop_call_count, stop_errors.size() - 1)];
    stop_call_count++;
    return err;
}

template<typename T>
Reader<T>::Reader(
    const std::vector<xerrors::Error>& start_errors,
    const std::vector<xerrors::Error>& stop_errors,
    std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses
) : Base(start_errors, stop_errors),
    read_responses(std::move(read_responses)),
    read_call_count(0) {}

template<typename T>
std::pair<size_t, xerrors::Error> Reader<T>::read(
    size_t samples_per_channel,
    std::vector<T>& data
) {
    auto response = read_responses[std::min(read_call_count, read_responses.size() - 1)];
    read_call_count++;
    if (!response.first.empty())
        std::copy(response.first.begin(), response.first.end(), data.begin());
    return {response.first.size(), response.second};
}

template<typename T>
Writer<T>::Writer(
    const std::vector<xerrors::Error>& start_errors,
    const std::vector<xerrors::Error>& stop_errors,
    std::vector<xerrors::Error> write_responses
) : Base(start_errors, stop_errors),
    write_responses(std::move(write_responses)),
    write_call_count(0) {}

template<typename T>
xerrors::Error Writer<T>::write(const T* data) {
    last_written_data = std::vector<T>(data, data + 1); // Assuming single value writes
    auto err = write_responses[std::min(write_call_count, write_responses.size() - 1)];
    write_call_count++;
    return err;
}

template class Reader<uint8_t>;
template class Reader<double>;
template class Writer<uint8_t>;
template class Writer<double>;

}