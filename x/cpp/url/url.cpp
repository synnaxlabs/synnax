// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <utility>

#include "x/cpp/url/url.h"

namespace url {
/// @brief joins the two paths together to form a valid url with a trailing slash.
std::string join_paths(const std::string &a, const std::string &b) {
    // Build result starting with "/" if a is empty, otherwise use a (adding "/" if
    // needed)
    std::string result;
    if (a.empty()) {
        result = "/";
    } else {
        result = (a[0] == '/') ? a : "/" + a;
    }

    // If b is empty, just ensure trailing slash
    if (b.empty()) {
        if (result.back() != '/') result += '/';
        return result;
    }

    // Add b to result (ensuring single slash between a and b)
    if (result.back() != '/') result += '/';
    result += (b[0] == '/') ? b.substr(1) : b;

    // Ensure trailing slash
    if (result.back() != '/') result += '/';
    return result;
}

URL::URL(std::string ip, const std::uint16_t port, const std::string &path):
    ip(std::move(ip)), port(port), path(join_paths("", path)) {}

URL::URL(const std::string &address) {
    if (address.empty()) {
        ip = "";
        port = 0;
        path = "";
        return;
    }

    const auto colon = address.find(':');
    if (colon == std::string::npos) {
        ip = address;
        port = 0;
        path = "";
        return;
    }

    ip = address.substr(0, colon);
    const auto path_start = address.find('/', colon + 1);
    port = static_cast<uint16_t>(
        std::atoi(address.substr(colon + 1, path_start - colon - 1).c_str())
    );
    path = path_start != std::string::npos ? join_paths("", address.substr(path_start))
                                           : "";
}

URL URL::child(const std::string &child_path) const {
    if (child_path.empty()) return {ip, port, path};
    if (ip.empty() && port == 0) return URL(child_path);
    return {ip, port, join_paths(path, child_path)};
}

std::string URL::to_string() const {
    return ip + ":" + std::to_string(port) + path;
}

std::string URL::host_address() const {
    return ip + ":" + std::to_string(port);
}
}
