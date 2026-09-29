// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>

#include "driver/arinc429/ddc/api.h"

namespace driver::arinc429::ddc {
namespace {
template<typename F>
bool resolve(const x::lib::Shared &lib, const std::string &name, F &fn) {
    fn = reinterpret_cast<F>(const_cast<void *>(lib.get_func_ptr(name)));
    return fn != nullptr;
}
}

std::pair<std::shared_ptr<API>, x::errors::Error> API::load() {
    auto lib = std::make_unique<x::lib::Shared>(LIBRARY_NAME);
    if (!lib->load()) return {nullptr, errors::missing_lib(LIBRARY_INFO)};
    auto api = std::make_shared<API>();
    const bool ok = resolve(*lib, "InitCard", api->init_card) &&
                    resolve(*lib, "FreeCard", api->free_card) &&
                    resolve(*lib, "GetChannelCount", api->get_channel_count) &&
                    resolve(*lib, "GetErrorMsg", api->get_error_msg) &&
                    resolve(*lib, "EnableRx", api->enable_rx) &&
                    resolve(*lib, "EnableTx", api->enable_tx) &&
                    resolve(*lib, "SetRxChannelSpeed", api->set_rx_channel_speed) &&
                    resolve(*lib, "SetTxSpeed", api->set_tx_speed) &&
                    resolve(*lib, "SetRxChannelParity", api->set_rx_channel_parity) &&
                    resolve(*lib, "SetTxParity", api->set_tx_parity) &&
                    resolve(*lib, "SetBitFormat", api->set_bit_format) &&
                    resolve(
                        *lib,
                        "ReadRxQueueIrigMore",
                        api->read_rx_queue_irig_more
                    ) &&
                    resolve(*lib, "LoadTxQueueMore", api->load_tx_queue_more);
    if (!ok) return {nullptr, errors::missing_lib(LIBRARY_INFO)};
    api->lib = std::move(lib);
    return {api, x::errors::NIL};
}

x::errors::Error API::error(const short code) const {
    if (code >= sdk::SUCCESS) return x::errors::NIL;
    std::array<char, sdk::ERROR_MESSAGE_SIZE> msg{};
    if (this->get_error_msg(code, msg.data()) != sdk::SUCCESS)
        return x::errors::Error(
            errors::CRITICAL_HARDWARE_ERROR,
            "DDC ARINC 429 error " + std::to_string(code)
        );
    return x::errors::Error(
        errors::CRITICAL_HARDWARE_ERROR,
        std::string(msg.data()) + " (DDC ARINC 429 error " + std::to_string(code) + ")"
    );
}
}
