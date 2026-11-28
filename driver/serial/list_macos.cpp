// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/serial/IOSerialKeys.h>

#include "x/cpp/errors/errors.h"

#include "driver/serial/list.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @returns the string property key of service, or of its parents when options say so,
/// or nothing when none holds it.
std::optional<std::string>
property(const io_object_t service, const CFStringRef key, const IOOptionBits options) {
    const CFTypeRef value = IORegistryEntrySearchCFProperty(
        service,
        kIOServicePlane,
        key,
        kCFAllocatorDefault,
        options
    );
    if (value == nullptr) return std::nullopt;
    std::optional<std::string> out;
    std::array<char, 1024> buf{};
    if (CFGetTypeID(value) == CFStringGetTypeID() &&
        CFStringGetCString(
            static_cast<CFStringRef>(value),
            buf.data(),
            buf.size(),
            kCFStringEncodingUTF8
        ) &&
        buf[0] != '\0')
        out = std::string(buf.data());
    CFRelease(value);
    return out;
}
}

std::pair<std::vector<Entry>, x::errors::Error> list() {
    CFMutableDictionaryRef match = IOServiceMatching(kIOSerialBSDServiceValue);
    if (match == nullptr)
        return {
            {},
            x::errors::Error(transport::CRITICAL_ERROR, "failed to list serial ports")
        };
    CFDictionarySetValue(
        match,
        CFSTR(kIOSerialBSDTypeKey),
        CFSTR(kIOSerialBSDAllTypes)
    );
    io_iterator_t it = 0;
    if (const auto status = IOServiceGetMatchingServices(MACH_PORT_NULL, match, &it);
        status != KERN_SUCCESS)
        return {
            {},
            x::errors::Error(
                transport::CRITICAL_ERROR,
                "failed to list serial ports: IOKit error " + std::to_string(status)
            )
        };
    std::vector<Entry> entries;
    while (const io_object_t service = IOIteratorNext(it)) {
        if (auto path = property(service, CFSTR(kIOCalloutDeviceKey), 0))
            entries.push_back({
                .path = std::move(*path),
                .description = property(
                    service,
                    CFSTR("USB Product Name"),
                    kIORegistryIterateRecursively | kIORegistryIterateParents
                ),
            });
        IOObjectRelease(service);
    }
    IOObjectRelease(it);
    std::ranges::sort(entries, {}, &Entry::path);
    return {std::move(entries), x::errors::NIL};
}
}
