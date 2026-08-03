// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
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
    uint32_t *out_index_keys,
    char *out_dtypes,
    const size_t out_dtypes_size,
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
        const std::vector<std::string> req = split_newlines(names);
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

        std::unordered_map<std::string, const synnax::channel::Channel *> by_name;
        by_name.reserve(channels.size());
        for (const auto &ch: channels)
            by_name[ch.name] = &ch;

        // retrieve omits unmatched names, so map by name and collect every miss to
        // report them together (missing names get key 0 and an empty data type).
        std::string missing;
        std::string types;
        for (size_t i = 0; i < name_count; i++) {
            if (i > 0) types += '\n';
            const auto it = by_name.find(req[i]);
            if (it == by_name.end()) {
                out_keys[i] = 0;
                if (out_index_keys != nullptr) out_index_keys[i] = 0;
                if (!missing.empty()) missing += ", ";
                missing += "\"" + req[i] + "\"";
            } else {
                out_keys[i] = it->second->key;
                if (out_index_keys != nullptr) out_index_keys[i] = it->second->index;
                types += it->second->data_type.name();
            }
        }
        if (out_dtypes != nullptr) {
            if (types.size() + 1 > out_dtypes_size) {
                set_err(
                    err,
                    CODE_INTERNAL,
                    "sy.validation",
                    "out_dtypes too small: need " + std::to_string(types.size() + 1) +
                        " bytes, have " + std::to_string(out_dtypes_size)
                );
                return CODE_INTERNAL;
            }
            copy_str(out_dtypes, out_dtypes_size, types);
        }
        if (!missing.empty()) {
            const auto nf = synnax::errors::not_found_error("channels", missing);
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

int32_t synnax_channel_create(
    SynnaxClient *client,
    const char *names,
    const char *data_types,
    const uint8_t *is_index,
    const uint32_t *index,
    const uint8_t *is_virtual,
    const size_t count,
    uint32_t *out_keys,
    SynnaxError *err
) {
    clear_err(err);
    if (client == nullptr || out_keys == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null client or out_keys");
        return CODE_INTERNAL;
    }
    if (count == 0) {
        set_err(
            err,
            CODE_INTERNAL,
            "sy.validation",
            "create requires at least one channel"
        );
        return CODE_INTERNAL;
    }
    try {
        const std::vector<std::string> name_list = split_newlines(names);
        const std::vector<std::string> dtype_list = split_newlines(data_types);
        if (name_list.size() != count || dtype_list.size() != count) {
            set_err(
                err,
                CODE_INTERNAL,
                "sy.validation",
                "parsed " + std::to_string(name_list.size()) + " names and " +
                    std::to_string(dtype_list.size()) + " data types but count is " +
                    std::to_string(count)
            );
            return CODE_INTERNAL;
        }
        std::vector<synnax::channel::Channel> chans;
        chans.reserve(count);
        for (size_t i = 0; i < count; i++) {
            synnax::channel::Channel ch;
            ch.name = name_list[i];
            ch.data_type = x::telem::DataType{dtype_list[i]};
            ch.is_index = is_index != nullptr && is_index[i] != 0;
            ch.index = index != nullptr ? index[i] : 0;
            ch.is_virtual = is_virtual != nullptr && is_virtual[i] != 0;
            chans.push_back(std::move(ch));
        }
        const auto c_err = client->client.channels.create(chans);
        if (!c_err.ok()) {
            set_err(err, CODE_ERROR, c_err.type, c_err.data);
            return CODE_ERROR;
        }
        for (size_t i = 0; i < count; i++) {
            if (chans[i].key == 0) {
                set_err(
                    err,
                    CODE_ERROR,
                    "sy.unexpected",
                    "server created fewer channels than requested"
                );
                return CODE_ERROR;
            }
            out_keys[i] = chans[i].key;
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
