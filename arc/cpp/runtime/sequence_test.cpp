// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/testutil/compile.h"
#include "arc/cpp/runtime/testutil/harness.h"
#include "arc/cpp/types/types.h"

/// Behavioral coverage for sequence/stage execution. Each test compiles a minimal Arc
/// program through the full text -> IR -> runtime pipeline and asserts observable
/// behavior via channel writes.
namespace arc::runtime {
/// @brief returns the final value written to a channel in a flushed frame. Fails if the
/// channel was not written.
template<typename T>
T last(const x::telem::Frame &fr, const types::ChannelKey key) {
    const x::telem::Series *found = nullptr;
    for (const auto &[ch, series]: fr)
        if (ch == key && series.size() > 0) found = &series;
    if (found == nullptr) {
        ADD_FAILURE() << "channel " << key << " not written";
        return T{};
    }
    return found->at<T>(static_cast<int>(found->size()) - 1);
}

/// @brief compiles a sequence spec and runs it through the scheduler. Every declared
/// channel is created under a generated name and substituted into the source, so
/// repeated runs against one cluster never collide.
class Sequence {
    synnax::Synnax client = new_test_client();
    testutil::Channels channels;
    testutil::Harness harness;

public:
    Sequence(
        const std::string &source,
        const std::vector<testutil::ChannelSpec> &specs
    ):
        channels(this->client, specs),
        harness(this->client, this->channels.substitute(source)) {}

    [[nodiscard]] types::ChannelKey key(const std::string &name) const {
        return this->channels.key(name);
    }

    /// @brief ticks the scheduler with the given elapsed time.
    void advance(const x::telem::TimeSpan elapsed) {
        this->harness.tick(elapsed);
        this->harness.clear_reads();
    }

    /// @brief ingests a u8=1 onto the given channel and ticks the scheduler long enough
    /// for the on-channel-read -> entry -> step cascade to settle.
    void trigger(const std::string &name) {
        this->harness.ingest(this->key(name), x::telem::Series(std::uint8_t(1)));
        for (int i = 0; i < 5; i++)
            this->advance(x::telem::MILLISECOND);
    }

    /// @brief ingests a sample onto the given channel and settles the scheduler.
    void push(const std::string &name, x::telem::Series &&data) {
        this->harness.ingest(this->key(name), std::move(data));
        for (int i = 0; i < 5; i++)
            this->advance(x::telem::MILLISECOND);
    }

    [[nodiscard]] x::telem::Frame flush() const { return this->harness.flush(); }
};

// A reassignment takes effect when its stage runs, even reached out of source order or
// after a skipped stage; guards against the old source-order chain.
namespace reactive_re_expression {
const std::string SRC = R"(
    sequence main {
        rx f32 := %rx_src% + 1
        stage rx_entry {
            rx -> %rx_out%
            %e_to_b% >= 1 => rx_b
            %e_to_c% >= 1 => rx_c
        }
        stage rx_a {
            rx = %rx_src% + 10
            rx -> %rx_out%
            %a_to_d% >= 1 => rx_d
        }
        stage rx_b {
            rx = %rx_src% + 20
            rx -> %rx_out%
        }
        stage rx_c {
            rx = %rx_src% + 30
            rx -> %rx_out%
            %c_to_a% >= 1 => rx_a
        }
        stage rx_d {
            rx = %rx_src% + 40
            rx -> %rx_out%
        }
    }
    %start_cmd% => main)";

Sequence new_h() {
    return Sequence(
        SRC,
        {{"start_cmd", x::telem::UINT8_T},
         {"rx_src", x::telem::FLOAT32_T},
         {"rx_out", x::telem::FLOAT32_T},
         {"e_to_c", x::telem::UINT8_T},
         {"e_to_b", x::telem::UINT8_T},
         {"c_to_a", x::telem::UINT8_T},
         {"a_to_d", x::telem::UINT8_T}}
    );
}

void push_src(Sequence &h, const float v) {
    h.push("rx_src", x::telem::Series(v));
}
}

TEST(ReactiveReExpressionTest, JumpsToAReExpressionReachedBySkippingEarlierStages) {
    using namespace reactive_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("e_to_c"); // entry => rx_c, skipping rx_a and rx_b
    push_src(h, 2);
    const auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 32); // rx_c: rx_src + 30
}

TEST(ReactiveReExpressionTest, ReExpressesToAnEarlierSourceStageAfterALaterOne) {
    using namespace reactive_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("e_to_c"); // entry => rx_c
    push_src(h, 2);
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 32);
    h.trigger("c_to_a"); // rx_c => rx_a (earlier in source order)
    push_src(h, 3);
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 13); // rx_a: rx_src + 10
}

TEST(ReactiveReExpressionTest, DoesNotEmitOnARebindUntilTheNextSourceValue) {
    using namespace reactive_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    push_src(h, 2);
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 3); // entry: rx_src + 1
    h.trigger("e_to_b"); // entry => rx_b rebinds rx
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("rx_out"))) << "a rebind alone must not emit";
    push_src(h, 5);
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 25); // rx_b: rx_src + 20
}

TEST(ReactiveReExpressionTest, ReEmitsAnUnchangedRecomputeOnAFreshSample) {
    using namespace reactive_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    push_src(h, 2);
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 3);
    push_src(h, 2);
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("rx_out")), 3)
        << "a fresh sample fires even when the value is unchanged";
}

namespace channel_alias_re_expression {
const std::string SRC = R"(
    sequence main {
        ra := %ch_init%
        stage r_entry {
            ra -> %ra_out%
            %e_to_c% >= 1 => r_c
        }
        stage r_a {
            ra -> %ra_out%
        }
        stage r_c {
            ra = %chc%
            ra -> %ra_out%
            %c_to_a% >= 1 => r_a
        }
    }
    %start_cmd% => main)";

Sequence new_h() {
    return Sequence(
        SRC,
        {{"start_cmd", x::telem::UINT8_T},
         {"ch_init", x::telem::UINT8_T},
         {"chc", x::telem::UINT8_T},
         {"ra_out", x::telem::UINT8_T},
         {"e_to_c", x::telem::UINT8_T},
         {"c_to_a", x::telem::UINT8_T}}
    );
}

void push_c(Sequence &h, const std::uint8_t v) {
    h.push("chc", x::telem::Series(v));
}
}

TEST(ChannelAliasReExpressionTest, ReadsARebindMadeInAStageReachedBySkippingStages) {
    using namespace channel_alias_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("e_to_c"); // r_entry => r_c, skipping r_a
    push_c(h, 42);
    const auto out = h.flush();
    // r_c: ra = chc
    EXPECT_EQ(last<std::uint8_t>(out, h.key("ra_out")), 42);
}

TEST(ChannelAliasReExpressionTest, ReadsARebindFromAnEarlierSourceStageReachedAfterIt) {
    using namespace channel_alias_re_expression;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("e_to_c"); // r_entry => r_c
    push_c(h, 42);
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("ra_out")), 42);
    h.trigger("c_to_a"); // r_c => r_a, compiled before the rebind
    push_c(h, 99);
    out = h.flush();
    // r_a reads chc's latest, not stale ch_init
    EXPECT_EQ(last<std::uint8_t>(out, h.key("ra_out")), 99);
}

TEST(ChannelAliasReExpressionTest, WritesThroughAnAliasFromAnEarlierStageAfterARebind) {
    Sequence h(
        R"(
    sequence main {
        ra := %ch_init%
        stage w_entry {
            %e_to_c% >= 1 => w_c
        }
        stage w_a {
            u8(9) -> ra
        }
        stage w_c {
            ra = %chc%
            %c_to_a% >= 1 => w_a
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"ch_init", x::telem::UINT8_T},
         {"chc", x::telem::UINT8_T},
         {"e_to_c", x::telem::UINT8_T},
         {"c_to_a", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.trigger("e_to_c"); // w_entry => w_c (ra = chc)
    h.trigger("c_to_a"); // w_c => w_a, compiled before the rebind
    const auto out = h.flush();
    // w_a writes the current binding chc, not ch_init
    EXPECT_EQ(last<std::uint8_t>(out, h.key("chc")), 9);
}

TEST(ChannelAliasReExpressionTest, ReadsTheLatestOfTwoRebindsAcrossStagesBeforeAJump) {
    Sequence h(
        R"(
    sequence main {
        ra := %ch_init%
        stage r_entry {
            %e_to_c% >= 1 => r_c
        }
        stage r_a {
            ra -> %ra_out%
        }
        stage r_c {
            ra = %chc%
            %c_to_d% >= 1 => r_d
        }
        stage r_d {
            ra = %chd%
            %d_to_a% >= 1 => r_a
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"ch_init", x::telem::UINT8_T},
         {"chc", x::telem::UINT8_T},
         {"chd", x::telem::UINT8_T},
         {"ra_out", x::telem::UINT8_T},
         {"e_to_c", x::telem::UINT8_T},
         {"c_to_d", x::telem::UINT8_T},
         {"d_to_a", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.trigger("e_to_c"); // r_entry => r_c (ra = chc)
    h.trigger("c_to_d"); // r_c => r_d (ra = chd)
    h.trigger("d_to_a"); // r_d => r_a, earliest in source
    h.push("chd", x::telem::Series(std::uint8_t(7)));
    const auto out = h.flush();
    // r_a reads chd, the last rebind, not chc or ch_init
    EXPECT_EQ(last<std::uint8_t>(out, h.key("ra_out")), 7);
}

namespace negative_literals {
x::telem::DataType chan_for(const std::string &arc_type) {
    if (arc_type == "i8") return x::telem::INT8_T;
    if (arc_type == "i16") return x::telem::INT16_T;
    if (arc_type == "i32") return x::telem::INT32_T;
    if (arc_type == "i64") return x::telem::INT64_T;
    if (arc_type == "f32") return x::telem::FLOAT32_T;
    return x::telem::FLOAT64_T;
}

template<typename T>
void expect_contains(const x::telem::Series &s, const double expected) {
    const auto vals = s.values<T>();
    const auto want = static_cast<T>(expected);
    EXPECT_TRUE(std::find(vals.begin(), vals.end(), want) != vals.end())
        << "series does not contain " << expected;
}

void assert_last(
    const x::telem::Series &s,
    const std::string &arc_type,
    const double expected
) {
    if (arc_type == "i8") return expect_contains<std::int8_t>(s, expected);
    if (arc_type == "i16") return expect_contains<std::int16_t>(s, expected);
    if (arc_type == "i32") return expect_contains<std::int32_t>(s, expected);
    if (arc_type == "i64") return expect_contains<std::int64_t>(s, expected);
    if (arc_type == "f32") return expect_contains<float>(s, expected);
    expect_contains<double>(s, expected);
}

struct TypeCase {
    std::string name;
    std::string arc_type;
    std::string literal;
    double expected;
};

struct LiteralCase {
    std::string name;
    std::string literal;
    std::int64_t want;
};

/// @brief returns every series written to the channel in a flushed frame.
std::vector<const x::telem::Series *>
written(const x::telem::Frame &fr, const types::ChannelKey key) {
    std::vector<const x::telem::Series *> out;
    for (const auto &[ch, series]: fr)
        if (ch == key) out.push_back(&series);
    return out;
}

/// @brief substitutes the declared literal into a source template.
std::string with_literal(const std::string &tmpl, const std::string &lit) {
    return testutil::replace_all(tmpl, "%lit%", lit);
}

const std::string DECLARED_SRC = R"(
    sequence main {
        a %type% := %lit%
        a -> %out%
    }
    %start_cmd% => main)";

const std::string STAGE_READ_SRC = R"(
    sequence main {
        a i64 := %lit%
        stage s {
            a -> %out%
        }
    }
    %start_cmd% => main)";

const std::string REASSIGN_SRC = R"(
    sequence main {
        a i64 := 0
        stage s {
            a = %lit%
            a -> %out%
        }
    }
    %start_cmd% => main)";
}

class NegativeLiteralTypesTest
    : public testing::TestWithParam<negative_literals::TypeCase> {};

INSTANTIATE_TEST_SUITE_P(
    Types,
    NegativeLiteralTypesTest,
    testing::Values(
        negative_literals::TypeCase{"i8", "i8", "-5", -5},
        negative_literals::TypeCase{"i16", "i16", "-5", -5},
        negative_literals::TypeCase{"i32", "i32", "-5", -5},
        negative_literals::TypeCase{"i64", "i64", "-5", -5},
        negative_literals::TypeCase{"i8_type_minimum", "i8", "-128", -128},
        negative_literals::TypeCase{"i16_type_minimum", "i16", "-32768", -32768},
        negative_literals::TypeCase{"f32", "f32", "-2.5", -2.5},
        negative_literals::TypeCase{"f64", "f64", "-2.5", -2.5}
    ),
    [](const testing::TestParamInfo<negative_literals::TypeCase> &info) {
        return info.param.name;
    }
);

TEST_P(NegativeLiteralTypesTest, ReadsBackTheDeclaredNegativeValueAcrossDataTypes) {
    using namespace negative_literals;
    const auto &p = GetParam();
    Sequence h(
        with_literal(
            testutil::replace_all(DECLARED_SRC, "%type%", p.arc_type),
            p.literal
        ),
        {{"start_cmd", x::telem::UINT8_T}, {"out", chan_for(p.arc_type)}}
    );
    h.trigger("start_cmd");
    const auto out = h.flush();
    const auto s = written(out, h.key("out"));
    ASSERT_FALSE(s.empty()) << "var channel not written";
    assert_last(*s.back(), p.arc_type, p.expected);
}

class NegativeLiteralConstantTest
    : public testing::TestWithParam<negative_literals::LiteralCase> {};

INSTANTIATE_TEST_SUITE_P(
    Literals,
    NegativeLiteralConstantTest,
    testing::Values(negative_literals::LiteralCase{"negated_literal", "-5", -5}),
    [](const testing::TestParamInfo<negative_literals::LiteralCase> &info) {
        return info.param.name;
    }
);

TEST_P(
    NegativeLiteralConstantTest,
    InitializesTheConstantSoAStageReadingItNeverSurfacesTheZeroValue
) {
    using namespace negative_literals;
    const auto &p = GetParam();
    Sequence h(
        with_literal(STAGE_READ_SRC, p.literal),
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::INT64_T}}
    );
    h.trigger("start_cmd");
    const auto out = h.flush();
    const auto s = written(out, h.key("out"));
    ASSERT_FALSE(s.empty()) << "var channel not written";
    for (const auto *ser: s)
        for (const auto v: ser->values<std::int64_t>())
            EXPECT_EQ(v, p.want)
                << "initialized constant must not glitch through its zero value";
}

class NegativeLiteralReassignTest
    : public testing::TestWithParam<negative_literals::LiteralCase> {};

INSTANTIATE_TEST_SUITE_P(
    Literals,
    NegativeLiteralReassignTest,
    testing::Values(negative_literals::LiteralCase{"negated_literal", "-100", -100}),
    [](const testing::TestParamInfo<negative_literals::LiteralCase> &info) {
        return info.param.name;
    }
);

TEST_P(NegativeLiteralReassignTest, ReassignsALiteralVariableToANegativeValue) {
    using namespace negative_literals;
    const auto &p = GetParam();
    Sequence h(
        with_literal(REASSIGN_SRC, p.literal),
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::INT64_T}}
    );
    h.trigger("start_cmd");
    const auto out = h.flush();
    const auto s = written(out, h.key("out"));
    ASSERT_FALSE(s.empty()) << "var channel not written";
    expect_contains<std::int64_t>(*s.back(), static_cast<double>(p.want));
}

TEST(StatefulFlowVariablesTest, FoldsAnUnwrittenStatefulsInitialValueIntoFlowReads) {
    Sequence h(
        R"(
    sequence main {
        x u8 $= 5
        stage s1 {
            x -> %out%
            x * 2 -> %out2%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"out2", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 5);
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out2")), 10);
}

TEST(StatefulFlowVariablesTest, InterpolatesAnUnwrittenStatefulsInitialValue) {
    Sequence h(
        R"(
    sequence main {
        x u8 $= 5
        stage s1 {
            f"x: {x}" -> %out%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "x: 5");
}

TEST(StatefulFlowVariablesTest, PersistsAWrittenStatefulAcrossStageReEntries) {
    Sequence h(
        R"(
    sequence main {
        x u8 $= 0
        stage s1 {
            x = x + 1
            x -> %out%
            %go2% => next
        }
        stage s2 {
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 1);
    h.trigger("go2");
    h.trigger("go1");
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 2);
}
}
