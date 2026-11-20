// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "x/cpp/fs/fs.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/x509.h"

#include "driver/secret/secret.h"

namespace driver::secret {
namespace {
/// @returns VALIDATION when item is a file of rack, and the Driver runs another rack.
x::errors::Error
check_rack(const std::string &item, const synnax::rack::Key rack, const Config &cfg) {
    if (rack == cfg.rack) return x::errors::NIL;
    return x::errors::Error(
        x::errors::VALIDATION,
        "Cannot read " + item + ": it is a file of rack " + std::to_string(rack) +
            ", and this Driver runs rack " + std::to_string(cfg.rack) + "."
    );
}
}

Reader::Reader(std::shared_ptr<synnax::Synnax> client, Config cfg):
    client(std::move(client)), cfg(std::move(cfg)) {}

std::pair<std::string, x::errors::Error>
Reader::read_file(const std::string &item, const std::string &name) const {
    if (this->cfg.secrets_dir.empty())
        return {
            "",
            x::errors::Error(
                x::errors::VALIDATION,
                "Cannot read " + item +
                    ": it is a file, and the Driver has no secrets folder. Set "
                    "secrets_dir in the Driver's config."
            )
        };
    auto [data, err] = x::fs::read_file((this->cfg.secrets_dir / name).string());
    if (err)
        return {"", x::errors::Error(err, "Failed to read " + item + ": " + err.data)};
    return {std::move(data), x::errors::NIL};
}

std::pair<synnax::secret::Values, x::errors::Error> Reader::read(
    const std::vector<synnax::secret::Key> &keys,
    const std::optional<synnax::secret::Grant> &grant
) const {
    if (grant.has_value())
        for (const auto &key: keys)
            if (!std::ranges::contains(grant->secrets, key))
                return {
                    {},
                    x::errors::Error(
                        x::errors::VALIDATION,
                        "The grant does not name secret " + key.to_string() + "."
                    )
                };
    auto [secrets, err] = this->client->secrets.retrieve(keys);
    if (err) return {{}, err};
    for (const auto &key: keys)
        if (std::ranges::none_of(secrets, [&](const auto &s) {
                return std::visit([](const auto &v) { return v.key; }, s) == key;
            }))
            return {
                {},
                x::errors::Error(
                    x::errors::NOT_FOUND,
                    "Found no secret with key " + key.to_string() + "."
                )
            };
    synnax::secret::Values values;
    std::vector<synnax::secret::Key> stored;
    for (const auto &s: secrets) {
        if (const auto *file = std::get_if<synnax::secret::FileSecret>(&s)) {
            const std::string item = "secret \"" + file->name + "\"";
            if (const auto rack_err = check_rack(item, file->rack, this->cfg))
                return {{}, rack_err};
            auto [value, file_err] = this->read_file(item, file->file_name);
            if (file_err) return {{}, file_err};
            values[file->key] = x::strings::trim_trailing_line_breaks(std::move(value));
            continue;
        }
        const auto &key = std::get<synnax::secret::StoredSecret>(s).key;
        if (!grant.has_value()) {
            stored.push_back(key);
            continue;
        }
        const auto it = grant->values.find(key);
        if (it == grant->values.end())
            return {
                {},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "The grant does not hold secret " + key.to_string() + "."
                )
            };
        values[key] = it->second;
    }
    if (stored.empty()) return {std::move(values), x::errors::NIL};
    auto [resolved, resolve_err] = this->client->secrets.resolve(stored);
    if (resolve_err) return {{}, resolve_err};
    values.merge(resolved);
    return {std::move(values), x::errors::NIL};
}

std::pair<std::optional<synnax::secret::Grant>, x::errors::Error>
Reader::spend(const synnax::task::Command &cmd) const {
    const auto arg = cmd.args.find("grant");
    if (arg == cmd.args.end()) return {std::nullopt, x::errors::NIL};
    const x::errors::Error invalid(
        x::errors::VALIDATION,
        "The grant argument must be the key of a grant."
    );
    if (!arg->second.is_string()) return {std::nullopt, invalid};
    auto [key, parse_err] = x::uuid::UUID::parse(arg->second.get<std::string>());
    if (parse_err) return {std::nullopt, invalid};
    auto [grant, err] = this->client->secrets.resolve(key, cmd.task);
    if (err) return {std::nullopt, err};
    grant.command.key = cmd.key;
    return {std::move(grant), x::errors::NIL};
}

std::pair<Certificate, x::errors::Error> Reader::read_certificate(
    const synnax::certificate::Key &key,
    const std::optional<synnax::secret::Grant> &grant
) const {
    auto [record, err] = this->client->certificates.retrieve(key);
    if (err) return {Certificate{}, err};
    if (const auto *stored = std::get_if<synnax::certificate::StoredCertificate>(
            &record
        )) {
        Certificate out{.pem = stored->pem};
        if (!stored->private_key.has_value()) return {std::move(out), x::errors::NIL};
        auto [values, read_err] = this->read({*stored->private_key}, grant);
        if (read_err) return {Certificate{}, read_err};
        out.private_key = std::move(values[*stored->private_key]);
        return {std::move(out), x::errors::NIL};
    }
    const auto &file = std::get<synnax::certificate::FileCertificate>(record);
    if (const auto rack_err = check_rack(
            "certificate \"" + file.name + "\"",
            file.rack,
            this->cfg
        ))
        return {Certificate{}, rack_err};
    Certificate out;
    if (file.private_key_file.has_value()) {
        auto [private_key, key_err] = this->read_file(
            "the private key of certificate \"" + file.name + "\"",
            *file.private_key_file
        );
        if (key_err) return {Certificate{}, key_err};
        out.private_key = std::move(private_key);
    }
    auto [data, read_err] = x::fs::read_file(file.certificate_path);
    if (read_err)
        return {
            Certificate{},
            x::errors::Error(
                read_err,
                "Failed to read certificate \"" + file.name + "\": " + read_err.data
            )
        };
    auto [cert, parse_err] = x::x509::Certificate::parse(data);
    if (parse_err)
        return {
            Certificate{},
            x::errors::Error(
                parse_err,
                "Failed to read certificate \"" + file.name + "\": " + parse_err.data
            )
        };
    const auto blocks = x::x509::Certificate::pem_blocks(data);
    if (blocks.empty()) out.pem = cert.pem();
    for (const auto &block: blocks)
        out.pem.append(block).push_back('\n');
    if (file.details.has_value() && file.details->sha256 == cert.sha256())
        return {std::move(out), x::errors::NIL};
    const synnax::certificate::Details details{
        .subject = cert.subject(),
        .issuer = cert.issuer(),
        .not_before = cert.not_before(),
        .not_after = cert.not_after(),
        .sha1 = cert.sha1(),
        .sha256 = cert.sha256(),
    };
    if (const auto refresh_err = this->client->certificates.refresh(key, details))
        return {Certificate{}, refresh_err};
    return {std::move(out), x::errors::NIL};
}
}
