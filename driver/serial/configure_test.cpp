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

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/serial/json.gen.h"
#include "client/cpp/serial/types.gen.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/serial/serial.h"
#include "driver/task/task.h"

namespace driver::serial {
namespace {
/// @brief SerialConfigure configures serial tasks on a device of a live Core.
class SerialConfigure : public ::testing::Test {
protected:
    x::breaker::Breaker cancel;
    std::shared_ptr<synnax::Synnax> client = std::make_shared<synnax::Synnax>(
        new_test_client()
    );
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        this->client
    );
    Factory factory;
    synnax::device::Device dev;

    void SetUp() override {
        const auto rack = ASSERT_NIL_P(this->client->racks.create("serial_configure"));
        synnax::serial::Properties props;
        props.port = "/dev/synnax-missing-port";
        this->dev = synnax::device::Device{
            .key = make_unique_channel_name("serial_device"),
            .rack = rack.key,
            .location = props.port,
            .make = "serial",
            .model = "serial",
            .name = "serial device",
        };
        this->dev.properties = props.to_json();
        ASSERT_NIL(this->client->devices.create(this->dev));
    }

    /// @returns the task of a commanded configure of a read task named name with entry,
    /// or nullptr when the configure fails.
    std::unique_ptr<task::Task> configure_task(
        const synnax::icd::ReadEntry &entry,
        const std::string &name = "serial read"
    ) {
        synnax::serial::ReadConfig cfg;
        cfg.device = this->dev.key;
        cfg.entries = {entry};
        this->cancel.start();
        auto [tsk, handled] = this->factory.configure_task(
            this->ctx,
            synnax::task::Task{
                .key = x::uuid::create(),
                .name = name,
                .type = READ_TASK_TYPE,
                .config = cfg.to_json(),
            },
            "start-cmd",
            this->cancel
        );
        this->cancel.stop();
        EXPECT_TRUE(handled);
        return std::move(tsk);
    }

    /// @returns the status of a commanded configure of a read task with entry that
    /// fails.
    synnax::task::Status configure(const synnax::icd::ReadEntry &entry) {
        EXPECT_EQ(this->configure_task(entry), nullptr);
        const auto statuses = this->ctx->snapshot();
        EXPECT_FALSE(statuses.empty());
        return statuses.empty() ? synnax::task::Status{} : statuses.back();
    }

    /// @returns an entry that reads lines of one text field into the channel with key,
    /// on an ICD that it creates.
    synnax::icd::ReadEntry lines(const synnax::channel::Key key) {
        synnax::icd::DelimitedTextField v;
        v.key = x::uuid::create();
        v.name = "v";
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "reading";
        m.payload = synnax::icd::TextPayload{.fields = {v}};
        synnax::icd::ICD ic;
        ic.key = x::uuid::create();
        ic.name = "sensor";
        ic.framing = synnax::icd::DelimitedFraming{};
        ic.messages = {m};
        EXPECT_NIL(this->client->icds.create(ic));
        return {
            .icd = ic.key,
            .messages = {{
                .message = m.key,
                .bindings = {{.field = v.key, .channel = key}},
            }},
        };
    }

    /// @returns a virtual float64 channel that it creates.
    synnax::channel::Channel channel() {
        synnax::channel::Channel ch{
            .name = make_unique_channel_name("serial_value"),
            .data_type = x::telem::FLOAT64_T,
            .is_virtual = true,
        };
        EXPECT_NIL(this->client->channels.create(ch));
        return ch;
    }
};
}

/// @brief it should fail to configure an entry whose ICD does not exist, and name it.
TEST_F(SerialConfigure, NamesMissingICD) {
    const synnax::icd::ReadEntry entry{
        .icd = x::uuid::create(),
        .messages = {{.message = x::uuid::create()}},
    };
    const auto status = this->configure(entry);
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(
        status.description,
        "entry 0: ICD " + entry.icd.to_string() + " does not exist"
    );
}

/// @brief it should fail to configure a binding to a channel that does not exist, and
/// name the channel.
TEST_F(SerialConfigure, NamesChannelOfBadBinding) {
    const auto missing = this->channel().key + 1000000;
    const auto status = this->configure(this->lines(missing));
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(
        status.description,
        "message reading: channel " + std::to_string(missing) + " does not exist"
    );
}

/// @brief it should refuse a second read task on a device and name the first, and
/// configure the second once the first is gone.
TEST_F(SerialConfigure, RefusesSecondReadTaskOnDevice) {
    const auto entry = this->lines(this->channel().key);
    auto first = this->configure_task(entry, "oven");
    ASSERT_NE(first, nullptr);
    const auto status = this->configure(entry);
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(
        status.description,
        "A serial device runs one read task at a time, and the task 'oven' reads from "
        "this device. Add these entries to that task."
    );
    first.reset();
    EXPECT_NE(this->configure_task(entry), nullptr);
}
}
