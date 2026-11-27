// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cctype>
#include <map>
#include <string>
#include <string_view>
#include <utility>

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <curl/curl.h>

#include "x/cpp/base64/base64.h"

#include "driver/http/device/device.h"

namespace driver::http::device {

/// @brief returns true if name is the Content-Type header name, in any letter case.
static bool is_content_type(const std::string_view name) {
    constexpr std::string_view CONTENT_TYPE = "content-type";
    return std::ranges::equal(name, CONTENT_TYPE, [](const char a, const char b) {
        return std::tolower(static_cast<unsigned char>(a)) == b;
    });
}

/// @brief builds a full URL from base_url + path + query params.
static std::string build_url(
    const std::string &base_url,
    const std::string &path,
    const std::map<std::string, std::string> &query_params
) {
    CURLU *u = curl_url();
    curl_url_set(u, CURLUPART_URL, base_url.c_str(), 0);
    if (!path.empty()) {
        char *base_path = nullptr;
        curl_url_get(u, CURLUPART_PATH, &base_path, 0);
        std::string joined = base_path == nullptr ? "" : base_path;
        curl_free(base_path);
        if (joined.ends_with('/')) joined.pop_back();
        if (!path.starts_with('/')) joined += '/';
        joined += path;
        curl_url_set(u, CURLUPART_PATH, joined.c_str(), 0);
    }
    for (const auto &[k, v]: query_params) {
        const auto param = k + "=" + v;
        curl_url_set(
            u,
            CURLUPART_QUERY,
            param.c_str(),
            CURLU_APPENDQUERY | CURLU_URLENCODE
        );
    }
    char *url = nullptr;
    curl_url_get(u, CURLUPART_URL, &url, 0);
    std::string result(url);
    curl_free(url);
    curl_url_cleanup(u);
    return result;
}

bool valid_url(const std::string &url) {
    CURLU *u = curl_url();
    const bool valid = curl_url_set(u, CURLUPART_URL, url.c_str(), 0) == CURLUE_OK;
    curl_url_cleanup(u);
    return valid;
}

Request build_request(const ConnectionConfig &conn, const RequestConfig &req) {
    std::map<std::string, std::string> query_params = req.query_params;
    if (conn.auth.type == "api_key" && conn.auth.send_as == "query_param")
        query_params[conn.auth.parameter] = conn.auth.key;

    Request r;
    r.url = build_url(conn.base_url, req.path, query_params);
    r.base_url = conn.base_url;
    r.max_concurrent_requests = conn.max_concurrent_requests;
    r.max_response_size = conn.max_response_size;
    r.method = req.method;
    r.timeout = conn.timeout;
    r.verify_ssl = conn.verify_ssl;

    // Merge headers into a map first to handle deduplication, then format.
    std::map<std::string, std::string> headers = req.headers;

    if (conn.auth.type == "bearer") {
        headers["Authorization"] = "Bearer " + conn.auth.token;
    } else if (conn.auth.type == "basic") {
        headers["Authorization"] = "Basic " +
                                   x::base64::encode(
                                       conn.auth.username + ":" + conn.auth.password
                                   );
    } else if (conn.auth.type == "api_key" && conn.auth.send_as != "query_param") {
        headers[conn.auth.header] = conn.auth.key;
    }

    if (!req.request_content_type.empty() &&
        std::ranges::none_of(headers, [](const auto &h) {
            return is_content_type(h.first);
        }))
        headers["Content-Type"] = req.request_content_type;

    r.headers.reserve(headers.size());
    for (const auto &[k, v]: headers)
        r.headers.push_back(k + ": " + v);

    return r;
}

std::pair<ConnectionConfig, x::errors::Error> retrieve_connection(
    const synnax::device::Client &devices,
    const std::string &device_key
) {
    auto [dev, dev_err] = devices.retrieve(device_key);
    if (dev_err)
        return {
            ConnectionConfig(x::json::Parser(x::json::json::object())),
            dev_err,
        };
    auto props = x::json::json(dev.properties);
    const bool secure = props.value("secure", true);
    const std::string protocol = secure ? "https://" : "http://";
    props["base_url"] = protocol + dev.location;
    auto parser = x::json::Parser(props);
    auto conn = ConnectionConfig(parser);
    return {std::move(conn), parser.error()};
}

}
