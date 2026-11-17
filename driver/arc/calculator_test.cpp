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
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/arc/calculator.h"

namespace driver::arc {
class CalculatorTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
    }

    /// @brief creates a virtual int64 channel.
    [[nodiscard]] synnax::channel::Channel create_base(const std::string &name) const {
        return ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name(name),
            x::telem::INT64_T
        ));
    }

    /// @brief creates a calculated int64 channel that returns expression.
    [[nodiscard]] synnax::channel::Channel
    create_calculated(const std::string &name, const std::string &expression) const {
        synnax::channel::Channel ch{
            .name = make_unique_channel_name(name),
            .data_type = x::telem::INT64_T,
            .is_virtual = true,
            .expression = "return " + expression,
        };
        EXPECT_NIL(this->client->channels.create(ch));
        return ch;
    }

    /// @brief loads a calculator for an automation that reads ch.
    [[nodiscard]] Calculator load(const synnax::channel::Channel &ch) const {
        const auto out = this->create_base("calc_out");
        synnax::arc::Arc arc = synnax::arc::AutomationArc{{
            .name = make_unique_channel_name("calculator"),
            .source = synnax::arc::TextSource{
                .text = {.raw = ch.name + " -> " + out.name}
            },
        }};
        EXPECT_NIL(this->client->arcs.create(arc));
        const auto retrieved = ASSERT_NIL_P(this->client->arcs.retrieve(
            synnax::arc::base(arc).key,
            synnax::arc::RetrieveOptions{.include_program = true}
        ));
        const ::arc::runtime::Config cfg{
            .retrieve_channels = [client = this->client](
                                     const std::vector<::arc::types::ChannelKey> &keys
                                 )
                -> std::pair<
                    std::vector<::arc::runtime::state::ChannelDigest>,
                    x::errors::Error> {
                auto [channels, err] = client->channels.retrieve(keys);
                if (err) return {{}, err};
                std::vector<::arc::runtime::state::ChannelDigest> digests;
                for (const auto &c: channels)
                    digests.push_back({c.key, c.data_type, c.index});
                return {digests, x::errors::NIL};
            },
            .report = [](const std::string &, const std::string &) {},
        };
        return ASSERT_NIL_P(
            Calculator::load(
                std::get<synnax::arc::AutomationArc>(retrieved).calculations,
                cfg,
                ::arc::runtime::scheduler::noop_handler
            )
        );
    }
};

/// @brief it should compute a calculated channel from its base input.
TEST_F(CalculatorTest, ComputesFromBaseInput) {
    const auto base = this->create_base("calc_base");
    const auto calc = this->create_calculated("calc", base.name + " * 2");
    const auto calculator = this->load(calc);
    x::telem::Frame frame(base.key, x::telem::Series(static_cast<int64_t>(3)));
    calculator.compute(frame);
    ASSERT_EQ(frame.at<int64_t>(calc.key, 0), 6);
}

/// @brief it should compute nested calculated channels in dependency order.
TEST_F(CalculatorTest, ComputesNestedInDependencyOrder) {
    const auto base = this->create_base("calc_base");
    const auto inner = this->create_calculated("calc_inner", base.name + " + 1");
    const auto outer = this->create_calculated("calc_outer", inner.name + " * 2");
    const auto calculator = this->load(outer);
    x::telem::Frame frame(base.key, x::telem::Series(static_cast<int64_t>(3)));
    calculator.compute(frame);
    ASSERT_EQ(frame.at<int64_t>(inner.key, 0), 4);
    ASSERT_EQ(frame.at<int64_t>(outer.key, 0), 8);
}

/// @brief it should compute nothing until a frame holds every input of the
/// calculation.
TEST_F(CalculatorTest, WaitsForMissingInputs) {
    const auto a = this->create_base("calc_a");
    const auto b = this->create_base("calc_b");
    const auto calc = this->create_calculated("calc_sum", a.name + " + " + b.name);
    const auto calculator = this->load(calc);
    x::telem::Frame first(a.key, x::telem::Series(static_cast<int64_t>(3)));
    calculator.compute(first);
    ASSERT_FALSE(first.contains(calc.key));
    x::telem::Frame second(b.key, x::telem::Series(static_cast<int64_t>(4)));
    calculator.compute(second);
    ASSERT_EQ(second.at<int64_t>(calc.key, 0), 7);
}

/// @brief it should compute only from the inputs it read since the last reset.
TEST_F(CalculatorTest, ResetDropsEarlierInputs) {
    const auto a = this->create_base("calc_a");
    const auto b = this->create_base("calc_b");
    const auto calc = this->create_calculated("calc_sum", a.name + " + " + b.name);
    auto calculator = this->load(calc);
    x::telem::Frame first(a.key, x::telem::Series(static_cast<int64_t>(3)));
    calculator.compute(first);
    ASSERT_NIL(calculator.reset());
    x::telem::Frame second(b.key, x::telem::Series(static_cast<int64_t>(4)));
    calculator.compute(second);
    ASSERT_FALSE(second.contains(calc.key));
}

/// @brief it should stream the base inputs in place of the calculated channels.
TEST_F(CalculatorTest, StreamsBaseInputs) {
    const auto base = this->create_base("calc_base");
    const auto other = this->create_base("calc_other");
    const auto inner = this->create_calculated("calc_inner", base.name + " + 1");
    const auto outer = this->create_calculated("calc_outer", inner.name + " * 2");
    const auto calculator = this->load(outer);
    ASSERT_EQ(
        calculator.streamed({outer.key, other.key}),
        std::vector({std::min(base.key, other.key), std::max(base.key, other.key)})
    );
}
}
