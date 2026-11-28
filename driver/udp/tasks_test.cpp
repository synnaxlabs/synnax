// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cstdint>
#include <map>
#include <memory>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/udp/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/bus/holders.h"
#include "driver/bus/tasks.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/common/write_task.h"
#include "driver/icd/receiver.h"
#include "driver/icd/sender.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "driver/task/task.h"
#include "driver/transport/errors.h"
#include "driver/udp/udp.h"

namespace driver::udp {
namespace {
/// @returns a channel of dt with key that index indexes.
synnax::channel::Channel channel(
    const synnax::channel::Key key,
    const x::telem::DataType &dt,
    const synnax::channel::Key index
) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.data_type = dt;
    ch.index = index;
    ch.is_index = dt == x::telem::TIMESTAMP_T;
    ch.is_virtual = index == 0 && !ch.is_index;
    return ch;
}

/// @returns an ICD of lines such as "T,12.5" whose message has the text field v.
synnax::icd::ICD lines(const synnax::icd::DelimitedTextField &v) {
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "reading";
    m.payload = synnax::icd::TextPayload{.prefix = "T", .fields = {v}};
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "sensor";
    ic.framing = synnax::icd::DelimitedFraming{};
    ic.messages = {m};
    return ic;
}

/// @brief Peer is a UDP socket on the loopback interface that plays the device.
struct Peer {
    asio::io_context ctx;
    asio::ip::udp::socket socket{
        ctx,
        asio::ip::udp::endpoint(asio::ip::make_address("127.0.0.1"), 0)
    };

    [[nodiscard]] std::uint16_t port() const {
        return this->socket.local_endpoint().port();
    }

    void send(const std::string_view text, const std::uint16_t port) {
        this->socket.send_to(
            asio::buffer(text),
            asio::ip::udp::endpoint(asio::ip::make_address("127.0.0.1"), port)
        );
    }

    /// @brief returns the text of the next datagram.
    [[nodiscard]] std::string receive() {
        std::array<char, 64> got{};
        const auto n = this->socket.receive(asio::buffer(got));
        return {got.data(), n};
    }
};

/// @returns a loopback port that no socket holds now.
std::uint16_t free_port() {
    const Peer p;
    return p.port();
}

/// @brief UDPTasks runs UDP tasks against a peer that plays the device.
class UDPTasks : public ::testing::Test {
protected:
    x::breaker::Breaker cancel;
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    Peer peer;
    std::uint16_t port = free_port();
    std::shared_ptr<Devices> devices = std::make_shared<Devices>();
    bus::Readers readers{"UDP"};
    std::shared_ptr<pipeline::testutil::WriterFactory>
        writes = std::make_shared<pipeline::testutil::WriterFactory>();
    std::unique_ptr<common::ReadTask> reader;
    std::unique_ptr<common::WriteTask> writer;

    void TearDown() override {
        if (this->reader != nullptr) this->reader->stop("", false);
        if (this->writer != nullptr) this->writer->stop("", false);
    }

    /// @returns the bus::Acquire of a device bound to port that sends to the peer.
    [[nodiscard]] bus::Acquire acquire() const {
        Device dev{.key = "device", .name = "device"};
        dev.props.local_address = "127.0.0.1";
        dev.props.local_port = this->port;
        dev.props.remote = synnax::udp::Remote{
            .host = "127.0.0.1",
            .port = this->peer.port()
        };
        return [devices = this->devices, dev] { return devices->acquire(dev); };
    }

    /// @brief creates a read task of lines whose field v writes to float64 channel 2.
    void create_read() {
        synnax::icd::DelimitedTextField v;
        v.key = x::uuid::create();
        v.name = "v";
        v.position = 1;
        const auto ic = lines(v);
        const std::map<synnax::channel::Key, synnax::channel::Channel> channels{
            {1, channel(1, x::telem::TIMESTAMP_T, 1)},
            {2, channel(2, x::telem::FLOAT64_T, 1)},
        };
        auto receiver = ASSERT_NIL_P(
            icd::Receiver::create(
                {{.icd = ic.key,
                  .messages =
                      {{.message = ic.messages[0].key,
                        .bindings = {{.field = v.key, .channel = 2}}}}}},
                {ic},
                channels,
                false
            )
        );
        synnax::task::Task t;
        t.key = x::uuid::create();
        t.name = "udp read";
        this->reader = std::make_unique<common::ReadTask>(
            t,
            this->ctx,
            common::unlimited_retry_config(t.name),
            std::make_unique<bus::Source>(
                this->acquire(),
                ASSERT_NIL_P(this->readers.claim("device", t)),
                std::move(receiver),
                std::vector{channels.at(2)},
                true
            ),
            this->writes
        );
    }

    /// @returns the last status whose message is message, or an empty status.
    [[nodiscard]] synnax::task::Status status(const std::string &message) const {
        const auto statuses = this->ctx->snapshot();
        const auto it = std::ranges::find_if(statuses, [&](const auto &st) {
            return st.message == message;
        });
        return it == statuses.end() ? synnax::task::Status{} : *it;
    }

    /// @returns the last status of the task.
    [[nodiscard]] synnax::task::Status last_status() const {
        const auto statuses = this->ctx->snapshot();
        return statuses.empty() ? synnax::task::Status{} : statuses.back();
    }
};
}

/// @brief it should read each message that the peer sends into a frame.
TEST_F(UDPTasks, ReadsMessageOfEachDatagram) {
    this->create_read();
    ASSERT_TRUE(this->reader->start("start", this->cancel));
    this->peer.send("T,12.5\n", this->port);
    this->peer.send("T,7\n", this->port);
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 2);
    ASSERT_EQ(this->writes->writes->at(0).at<double>(2, 0), 12.5);
    ASSERT_EQ(this->writes->writes->at(1).at<double>(2, 0), 7);
}

/// @brief it should send the message of a command to the remote.
TEST_F(UDPTasks, SendsCommandedMessage) {
    synnax::icd::DelimitedTextField v;
    v.key = x::uuid::create();
    v.name = "v";
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "setpoint";
    m.payload = synnax::icd::TextPayload{.fields = {v}};
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "valve";
    ic.framing = synnax::icd::DelimitedFraming{};
    ic.messages = {m};
    synnax::icd::ChannelWriteBinding b;
    b.field = v.key;
    b.channel = 2;
    auto sender = ASSERT_NIL_P(
        icd::Sender::create(
            {{.icd = ic.key, .message = {.message = m.key, .bindings = {b}}}},
            {ic},
            {{2, channel(2, x::telem::FLOAT64_T, 0)}},
            false
        )
    );
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(2, x::telem::Series(4.5));
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = "udp write";
    this->writer = std::make_unique<common::WriteTask>(
        t,
        this->ctx,
        common::unlimited_retry_config(t.name),
        std::make_unique<bus::Sink>(
            this->acquire(),
            std::move(sender),
            std::vector<synnax::channel::Key>{2}
        ),
        std::make_shared<pipeline::testutil::WriterFactory>(),
        pipeline::testutil::simple_streamer_factory(
            {2},
            std::make_shared<std::vector<x::telem::Frame>>(std::move(commands))
        )
    );
    ASSERT_TRUE(this->writer->start("start", this->cancel));
    ASSERT_EQ(this->peer.receive(), "4.5\n");
}

/// @brief it should keep the socket open for a read task on the device when a write
/// refuses a message too large for a datagram.
TEST_F(UDPTasks, ReadKeepsDatagramsAfterTooLargeWrite) {
    this->create_read();
    ASSERT_TRUE(this->reader->start("start", this->cancel));
    synnax::icd::StringJSONField blob;
    blob.key = x::uuid::create();
    blob.name = "blob";
    blob.pointer = "/blob";
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "dump";
    m.payload = synnax::icd::JSONPayload{.fields = {blob}};
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "logger";
    ic.messages = {m};
    synnax::icd::ChannelWriteBinding b;
    b.field = blob.key;
    b.channel = 3;
    auto sender = ASSERT_NIL_P(
        icd::Sender::create(
            {{.icd = ic.key, .message = {.message = m.key, .bindings = {b}}}},
            {ic},
            {{3, channel(3, x::telem::STRING_T, 0)}},
            true
        )
    );
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(3, x::telem::Series(std::vector{std::string(70000, 'x')}));
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = "udp write";
    this->writer = std::make_unique<common::WriteTask>(
        t,
        this->ctx,
        common::unlimited_retry_config(t.name),
        std::make_unique<bus::Sink>(
            this->acquire(),
            std::move(sender),
            std::vector<synnax::channel::Key>{3}
        ),
        std::make_shared<pipeline::testutil::WriterFactory>(),
        pipeline::testutil::simple_streamer_factory(
            {3},
            std::make_shared<std::vector<x::telem::Frame>>(std::move(commands))
        )
    );
    ASSERT_TRUE(this->writer->start("start", this->cancel));
    const std::string failed = "Failed to write commands";
    ASSERT_EVENTUALLY_EQ(this->status(failed).message, failed);
    const auto description = this->status(failed).description;
    EXPECT_TRUE(description.ends_with("bytes exceed the 65507 bytes of a datagram"))
        << description;
    const auto conn = ASSERT_NIL_P(this->acquire()());
    {
        auto guard = conn->lock();
        ASSERT_NIL_P(guard.transport());
        ASSERT_EQ(guard.opens(), 1);
    }
    this->peer.send("T,7\n", this->port);
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 1);
    ASSERT_EQ(this->writes->writes->at(0).at<double>(2, 0), 7);
}

/// @brief it should fail to acquire a device whose local port another device holds.
TEST(Devices, FailsOnDeviceWhosePortIsHeld) {
    Devices devices;
    Device first{.key = "first", .name = "first"};
    first.props.local_address = "127.0.0.1";
    first.props.local_port = free_port();
    const auto held = ASSERT_NIL_P(devices.acquire(first));
    Device second{.key = "second", .name = "second"};
    second.props = first.props;
    const auto [conn, err] = devices.acquire(second);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_TRUE(err.data().starts_with(
        "failed to bind UDP port 127.0.0.1:" + std::to_string(first.props.local_port) +
        ": "
    )) << err.data();
}

/// @brief it should acquire a device whose remote host does not resolve, and report the
/// failed resolve on its turn.
TEST(Devices, AcquiresDeviceWhoseRemoteDoesNotResolve) {
    Devices devices;
    Device dev{.key = "unresolved", .name = "unresolved"};
    dev.props.local_address = "127.0.0.1";
    dev.props.local_port = free_port();
    dev.props.remote = synnax::udp::Remote{.host = "nonexistent.invalid", .port = 9000};
    const auto conn = ASSERT_NIL_P(devices.acquire(dev));
    auto guard = conn->lock();
    const auto [socket, err] = guard.transport();
    EXPECT_EQ(socket, nullptr);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data().starts_with("failed to resolve nonexistent.invalid: "))
        << err.data();
}

/// @brief it should fail to configure a task with no device and no entry or target, and
/// name each.
TEST(UDPFactory, NamesMissingFields) {
    x::breaker::Breaker cancel;
    Factory factory;
    for (const auto &[type, description]:
         std::vector<std::pair<std::string, std::string>>{
             {READ_TASK_TYPE,
              "device: this field is required\nentries: at least one entry is "
              "required"},
             {WRITE_TASK_TYPE,
              "device: this field is required\ntargets: at least one target is "
              "required"},
         }) {
        const auto ctx = std::make_shared<task::MockContext>(nullptr);
        cancel.start();
        const auto [tsk, handled] = factory.configure_task(
            ctx,
            synnax::task::Task{.name = "t", .type = type},
            "start-cmd",
            cancel
        );
        cancel.stop();
        ASSERT_TRUE(handled);
        ASSERT_EQ(tsk, nullptr);
        const auto statuses = ctx->snapshot();
        ASSERT_EQ(statuses.size(), 1);
        ASSERT_EQ(statuses[0].message, "Failed to configure task");
        ASSERT_EQ(statuses[0].description, description);
    }
}
}
