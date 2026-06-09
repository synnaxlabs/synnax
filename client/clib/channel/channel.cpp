// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

#include "client/clib/channel/channel.h"
#include "client/clib/internal.h"
#include "client/cpp/errors/errors.h"

using namespace synnax::clib;

int32_t synnax_channel_retrieve_keys(
    SynnaxClient *client,
    const char *names,
    const size_t name_count,
    uint32_t *out_keys,
    SynnaxError *err
) {
    clear_err(err);
    if (client == nullptr || names == nullptr || out_keys == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null client, names, or out_keys");
        return CODE_INTERNAL;
    }
    try {
        // LabVIEW's CLFN has no array-of-C-string-pointers type, so names arrives as a
        // single '\n'-delimited string (Array To Spreadsheet String); split it back
        // out.
        std::vector<std::string> req;
        req.reserve(name_count);
        std::stringstream ss(str_or(names, ""));
        std::string token;
        while (std::getline(ss, token, '\n'))
            req.push_back(token);
        if (req.size() != name_count) {
            set_err(
                err,
                CODE_INTERNAL,
                "sy.validation",
                "parsed " + std::to_string(req.size()) + " names but name_count is " +
                    std::to_string(name_count)
            );
            return CODE_INTERNAL;
        }

        auto [channels, c_err] = client->client.channels.retrieve(req);
        if (!c_err.ok()) {
            set_err(err, CODE_ERROR, c_err.type, c_err.data);
            return CODE_ERROR;
        }

        std::unordered_map<std::string, uint32_t> by_name;
        by_name.reserve(channels.size());
        for (const auto &ch: channels)
            by_name[ch.name] = ch.key;

        // retrieve omits unmatched names, so map by name and collect every miss to
        // report them together (missing names get key 0).
        std::string missing;
        for (size_t i = 0; i < name_count; i++) {
            const auto it = by_name.find(req[i]);
            if (it == by_name.end()) {
                out_keys[i] = 0;
                if (!missing.empty()) missing += ", ";
                missing += req[i];
            } else
                out_keys[i] = it->second;
        }
        if (!missing.empty()) {
            const auto nf = synnax::errors::not_found_error(
                "channels",
                "names " + missing
            );
            set_err(err, CODE_ERROR, nf.type, nf.data);
            return CODE_ERROR;
        }
        return CODE_OK;
    } catch (const std::exception &e) {
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}
