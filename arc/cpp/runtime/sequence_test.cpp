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
#include <iterator>
#include <string>
#include <type_traits>
#include <utility>
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

    /// @brief ticks the scheduler long enough for the on-channel-read -> entry -> step
    /// cascade to settle.
    void settle() {
        for (int i = 0; i < 5; i++)
            this->advance(x::telem::MILLISECOND);
    }

    /// @brief ingests a sample onto the given channel.
    void ingest(const std::string &name, x::telem::Series &&data) {
        this->harness.ingest(this->key(name), std::move(data));
    }

    /// @brief ingests a u8=1 onto the given channel and ticks the scheduler long enough
    /// for the on-channel-read -> entry -> step cascade to settle.
    void trigger(const std::string &name) {
        this->ingest(name, x::telem::Series(std::uint8_t(1)));
        this->settle();
    }

    /// @brief ingests a sample onto the given channel and settles the scheduler.
    void push(const std::string &name, x::telem::Series &&data) {
        this->ingest(name, std::move(data));
        this->settle();
    }

    [[nodiscard]] x::telem::Frame flush() const { return this->harness.flush(); }
};

/// @brief returns every sample in a series as T.
template<typename T>
std::vector<T> samples(const x::telem::Series &s) {
    if constexpr (std::is_same_v<T, std::string>)
        return s.strings();
    else
        return s.values<T>();
}

/// @brief collects every value a flushed frame wrote to a channel, in order.
template<typename T>
std::vector<T> collect(const x::telem::Frame &fr, const types::ChannelKey key) {
    std::vector<T> out;
    for (const auto &[ch, series]: fr) {
        if (ch != key) continue;
        for (auto &v: samples<T>(series))
            out.push_back(std::move(v));
    }
    return out;
}

/// @brief returns how many samples on a channel equal want.
int count_of(
    const x::telem::Frame &fr,
    const types::ChannelKey key,
    const std::string &want
) {
    int n = 0;
    for (const auto &v: collect<std::string>(fr, key))
        if (v == want) n++;
    return n;
}

/// @brief appends every sample a flushed frame wrote to the given channel.
template<typename T>
void drain(const Sequence &h, const std::string &name, std::vector<T> &got) {
    auto vals = collect<T>(h.flush(), h.key(name));
    got.insert(
        got.end(),
        std::make_move_iterator(vals.begin()),
        std::make_move_iterator(vals.end())
    );
}

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

TEST(
    VariableInitializationOnDeclarationTest,
    InitializesAVariableDeclaredInASequenceBody
) {
    Sequence h(
        R"(
    sequence s {
        my_var := "hello"
        my_var -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "hello");
}

TEST(
    VariableInitializationOnDeclarationTest,
    InitializesAVariableDeclaredInAStageBody
) {
    Sequence h(
        R"(
    sequence s {
        stage read {
            my_var := "hello"
            my_var -> %out%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "hello");
}

TEST(
    VariableInitializationOnDeclarationTest,
    ReflectsAFlowWriteToAnInitializedVariable
) {
    Sequence h(
        R"(
    sequence s {
        my_var := "hello"
        "updated" -> my_var
        my_var -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "updated");
}

TEST(
    VariableInitializationOnDeclarationTest,
    SharesASequenceScopedVariableWithANestedStage
) {
    Sequence h(
        R"(
    sequence s {
        my_var := "initial"
        stage write {
            "updated" -> my_var
            my_var -> %out%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "updated");
}

// Declarations are bindings, not steps: they must not block step advance.
TEST(VariableInitializationOnDeclarationTest, AdvancesPastSequenceScopedDeclarations) {
    Sequence h(
        R"(
    sequence s {
        my_var := "my_var"
        1 -> %out%
        other := "other"
        2 -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 2);
}

TEST(ReactiveVariableReadsTest, LogsAVariableOncePerChangeAcrossStageReActivations) {
    Sequence h(
        R"(import time
    sequence main {
        stage s1 {
            counter $= 0
            1 => counter + 1 => counter
            str(counter) => %log%
            time.wait{100ms} => next
        }
        stage s2 {
            1 => s1
        }
    }
    1 => main)",
        {{"log", x::telem::STRING_T}}
    );

    // Two scheduler passes settle each wait -> s2 -> s1 re-entry; the logged value
    // lags the increment by one activation, so activations log 0, 1, 2, ...
    const auto step = [&](const x::telem::TimeSpan now) {
        h.advance(now);
        h.advance(now);
    };
    step(x::telem::TimeSpan(0));
    step(100 * x::telem::MILLISECOND);
    step(200 * x::telem::MILLISECOND);
    step(300 * x::telem::MILLISECOND);

    const auto out = h.flush();
    const auto logged = collect<std::string>(out, h.key("log"));
    EXPECT_EQ(logged, (std::vector<std::string>{"0", "1", "2", "3", "4"}));
}

TEST(ReactiveVariableReadsTest, ReEmitsRepeatedDerivationsOfTheSameValue) {
    Sequence h(
        R"(
    stage {
        x := %cpu% * 2
        x -> %out%
    })",
        {{"cpu", x::telem::UINT8_T}, {"out", x::telem::UINT8_T}}
    );

    for (int i = 0; i < 4; i++)
        h.ingest("cpu", x::telem::Series(std::uint8_t(5)));
    h.settle();
    const auto out = h.flush();
    EXPECT_EQ(collect<std::uint8_t>(out, h.key("out")).size(), size_t{4})
        << "every fresh sample fires even when the value is unchanged";
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 10);
}

TEST(ReactiveVariableReadsTest, ReFiresADerivationOnEveryFreshSample) {
    Sequence h(
        R"(
    stage {
        x := %cpu% * 2
        x -> %out%
    })",
        {{"cpu", x::telem::UINT8_T}, {"out", x::telem::UINT8_T}}
    );

    std::vector<std::uint8_t> got;
    const auto feed = [&](const std::uint8_t v) {
        h.push("cpu", x::telem::Series(v));
        drain(h, "out", got);
    };
    feed(5);
    feed(5);
    feed(7);
    feed(7);
    feed(5);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{10, 10, 14, 14, 10}));
}

TEST(ReactiveVariableReadsTest, ReEmitsRepeatedFormatStringDerivations) {
    Sequence h(
        R"(
    stage {
        x := "v: " + str(%cpu%)
        x -> %out%
    })",
        {{"cpu", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    std::vector<std::string> got;
    const auto feed = [&](const std::uint8_t v) {
        h.push("cpu", x::telem::Series(v));
        drain(h, "out", got);
    };
    feed(5);
    feed(5);
    feed(7);
    EXPECT_EQ(got, (std::vector<std::string>{"v: 5", "v: 5", "v: 7"}));
}

TEST(ReactiveVariableReadsTest, DerivesAChannelReadVariableAndReadsIt) {
    Sequence h(
        R"(
    stage {
        x := %cpu% * 2
        x -> %out%
        x + 1 -> %out2%
    })",
        {{"cpu", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"out2", x::telem::UINT8_T}}
    );

    h.push("cpu", x::telem::Series(std::uint8_t(5)));
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 10);
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out2")), 11);

    h.push("cpu", x::telem::Series(std::uint8_t(7)));
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 14);
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out2")), 15);
}

TEST(VariableReassignmentTest, ReflectsAReassignmentToAnInitializedVariable) {
    Sequence h(
        R"(
    sequence s {
        my_var := "hello"
        my_var = "updated"
        my_var -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "updated");
}

TEST(VariableReassignmentTest, ReassignsAVariableInsideAStageBody) {
    Sequence h(
        R"(
    sequence s {
        my_var := "initial"
        stage write {
            my_var = "updated"
            my_var -> %out%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "updated");
}

TEST(VariableReassignmentTest, ReassignsASequenceDeclaredVariableFromInsideAStage) {
    Sequence h(
        R"(
    sequence s {
        my_var := "top"
        stage write {
            my_var = "updated"
            my_var -> %out%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "updated");
}

TEST(
    VariableReassignmentTest,
    IncrementsAVariableViaSelfReferentialReassignmentExactlyOnce
) {
    Sequence h(
        R"(
    sequence s {
        counter u8 := 5
        counter = counter + 1
        counter -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 6);
}

TEST(VariableReassignmentTest, RedirectsWritesThroughAnAliasAcrossARebind) {
    Sequence h(
        R"(
    sequence s {
        sink := %out_a%
        u8(1) -> sink
        sink = %out_b%
        u8(2) -> sink
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out_a", x::telem::UINT8_T},
         {"out_b", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_a")), 1);
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_b")), 2);
}

TEST(VariableReassignmentTest, ReadsThroughAnAliasThatFollowsARebind) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor_a%
        src = %sensor_b%
        src -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor_a", x::telem::UINT8_T},
         {"sensor_b", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor_a", x::telem::Series(std::uint8_t(3)));
    h.ingest("sensor_b", x::telem::Series(std::uint8_t(7)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 7);
}

TEST(VariableReassignmentTest, RebindsAnAliasInsideAStageBody) {
    Sequence h(
        R"(
    sequence s {
        stage w {
            sink := %out_a%
            sink = %out_b%
            %trig% -> sink
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"trig", x::telem::UINT8_T},
         {"out_a", x::telem::UINT8_T},
         {"out_b", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("trig", x::telem::Series(std::uint8_t(2)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_b")), 2);
}

TEST(
    VariableReassignmentTest,
    WalksMultipleSequentialReadAliasRebindsToTheFinalBinding
) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor_a%
        src = %sensor_b%
        src = %sensor_c%
        src -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor_a", x::telem::UINT8_T},
         {"sensor_b", x::telem::UINT8_T},
         {"sensor_c", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor_a", x::telem::Series(std::uint8_t(1)));
    h.ingest("sensor_b", x::telem::Series(std::uint8_t(2)));
    h.ingest("sensor_c", x::telem::Series(std::uint8_t(3)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 3);
}

TEST(
    VariableReassignmentTest,
    WalksMultipleSequentialWriteAliasRebindsToTheFinalBinding
) {
    Sequence h(
        R"(
    sequence s {
        sink := %out_a%
        sink = %out_b%
        sink = %out_c%
        u8(9) -> sink
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out_a", x::telem::UINT8_T},
         {"out_b", x::telem::UINT8_T},
         {"out_c", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_c")), 9);
    EXPECT_FALSE(out.contains(h.key("out_a"))) << "original target must not be written";
    EXPECT_FALSE(out.contains(h.key("out_b")))
        << "intermediate target must not be written";
}

TEST(VariableReassignmentTest, RebindsAnAliasToTheSameChannelWithoutCrashing) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor%
        src = %sensor%
        src -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor", x::telem::Series(std::uint8_t(5)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 5);
}

TEST(
    VariableReassignmentTest,
    ReadsThroughAnAliasBeforeItsSourceHasDataWithoutCrashing
) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor%
        src -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.advance(x::telem::MILLISECOND);
    EXPECT_TRUE(h.flush().empty()) << "no source data means nothing should be emitted";
}

TEST(
    VariableReassignmentTest,
    WritesThroughAnAliasBoundToAnUnbackedChannelWithoutCrashing
) {
    Sequence h(
        R"(
    sequence s {
        sink := %ghost%
        u8(1) -> sink
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"ghost", x::telem::UINT8_T}}
    );

    EXPECT_NO_THROW(h.trigger("start_cmd"));
}

TEST(VariableReassignmentTest, SwitchesAChannelReadVariablesExpressionOnReExpression) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        r = %in_val% + u8(100)
        r -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("in_val", x::telem::Series(std::uint8_t(5)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 105);
}

TEST(VariableReassignmentTest, AdvancesThroughMultipleReExpressionsToTheFinalFeeder) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        r = %in_val% + u8(10)
        r = %in_val% + u8(100)
        r -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("in_val", x::telem::Series(std::uint8_t(5)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 105);
}

TEST(VariableReassignmentTest, ReGatesAReactiveReaderOnAFreshInputAfterReExpression) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        r -> %out%
        r = %in_val% + u8(100)
        r -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    std::vector<std::uint8_t> got;
    const auto step = [&](const std::uint8_t val) {
        h.push("in_val", x::telem::Series(val));
        drain(h, "out", got);
    };

    h.trigger("start_cmd");
    drain(h, "out", got);
    // first reader: in=10 -> r=11
    step(10);
    // second reader (post re-expr): in=100 -> r=200, must not surface stale 11
    step(100);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{11, 200}));
}

TEST(VariableReassignmentTest, ReFiresOnAnEqualInputValueAfterReExpression) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        r -> %out%
        r = %in_val% + u8(100)
        r -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("in_val", x::telem::Series(std::uint8_t(2)));
    h.push("in_val", x::telem::Series(std::uint8_t(2)));
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 102);
}

TEST(VariableReassignmentTest, ComputesAnExpressionOverAReboundAlias) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor_a%
        src = %sensor_b%
        src + u8(1) -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor_a", x::telem::UINT8_T},
         {"sensor_b", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor_a", x::telem::Series(std::uint8_t(3)));
    h.ingest("sensor_b", x::telem::Series(std::uint8_t(7)));
    h.settle();
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 8);
}

TEST(VariableReassignmentTest, ComputesAnExpressionMixingAStaticAndAReboundAlias) {
    Sequence h(
        R"(
    sequence s {
        a := %sensor_a%
        b := %sensor_b%
        b = %sensor_c%
        a + b -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor_a", x::telem::UINT8_T},
         {"sensor_b", x::telem::UINT8_T},
         {"sensor_c", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor_a", x::telem::Series(std::uint8_t(1)));
    h.ingest("sensor_c", x::telem::Series(std::uint8_t(5)));
    h.settle();
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 6);
}

TEST(VariableReassignmentTest, DoesNotReplayADerivationIntoAReEnteredStage) {
    Sequence h(
        R"(
    sequence main {
        r := %in_val% + u8(1)
        stage s1 {
            r -> %out%
            %go2% => next
        }
        stage s2 {
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    std::vector<std::uint8_t> got;
    h.trigger("start_cmd");
    h.push("in_val", x::telem::Series(std::uint8_t(5)));
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{6}));

    h.trigger("go2"); // s1 -> s2
    h.trigger("go1"); // s2 -> s1 re-entry
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{6}))
        << "re-entry must not replay the stale value";

    h.push("in_val", x::telem::Series(std::uint8_t(9)));
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{6, 10}));
}

TEST(VariableReassignmentTest, SwallowsADerivationValuePendingAtTheRePoint) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        u8(1) -> %gate%
        r = %in_val% + u8(100)
        r -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"gate", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    // Data lands on the same tick the sequence enters, so the pre-rebind derivation
    // value is pending exactly when the re-point arrives.
    h.ingest("start_cmd", x::telem::Series(std::uint8_t(1)));
    h.ingest("in_val", x::telem::Series(std::uint8_t(5)));
    for (int i = 0; i < 6; i++)
        h.advance(x::telem::MILLISECOND);
    auto out = h.flush();
    EXPECT_FALSE(out.contains(h.key("out")))
        << "a value predating the re-point must not fire";

    h.push("in_val", x::telem::Series(std::uint8_t(7)));
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 107);
}

TEST(VariableReassignmentTest, ReflectsAStageReassignmentOnReEnteringAReaderStage) {
    Sequence h(
        R"(
    sequence main {
        my_var u8 := 1
        stage s1 {
            my_var -> %out%
            %go2% => next
        }
        stage s2 {
            my_var = 2
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

TEST(VariableReassignmentTest, IncrementsAVariableOncePerStageEntry) {
    Sequence h(
        R"(
    sequence main {
        count u8 := 0
        stage s1 {
            count + 1 -> count
            count -> %out%
            %go2% => next
        }
        stage s2 {
            count = count + 1
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    std::vector<std::uint8_t> got;
    // Increments fire once per scope entry; reads fire only on unconsumed values.
    h.trigger("start_cmd");
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{0, 1}));
    h.trigger("go2");
    h.trigger("go1");
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{0, 1, 2, 3}));
}

// Re-entry without a reassignment while away: the read must stay silent on entry
// (already-seen value) and fire once with the new increment.
TEST(VariableReassignmentTest, EmitsOncePerReEntryOfAnIncrementLoop) {
    Sequence h(
        R"(
    sequence main {
        count u8 := 0
        stage s1 {
            count + 1 -> count
            count -> %out%
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

    std::vector<std::uint8_t> got;
    h.trigger("start_cmd");
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{0, 1}));
    h.trigger("go2");
    h.trigger("go1");
    drain(h, "out", got);
    EXPECT_EQ(got, (std::vector<std::uint8_t>{0, 1, 2}));
}

TEST(VariableReassignmentTest, RoutesAliasWritesToTheReboundChannelAcrossStages) {
    Sequence h(
        R"(
    l := %out_a%
    sequence main {
        stage s1 {
            u8(1) -> l
            %go2% => next
        }
        stage s2 {
            l = %out_b%
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"out_a", x::telem::UINT8_T},
         {"out_b", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_a")), 1);
    EXPECT_FALSE(out.contains(h.key("out_b")));
    h.trigger("go2");
    h.trigger("go1");
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_b")), 1);
}

TEST(VariableReassignmentTest, RoutesAliasReadsToTheReboundChannelAcrossStages) {
    Sequence h(
        R"(
    r := %in_a%
    sequence main {
        stage s1 {
            r -> %out%
            %go2% => next
        }
        stage s2 {
            r = %in_b%
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_a", x::telem::UINT8_T},
         {"in_b", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("in_a", x::telem::Series(std::uint8_t(10)));
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 10);

    h.trigger("go2");
    h.trigger("go1");
    h.ingest("in_b", x::telem::Series(std::uint8_t(20)));
    h.ingest("in_a", x::telem::Series(std::uint8_t(99)));
    h.settle();
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 20);
}

TEST(
    VariableReassignmentTest,
    RoutesAliasExpressionReadsToTheReboundChannelAcrossStages
) {
    Sequence h(
        R"(
    r := %in_a%
    sequence main {
        stage s1 {
            r * 2 -> %out%
            %go2% => next
        }
        stage s2 {
            r = %in_b%
            %go1% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_a", x::telem::UINT8_T},
         {"in_b", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T},
         {"go1", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("in_a", x::telem::Series(std::uint8_t(10)));
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 20);

    h.trigger("go2");
    h.trigger("go1");
    h.push("in_b", x::telem::Series(std::uint8_t(30)));
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 60);
}

TEST(VariableReassignmentTest, RoutesATopLevelAliasExpressionReadAcrossAStageRebind) {
    Sequence h(
        R"(
    r := %in_a%
    r * 2 -> %out%
    sequence main {
        stage s1 {
            %go2% => next
        }
        stage s2 {
            r = %in_b%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_a", x::telem::UINT8_T},
         {"in_b", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("in_a", x::telem::Series(std::uint8_t(10)));
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 20);

    h.trigger("go2");
    h.push("in_b", x::telem::Series(std::uint8_t(30)));
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 60);
}

TEST(VariableReassignmentTest, RePointsAChannelReadVariableFromAnInlineSequence) {
    Sequence h(
        R"(
    stage {
        x := %cpu% * 2
        x -> %out%
        %go2% -> sequence {
            x = %cpu% * 3
        }
    })",
        {{"cpu", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T},
         {"go2", x::telem::UINT8_T}}
    );

    h.push("cpu", x::telem::Series(std::uint8_t(5)));
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 10);

    h.trigger("go2");
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("out"))) << "a re-point alone must not emit";

    h.push("cpu", x::telem::Series(std::uint8_t(7)));
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 21);
}

namespace scope_entry_variable_reset {
/// @brief drives four s1 -> s2 -> s1 re-entries via a 100ms wait, two scheduler passes
/// settling each re-entry.
void loop(Sequence &h) {
    const auto step = [&](const x::telem::TimeSpan now) {
        h.advance(now);
        h.advance(now);
    };
    step(x::telem::TimeSpan(0));
    step(100 * x::telem::MILLISECOND);
    step(200 * x::telem::MILLISECOND);
    step(300 * x::telem::MILLISECOND);
}
}

TEST(
    ScopeEntryVariableResetTest,
    ResetsAColonEqualsVariableDeclaredInTheReEnteredStage
) {
    Sequence h(
        R"(import time
    sequence main {
        stage s1 {
            counter := 0
            1 => counter + 1 => counter
            str(counter) => %log%
            time.wait{100ms} => next
        }
        stage s2 {
            1 => s1
        }
    }
    1 => main)",
        {{"log", x::telem::STRING_T}}
    );

    scope_entry_variable_reset::loop(h);
    const auto out = h.flush();
    EXPECT_EQ(
        collect<std::string>(out, h.key("log")),
        (std::vector<std::string>{"0", "1", "0", "1", "0", "1", "0", "1"})
    );
}

TEST(
    ScopeEntryVariableResetTest,
    PersistsADollarEqualsVariableDeclaredInTheReEnteredStage
) {
    Sequence h(
        R"(import time
    sequence main {
        stage s1 {
            counter $= 0
            1 => counter + 1 => counter
            str(counter) => %log%
            time.wait{100ms} => next
        }
        stage s2 {
            1 => s1
        }
    }
    1 => main)",
        {{"log", x::telem::STRING_T}}
    );

    scope_entry_variable_reset::loop(h);
    const auto out = h.flush();
    EXPECT_EQ(
        collect<std::string>(out, h.key("log")),
        (std::vector<std::string>{"0", "1", "2", "3", "4"})
    );
}

TEST(
    ScopeEntryVariableResetTest,
    DoesNotResetAVariableDeclaredAboveTheSubScopeThatWritesIt
) {
    Sequence h(
        R"(import time
    sequence main {
        counter_c := 0
        counter_s $= 0
        stage s1 {
            1 => counter_c + 1 => counter_c
            1 => counter_s + 1 => counter_s
            str(counter_c) => %log_c%
            str(counter_s) => %log_s%
            time.wait{100ms} => next
        }
        stage s2 {
            1 => s1
        }
    }
    1 => main)",
        {{"log_c", x::telem::STRING_T}, {"log_s", x::telem::STRING_T}}
    );

    scope_entry_variable_reset::loop(h);
    const auto out = h.flush();
    EXPECT_EQ(
        collect<std::string>(out, h.key("log_c")),
        (std::vector<std::string>{"0", "1", "2", "3", "4"})
    );
    EXPECT_EQ(
        collect<std::string>(out, h.key("log_s")),
        (std::vector<std::string>{"0", "1", "2", "3", "4"})
    );
}

TEST(FormatStringInterpolationOfVariablesTest, InterpolatesAReassignedLiteralVariable) {
    Sequence h(
        R"(
    sequence s {
        my_var := "hello"
        my_var = "updated"
        f"val={my_var}" -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "val=updated");
}

TEST(
    FormatStringInterpolationOfVariablesTest,
    InterpolatesASelfReferentiallyIncrementedVariable
) {
    Sequence h(
        R"(
    sequence s {
        counter u8 := 5
        counter = counter + 1
        f"n={counter}" -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T}, {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "n=6");
}

TEST(
    FormatStringInterpolationOfVariablesTest,
    InterpolatesAChannelReadWriteThatFollowsARebind
) {
    Sequence h(
        R"(
    sequence s {
        src := %sensor_a%
        src = %sensor_b%
        f"v={src}" -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor_a", x::telem::UINT8_T},
         {"sensor_b", x::telem::UINT8_T},
         {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor_a", x::telem::Series(std::uint8_t(3)));
    h.ingest("sensor_b", x::telem::Series(std::uint8_t(7)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=7");
}

TEST(
    FormatStringInterpolationOfVariablesTest,
    InterpolatesAReExpressedChannelReadVariable
) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        r = %in_val% + u8(100)
        f"r={r}" -> %out%
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    h.ingest("in_val", x::telem::Series(std::uint8_t(5)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "r=105");
}

TEST(
    FormatStringInterpolationOfVariablesTest,
    ReadsAndInterpolatesAnInheritedChannelReadWriteFromANestedStage
) {
    Sequence h(
        R"(
    sequence s {
        ia := %sensor%
        stage {
            ia -> %out_direct%
            f"a={ia}" -> %out_fmt%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"sensor", x::telem::UINT8_T},
         {"out_direct", x::telem::UINT8_T},
         {"out_fmt", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    h.ingest("sensor", x::telem::Series(std::uint8_t(7)));
    h.advance(x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_direct")), 7);
    EXPECT_EQ(last<std::string>(out, h.key("out_fmt")), "a=7");
}

TEST(
    FormatStringInterpolationOfVariablesTest,
    ReadsAnInheritedChannelReadVariableFromANestedStage
) {
    Sequence h(
        R"(
    sequence s {
        r := %in_val% + u8(1)
        stage {
            r -> %out%
        }
    }
    %start_cmd% => s)",
        {{"start_cmd", x::telem::UINT8_T},
         {"in_val", x::telem::UINT8_T},
         {"out", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("in_val", x::telem::Series(std::uint8_t(5)));
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out")), 6);
}

// A triggered format string or expression samples a variable's latest value at each
// fire; writes alone never fire it.
namespace triggered_expression_variable_reads {
const std::string FMT_SRC = R"(
    sequence main {
        v u8 := 3
        stage s1 {
            %set_ch% -> v
            %go_ch% -> f"v={v}" -> %out%
        }
    }
    %start_cmd% => main)";

Sequence new_h(const std::string &src) {
    return Sequence(
        src,
        {{"start_cmd", x::telem::UINT8_T},
         {"set_ch", x::telem::UINT8_T},
         {"go_ch", x::telem::UINT8_T},
         {"out", x::telem::STRING_T}}
    );
}

void push_val(Sequence &h, const std::uint8_t v) {
    h.push("set_ch", x::telem::Series(v));
}
}

TEST(TriggeredExpressionVariableReadsTest, ReadsTheLatestValueAtEachTrigger) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(FMT_SRC);
    h.trigger("start_cmd");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=3");
    push_val(h, 9);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=9");
}

TEST(
    TriggeredExpressionVariableReadsTest,
    FiresExactlyOncePerTriggerAndNeverOnIdlePasses
) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(FMT_SRC);
    h.trigger("start_cmd");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out"), "v=3"), 1);
    for (int i = 0; i < 10; i++)
        h.advance(x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out"), "v=3"), 0);
}

TEST(TriggeredExpressionVariableReadsTest, DoesNotFireOnAVariableWriteAlone) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(FMT_SRC);
    h.trigger("start_cmd");
    static_cast<void>(h.flush());
    push_val(h, 9);
    const auto out = h.flush();
    EXPECT_FALSE(out.contains(h.key("out")))
        << "a write must not fire the triggered read";
}

TEST(TriggeredExpressionVariableReadsTest, ReEmitsAnUnchangedValueOnEveryTrigger) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(FMT_SRC);
    h.trigger("start_cmd");
    h.trigger("go_ch");
    h.trigger("go_ch");
    const auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out"), "v=3"), 2);
}

TEST(
    TriggeredExpressionVariableReadsTest,
    ReadsTheLatestValueThroughATriggeredExpression
) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(R"(
    sequence main {
        v u8 := 3
        stage s1 {
            %set_ch% -> v
            %go_ch% -> str(v) -> %out%
        }
    }
    %start_cmd% => main)");
    h.trigger("start_cmd");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "3");
    push_val(h, 9);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "9");
}

TEST(TriggeredExpressionVariableReadsTest, ReadsTheLatestValueOfAStatefulVariable) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(R"(
    sequence main {
        v u8 $= 3
        stage s1 {
            %set_ch% -> v
            %go_ch% -> f"v={v}" -> %out%
        }
    }
    %start_cmd% => main)");
    h.trigger("start_cmd");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=3");
    push_val(h, 9);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=9");
}

TEST(
    TriggeredExpressionVariableReadsTest,
    SamplesEveryVariableInAMultiPlaceholderFormatString
) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(R"(
    sequence main {
        a u8 := 3
        b u8 := 3
        stage s1 {
            %set_ch% -> b
            %go_ch% -> f"{a}-{b}" -> %out%
        }
    }
    %start_cmd% => main)");
    h.trigger("start_cmd");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "3-3");
    push_val(h, 9);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "3-9");
}

TEST(TriggeredExpressionVariableReadsTest, ReadsTheResetInitialAfterStageReEntry) {
    Sequence h(
        R"(
    sequence main {
        stage s1 {
            v u8 := 3
            %set_ch% -> v
            %go_ch% -> f"v={v}" -> %out%
            %next_ch% => next
        }
        stage s2 {
            %back_ch% => s1
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"set_ch", x::telem::UINT8_T},
         {"go_ch", x::telem::UINT8_T},
         {"out", x::telem::STRING_T},
         {"next_ch", x::telem::UINT8_T},
         {"back_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("set_ch", x::telem::Series(std::uint8_t(9)));
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=9");
    h.trigger("next_ch");
    h.trigger("back_ch");
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("out")), "v=3");
}

TEST(TriggeredExpressionVariableReadsTest, FiresPerIntervalTickWithTheLiveValue) {
    using namespace triggered_expression_variable_reads;
    auto h = new_h(R"(import time
    stage main {
        x u8 := 3
        %set_ch% -> x
        time.interval{50ms} -> f"v={x}" -> %out%
    }
    1 -> main)");
    h.advance(x::telem::TimeSpan(0));
    h.advance(60 * x::telem::MILLISECOND);
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out"), "v=3"), 1);
    h.ingest("set_ch", x::telem::Series(std::uint8_t(9)));
    h.advance(65 * x::telem::MILLISECOND);
    h.advance(115 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out"), "v=9"), 1);
    EXPECT_EQ(count_of(out, h.key("out"), "v=3"), 0);
}

// A func brace input bound to a variable reads the variable's value at
// each fire; the node fires only on its own trigger.
namespace variable_brace_inputs {
const std::string SRC = R"(
    func echo{tag str} (n u8) str {
        return tag
    }
    sequence main {
        v str := "initial"
        stage s1 {
            %tag_ch% -> v
            %go_ch% -> echo{tag=v} -> %echo_out%
        }
    }
    %start_cmd% => main)";

Sequence new_h() {
    return Sequence(
        SRC,
        {{"start_cmd", x::telem::UINT8_T},
         {"tag_ch", x::telem::STRING_T},
         {"go_ch", x::telem::UINT8_T},
         {"echo_out", x::telem::STRING_T}}
    );
}

void push_tag(Sequence &h, const std::string &v) {
    h.push("tag_ch", x::telem::Series(v));
}
}

TEST(VariableBraceInputsTest, PassesTheVariablesLatestValueAtEachFire) {
    using namespace variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    push_tag(h, "alpha");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "alpha");
    push_tag(h, "beta");
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "beta");
}

TEST(VariableBraceInputsTest, UsesTheDeclaredInitialBeforeAnyWrite) {
    using namespace variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("go_ch");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "initial");
}

TEST(VariableBraceInputsTest, FiresTheConsumerExactlyOncePerTrigger) {
    using namespace variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    push_tag(h, "alpha");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 1);
    // Idle cycles must not re-fire the echo.
    for (int i = 0; i < 10; i++)
        h.advance(x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 0);
    // A fresh trigger still fires, exactly once.
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 1);
}

namespace stateful_variable_brace_inputs {
const std::string SRC = R"(
    func echo{tag str} (n u8) str {
        return tag
    }
    sequence main {
        stage s1 {
            acc str $= "initial"
            %tag_ch% -> acc
            %go_ch% -> echo{tag=acc} -> %echo_out%
            %hop_ch% >= 1 => s2
        }
        stage s2 {
            %back_ch% >= 1 => s1
        }
    }
    %start_cmd% => main)";

Sequence new_h() {
    return Sequence(
        SRC,
        {{"start_cmd", x::telem::UINT8_T},
         {"tag_ch", x::telem::STRING_T},
         {"go_ch", x::telem::UINT8_T},
         {"echo_out", x::telem::STRING_T},
         {"hop_ch", x::telem::UINT8_T},
         {"back_ch", x::telem::UINT8_T}}
    );
}

void push_tag(Sequence &h, const std::string &v) {
    h.push("tag_ch", x::telem::Series(v));
}
}

TEST(StatefulVariableBraceInputsTest, PassesTheStatefulVariablesLatestValueAtEachFire) {
    using namespace stateful_variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    push_tag(h, "alpha");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "alpha");
    push_tag(h, "beta");
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "beta");
}

TEST(StatefulVariableBraceInputsTest, UsesTheDeclaredInitialBeforeAnyWrite) {
    using namespace stateful_variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    h.trigger("go_ch");
    const auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "initial");
}

TEST(StatefulVariableBraceInputsTest, FiresTheConsumerExactlyOncePerTrigger) {
    using namespace stateful_variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    push_tag(h, "alpha");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 1);
    for (int i = 0; i < 10; i++)
        h.advance(x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 0);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("echo_out"), "alpha"), 1);
}

TEST(StatefulVariableBraceInputsTest, RetainsTheWrittenValueAcrossAStageReEntry) {
    using namespace stateful_variable_brace_inputs;
    auto h = new_h();
    h.trigger("start_cmd");
    push_tag(h, "alpha");
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "alpha");
    // Hop s1 -> s2 -> s1; a := variable would reset to "initial" here.
    h.trigger("hop_ch");
    h.trigger("back_ch");
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("echo_out")), "alpha");
}

namespace channel_brace_inputs {
Sequence new_h(const std::string &src) {
    return Sequence(
        src,
        {{"start_cmd", x::telem::UINT8_T},
         {"data_ch", x::telem::FLOAT32_T},
         {"reader_out", x::telem::FLOAT32_T},
         {"go_ch", x::telem::UINT8_T},
         {"sink_ch", x::telem::FLOAT32_T},
         {"data2_ch", x::telem::FLOAT32_T}}
    );
}

void push_data(Sequence &h, const float v) {
    h.push("data_ch", x::telem::Series(v));
}

void run(const std::string &src) {
    auto h = new_h(src);
    h.trigger("start_cmd");
    push_data(h, 1.5f);
    h.trigger("go_ch");
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("reader_out")), 1.5f);
    push_data(h, 2.5f);
    h.trigger("go_ch");
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("reader_out")), 2.5f);
}

void run_write(const std::string &src) {
    auto h = new_h(src);
    h.trigger("start_cmd");
    push_data(h, 1.5f);
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("sink_ch")), 1.5f);
    push_data(h, 2.5f);
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("sink_ch")), 2.5f);
}
}

TEST(ChannelBraceInputsTest, ReadsTheLatestChannelDataAtEachFire) {
    channel_brace_inputs::run(R"(
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        stage s1 {
            %go_ch% -> reader{channel=%data_ch%} -> %reader_out%
        }
    }
    %start_cmd% => main)");
}

TEST(ChannelBraceInputsTest, ReadsTheLatestDataThroughAChannelAlias) {
    channel_brace_inputs::run(R"(
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        a := %data_ch%
        stage s1 {
            %go_ch% -> reader{channel=a} -> %reader_out%
        }
    }
    %start_cmd% => main)");
}

TEST(ChannelBraceInputsTest, WritesEachValueToTheChannel) {
    channel_brace_inputs::run_write(R"(
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        stage s1 {
            %data_ch% -> writer{channel=%sink_ch%}
        }
    }
    %start_cmd% => main)");
}

TEST(ChannelBraceInputsTest, WritesEachValueThroughAChannelAlias) {
    channel_brace_inputs::run_write(R"(
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        w := %sink_ch%
        stage s1 {
            %data_ch% -> writer{channel=w}
        }
    }
    %start_cmd% => main)");
}

TEST(ChannelBraceInputsTest, WritesThroughTheDeclaredBindingThenTheReboundOne) {
    Sequence h(
        R"(
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        w := %sink_a%
        stage s1 {
            %data_ch% -> writer{channel=w}
            %hop_ch% >= 1 => s2
        }
        stage s2 {
            w = %sink_b%
            %data_ch% -> writer{channel=w}
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"data_ch", x::telem::FLOAT32_T},
         {"sink_a", x::telem::FLOAT32_T},
         {"sink_b", x::telem::FLOAT32_T},
         {"hop_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    h.push("data_ch", x::telem::Series(1.5f));
    auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("sink_a")), 1.5f);
    h.trigger("hop_ch");
    h.push("data_ch", x::telem::Series(2.5f));
    out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("sink_b")), 2.5f);
}

TEST(ChannelBraceInputsTest, ReadsThroughTheReboundChannelNotTheDeclaredOne) {
    auto h = channel_brace_inputs::new_h(R"(
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        a := %data_ch%
        stage s1 {
            a = %data2_ch%
            %go_ch% -> reader{channel=a} -> %reader_out%
        }
    }
    %start_cmd% => main)");
    h.trigger("start_cmd");
    h.push("data2_ch", x::telem::Series(7.5f));
    h.trigger("go_ch");
    const auto out = h.flush();
    EXPECT_FLOAT_EQ(last<float>(out, h.key("reader_out")), 7.5f);
}

TEST(TimerBraceInputsTest, AdoptsTheReassignedIntervalPeriodAtTheNextFire) {
    Sequence h(
        R"(
    sequence main {
        rate := i64 ns(100ms)
        stage s1 {
            interval{rate} -> %tick_ch%
            wait{1s} -> sequence {
                rate = i64 ns(2s)
            }
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"tick_ch", x::telem::UINT8_T}}
    );

    // Stage entry: the interval fires immediately.
    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_TRUE(out.contains(h.key("tick_ch")));
    // 100ms cadence while the declared period holds.
    h.advance(141 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_TRUE(out.contains(h.key("tick_ch")));
    h.advance(171 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("tick_ch")))
        << "30ms after the last fire must not tick at a 100ms period";
    // The 1s wait fires and reassigns rate to 2s; the interval also
    // ticks here, which the flush drains.
    h.advance(1101 * x::telem::MILLISECOND);
    static_cast<void>(h.flush());
    h.advance(1400 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("tick_ch")))
        << "the interval must adopt the reassigned 2s period";
    h.advance(3 * x::telem::SECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("tick_ch")));
    // 2s after the last fire: ticks again.
    h.advance(3200 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_TRUE(out.contains(h.key("tick_ch")));
}

TEST(TimerBraceInputsTest, FiresTheWaitEarlierWhenItsDurationIsShortenedMidWait) {
    Sequence h(
        R"(
    sequence main {
        hold := i64 ns(5s)
        stage s1 {
            wait{300ms} -> sequence {
                hold = i64 ns(1s)
            }
            wait{duration=hold} -> %done_ch%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"done_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    // The 300ms wait fires and shortens hold to 1s.
    h.advance(400 * x::telem::MILLISECOND);
    auto out = h.flush();
    EXPECT_FALSE(out.contains(h.key("done_ch")));
    h.advance(700 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("done_ch")));
    // 1s after entry: fires under the shortened duration, not at 5s.
    h.advance(1100 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("done_ch")), 1);
}

TEST(TimerBraceInputsTest, HoldsTheWaitLongerWhenItsDurationIsLengthenedMidWait) {
    Sequence h(
        R"(
    sequence main {
        hold := i64 ns(500ms)
        stage s1 {
            wait{200ms} -> sequence {
                hold = i64 ns(3s)
            }
            wait{duration=hold} -> %done_ch%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"done_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    // The 200ms wait fires and lengthens hold to 3s.
    h.advance(260 * x::telem::MILLISECOND);
    auto out = h.flush();
    EXPECT_FALSE(out.contains(h.key("done_ch")));
    // 600ms after entry: the declared 500ms has passed, but the live
    // duration is now 3s.
    h.advance(600 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("done_ch")))
        << "the wait must adopt the lengthened duration";
    h.advance(3100 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("done_ch")), 1);
}

// A triggered mid-chain variable read emits the variable's live value on
// each fire; the write itself never fires the chain.
TEST(MidChainVariableReadsTest, EmitsTheLiveStringValueAtEachIntervalFire) {
    Sequence h(
        R"(
    sequence main {
        text := "hello"
        stage s1 {
            interval{300ms} -> text -> %log_ch%
            wait{100ms} -> sequence {
                text = "goodbye"
            }
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"log_ch", x::telem::STRING_T}}
    );

    // Stage entry: the interval fires and the read emits the initial.
    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("log_ch")), "hello");
    // The wait fires and reassigns text; the write alone must not
    // re-fire the chain.
    h.advance(171 * x::telem::MILLISECOND);
    h.advance(172 * x::telem::MILLISECOND);
    h.advance(173 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("log_ch")))
        << "a variable write must not fire the reading chain";
    // Next interval fire: the read emits the reassigned value.
    h.advance(331 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("log_ch")), "goodbye");
    EXPECT_EQ(count_of(out, h.key("log_ch"), "hello"), 0)
        << "the stale initial must not be emitted after the write";
    // The reassigned value persists on later fires.
    h.advance(700 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::string>(out, h.key("log_ch")), "goodbye");
}

TEST(MidChainVariableReadsTest, EmitsTheLiveNumericValueAtEachIntervalFire) {
    Sequence h(
        R"(
    sequence main {
        k u8 := 5
        stage s1 {
            interval{300ms} -> k -> %out_ch%
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"out_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_ch")), 5);
    h.advance(171 * x::telem::MILLISECOND);
    h.advance(172 * x::telem::MILLISECOND);
    h.advance(173 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("out_ch")))
        << "a variable write must not fire the reading chain";
    h.advance(331 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_ch")), 9);
}

TEST(
    MidChainVariableReadsTest,
    KeepsHeadReadsWriteDrivenWhileMidChainReadsStayTriggerDriven
) {
    Sequence h(
        R"(
    sequence main {
        k u8 := 5
        stage s1 {
            interval{10s} -> k -> %slow_out%
            k -> %live_out%
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"slow_out", x::telem::UINT8_T},
         {"live_out", x::telem::UINT8_T}}
    );

    // Entry: the interval fires once and reads the initial.
    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("slow_out")), 5);
    // The reassignment fires the head read but not the interval chain.
    h.advance(171 * x::telem::MILLISECOND);
    h.advance(172 * x::telem::MILLISECOND);
    h.advance(173 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("live_out")), 9)
        << "the head read must fire on the write";
    EXPECT_FALSE(out.contains(h.key("slow_out")))
        << "the write must not fire the interval chain";
    h.advance(500 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("slow_out")))
        << "a 10s interval must stay quiet at 500ms";
}

TEST(MidChainVariableReadsTest, EmitsTheLiveStatefulVariableValueAtEachIntervalFire) {
    Sequence h(
        R"(
    sequence main {
        k u8 $= 5
        stage s1 {
            interval{300ms} -> k -> %out_ch%
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"out_ch", x::telem::UINT8_T}}
    );

    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_ch")), 5);
    h.advance(171 * x::telem::MILLISECOND);
    h.advance(172 * x::telem::MILLISECOND);
    h.advance(173 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_FALSE(out.contains(h.key("out_ch")))
        << "a variable write must not fire the reading chain";
    h.advance(331 * x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("out_ch")), 9);
}

// A reactive read variable heads a chain; each source sample must fire
// the chain exactly once.
namespace reactive_read_chain_cadence {
const std::string SRC = R"(
    v := %src_ch% - f32(1)
    v -> "tick" -> %out_str%)";

Sequence new_h(const std::string &src) {
    return Sequence(
        src,
        {{"src_ch", x::telem::FLOAT32_T}, {"out_str", x::telem::STRING_T}}
    );
}
}

TEST(ReactiveReadChainCadenceTest, FiresOncePerSourceSample) {
    using namespace reactive_read_chain_cadence;
    auto h = new_h(SRC);
    h.push("src_ch", x::telem::Series(2.5f));
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1)
        << "one sample must fire the chain exactly once";
    h.settle();
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 0)
        << "idle cycles must not re-fire the chain";
    h.push("src_ch", x::telem::Series(3.5f));
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1)
        << "the next sample must fire the chain exactly once";
}

TEST(ReactiveReadChainCadenceTest, KeepsTheCadenceWhenAnUnrelatedTimerChainAddsCycles) {
    Sequence h(
        R"(
    v := %src_ch% - f32(1)
    v -> "tick" -> %out_str%
    interval{50ms} -> %beat_ch%)",
        {{"src_ch", x::telem::FLOAT32_T},
         {"out_str", x::telem::STRING_T},
         {"beat_ch", x::telem::UINT8_T}}
    );

    h.push("src_ch", x::telem::Series(2.5f));
    h.advance(60 * x::telem::MILLISECOND);
    h.advance(120 * x::telem::MILLISECOND);
    h.advance(180 * x::telem::MILLISECOND);
    const auto out = h.flush();
    EXPECT_EQ(last<std::uint8_t>(out, h.key("beat_ch")), 1)
        << "the timer chain must be adding cycles";
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1)
        << "timer cycles must not re-fire the reactive chain";
}

TEST(ReactiveReadChainCadenceTest, FiresOncePerDistinctSample) {
    using namespace reactive_read_chain_cadence;
    auto h = new_h(SRC);
    for (const float sample: {2.5f, 3.5f, 4.5f})
        h.push("src_ch", x::telem::Series(sample));
    const auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 3);
}

TEST(ReactiveReadChainCadenceTest, ReFiresOnARecomputedIdenticalValue) {
    using namespace reactive_read_chain_cadence;
    auto h = new_h(SRC);
    h.push("src_ch", x::telem::Series(2.5f));
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1);
    h.push("src_ch", x::telem::Series(2.5f));
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1)
        << "a fresh sample fires even when the value is unchanged";
}

TEST(ReactiveReadChainCadenceTest, FiresOncePerSampleWhenTheChainLivesInAStage) {
    Sequence h(
        R"(
    sequence main {
        v := %src_ch% - f32(1)
        stage s1 {
            v -> "tick" -> %out_str%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T},
         {"src_ch", x::telem::FLOAT32_T},
         {"out_str", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    h.push("src_ch", x::telem::Series(2.5f));
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1)
        << "one sample must fire the staged chain exactly once";
    h.settle();
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 0)
        << "idle cycles must not re-fire the staged chain";
    h.push("src_ch", x::telem::Series(3.5f));
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("out_str"), "tick"), 1);
}

// A one-shot entry drives a func whose output rebinds a variable;
// the loopback must not re-fire the chain.
TEST(VariableCaptureDispatchTest, ExecutesAOneShotCaptureChainExactlyOnce) {
    Sequence h(
        R"(
    func stamp{tag str} (n u8) str {
        return tag
    }
    sequence main {
        fp str := ""
        stage s1 {
            1 -> stamp{tag="parent"} -> fp
            fp -> %fp_out%
        }
    }
    %start_cmd% => main)",
        {{"start_cmd", x::telem::UINT8_T}, {"fp_out", x::telem::STRING_T}}
    );

    h.trigger("start_cmd");
    auto out = h.flush();
    EXPECT_EQ(count_of(out, h.key("fp_out"), "parent"), 1);
    // Settle passes and later cycles must not re-run the capture.
    for (int i = 0; i < 10; i++)
        h.advance(x::telem::MILLISECOND);
    out = h.flush();
    EXPECT_EQ(count_of(out, h.key("fp_out"), "parent"), 0);
}
}
