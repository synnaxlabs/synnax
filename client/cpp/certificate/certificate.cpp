// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/certificate/certificate.h"
#include "client/cpp/errors/errors.h"
#include "x/cpp/errors/errors.h"

namespace synnax::certificate {
namespace {
std::pair<grpc::certificate::Input, x::errors::Error>
to_input(const Certificate &certificate, const std::string &private_key_pem) {
    grpc::certificate::Input in;
    in.set_private_key_pem(private_key_pem);
    if (const auto *stored = std::get_if<StoredCertificate>(&certificate)) {
        if (!stored->key.is_nil()) in.set_key(stored->key.to_string());
        in.set_name(stored->name);
        in.set_source(stored->source);
        in.set_pem(stored->pem);
        return {std::move(in), x::errors::NIL};
    }
    const auto &file = std::get<FileCertificate>(certificate);
    if (!file.key.is_nil()) in.set_key(file.key.to_string());
    in.set_name(file.name);
    in.set_source(file.source);
    in.set_certificate_path(file.certificate_path);
    if (file.private_key_file.has_value())
        in.set_private_key_file(*file.private_key_file);
    if (file.details.has_value()) {
        auto [details, err] = file.details->to_proto();
        if (err) return {grpc::certificate::Input{}, err};
        *in.mutable_details() = std::move(details);
    }
    in.set_rack(file.rack);
    return {std::move(in), x::errors::NIL};
}
}

Client::Client(
    std::shared_ptr<RetrieveClient> retrieve_client,
    std::shared_ptr<SetClient> set_client,
    std::shared_ptr<RefreshClient> refresh_client
):
    retrieve_client(std::move(retrieve_client)),
    set_client(std::move(set_client)),
    refresh_client(std::move(refresh_client)) {}

std::pair<Certificate, x::errors::Error> Client::retrieve(const Key &key) const {
    auto [certificates, err] = this->retrieve(std::vector{key});
    if (err) return {Certificate{}, err};
    if (const auto one_err = errors::check_one(
            "certificate",
            Where{.key = key}.to_json(),
            "",
            certificates.size()
        ))
        return {Certificate{}, one_err};
    return {std::move(certificates[0]), x::errors::NIL};
}

std::pair<std::vector<Certificate>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    auto [req, pb_err] = RetrieveRequest::from(Where{.key = keys}).to_proto();
    if (pb_err) return {{}, pb_err};
    auto [res, err] = this->retrieve_client->send("/certificate/retrieve", req);
    if (err) return {{}, err};
    auto [decoded, decode_err] = RetrieveResponse::from_proto(res);
    if (decode_err) return {{}, decode_err};
    return {std::move(decoded.certificates), x::errors::NIL};
}

x::errors::Error
Client::set(Certificate &certificate, const std::string &private_key_pem) const {
    auto [in, in_err] = to_input(certificate, private_key_pem);
    if (in_err) return in_err;
    grpc::certificate::SetRequest req;
    *req.add_certificates() = std::move(in);
    auto [res, err] = this->set_client->send("/certificate/set", req);
    if (err) return err;
    if (res.certificates_size() == 0)
        return errors::unexpected_missing_error("certificate");
    auto [saved, decode_err] = certificate_from_proto(res.certificates(0));
    if (decode_err) return decode_err;
    certificate = std::move(saved);
    return x::errors::NIL;
}

x::errors::Error Client::refresh(const Key &key, const Details &details) const {
    grpc::certificate::RefreshRequest req;
    req.set_key(key.to_string());
    auto [pb, pb_err] = details.to_proto();
    if (pb_err) return pb_err;
    *req.mutable_details() = std::move(pb);
    return this->refresh_client->send("/certificate/refresh", req).second;
}
}
