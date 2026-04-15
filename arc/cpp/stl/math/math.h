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
#include <cmath>
#include <cstdint>
#include <limits>
#include <optional>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::math {

template<typename T>
T int_pow(T base, T exp) {
    if (exp == 0) return 1;
    T result = 1;
    for (T i = 0; i < exp; ++i)
        result *= base;
    return result;
}

class Module : public stl::Module {
public:
    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        bind_float<float>(linker, "f32");
        bind_float<double>(linker, "f64");
        bind_int<uint8_t>(linker, "u8");
        bind_int<uint16_t>(linker, "u16");
        bind_int<uint32_t>(linker, "u32");
        bind_int<uint64_t>(linker, "u64");
        bind_int<int8_t>(linker, "i8");
        bind_int<int16_t>(linker, "i16");
        bind_int<int32_t>(linker, "i32");
        bind_int<int64_t>(linker, "i64");
        bind_signed_unary<int8_t>(linker, "i8");
        bind_signed_unary<int16_t>(linker, "i16");
        bind_signed_unary<int32_t>(linker, "i32");
        bind_signed_unary<int64_t>(linker, "i64");
        bind_float_unary<float>(linker, "f32");
        bind_float_unary<double>(linker, "f64");
    }

private:
    template<typename T>
    static void bind_float(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "pow_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(
                        std::pow(static_cast<T>(a), static_cast<T>(b))
                    );
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "add_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) + static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "subtract_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) - static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "multiply_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) * static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "divide_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) / static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "mod_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(
                        std::fmod(static_cast<double>(a), static_cast<double>(b))
                    );
                }
            )
            .unwrap();
    }

    template<typename T>
    static void bind_int(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "pow_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(
                        int_pow(static_cast<T>(a), static_cast<T>(b))
                    );
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "add_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) + static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "subtract_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) - static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "multiply_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) * static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "divide_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) / static_cast<T>(b));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "math",
                "mod_" + suffix,
                [](W a, W b) -> W {
                    return static_cast<W>(static_cast<T>(a) % static_cast<T>(b));
                }
            )
            .unwrap();
    }

    template<typename T>
    static void bind_signed_unary(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "neg_" + suffix,
                [](W a) -> W { return static_cast<W>(-static_cast<T>(a)); }
            )
            .unwrap();
    }

    template<typename T>
    static void bind_float_unary(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "neg_" + suffix,
                [](W a) -> W { return static_cast<W>(-static_cast<T>(a)); }
            )
            .unwrap();
    }
};

struct WindowConfig {
    x::telem::TimeSpan duration{0};
    int64_t count = 0;

    static std::pair<WindowConfig, x::errors::Error>
    create(const types::Params &params) {
        WindowConfig cfg;
        for (size_t i = 0; i < params.size(); i++) {
            const auto &p = params[i];
            if (p.name == "duration") {
                auto sv = types::to_sample_value(p.value, p.type);
                if (sv.has_value())
                    cfg.duration = x::telem::TimeSpan(x::telem::cast<int64_t>(*sv));
            } else if (p.name == "count") {
                auto sv = types::to_sample_value(p.value, p.type);
                if (sv.has_value()) cfg.count = x::telem::cast<int64_t>(*sv);
            }
        }
        return {cfg, x::errors::NIL};
    }
};

/// @brief Aggregator is a reduction node that computes running statistics (avg, min,
/// max) over input data with optional reset via duration, count, or signal.
class Aggregator : public runtime::node::Node {
public:
    enum class Op { Avg, Min, Max };

private:
    runtime::state::Node state;
    types::Kind kind;
    Op op;
    WindowConfig cfg;
    int64_t sample_count = 0;
    x::telem::TimeStamp start_time{0};
    x::telem::TimeStamp last_reset_time{0};
    int reset_idx;

public:
    Aggregator(
        runtime::state::Node &&state,
        types::Kind kind,
        Op op,
        WindowConfig cfg,
        int reset_idx
    ):
        state(std::move(state)),
        kind(kind),
        op(op),
        cfg(std::move(cfg)),
        reset_idx(reset_idx) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &input_time = this->state.input_time(0);
        if (this->start_time == x::telem::TimeStamp(0) && input_time->size() > 0)
            this->start_time = x::telem::TimeStamp(input_time->at<int64_t>(0));

        bool should_reset = false;

        if (this->reset_idx >= 0) {
            const auto &reset_data = this->state.input(this->reset_idx);
            const auto &reset_time = this->state.input_time(this->reset_idx);
            for (size_t i = 0; i < reset_data->size(); i++) {
                auto ts = x::telem::TimeStamp(reset_time->at<int64_t>(i));
                if (ts > this->last_reset_time && reset_data->at<uint8_t>(i) == 1)
                    should_reset = true;
            }
            if (reset_time->size() > 0)
                this->last_reset_time = x::telem::TimeStamp(
                    reset_time->at<int64_t>(-1)
                );
        }

        if (this->cfg.duration > x::telem::TimeSpan(0) && input_time->size() > 0) {
            auto current_time = x::telem::TimeStamp(input_time->at<int64_t>(-1));
            if (x::telem::TimeSpan(current_time - this->start_time) >=
                this->cfg.duration) {
                should_reset = true;
                this->start_time = current_time;
            }
        }

        if (this->cfg.count > 0 && this->sample_count >= this->cfg.count)
            should_reset = true;

        if (should_reset) {
            this->sample_count = 0;
            this->state.output(0)->resize(0);
        }

        const auto &input_data = this->state.input(0);
        if (input_data->size() == 0) return x::errors::NIL;

        switch (this->kind) {
            case types::Kind::F64:
                this->reduce<double>(input_data);
                break;
            case types::Kind::F32:
                this->reduce<float>(input_data);
                break;
            case types::Kind::I64:
                this->reduce<int64_t>(input_data);
                break;
            case types::Kind::I32:
                this->reduce<int32_t>(input_data);
                break;
            case types::Kind::I16:
                this->reduce<int16_t>(input_data);
                break;
            case types::Kind::I8:
                this->reduce<int8_t>(input_data);
                break;
            case types::Kind::U64:
                this->reduce<uint64_t>(input_data);
                break;
            case types::Kind::U32:
                this->reduce<uint32_t>(input_data);
                break;
            case types::Kind::U16:
                this->reduce<uint16_t>(input_data);
                break;
            case types::Kind::U8:
                this->reduce<uint8_t>(input_data);
                break;
            default:
                break;
        }

        if (this->state.input_time(0)->size() > 0) {
            auto last_ts = this->state.input_time(0)->at<int64_t>(-1);
            *this->state.output_time(0) = x::telem::Series(
                std::vector<int64_t>{last_ts}
            );
        }

        auto &output = this->state.output(0);
        auto &output_time = this->state.output_time(0);
        auto alignment = input_data->alignment;
        auto time_range = input_data->time_range;
        if (this->reset_idx >= 0) {
            const auto &reset_data = this->state.input(this->reset_idx);
            alignment += reset_data->alignment;
            if (reset_data->time_range.start != x::telem::TimeStamp(0) &&
                (time_range.start == x::telem::TimeStamp(0) ||
                 reset_data->time_range.start < time_range.start))
                time_range.start = reset_data->time_range.start;
            if (reset_data->time_range.end > time_range.end)
                time_range.end = reset_data->time_range.end;
        }
        output->alignment = alignment;
        output->time_range = time_range;
        output_time->alignment = alignment;
        output_time->time_range = time_range;
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override {
        this->sample_count = 0;
        this->start_time = x::telem::TimeStamp(0);
        this->last_reset_time = x::telem::TimeStamp(0);
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return this->state.is_output_truthy(param);
    }

private:
    template<typename T>
    void reduce(const runtime::state::Series &input_data) {
        const auto n = static_cast<int64_t>(input_data->size());
        auto &output = this->state.output(0);
        const bool fresh = (output->size() == 0);
        switch (this->op) {
            case Op::Avg:
                this->reduce_avg<T>(input_data, n, fresh);
                break;
            case Op::Min:
                this->reduce_min<T>(input_data, n, fresh);
                break;
            case Op::Max:
                this->reduce_max<T>(input_data, n, fresh);
                break;
        }
        this->sample_count += n;
    }

    template<typename T>
    void reduce_avg(const runtime::state::Series &input_data, int64_t n, bool fresh) {
        double new_sum = 0;
        for (int64_t i = 0; i < n; i++)
            new_sum += static_cast<double>(input_data->at<T>(i));
        auto &output = this->state.output(0);
        if (fresh) {
            *output = x::telem::Series(
                static_cast<T>(new_sum / static_cast<double>(n))
            );
        } else {
            auto prev_avg = static_cast<double>(output->at<T>(0));
            auto total = static_cast<double>(this->sample_count + n);
            auto result = (prev_avg * static_cast<double>(this->sample_count) +
                           new_sum) /
                          total;
            *output = x::telem::Series(static_cast<T>(result));
        }
    }

    template<typename T>
    void reduce_min(const runtime::state::Series &input_data, int64_t n, bool fresh) {
        T new_min = input_data->at<T>(0);
        for (int64_t i = 1; i < n; i++) {
            auto val = input_data->at<T>(i);
            if (val < new_min) new_min = val;
        }
        auto &output = this->state.output(0);
        if (fresh || new_min < output->at<T>(0)) *output = x::telem::Series(new_min);
    }

    template<typename T>
    void reduce_max(const runtime::state::Series &input_data, int64_t n, bool fresh) {
        T new_max = input_data->at<T>(0);
        for (int64_t i = 1; i < n; i++) {
            auto val = input_data->at<T>(i);
            if (val > new_max) new_max = val;
        }
        auto &output = this->state.output(0);
        if (fresh || new_max > output->at<T>(0)) *output = x::telem::Series(new_max);
    }
};

/// @brief Derivative computes the pointwise rate of change (units per second)
/// of an input signal using timestamps.
class Derivative : public runtime::node::Node {
    runtime::state::Node state;
    types::Kind kind;
    double prev_value = 0.0;
    int64_t prev_timestamp_ns = 0;
    bool has_prev = false;

public:
    Derivative(runtime::state::Node &&state, types::Kind kind):
        state(std::move(state)), kind(kind) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &input_data = this->state.input(0);
        const auto &input_time = this->state.input_time(0);
        if (input_data->size() == 0) return x::errors::NIL;
        switch (this->kind) {
            case types::Kind::F64:
                this->compute<double>(input_data, input_time, ctx);
                break;
            case types::Kind::F32:
                this->compute<float>(input_data, input_time, ctx);
                break;
            case types::Kind::I64:
                this->compute<int64_t>(input_data, input_time, ctx);
                break;
            case types::Kind::I32:
                this->compute<int32_t>(input_data, input_time, ctx);
                break;
            case types::Kind::I16:
                this->compute<int16_t>(input_data, input_time, ctx);
                break;
            case types::Kind::I8:
                this->compute<int8_t>(input_data, input_time, ctx);
                break;
            case types::Kind::U64:
                this->compute<uint64_t>(input_data, input_time, ctx);
                break;
            case types::Kind::U32:
                this->compute<uint32_t>(input_data, input_time, ctx);
                break;
            case types::Kind::U16:
                this->compute<uint16_t>(input_data, input_time, ctx);
                break;
            case types::Kind::U8:
                this->compute<uint8_t>(input_data, input_time, ctx);
                break;
            default:
                break;
        }
        return x::errors::NIL;
    }

    void reset() override {
        this->has_prev = false;
        this->prev_value = 0.0;
        this->prev_timestamp_ns = 0;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return this->state.is_output_truthy(param);
    }

private:
    template<typename T>
    void compute(
        const runtime::state::Series &input_data,
        const runtime::state::Series &input_time,
        runtime::node::Context &ctx
    ) {
        const auto n = static_cast<int64_t>(input_data->size());
        auto &output = this->state.output(0);
        auto &output_time = this->state.output_time(0);
        output->resize(n);
        output_time->resize(n);
        for (int64_t i = 0; i < n; i++) {
            const auto current_val = static_cast<double>(input_data->at<T>(i));
            const auto current_ts = input_time->at<int64_t>(i);
            output_time->set(static_cast<int>(i), current_ts);
            if (!this->has_prev) {
                output->set(static_cast<int>(i), 0.0);
            } else {
                const double dt_seconds = static_cast<double>(
                                              current_ts - this->prev_timestamp_ns
                                          ) /
                                          1e9;
                if (dt_seconds <= 0)
                    output->set(static_cast<int>(i), 0.0);
                else
                    output->set(
                        static_cast<int>(i),
                        (current_val - this->prev_value) / dt_seconds
                    );
            }
            this->prev_value = current_val;
            this->prev_timestamp_ns = current_ts;
            this->has_prev = true;
        }
        output->alignment = input_data->alignment;
        output->time_range = input_data->time_range;
        output_time->alignment = input_data->alignment;
        output_time->time_range = input_data->time_range;
        ctx.mark_changed(ir::default_output_param);
    }
};

/// @brief ArithmeticBinary applies a binary arithmetic operation (add, subtract,
/// multiply, divide, mod) element-wise to two input series.
class ArithmeticBinary : public runtime::node::Node {
public:
    enum class Op { Add, Subtract, Multiply, Divide, Mod };

private:
    runtime::state::Node state;
    types::Kind kind;
    Op op;

public:
    ArithmeticBinary(runtime::state::Node &&state, types::Kind kind, Op op):
        state(std::move(state)), kind(kind), op(op) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &lhs = this->state.input(0);
        const auto &rhs = this->state.input(1);
        switch (this->kind) {
            case types::Kind::F64:
                this->compute<double>(lhs, rhs);
                break;
            case types::Kind::F32:
                this->compute<float>(lhs, rhs);
                break;
            case types::Kind::I64:
                this->compute<int64_t>(lhs, rhs);
                break;
            case types::Kind::I32:
                this->compute<int32_t>(lhs, rhs);
                break;
            case types::Kind::I16:
                this->compute<int16_t>(lhs, rhs);
                break;
            case types::Kind::I8:
                this->compute<int8_t>(lhs, rhs);
                break;
            case types::Kind::U64:
                this->compute<uint64_t>(lhs, rhs);
                break;
            case types::Kind::U32:
                this->compute<uint32_t>(lhs, rhs);
                break;
            case types::Kind::U16:
                this->compute<uint16_t>(lhs, rhs);
                break;
            case types::Kind::U8:
                this->compute<uint8_t>(lhs, rhs);
                break;
            default:
                break;
        }
        auto &output = this->state.output(0);
        auto &output_time = this->state.output_time(0);
        output_time = this->state.input_time(0);
        auto alignment = lhs->alignment + rhs->alignment;
        auto time_range = lhs->time_range;
        if (rhs->time_range.start != 0 &&
            (time_range.start == 0 || rhs->time_range.start < time_range.start))
            time_range.start = rhs->time_range.start;
        if (rhs->time_range.end > time_range.end) time_range.end = rhs->time_range.end;
        output->alignment = alignment;
        output->time_range = time_range;
        output_time->alignment = alignment;
        output_time->time_range = time_range;
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override {}

    [[nodiscard]] bool is_output_truthy(const std::string &) const override {
        return false;
    }

private:
    template<typename T>
    void compute(const runtime::state::Series &lhs, const runtime::state::Series &rhs) {
        const auto n = std::max(lhs->size(), rhs->size());
        auto &output = this->state.output(0);
        output->resize(n);
        for (size_t i = 0; i < n; i++) {
            auto a = lhs->at<T>(static_cast<int>(std::min(i, lhs->size() - 1)));
            auto b = rhs->at<T>(static_cast<int>(std::min(i, rhs->size() - 1)));
            T result;
            switch (this->op) {
                case Op::Add:
                    result = a + b;
                    break;
                case Op::Subtract:
                    result = a - b;
                    break;
                case Op::Multiply:
                    result = a * b;
                    break;
                case Op::Divide:
                    result = a / b;
                    break;
                case Op::Mod:
                    result = mod_impl(a, b);
                    break;
            }
            output->set(static_cast<int>(i), result);
        }
    }

    template<typename T>
    static T mod_impl(T a, T b) {
        if constexpr (std::is_floating_point_v<T>)
            return static_cast<T>(
                std::fmod(static_cast<double>(a), static_cast<double>(b))
            );
        else
            return a % b;
    }
};

/// @brief ArithmeticUnary applies a unary arithmetic operation (neg)
/// element-wise to an input series.
class ArithmeticUnary : public runtime::node::Node {
    runtime::state::Node state;
    types::Kind kind;

public:
    ArithmeticUnary(runtime::state::Node &&state, types::Kind kind):
        state(std::move(state)), kind(kind) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &input = this->state.input(0);
        switch (this->kind) {
            case types::Kind::F64:
                this->compute<double>(input);
                break;
            case types::Kind::F32:
                this->compute<float>(input);
                break;
            case types::Kind::I64:
                this->compute<int64_t>(input);
                break;
            case types::Kind::I32:
                this->compute<int32_t>(input);
                break;
            case types::Kind::I16:
                this->compute<int16_t>(input);
                break;
            case types::Kind::I8:
                this->compute<int8_t>(input);
                break;
            default:
                break;
        }
        auto &output = this->state.output(0);
        auto &output_time = this->state.output_time(0);
        output_time = this->state.input_time(0);
        output->alignment = input->alignment;
        output->time_range = input->time_range;
        output_time->alignment = input->alignment;
        output_time->time_range = input->time_range;
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override {}

    [[nodiscard]] bool is_output_truthy(const std::string &) const override {
        return false;
    }

private:
    template<typename T>
    void compute(const runtime::state::Series &input) {
        const auto n = input->size();
        auto &output = this->state.output(0);
        output->resize(n);
        for (size_t i = 0; i < n; i++)
            output->set(static_cast<int>(i), -input->at<T>(i));
    }
};

class FlowModule : public stl::Module {
public:
    [[nodiscard]] std::string module_name() const override { return "math"; }

    bool handles(const std::string &node_type) const override {
        return node_type == "avg" || node_type == "min" || node_type == "max" ||
               node_type == "derivative" || node_type == "add" ||
               node_type == "subtract" || node_type == "multiply" ||
               node_type == "divide" || node_type == "mod" || node_type == "neg";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};

        types::Kind kind = types::Kind::Invalid;
        if (!cfg.node.inputs.empty()) kind = cfg.node.inputs[0].type.kind;

        if (cfg.node.type == "derivative")
            return {
                std::make_unique<Derivative>(std::move(cfg.state), kind),
                x::errors::NIL
            };

        if (cfg.node.type == "neg")
            return {
                std::make_unique<ArithmeticUnary>(std::move(cfg.state), kind),
                x::errors::NIL
            };

        static const std::unordered_map<std::string, ArithmeticBinary::Op> arith_ops = {
            {"add", ArithmeticBinary::Op::Add},
            {"subtract", ArithmeticBinary::Op::Subtract},
            {"multiply", ArithmeticBinary::Op::Multiply},
            {"divide", ArithmeticBinary::Op::Divide},
            {"mod", ArithmeticBinary::Op::Mod},
        };
        if (auto it = arith_ops.find(cfg.node.type); it != arith_ops.end())
            return {
                std::make_unique<ArithmeticBinary>(
                    std::move(cfg.state),
                    kind,
                    it->second
                ),
                x::errors::NIL
            };

        auto [window_cfg, err] = WindowConfig::create(cfg.node.config);
        if (err) return {nullptr, err};

        Aggregator::Op op;
        if (cfg.node.type == "avg")
            op = Aggregator::Op::Avg;
        else if (cfg.node.type == "min")
            op = Aggregator::Op::Min;
        else
            op = Aggregator::Op::Max;

        int reset_idx = -1;
        auto edge = cfg.prog.edge_to(ir::Handle(cfg.node.key, "reset"));
        if (edge.has_value()) {
            reset_idx = 1;
            cfg.state.init_input(
                reset_idx,
                x::mem::make_local_shared<x::telem::Series>(static_cast<uint8_t>(0)),
                x::mem::make_local_shared<x::telem::Series>(x::telem::TimeStamp(1))
            );
        }

        return {
            std::make_unique<
                Aggregator>(std::move(cfg.state), kind, op, window_cfg, reset_idx),
            x::errors::NIL
        };
    }
};

}
