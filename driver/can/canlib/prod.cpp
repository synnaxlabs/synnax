// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "driver/can/canlib/prod.h"
#include "driver/can/symbols.h"
#include "driver/errors/errors.h"

namespace driver::can::canlib {
namespace {
const LibraryInfo LIBRARY_INFO = {
    "Kvaser CANlib",
    "https://www.kvaser.com/downloads-kvaser/"
};

#ifdef _WIN32
const std::string LIBRARY_NAME = "canlib32.dll";
#else
const std::string LIBRARY_NAME;
#endif
}

std::pair<std::shared_ptr<API>, x::errors::Error> ProdAPI::load() {
    if (LIBRARY_NAME.empty())
        return {
            nullptr,
            x::errors::Error(
                UNSUPPORTED_ERROR,
                "CANlib runs on Windows. Use the socketcan backend for Kvaser "
                "adapters on Linux"
            )
        };
    auto lib = std::make_unique<x::lib::Shared>(LIBRARY_NAME);
    if (!lib->load()) return {nullptr, driver::errors::missing_lib(LIBRARY_INFO)};
    auto api = std::make_shared<ProdAPI>(std::move(lib));
    Symbols symbols(*api->lib);
    symbols.resolve("canInitializeLibrary", api->initialize_library);
    symbols.resolve("canOpenChannel", api->open_channel);
    symbols.resolve("canClose", api->close);
    symbols.resolve("canBusOn", api->bus_on);
    symbols.resolve("canBusOff", api->bus_off);
    symbols.resolve("canSetBusParams", api->set_bus_params);
    symbols.resolve("canSetBusParamsFd", api->set_bus_params_fd);
    symbols.resolve("canSetBusOutputControl", api->set_bus_output_control);
    symbols.resolve("canIoCtl", api->io_ctl);
    symbols.resolve("canReadWait", api->read_wait);
    symbols.resolve("canWrite", api->write);
    symbols.resolve("canGetNumberOfChannels", api->get_number_of_channels);
    symbols.resolve("canGetChannelData", api->get_channel_data);
    symbols.resolve("canGetErrorText", api->get_error_text);
    if (auto err = symbols.error(LIBRARY_INFO.name)) return {nullptr, err};
    api->initialize_library();
    return {api, x::errors::NIL};
}

canHandle ProdAPI::OpenChannel(const int channel, const int flags) {
    return this->open_channel(channel, flags);
}

canStatus ProdAPI::Close(const canHandle hnd) {
    return this->close(hnd);
}

canStatus ProdAPI::BusOn(const canHandle hnd) {
    return this->bus_on(hnd);
}

canStatus ProdAPI::BusOff(const canHandle hnd) {
    return this->bus_off(hnd);
}

canStatus ProdAPI::SetBusParams(
    const canHandle hnd,
    const long freq,
    const unsigned int tseg1,
    const unsigned int tseg2,
    const unsigned int sjw,
    const unsigned int no_samp,
    const unsigned int syncmode
) {
    return this->set_bus_params(hnd, freq, tseg1, tseg2, sjw, no_samp, syncmode);
}

canStatus ProdAPI::SetBusParamsFd(
    const canHandle hnd,
    const long freq_brs,
    const unsigned int tseg1_brs,
    const unsigned int tseg2_brs,
    const unsigned int sjw_brs
) {
    return this->set_bus_params_fd(hnd, freq_brs, tseg1_brs, tseg2_brs, sjw_brs);
}

canStatus ProdAPI::SetBusOutputControl(const canHandle hnd, const unsigned int type) {
    return this->set_bus_output_control(hnd, type);
}

canStatus ProdAPI::IoCtl(
    const canHandle hnd,
    const unsigned int func,
    void *buf,
    const unsigned int buflen
) {
    return this->io_ctl(hnd, func, buf, buflen);
}

canStatus ProdAPI::ReadWait(
    const canHandle hnd,
    long *id,
    void *msg,
    unsigned int *dlc,
    unsigned int *flag,
    unsigned long *time,
    const unsigned long timeout
) {
    return this->read_wait(hnd, id, msg, dlc, flag, time, timeout);
}

canStatus ProdAPI::Write(
    const canHandle hnd,
    const long id,
    void *msg,
    const unsigned int dlc,
    const unsigned int flag
) {
    return this->write(hnd, id, msg, dlc, flag);
}

canStatus ProdAPI::GetNumberOfChannels(int *count) {
    return this->get_number_of_channels(count);
}

canStatus ProdAPI::GetChannelData(
    const int channel,
    const int item,
    void *buffer,
    const std::size_t size
) {
    return this->get_channel_data(channel, item, buffer, size);
}

canStatus
ProdAPI::GetErrorText(const canStatus err, char *buf, const unsigned int size) {
    return this->get_error_text(err, buf, size);
}
}
