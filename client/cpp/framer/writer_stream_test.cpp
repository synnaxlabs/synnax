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
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/framer/framer.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::framer {
namespace {
/// @brief a writer stream that answers the open, and then ends with the given status.
class EndedStream final : public WriterStream {
    x::errors::Error status;
    mutable bool opened = false;
    bool answered = false;

public:
    explicit EndedStream(const x::errors::Error &status): status(status) {}

    std::pair<grpc::framer::WriterResponse, x::errors::Error> receive() override {
        if (this->answered) return {grpc::framer::WriterResponse(), this->status};
        this->answered = true;
        return {grpc::framer::WriterResponse(), x::errors::NIL};
    }

    x::errors::Error send(grpc::framer::WriterRequest &) const override {
        if (this->opened) return freighter::STREAM_CLOSED;
        this->opened = true;
        return x::errors::NIL;
    }

    void close_send() override {}
};

/// @brief a writer transport that opens streams that end with the given status.
class EndedStreamClient final : public WriterClient {
    x::errors::Error status;

public:
    explicit EndedStreamClient(const x::errors::Error &status): status(status) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<std::unique_ptr<WriterStream>, x::errors::Error>
    stream(const std::string &) override {
        return {std::make_unique<EndedStream>(this->status), x::errors::NIL};
    }
};

/// @brief a writer stream that accepts every send, answers the open, and then ends
/// with the given status.
class UnansweredStream final : public WriterStream {
    x::errors::Error status;
    bool answered = false;

public:
    explicit UnansweredStream(const x::errors::Error &status): status(status) {}

    std::pair<grpc::framer::WriterResponse, x::errors::Error> receive() override {
        if (this->answered) return {grpc::framer::WriterResponse(), this->status};
        this->answered = true;
        return {grpc::framer::WriterResponse(), x::errors::NIL};
    }

    x::errors::Error send(grpc::framer::WriterRequest &) const override {
        return x::errors::NIL;
    }

    void close_send() override {}
};

/// @brief a writer transport that opens streams that never answer after the open.
class UnansweredStreamClient final : public WriterClient {
    x::errors::Error status;

public:
    explicit UnansweredStreamClient(const x::errors::Error &status): status(status) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<std::unique_ptr<WriterStream>, x::errors::Error>
    stream(const std::string &) override {
        return {std::make_unique<UnansweredStream>(this->status), x::errors::NIL};
    }
};

/// @brief a client whose writers end with the given status after they open.
Client ended_client(const x::errors::Error &status) {
    return {nullptr, std::make_unique<EndedStreamClient>(status), channel::Client()};
}
}

/// @brief it should wait for the Core to acknowledge an authority change by default.
TEST(WriterStream, WaitsForAuthorityAcknowledgementByDefault) {
    ASSERT_FALSE(SetAuthorityOptions{}.unacknowledged);
    const Client client{
        nullptr,
        std::make_unique<UnansweredStreamClient>(freighter::UNREACHABLE),
        channel::Client()
    };
    auto writer = ASSERT_NIL_P(client.open_writer(WriterConfig{}));
    ASSERT_OCCURRED_AS(
        writer.set_authority(
            std::vector<channel::Key>{1},
            std::vector<x::control::Authority>{200}
        ),
        freighter::UNREACHABLE
    );
}

/// @brief it should not wait for the Core to acknowledge an unacknowledged authority
/// change.
TEST(WriterStream, SkipsAuthorityAcknowledgementWhenUnacknowledged) {
    const Client client{
        nullptr,
        std::make_unique<UnansweredStreamClient>(freighter::UNREACHABLE),
        channel::Client()
    };
    auto writer = ASSERT_NIL_P(client.open_writer(WriterConfig{}));
    ASSERT_NIL(writer.set_authority(
        std::vector<channel::Key>{1},
        std::vector<x::control::Authority>{200},
        {.unacknowledged = true}
    ));
}

/// @brief it should return the status of a stream that the Core ended, not a closed
/// stream error, when a send fails.
TEST(WriterStream, ReturnsStatusOfEndedStream) {
    const auto client = ended_client(freighter::UNREACHABLE);
    auto writer = ASSERT_NIL_P(client.open_writer(WriterConfig{}));
    ASSERT_OCCURRED_AS_P(writer.commit(), freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(writer.close(), freighter::UNREACHABLE);
}

/// @brief it should return the failed send error when the Core ended the stream with
/// no error.
TEST(WriterStream, ReturnsSendErrorOfStreamEndedWithoutError) {
    const auto client = ended_client(freighter::ERR_EOF);
    auto writer = ASSERT_NIL_P(client.open_writer(WriterConfig{}));
    ASSERT_OCCURRED_AS_P(writer.commit(), freighter::STREAM_CLOSED);
    ASSERT_OCCURRED_AS(writer.close(), freighter::STREAM_CLOSED);
}

/// @brief it should keep a permanent error that the Core ended the stream with.
TEST(WriterStream, ReturnsPermanentErrorOfEndedStream) {
    const auto client = ended_client(x::errors::VALIDATION);
    auto writer = ASSERT_NIL_P(client.open_writer(WriterConfig{}));
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.close(), x::errors::VALIDATION);
}
}
