// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <utility>

/// external
#include "freighter/cpp/freighter.h"

using namespace freighter;

/// @brief joins the two paths together to form a valid url with a trailing slash.
std::string join_paths(const std::string &a, const std::string &b) {
    if (a.empty() && b.empty()) return "";
    auto adjusted = b[0] == '/' ? b.substr(1) : b;
    adjusted = b[b.size() - 1] == '/' ? b : b + "/";
    return a + adjusted;
}

URL::URL(
    std::string ip,
    const std::uint16_t port,
    const std::string &path
) : ip(std::move(ip)), port(port),
    path(join_paths("", path)) {
}

URL::URL(const std::string &address) {
    const auto colon = address.find(':');
    ip = address.substr(0, colon);
    const auto path_start = address.find('/');
    port = std::stoi(address.substr(colon + 1, path_start - colon - 1));
    if (path_start != std::string::npos) path = join_paths("", address.substr(path_start));
}

URL URL::child(const std::string &child_path) const {
    if (child_path.empty()) return {ip, port, path};
    if (ip.empty() && port == 0) return URL(child_path);
    return {ip, port, join_paths(path, child_path)};
}

std::string URL::to_string() const {
    return ip + ":" + std::to_string(port) + path;
}
