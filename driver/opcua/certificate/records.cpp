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
#include <variant>

#include "driver/opcua/certificate/records.h"

namespace driver::opcua::certificate {
namespace {
const std::string CURRENT_SUFFIX = " Driver certificate";
const std::string NEXT_SUFFIX = " next Driver certificate";

synnax::certificate::Details details_of(const x::x509::Certificate &cert) {
    return {
        .subject = cert.subject(),
        .issuer = cert.issuer(),
        .not_before = cert.not_before(),
        .not_after = cert.not_after(),
        .sha1 = cert.sha1(),
        .sha256 = cert.sha256(),
    };
}
}

Records::Records(
    std::shared_ptr<Store> store,
    std::shared_ptr<synnax::Synnax> client,
    const synnax::rack::Key rack
):
    store(std::move(store)), client(std::move(client)), rack(rack) {}

x::errors::Error Records::sync(const std::string &rack_name) {
    auto [ids, err] = this->store->list();
    if (err) return err;
    std::vector<synnax::certificate::FileCertificate> wanted;
    std::vector<std::pair<std::string, std::string>> state;
    for (std::size_t i = 0; i < ids.size(); i++) {
        synnax::certificate::FileCertificate file;
        file.name = rack_name + (i == 0 ? CURRENT_SUFFIX : NEXT_SUFFIX);
        file.certificate_path = this->store->cert_path(ids[i]).string();
        file.details = details_of(ids[i].certificate);
        file.rack = this->rack;
        state.emplace_back(file.name, file.certificate_path);
        wanted.push_back(std::move(file));
    }
    if (this->synced == state) return x::errors::NIL;
    auto [records, retrieve_err] = this->client->certificates.retrieve(
        synnax::certificate::Where{.racks = this->rack}
    );
    if (retrieve_err) return retrieve_err;
    std::vector<synnax::certificate::Key> stale;
    for (const auto &record: records) {
        const auto *file = std::get_if<synnax::certificate::FileCertificate>(&record);
        if (file == nullptr ||
            std::filesystem::path(file->certificate_path).parent_path() !=
                this->store->certs_dir())
            continue;
        const auto match = std::ranges::find(
            wanted,
            file->certificate_path,
            &synnax::certificate::FileCertificate::certificate_path
        );
        if (match == wanted.end())
            stale.push_back(file->key);
        else
            match->key = file->key;
    }
    if (!stale.empty())
        if (const auto del_err = this->client->certificates.del(stale)) return del_err;
    for (auto &file: wanted) {
        synnax::certificate::Certificate c = std::move(file);
        if (const auto set_err = this->client->certificates.set(c)) return set_err;
    }
    this->synced = std::move(state);
    return x::errors::NIL;
}
}
