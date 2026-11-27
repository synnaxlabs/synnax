// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <map>
#include <optional>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "driver/serial/errors.h"
#include "driver/serial/list.h"

namespace driver::serial {
namespace fs = std::filesystem;

namespace {
/// @returns the first line of the file at path, or nothing when it is missing or empty.
std::optional<std::string> first_line(const fs::path &path) {
    std::ifstream in(path);
    std::string line;
    if (!std::getline(in, line) || line.empty()) return std::nullopt;
    return line;
}

/// @returns the product string of the USB device that holds device, or nothing when no
/// USB device under root holds it.
std::optional<std::string> product(const fs::path &root, fs::path device) {
    std::error_code ec;
    for (; device != root && device.has_relative_path(); device = device.parent_path())
        if (fs::exists(device / "idVendor", ec)) return first_line(device / "product");
    return std::nullopt;
}

/// @returns true when tty is a UART port in which the kernel found no UART, such as the
/// spare 8250 ports that it makes at each boot.
bool empty_uart(const fs::path &tty) {
    return first_line(tty / "type") == "0";
}
}

std::pair<std::vector<Entry>, x::errors::Error>
list(const fs::path &sys, const fs::path &dev) {
    std::error_code ec;
    std::map<std::string, std::string> links;
    for (const auto &link: fs::directory_iterator(dev / "serial" / "by-id", ec)) {
        std::error_code target_ec;
        const auto target = fs::canonical(link.path(), target_ec);
        if (!target_ec)
            links.try_emplace(target.filename().string(), link.path().string());
    }
    if (ec && ec != std::errc::no_such_file_or_directory)
        return {{}, translate("failed to list serial ports", ec)};
    const auto root = fs::canonical(sys, ec);
    if (ec) return {{}, translate("failed to list serial ports", ec)};
    std::vector<Entry> entries;
    for (const auto &tty: fs::directory_iterator(root / "class" / "tty", ec)) {
        std::error_code device_ec;
        const auto device = fs::canonical(tty.path() / "device", device_ec);
        if (device_ec || empty_uart(tty.path())) continue;
        const auto name = tty.path().filename().string();
        const auto link = links.find(name);
        entries.push_back({
            .path = link != links.end() ? link->second : (dev / name).string(),
            .description = product(root, device),
        });
    }
    if (ec) return {{}, translate("failed to list serial ports", ec)};
    std::ranges::sort(entries, {}, &Entry::path);
    return {std::move(entries), x::errors::NIL};
}

std::pair<std::vector<Entry>, x::errors::Error> list() {
    return list("/sys", "/dev");
}
}
