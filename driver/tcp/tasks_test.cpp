// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <map>
#include <memory>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/tcp/types.gen.h"
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
#include "driver/tcp/tcp.h"
#include "driver/transport/errors.h"

namespace driver::tcp {
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

/// @brief Server is a TCP server on the loopback interface that plays the device.
struct Server {
    asio::io_context ctx;
    asio::ip::tcp::acceptor acceptor{
        ctx,
        asio::ip::tcp::endpoint(asio::ip::make_address("127.0.0.1"), 0)
    };
    asio::ip::tcp::socket peer{ctx};

    /// @brief accepts the next connection as peer, and closes the last one.
    void accept() {
        if (this->peer.is_open()) this->peer.close();
        this->acceptor.accept(this->peer);
    }

    void write(const std::string_view text) {
        asio::write(this->peer, asio::buffer(text));
    }

    /// @brief reads exactly n bytes from peer.
    [[nodiscard]] std::string read(const std::size_t n) {
        std::string out(n, '\0');
        asio::read(this->peer, asio::buffer(out));
        return out;
    }
};

/// @brief TCPTasks runs TCP tasks against a server that plays the device.
class TCPTasks : public ::testing::Test {
protected:
    x::breaker::Breaker cancel;
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    Server server;
    std::shared_ptr<Devices> devices = std::make_shared<Devices>();
    bus::Readers readers{"TCP"};
    std::shared_ptr<pipeline::testutil::WriterFactory>
        writes = std::make_shared<pipeline::testutil::WriterFactory>();
    std::unique_ptr<common::ReadTask> reader;
    std::unique_ptr<common::WriteTask> writer;

    void TearDown() override {
        if (this->reader != nullptr) this->reader->stop("", false);
        if (this->writer != nullptr) this->writer->stop("", false);
    }

    /// @returns the bus::Acquire of the device on the server.
    [[nodiscard]] bus::Acquire acquire() const {
        Device dev{.key = "device", .name = "device"};
        dev.props.host = "127.0.0.1";
        dev.props.port = this->server.acceptor.local_endpoint().port();
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
        t.name = "tcp read";
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

    /// @returns the last status of the task.
    [[nodiscard]] synnax::task::Status last_status() const {
        const auto statuses = this->ctx->snapshot();
        return statuses.empty() ? synnax::task::Status{} : statuses.back();
    }
};
}

/// @brief it should read a framed message from the server into a frame.
TEST_F(TCPTasks, ReadsFramedMessage) {
    this->create_read();
    ASSERT_TRUE(this->reader->start("start", this->cancel));
    this->server.accept();
    this->server.write("T,12.5\n");
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 1);
    ASSERT_EQ(this->writes->writes->at(0).at<double>(2, 0), 12.5);
}

/// @brief it should warn when the server closes the connection, and read again once it
/// connects.
TEST_F(TCPTasks, ReconnectsAfterServerCloses) {
    this->create_read();
    ASSERT_TRUE(this->reader->start("start", this->cancel));
    this->server.accept();
    this->server.write("T,5\n");
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 1);
    this->server.peer.close();
    ASSERT_EVENTUALLY_EQ(this->last_status().message, "Failed to read data");
    ASSERT_EQ(this->last_status().variant, synnax::status::VARIANT_WARNING);
    this->server.accept();
    this->server.write("T,7\n");
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 2);
    ASSERT_EQ(this->writes->writes->at(1).at<double>(2, 0), 7);
}

/// @brief it should send the message of a command to the server.
TEST_F(TCPTasks, SendsCommandedMessage) {
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
    t.name = "tcp write";
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
    this->server.accept();
    ASSERT_EQ(this->server.read(4), "4.5\n");
}

/// @brief it should acquire a device whose server does not answer, and fail on a device
/// with no host.
TEST(Devices, FailsOnlyOnDeviceThatCannotConnect) {
    Devices devices;
    Device closed{.key = "closed", .name = "closed"};
    closed.props.host = "127.0.0.1";
    {
        Server server;
        closed.props.port = server.acceptor.local_endpoint().port();
    }
    ASSERT_NIL_P(devices.acquire(closed));
    Device hostless{.key = "hostless", .name = "hostless"};
    hostless.props.port = 502;
    const auto [conn, err] = devices.acquire(hostless);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    ASSERT_EQ(err.data(), "the host is empty");
}

/// @brief it should fail to configure a task with no device and no entry or target, and
/// name each.
TEST(TCPFactory, NamesMissingFields) {
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
