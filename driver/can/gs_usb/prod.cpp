// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "driver/can/gs_usb/prod.h"
#include "driver/can/symbols.h"
#include "driver/errors/errors.h"

namespace driver::can::gs_usb {
namespace {
const LibraryInfo LIBRARY_INFO = {"libusb", "https://libusb.info"};

#ifdef _WIN32
const std::vector<std::string> LIBRARY_NAMES = {"libusb-1.0.dll"};
#elif defined(__APPLE__)
const std::vector<std::string> LIBRARY_NAMES = {
    "libusb-1.0.0.dylib",
    "/opt/homebrew/lib/libusb-1.0.0.dylib",
    "/usr/local/lib/libusb-1.0.0.dylib",
};
#else
const std::vector<std::string> LIBRARY_NAMES = {"libusb-1.0.so.0"};
#endif
}

std::pair<std::shared_ptr<API>, x::errors::Error> ProdAPI::load() {
    std::unique_ptr<x::lib::Shared> lib;
    for (const auto &name: LIBRARY_NAMES) {
        auto candidate = std::make_unique<x::lib::Shared>(name);
        if (!candidate->load()) continue;
        lib = std::move(candidate);
        break;
    }
    if (lib == nullptr) return {nullptr, driver::errors::missing_lib(LIBRARY_INFO)};
    auto api = std::make_shared<ProdAPI>(std::move(lib));
    Symbols symbols(*api->lib);
    symbols.resolve("libusb_init", api->init);
    symbols.resolve("libusb_exit", api->exit);
    symbols.resolve("libusb_get_device_list", api->get_device_list);
    symbols.resolve("libusb_free_device_list", api->free_device_list);
    symbols.resolve("libusb_get_device_descriptor", api->get_device_descriptor);
    symbols.resolve("libusb_get_bus_number", api->get_bus_number);
    symbols.resolve("libusb_get_device_address", api->get_device_address);
    symbols.resolve("libusb_open", api->open);
    symbols.resolve("libusb_close", api->close);
    symbols.resolve(
        "libusb_get_string_descriptor_ascii",
        api->get_string_descriptor_ascii
    );
    symbols.resolve(
        "libusb_set_auto_detach_kernel_driver",
        api->set_auto_detach_kernel_driver
    );
    symbols.resolve("libusb_claim_interface", api->claim_interface);
    symbols.resolve("libusb_release_interface", api->release_interface);
    symbols.resolve("libusb_control_transfer", api->control_transfer);
    symbols.resolve("libusb_bulk_transfer", api->bulk_transfer);
    symbols.resolve("libusb_error_name", api->error_name);
    if (auto err = symbols.error(LIBRARY_INFO.name)) return {nullptr, err};
    if (const int rc = api->init(&api->ctx); rc != LIBUSB_SUCCESS)
        return {
            nullptr,
            x::errors::Error(
                CRITICAL_HARDWARE_ERROR,
                std::string("libusb failed to start: ") + api->error_name(rc)
            )
        };
    return {api, x::errors::NIL};
}

ProdAPI::~ProdAPI() {
    if (this->ctx != nullptr) this->exit(this->ctx);
}

std::ptrdiff_t ProdAPI::GetDeviceList(libusb_device ***list) {
    return this->get_device_list(this->ctx, list);
}

void ProdAPI::FreeDeviceList(libusb_device **list, const int unref_devices) {
    this->free_device_list(list, unref_devices);
}

int ProdAPI::GetDeviceDescriptor(libusb_device *dev, libusb_device_descriptor *desc) {
    return this->get_device_descriptor(dev, desc);
}

std::uint8_t ProdAPI::GetBusNumber(libusb_device *dev) {
    return this->get_bus_number(dev);
}

std::uint8_t ProdAPI::GetDeviceAddress(libusb_device *dev) {
    return this->get_device_address(dev);
}

int ProdAPI::Open(libusb_device *dev, libusb_device_handle **handle) {
    return this->open(dev, handle);
}

void ProdAPI::Close(libusb_device_handle *handle) {
    this->close(handle);
}

int ProdAPI::GetStringDescriptorAscii(
    libusb_device_handle *handle,
    const std::uint8_t index,
    unsigned char *data,
    const int length
) {
    return this->get_string_descriptor_ascii(handle, index, data, length);
}

int ProdAPI::SetAutoDetachKernelDriver(libusb_device_handle *handle, const int enable) {
    return this->set_auto_detach_kernel_driver(handle, enable);
}

int ProdAPI::ClaimInterface(libusb_device_handle *handle, const int number) {
    return this->claim_interface(handle, number);
}

int ProdAPI::ReleaseInterface(libusb_device_handle *handle, const int number) {
    return this->release_interface(handle, number);
}

int ProdAPI::ControlTransfer(
    libusb_device_handle *handle,
    const std::uint8_t request_type,
    const std::uint8_t request,
    const std::uint16_t value,
    const std::uint16_t index,
    unsigned char *data,
    const std::uint16_t length,
    const unsigned int timeout
) {
    return this->control_transfer(
        handle,
        request_type,
        request,
        value,
        index,
        data,
        length,
        timeout
    );
}

int ProdAPI::BulkTransfer(
    libusb_device_handle *handle,
    const unsigned char endpoint,
    unsigned char *data,
    const int length,
    int *transferred,
    const unsigned int timeout
) {
    return this->bulk_transfer(handle, endpoint, data, length, transferred, timeout);
}

const char *ProdAPI::ErrorName(const int code) {
    return this->error_name(code);
}
}
