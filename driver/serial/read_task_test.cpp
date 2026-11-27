// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/serial/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/icd/receiver.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/serial/read_task.h"
#include "driver/task/task.h"

namespace driver::serial {
namespace {
/// @brief returns a channel of dt with key that index indexes.
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
    return ch;
}

/// @brief Lines is an ICD of lines such as "T,12.5" whose second item writes to float64
/// channel 2, on index channel 1.
struct Lines {
    synnax::icd::ICD ic;
    synnax::icd::ReadMessage read;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels = {
        {1, channel(1, x::telem::TIMESTAMP_T, 1)},
        {2, channel(2, x::telem::FLOAT64_T, 1)},
    };

    Lines() {
        synnax::icd::DelimitedTextField v;
        v.key = x::uuid::create();
        v.name = "v";
        v.position = 1;
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "reading";
        m.payload = synnax::icd::TextPayload{.prefix = "T", .fields = {v}};
        this->ic.key = x::uuid::create();
        this->ic.name = "sensor";
        this->ic.framing = synnax::icd::DelimitedFraming{};
        this->ic.messages = {m};
        this->read = {.message = m.key, .bindings = {{.field = v.key, .channel = 2}}};
    }
};

/// @brief SerialReadTask runs read tasks on a pseudo-terminal that plays the device.
class SerialReadTask : public PtyTest {
protected:
    x::breaker::Breaker cancel;
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    std::shared_ptr<pipeline::testutil::WriterFactory>
        writes = std::make_shared<pipeline::testutil::WriterFactory>();
    std::unique_ptr<common::ReadTask> task;

    /// @brief creates the task of an entry that reads Lines on the port of props, and
    /// sends poll.
    void create(
        const synnax::serial::Properties &props,
        std::optional<synnax::icd::Poll> poll = std::nullopt
    ) {
        const Lines lines;
        auto receiver = ASSERT_NIL_P(
            icd::Receiver::create(
                {{.icd = lines.ic.key,
                  .messages = {lines.read},
                  .poll = std::move(poll)}},
                {lines.ic},
                lines.channels,
                false
            )
        );
        synnax::task::Task t;
        t.key = x::uuid::create();
        t.name = "serial read";
        const auto ports = std::make_shared<Ports>();
        auto claim = ASSERT_NIL_P(ports->claim_read(props.port, t));
        this->task = std::make_unique<common::ReadTask>(
            t,
            this->ctx,
            common::unlimited_retry_config(t.name),
            std::make_unique<Source>(
                ports,
                Device{.key = "device", .name = "device", .props = props},
                std::move(claim),
                std::move(receiver),
                std::vector{lines.channels.at(2)},
                true
            ),
            this->writes
        );
    }

    void TearDown() override {
        if (this->task != nullptr) this->task->stop("", false);
        PtyTest::TearDown();
    }

    /// @returns the last status of the task.
    [[nodiscard]] synnax::task::Status last_status() const {
        const auto statuses = this->ctx->snapshot();
        EXPECT_FALSE(statuses.empty());
        if (statuses.empty()) return {};
        return statuses.back();
    }
};
}

/// @brief it should decode a framed message from the port into a frame.
TEST_F(SerialReadTask, DecodesFramedMessageIntoFrame) {
    this->create(this->props());
    ASSERT_TRUE(this->task->start("start", this->cancel));
    this->peer_write("T,12.5\n");
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 1);
    ASSERT_EQ(this->writes->config.channels, (std::vector<synnax::channel::Key>{2, 1}));
    const auto &fr = this->writes->writes->at(0);
    ASSERT_EQ(fr.at<double>(2, 0), 12.5);
    ASSERT_GT(fr.at<std::int64_t>(1, 0), 0);
}

/// @brief it should send the poll query of an entry on its rate and read the reply.
TEST_F(SerialReadTask, SendsPollQueryOnItsRate) {
    this->create(
        this->props(),
        synnax::icd::Poll{.query = {'?'}, .rate = x::telem::Rate(5)}
    );
    ASSERT_TRUE(this->task->start("start", this->cancel));
    const x::telem::Stopwatch sw;
    ASSERT_EQ(this->peer_read(6), "?\n?\n?\n");
    ASSERT_GE(sw.elapsed(), 400 * x::telem::MILLISECOND);
    this->peer_write("T,7\n");
    ASSERT_EVENTUALLY_GE(this->writes->writes->size(), 1);
    ASSERT_EQ(this->writes->writes->at(0).at<double>(2, 0), 7);
}

/// @brief it should release the port when the task stops.
TEST_F(SerialReadTask, ReleasesPortOnStop) {
    this->create(this->props());
    ASSERT_TRUE(this->task->start("start", this->cancel));
    this->task->stop("stop", true);
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
}

/// @brief it should fail to start when the port refuses its settings, and name the
/// setting.
TEST_F(SerialReadTask, FailsStartWhenPortRefusesSettings) {
    auto props = this->props();
    props.data_bits = 9;
    this->create(props);
    ASSERT_FALSE(this->task->start("start", this->cancel));
    const auto status = this->last_status();
    ASSERT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(status.message, "Failed to start task");
    ASSERT_EQ(status.description, "data bits must be 5, 6, 7, or 8, got 9");
}

/// @brief it should start with a missing port, and warn while it tries to open it.
TEST_F(SerialReadTask, WarnsWhilePortIsMissing) {
    auto props = this->props();
    props.port = "/dev/synnax-missing-port";
    this->create(props);
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EVENTUALLY_EQ(this->last_status().message, "Failed to read data");
    const auto status = this->last_status();
    ASSERT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    ASSERT_TRUE(status.details.running);
    ASSERT_TRUE(
        status.description.starts_with("failed to open serial port " + props.port)
    ) << status.description;
}
}
