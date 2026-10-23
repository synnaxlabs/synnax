// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/can/can.h"
#include "driver/can/pcan/prod.h"

namespace driver::can::pcan {
std::pair<std::shared_ptr<API>, x::errors::Error> ProdAPI::load() {
    const auto library = pcan::library();
    auto [lib, err] = vendor::load(library);
    if (err) return {nullptr, err};
    auto api = std::make_shared<ProdAPI>(std::move(lib));
    x::lib::Symbols symbols(*api->lib);
    symbols.resolve("CAN_Initialize", api->initialize);
    symbols.resolve("CAN_InitializeFD", api->initialize_fd);
    symbols.resolve("CAN_Uninitialize", api->uninitialize);
    symbols.resolve("CAN_Read", api->read);
    symbols.resolve("CAN_ReadFD", api->read_fd);
    symbols.resolve("CAN_Write", api->write);
    symbols.resolve("CAN_WriteFD", api->write_fd);
    symbols.resolve("CAN_GetStatus", api->get_status);
    symbols.resolve("CAN_GetValue", api->get_value);
    symbols.resolve("CAN_SetValue", api->set_value);
    symbols.resolve("CAN_GetErrorText", api->get_error_text);
    err = symbols.error(CRITICAL_HARDWARE_ERROR, library.info.name);
    if (err) return {nullptr, err};
    return {api, x::errors::NIL};
}

TPCANStatus ProdAPI::Initialize(
    const TPCANHandle channel,
    const TPCANBaudrate btr0btr1,
    const TPCANType hw_type,
    const std::uint32_t io_port,
    const std::uint16_t interrupt
) {
    return this->initialize(channel, btr0btr1, hw_type, io_port, interrupt);
}

TPCANStatus ProdAPI::InitializeFD(const TPCANHandle channel, TPCANBitrateFD bitrate) {
    return this->initialize_fd(channel, bitrate);
}

TPCANStatus ProdAPI::Uninitialize(const TPCANHandle channel) {
    return this->uninitialize(channel);
}

TPCANStatus
ProdAPI::Read(const TPCANHandle channel, TPCANMsg *msg, TPCANTimestamp *timestamp) {
    return this->read(channel, msg, timestamp);
}

TPCANStatus ProdAPI::ReadFD(
    const TPCANHandle channel,
    TPCANMsgFD *msg,
    TPCANTimestampFD *timestamp
) {
    return this->read_fd(channel, msg, timestamp);
}

TPCANStatus ProdAPI::Write(const TPCANHandle channel, TPCANMsg *msg) {
    return this->write(channel, msg);
}

TPCANStatus ProdAPI::WriteFD(const TPCANHandle channel, TPCANMsgFD *msg) {
    return this->write_fd(channel, msg);
}

TPCANStatus ProdAPI::GetStatus(const TPCANHandle channel) {
    return this->get_status(channel);
}

TPCANStatus ProdAPI::GetValue(
    const TPCANHandle channel,
    const TPCANParameter parameter,
    void *buffer,
    const std::uint32_t length
) {
    return this->get_value(channel, parameter, buffer, length);
}

TPCANStatus ProdAPI::SetValue(
    const TPCANHandle channel,
    const TPCANParameter parameter,
    void *buffer,
    const std::uint32_t length
) {
    return this->set_value(channel, parameter, buffer, length);
}

TPCANStatus ProdAPI::GetErrorText(
    const TPCANStatus error,
    const std::uint16_t language,
    char *buffer
) {
    return this->get_error_text(error, language, buffer);
}
}
