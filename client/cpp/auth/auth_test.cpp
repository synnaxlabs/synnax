// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>

#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "freighter/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "core/pkg/transport/grpc/auth/auth.pb.h"

namespace synnax::auth {
namespace {
grpc::auth::SessionCreateResponse session_response(const std::string &token) {
    grpc::auth::SessionCreateResponse res;
    res.set_token(token);
    res.mutable_session()->set_key("3f3a1b0e-7c2e-4a8f-9d6b-2a1c5e8f0b41");
    res.mutable_session()->mutable_subject()->set_type("user");
    res.mutable_session()->mutable_subject()->set_key("u");
    return res;
}

std::shared_ptr<Middleware> middleware_with(
    std::unique_ptr<SessionCreateClient> create,
    std::unique_ptr<SessionDeleteClient> del = nullptr,
    std::unique_ptr<SessionRenewClient> renew = nullptr,
    std::unique_ptr<SessionRetrieveClient> retrieve = nullptr
) {
    if (renew == nullptr)
        renew = std::make_unique<freighter::testutil::UnaryClient<
            google::protobuf::Empty,
            grpc::auth::SessionRenewResponse>>(
            grpc::auth::SessionRenewResponse(),
            x::errors::NIL
        );
    return std::make_shared<Middleware>(
        std::move(create),
        std::move(renew),
        std::move(retrieve),
        std::move(del),
        password_proof("synnax", "seldon")
    );
}
}

/// @brief it should correctly authenticate with a Synnax cluster.
TEST(TestAuth, LoginHappyPath) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    const auto mw = middleware_with(std::move(mock_create_client));
    auto mock_client = freighter::testutil::UnaryClient<int, int>{1, x::errors::NIL};
    mock_client.use(mw);
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
    ASSERT_EQ(mw->current_session().subject.type, "user");
}

/// @brief it should send a password proof when logging in.
TEST(TestAuth, LoginSendsPasswordProof) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto *create = mock_create_client.get();
    const auto mw = middleware_with(std::move(mock_create_client));
    ASSERT_NIL(mw->authenticate());
    ASSERT_EQ(create->requests.size(), 1);
    ASSERT_EQ(create->requests[0].proof().method(), "password");
    ASSERT_EQ(create->requests[0].proof().username(), "synnax");
    ASSERT_EQ(create->requests[0].proof().password(), "seldon");
}

/// @brief it should log in with an API key proof.
TEST(TestAuth, LoginSendsAPIKeyProof) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto *create = mock_create_client.get();
    const auto mw = std::make_shared<Middleware>(
        std::move(mock_create_client),
        nullptr,
        nullptr,
        nullptr,
        api_key_proof("syk_key")
    );
    ASSERT_NIL(mw->authenticate());
    ASSERT_EQ(create->requests.size(), 1);
    ASSERT_EQ(create->requests[0].proof().method(), "api_key");
    ASSERT_EQ(create->requests[0].proof().key(), "syk_key");
    ASSERT_TRUE(create->requests[0].proof().username().empty());
}

/// @brief it should return an error if credentials are invalid.
TEST(TestAuth, LoginInvalidCredentials) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(
        session_response("abc"),
        ERR_INVALID_CREDENTIALS
    );
    const auto mw = middleware_with(std::move(mock_create_client));
    auto mock_client = freighter::testutil::UnaryClient<int, int>{1, x::errors::NIL};
    mock_client.use(mw);
    auto v = 1;
    auto [r, err] = mock_client.send("", v);
    ASSERT_OCCURRED_AS(err, ERR_INVALID_CREDENTIALS);
}

/// @brief it should retry authentication if the authentication token is invalid.
TEST(TestAuth, LoginRetry) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(
        std::vector{session_response("abc"), session_response("def")},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    const auto mw = middleware_with(std::move(mock_create_client));
    auto mock_client = freighter::testutil::UnaryClient<int, int>{
        {1, 1},
        {x::errors::Error(ERR_INVALID_TOKEN, ""), x::errors::NIL}
    };
    mock_client.use(mw);
    auto v = 1;
    const auto r = ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(r, 1);
}

/// @brief it should return a session-ended error instead of retrying, and log in again
/// on the next request.
TEST(TestAuth, SessionEndedReturnsWithoutRetry) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(
        std::vector{session_response("abc"), session_response("def")},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto *create = mock_create_client.get();
    const auto mw = middleware_with(std::move(mock_create_client));
    auto mock_client = freighter::testutil::UnaryClient<int, int>{
        {1, 1},
        {x::errors::Error(ERR_SESSION_ENDED, ""), x::errors::NIL}
    };
    mock_client.use(mw);
    auto v = 1;
    ASSERT_OCCURRED_AS_P(mock_client.send("", v), ERR_SESSION_ENDED);
    ASSERT_EQ(create->requests.size(), 1);
    ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(create->requests.size(), 2);
}

/// @brief logs in again once while a request is in flight, then fails that request as
/// if its older session had ended.
class ReloginThenEnd final : public freighter::PassthroughMiddleware {
    auth::Middleware &mw;
    bool done = false;

public:
    explicit ReloginThenEnd(auth::Middleware &mw): mw(mw) {}

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        if (this->done) return next(context);
        this->done = true;
        if (const auto err = this->mw.authenticate()) return {context, err};
        return {context, x::errors::Error(ERR_SESSION_ENDED, "")};
    }
};

/// @brief it should keep a newer session when a request on an older one fails.
TEST(TestAuth, StaleFailureKeepsNewerSession) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(
        std::vector{session_response("abc"), session_response("def")},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto *create = mock_create_client.get();
    const auto mw = middleware_with(std::move(mock_create_client));
    auto mock_client = freighter::testutil::UnaryClient<int, int>{
        {1, 1},
        {x::errors::NIL, x::errors::NIL}
    };
    mock_client.use(mw);
    mock_client.use(std::make_shared<ReloginThenEnd>(*mw));
    auto v = 1;
    ASSERT_OCCURRED_AS_P(mock_client.send("", v), ERR_SESSION_ENDED);
    ASSERT_EQ(create->requests.size(), 2);
    ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(create->requests.size(), 2);
}

/// @brief it should end the session on logout and send the token with the request.
TEST(TestAuth, Logout) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto mock_delete_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionDeleteRequest,
        google::protobuf::Empty>>(google::protobuf::Empty(), x::errors::NIL);
    auto *del = mock_delete_client.get();
    const auto mw = middleware_with(
        std::move(mock_create_client),
        std::move(mock_delete_client)
    );
    ASSERT_NIL(mw->authenticate());
    const auto key = mw->current_session().key;
    ASSERT_NIL(mw->logout());
    ASSERT_EQ(del->requests.size(), 1);
    ASSERT_EQ(del->requests[0].keys_size(), 1);
    ASSERT_EQ(del->requests[0].keys(0), key);
    ASSERT_EQ(mw->current_session().key, "");
    ASSERT_NIL(mw->logout());
    ASSERT_EQ(del->requests.size(), 1);
}

class DeadlineProbe final : public freighter::PassthroughMiddleware {
public:
    std::vector<bool> deadlines;

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        deadlines.push_back(context.deadline.has_value());
        return next(context);
    }
};

/// @brief it should bound the logout request so an unanswered Core cannot hang it.
TEST(TestAuth, LogoutCarriesDeadline) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto mock_delete_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionDeleteRequest,
        google::protobuf::Empty>>(google::protobuf::Empty(), x::errors::NIL);
    auto *del = mock_delete_client.get();
    const auto mw = middleware_with(
        std::move(mock_create_client),
        std::move(mock_delete_client)
    );
    const auto probe = std::make_shared<DeadlineProbe>();
    del->use(probe);
    ASSERT_NIL(mw->authenticate());
    ASSERT_NIL(mw->logout());
    ASSERT_EQ(probe->deadlines, std::vector{true});
}

/// @brief it should forget a session the Core no longer accepts on renew, so the next
/// request logs in again instead of resending the dead token.
TEST(TestAuth, RenewForgetsEndedSession) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(
        std::vector{session_response("abc"), session_response("def")},
        std::vector<x::errors::Error>{x::errors::NIL, x::errors::NIL}
    );
    auto *create = mock_create_client.get();
    auto mock_renew_client = std::make_unique<freighter::testutil::UnaryClient<
        google::protobuf::Empty,
        grpc::auth::SessionRenewResponse>>(
        grpc::auth::SessionRenewResponse(),
        x::errors::Error(ERR_SESSION_ENDED, "")
    );
    const auto mw = middleware_with(
        std::move(mock_create_client),
        nullptr,
        std::move(mock_renew_client)
    );
    ASSERT_NIL(mw->authenticate());
    ASSERT_OCCURRED_AS(mw->renew(), ERR_SESSION_ENDED);
    ASSERT_EQ(mw->current_session().key, "");
    auto mock_client = freighter::testutil::UnaryClient<int, int>{1, x::errors::NIL};
    mock_client.use(mw);
    auto v = 1;
    ASSERT_NIL_P(mock_client.send("", v));
    ASSERT_EQ(create->requests.size(), 2);
}

/// @brief it should treat a session the Core already ended as logged out.
TEST(TestAuth, LogoutAfterSessionEnded) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto mock_delete_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionDeleteRequest,
        google::protobuf::Empty>>(
        google::protobuf::Empty(),
        x::errors::Error(ERR_SESSION_ENDED, "")
    );
    const auto mw = middleware_with(
        std::move(mock_create_client),
        std::move(mock_delete_client)
    );
    ASSERT_NIL(mw->authenticate());
    ASSERT_NIL(mw->logout());
    ASSERT_EQ(mw->current_session().key, "");
}

/// @brief it should list sessions for the requested subjects.
TEST(TestAuth, RetrieveSessions) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    grpc::auth::SessionRetrieveResponse res;
    *res.add_sessions() = session_response("abc").session();
    res.mutable_sessions(0)->set_credential("c");
    auto mock_retrieve_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionRetrieveRequest,
        grpc::auth::SessionRetrieveResponse>>(res, x::errors::NIL);
    auto *retrieve = mock_retrieve_client.get();
    const auto mw = middleware_with(
        std::move(mock_create_client),
        nullptr,
        nullptr,
        std::move(mock_retrieve_client)
    );
    const auto sessions = ASSERT_NIL_P(mw->retrieve_sessions({{"user", "u"}}));
    ASSERT_EQ(sessions.size(), 1);
    ASSERT_EQ(sessions[0].key, mw->current_session().key);
    ASSERT_EQ(sessions[0].subject.key, "u");
    ASSERT_EQ(sessions[0].credential, "c");
    ASSERT_EQ(retrieve->requests.size(), 1);
    ASSERT_EQ(retrieve->requests[0].subjects_size(), 1);
    ASSERT_EQ(retrieve->requests[0].subjects(0).type(), "user");
}

/// @brief it should keep its own session when ending another one.
TEST(TestAuth, DeleteOtherSession) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto mock_delete_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionDeleteRequest,
        google::protobuf::Empty>>(google::protobuf::Empty(), x::errors::NIL);
    auto *del = mock_delete_client.get();
    const auto mw = middleware_with(
        std::move(mock_create_client),
        std::move(mock_delete_client)
    );
    ASSERT_NIL(mw->delete_sessions({"other"}));
    ASSERT_EQ(del->requests.size(), 1);
    ASSERT_EQ(del->requests[0].keys(0), "other");
    ASSERT_NE(mw->current_session().key, "");
}

/// @brief it should forget its own session when ending it.
TEST(TestAuth, DeleteOwnSession) {
    auto mock_create_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionCreateRequest,
        grpc::auth::SessionCreateResponse>>(session_response("abc"), x::errors::NIL);
    auto mock_delete_client = std::make_unique<freighter::testutil::UnaryClient<
        grpc::auth::SessionDeleteRequest,
        google::protobuf::Empty>>(google::protobuf::Empty(), x::errors::NIL);
    const auto mw = middleware_with(
        std::move(mock_create_client),
        std::move(mock_delete_client)
    );
    ASSERT_NIL(mw->authenticate());
    ASSERT_NIL(mw->delete_sessions({mw->current_session().key}));
    ASSERT_EQ(mw->current_session().key, "");
}
}
