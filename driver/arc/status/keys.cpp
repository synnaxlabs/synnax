// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <limits>

#include "x/cpp/telem/telem.h"

#include "driver/arc/status/keys.h"

namespace driver::arc::status {
namespace {
std::filesystem::path temp_path(const std::filesystem::path &path) {
    auto tmp = path;
    tmp += ".tmp";
    return tmp;
}

/// @brief encodes a name and its key as a u32 little-endian name size, the name,
/// and the key.
std::string encode(const std::string &name, const std::string &key) {
    const auto size = static_cast<std::uint32_t>(name.size());
    std::string out(4, '\0');
    for (int i = 0; i < 4; i++)
        out[i] = static_cast<char>((size >> (8 * i)) & 0xFF);
    return out + name + key;
}

std::pair<std::pair<std::string, std::string>, x::errors::Error>
decode(const std::string &payload) {
    if (payload.size() < 4)
        return {{}, x::errors::Error(journal::CORRUPT, "status key record too short")};
    std::uint32_t size = 0;
    for (int i = 0; i < 4; i++)
        size |= static_cast<std::uint32_t>(static_cast<unsigned char>(payload[i]))
             << (8 * i);
    if (size > payload.size() - 4)
        return {{}, x::errors::Error(journal::CORRUPT, "status key name too long")};
    return {
        {payload.substr(4, size), payload.substr(4 + size)},
        x::errors::NIL,
    };
}

x::errors::Error fs_error(const std::string &action, const std::error_code &ec) {
    return x::errors::Error(journal::BASE_ERROR, action + ": " + ec.message());
}
}

Keys::Keys(
    std::filesystem::path path,
    std::string cluster,
    const std::size_t max,
    x::fs::OnCreate on_create
):
    path(std::move(path)),
    cluster(std::move(cluster)),
    max(max),
    on_create(std::move(on_create)) {}

void Keys::put(const std::string &name, const std::string &key) {
    if (const auto it = this->index.find(name); it != this->index.end())
        this->entries.erase(it->second);
    this->entries.emplace_front(name, key);
    this->index[name] = this->entries.begin();
    if (this->entries.size() <= this->max) return;
    this->index.erase(this->entries.back().first);
    this->entries.pop_back();
}

std::pair<std::shared_ptr<Keys>, x::errors::Error> Keys::open(
    const std::filesystem::path &path,
    const std::string &cluster,
    const std::size_t max,
    x::fs::OnCreate on_create
) {
    auto keys = std::shared_ptr<Keys>(
        new Keys(path, cluster, max, std::move(on_create))
    );
    std::error_code ec;
    std::filesystem::remove(temp_path(path), ec);
    if (ec)
        return {nullptr, fs_error("failed to remove " + temp_path(path).string(), ec)};
    // A writer creates the temp through a temp of its own.
    if (auto err = x::fs::remove_temps(temp_path(path))) return {nullptr, err};
    if (!std::filesystem::exists(path, ec)) {
        if (ec) return {nullptr, fs_error("failed to read " + path.string(), ec)};
        return {keys, x::errors::NIL};
    }
    // A save syncs the whole file before it takes the place of the old one, so any
    // damage is corruption.
    auto [reader, err] = journal::Reader::open({
        .path = path,
        .magic = KEYS_MAGIC,
        .types = {KEY_TYPE, CLUSTER_TYPE},
    });
    if (err) return {nullptr, err};
    bool same_cluster = false;
    while (true) {
        auto [record, next_err] = reader->next();
        if (next_err) return {nullptr, next_err};
        if (!record) break;
        if (record->type == CLUSTER_TYPE) {
            same_cluster = record->payload == cluster;
            continue;
        }
        auto [entry, decode_err] = decode(record->payload);
        if (decode_err) return {nullptr, decode_err};
        keys->put(entry.first, entry.second);
    }
    if (reader->cut() > 0)
        return {
            nullptr,
            x::errors::Error(
                journal::CORRUPT,
                path.string() + " is cut off after byte " +
                    std::to_string(reader->pos())
            )
        };
    // Another cluster gave the keys, so its keys name no status of this one.
    if (!same_cluster) {
        keys->entries.clear();
        keys->index.clear();
    }
    return {keys, x::errors::NIL};
}

x::errors::Error
Keys::save(const std::vector<std::pair<std::string, std::string>> &entries) const {
    const auto tmp = temp_path(this->path);
    std::error_code ec;
    std::filesystem::remove(tmp, ec);
    if (ec) return fs_error("failed to remove " + tmp.string(), ec);
    auto [writer, err] = journal::Writer::open({
        .path = tmp,
        .magic = KEYS_MAGIC,
        .sync_interval = x::telem::TimeSpan(std::numeric_limits<std::int64_t>::max()),
        .sync_bytes = std::numeric_limits<std::uint64_t>::max(),
        .on_create = this->on_create,
    });
    if (err) return err;
    if (err = writer->write({.type = CLUSTER_TYPE, .payload = this->cluster}); err)
        return err;
    for (const auto &[name, key]: entries)
        if (err = writer->write({.type = KEY_TYPE, .payload = encode(name, key)}); err)
            return err;
    if (err = writer->close(); err) return err;
    if (err = x::fs::replace(tmp, this->path); err) return err;
    const auto dir = this->path.parent_path();
    return x::fs::sync_dir(dir.empty() ? "." : dir);
}

std::string Keys::get(const std::string &name) {
    std::lock_guard lock(this->mu);
    const auto it = this->index.find(name);
    if (it == this->index.end()) return "";
    this->entries.splice(this->entries.begin(), this->entries, it->second);
    return it->second->second;
}

x::errors::Error Keys::set(const std::string &name, const std::string &key) {
    std::lock_guard save_lock(this->save_mu);
    std::vector<std::pair<std::string, std::string>> entries;
    {
        std::lock_guard lock(this->mu);
        if (const auto it = this->index.find(name);
            it != this->index.end() && it->second->second == key) {
            this->entries.splice(this->entries.begin(), this->entries, it->second);
            return x::errors::NIL;
        }
        this->put(name, key);
        // The least recently used name goes first, so a load puts the last one first.
        entries.assign(this->entries.rbegin(), this->entries.rend());
    }
    return this->save(entries);
}
}
