// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <atomic>
#include <chrono>
#include <functional>
#include <mutex>
#include <string>
#include <vector>

#include "google/protobuf/empty.pb.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/transport/grpc/auth/auth.pb.h"

namespace synnax::auth {
/// @brief auth metadata key. NOTE: This must be lowercase, GRPC will panic on
/// capitalized or uppercase keys.
const std::string HEADER_KEY = "authorization";
/// @brief auth token prefix that will be parsed by the cluster.
const std::string HEADER_VALUE_PREFIX = "Bearer ";

/// @brief transport that opens a session from a proof.
using SessionCreateClient = freighter::
    UnaryClient<grpc::auth::SessionCreateRequest, grpc::auth::SessionCreateResponse>;
/// @brief transport that reports activity on the caller's session.
using SessionRenewClient = freighter::
    UnaryClient<google::protobuf::Empty, grpc::auth::SessionRenewResponse>;
/// @brief transport that lists sessions.
using SessionRetrieveClient = freighter::UnaryClient<
    grpc::auth::SessionRetrieveRequest,
    grpc::auth::SessionRetrieveResponse>;
/// @brief transport that ends sessions.
using SessionDeleteClient = freighter::
    UnaryClient<grpc::auth::SessionDeleteRequest, google::protobuf::Empty>;

const x::errors::Error ERR = x::errors::SY.sub("auth");
const x::errors::Error ERR_INVALID_TOKEN = ERR.sub("invalid_token");
const x::errors::Error ERR_EXPIRED_TOKEN = ERR.sub("expired_token");
const x::errors::Error ERR_SESSION_ENDED = ERR.sub("session_ended");
const x::errors::Error ERR_INVALID_CREDENTIALS = ERR.sub("invalid-credentials");
/// @brief errors after which a request logs in again and retries.
const std::vector RETRY_ON_ERRORS = {ERR_INVALID_TOKEN, ERR_EXPIRED_TOKEN};
/// @brief errors meaning the Core no longer holds the session.
const std::vector ENDED_ERRORS = {
    ERR_INVALID_TOKEN,
    ERR_EXPIRED_TOKEN,
    ERR_SESSION_ENDED
};

/// @brief the subject a session belongs to: a user or a rack.
struct Subject {
    /// @brief the resource type of the subject.
    std::string type;
    /// @brief the key of the subject within its type.
    std::string key;
};

/// @brief a session a login opened, without its token.
struct Session {
    /// @brief the unique key of the session.
    std::string key;
    /// @brief the subject that logged in.
    Subject subject;
    /// @brief the key of the credential the login matched.
    std::string credential;
    /// @brief the last time a request used the session.
    x::telem::TimeStamp last_active = x::telem::TimeStamp(0);

    Session() = default;

    explicit Session(const grpc::auth::Session &s):
        key(s.key()),
        subject{s.subject().type(), s.subject().key()},
        credential(s.credential()),
        last_active(s.last_active()) {}
};

/// @brief auth::Middleware logs in with a username and password, holds the session,
/// and attaches its token to every request. Middleware has no preference on order when
/// provided to use. It is safe to use concurrently.
class Middleware final : public freighter::PassthroughMiddleware {
    /// @brief how long a logout waits for the Core before giving up. The client's
    /// destructor logs out, so an unanswered request must not hang it.
    static constexpr auto LOGOUT_TIMEOUT = std::chrono::seconds(5);

    /// @brief adds the token to the session requests the Middleware sends itself.
    class TokenHeader final : public freighter::PassthroughMiddleware {
        std::function<std::string()> token;

    public:
        explicit TokenHeader(std::function<std::string()> token):
            token(std::move(token)) {}

        std::pair<freighter::Context, x::errors::Error>
        operator()(freighter::Context context, freighter::Next &next) override {
            context.set(HEADER_KEY, HEADER_VALUE_PREFIX + this->token());
            return next(context);
        }
    };

    class Deadline final : public freighter::PassthroughMiddleware {
        std::chrono::milliseconds timeout;

    public:
        explicit Deadline(const std::chrono::milliseconds timeout): timeout(timeout) {}

        std::pair<freighter::Context, x::errors::Error>
        operator()(freighter::Context context, freighter::Next &next) override {
            context.deadline = std::chrono::system_clock::now() + this->timeout;
            return next(context);
        }
    };

    std::string token;
    /// Valid only when authenticated.
    Session session;
    /// Whether the middleware has successfully authenticated with the server.
    std::atomic<bool> authenticated = false;
    std::unique_ptr<SessionCreateClient> create_client;
    std::unique_ptr<SessionRenewClient> renew_client;
    std::unique_ptr<SessionRetrieveClient> retrieve_client;
    std::unique_ptr<SessionDeleteClient> delete_client;
    /// Username to be used for authentication.
    std::string username;
    /// Password to be used for authentication.
    std::string password;
    /// Guards token and session.
    std::mutex mu;

    std::string current_token() {
        std::lock_guard lock(mu);
        return this->token;
    }

    /// Clears the session only if it still holds token, so a late failure from an
    /// older session does not erase a newer one.
    void forget(const std::string &token) {
        std::lock_guard lock(mu);
        if (this->token != token) return;
        this->authenticated = false;
        this->token.clear();
        this->session = Session();
    }

public:
    Middleware(
        std::unique_ptr<SessionCreateClient> create_client,
        std::unique_ptr<SessionRenewClient> renew_client,
        std::unique_ptr<SessionRetrieveClient> retrieve_client,
        std::unique_ptr<SessionDeleteClient> delete_client,
        std::string username,
        std::string password
    ):
        create_client(std::move(create_client)),
        renew_client(std::move(renew_client)),
        retrieve_client(std::move(retrieve_client)),
        delete_client(std::move(delete_client)),
        username(std::move(username)),
        password(std::move(password)) {
        const auto header = std::make_shared<TokenHeader>([this] {
            return this->current_token();
        });
        if (this->renew_client) this->renew_client->use(header);
        if (this->retrieve_client) this->retrieve_client->use(header);
        if (this->delete_client) {
            this->delete_client->use(header);
            this->delete_client->use(std::make_shared<Deadline>(LOGOUT_TIMEOUT));
        }
    }

    /// @brief opens a session with the credentials provided when constructing the
    /// Synnax client.
    x::errors::Error authenticate() {
        std::lock_guard lock(mu);
        grpc::auth::SessionCreateRequest req;
        req.mutable_proof()->set_method("password");
        req.mutable_proof()->set_username(this->username);
        req.mutable_proof()->set_password(this->password);
        auto [res, err] = create_client->send("/auth/session/create", req);
        if (err) return err;
        this->token = res.token();
        this->session = Session(res.session());
        this->authenticated = true;
        return x::errors::NIL;
    }

    /// @brief returns the session the client holds. Only valid once authenticated.
    Session current_session() {
        std::lock_guard lock(mu);
        return this->session;
    }

    /// @brief reports activity on the session so it does not idle out. A session the
    /// Core no longer accepts is forgotten, and the next request logs in again.
    x::errors::Error renew() {
        if (!this->authenticated)
            if (const auto err = this->authenticate()) return err;
        const auto sent = this->current_token();
        google::protobuf::Empty req;
        auto [_, err] = renew_client->send("/auth/session/renew", req);
        if (err.matches(ENDED_ERRORS)) this->forget(sent);
        return err;
    }

    /// @brief lists sessions: the client's own when subjects is empty, otherwise the
    /// sessions of the given subjects. Returns an access-denied error when the client
    /// may not retrieve another subject.
    std::pair<std::vector<Session>, x::errors::Error>
    retrieve_sessions(const std::vector<Subject> &subjects = {}) {
        if (!this->authenticated)
            if (const auto err = this->authenticate()) return {{}, err};
        const auto sent = this->current_token();
        grpc::auth::SessionRetrieveRequest req;
        for (const auto &subject: subjects) {
            auto *s = req.add_subjects();
            s->set_type(subject.type);
            s->set_key(subject.key);
        }
        auto [res, err] = retrieve_client->send("/auth/session/retrieve", req);
        if (err) {
            if (err.matches(ENDED_ERRORS)) this->forget(sent);
            return {{}, err};
        }
        std::vector<Session> sessions;
        sessions.reserve(res.sessions_size());
        for (const auto &s: res.sessions())
            sessions.emplace_back(s);
        return {sessions, x::errors::NIL};
    }

    /// @brief ends the given sessions. Unknown keys are ignored. Returns an
    /// access-denied error when the client may not end another subject's sessions.
    /// Ending the client's own session forgets it, and a later request logs in again.
    x::errors::Error delete_sessions(const std::vector<std::string> &keys) {
        if (!this->authenticated)
            if (const auto err = this->authenticate()) return err;
        const auto sent = this->current_token();
        const auto own = this->current_session().key;
        grpc::auth::SessionDeleteRequest req;
        for (const auto &key: keys)
            req.add_keys(key);
        auto [_, err] = delete_client->send("/auth/session/delete", req);
        if (err.matches(ENDED_ERRORS)) this->forget(sent);
        if (err) return err;
        if (std::ranges::find(keys, own) != keys.end()) this->forget(sent);
        return x::errors::NIL;
    }

    /// @brief ends the client's own session and forgets it. A later request logs in
    /// again. Does nothing when no session is open, and a session the Core already
    /// ended counts as logged out.
    x::errors::Error logout() {
        if (!this->authenticated) return x::errors::NIL;
        const auto sent = this->current_token();
        grpc::auth::SessionDeleteRequest req;
        req.add_keys(this->current_session().key);
        auto [_, err] = delete_client->send("/auth/session/delete", req);
        if (err && !err.matches(ENDED_ERRORS)) return err;
        this->forget(sent);
        return x::errors::NIL;
    }

    /// @brief implements freighter::Middleware, ensuring that all requests to the
    /// Synnax Core are appropriately authenticated.
    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        if (!this->authenticated)
            if (const auto err = this->authenticate()) return {context, err};
        const auto sent = this->current_token();
        context.set(HEADER_KEY, HEADER_VALUE_PREFIX + sent);
        auto [res_ctx, err] = next(context);
        // A session ending under an open stream closes that stream for good, so it
        // goes back to the caller to reopen.
        if (err.matches(ERR_SESSION_ENDED)) {
            this->forget(sent);
            return {res_ctx, err};
        }
        if (err.matches(RETRY_ON_ERRORS)) {
            this->forget(sent);
            return this->operator()(context, next);
        }
        return {res_ctx, err};
    }
};
}
