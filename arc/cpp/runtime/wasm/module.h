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
#include <cstring>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <tuple>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "wasmtime.hh"

#include "x/cpp/errors/errors.h"

#include "arc/cpp/program/program.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::wasm {

/// @brief returns the cause in a Wasmtime trap message, without its backtrace.
inline std::string trap_reason(std::string_view message) {
    constexpr std::string_view CAUSE = "Caused by:\n    ";
    if (const auto pos = message.rfind(CAUSE); pos != std::string_view::npos)
        message.remove_prefix(pos + CAUSE.size());
    constexpr std::string_view PREFIX = "wasm trap: ";
    if (message.starts_with(PREFIX)) message.remove_prefix(PREFIX.size());
    while (message.ends_with('\n'))
        message.remove_suffix(1);
    return std::string(message);
}

/// @brief suffix of the batch wrapper that the compiler exports beside an
/// element-wise function.
constexpr auto BATCH_SUFFIX = "$batch";

/// @brief returns the raw bits of a guest value, zero-extended to 64 bits.
inline uint64_t wasm_bits(const wasmtime::Val &val) {
    uint64_t bits = 0;
    switch (val.kind()) {
        case wasmtime::ValKind::I32:
            bits = static_cast<uint32_t>(val.i32());
            break;
        case wasmtime::ValKind::I64:
            bits = static_cast<uint64_t>(val.i64());
            break;
        case wasmtime::ValKind::F32: {
            const float f = val.f32();
            uint32_t b = 0;
            std::memcpy(&b, &f, sizeof(f));
            bits = b;
            break;
        }
        case wasmtime::ValKind::F64: {
            const double d = val.f64();
            std::memcpy(&bits, &d, sizeof(d));
            break;
        }
        default:
            break;
    }
    return bits;
}

/// @brief reports whether the raw bytes of a sample of data_type give the guest the
/// same value as the conversion of one sample at a time.
inline bool same_bits(const x::telem::DataType &data_type, const types::Type &type) {
    if (data_type.density() != type.density()) return false;
    const bool is_float = data_type == x::telem::FLOAT32_T ||
                          data_type == x::telem::FLOAT64_T;
    if (is_float != (type.kind == types::Kind::F32 || type.kind == types::Kind::F64))
        return false;
    // A narrow integer widens to i32, so its sign must match.
    if (type.density() > 2) return true;
    const bool is_signed = data_type == x::telem::INT8_T ||
                           data_type == x::telem::INT16_T;
    return is_signed == (type.kind == types::Kind::I8 || type.kind == types::Kind::I16);
}

/// @brief Convert SampleValue to wasmtime::Val for WASM function calls.
inline wasmtime::Val sample_to_wasm(const x::telem::SampleValue &val) {
    return std::visit(
        []<typename T0>(const T0 &arg) -> wasmtime::Val {
            using T = std::decay_t<T0>;
            if constexpr (
                std::is_same_v<T, double> || std::is_same_v<T, float> ||
                std::is_same_v<T, int64_t>
            ) {
                return wasmtime::Val(arg);
            } else if constexpr (std::is_same_v<T, uint64_t>) {
                return wasmtime::Val(static_cast<int64_t>(arg));
            } else if constexpr (std::is_same_v<T, x::telem::TimeStamp>) {
                return wasmtime::Val(static_cast<int64_t>(arg.nanoseconds()));
            } else if constexpr (std::is_same_v<T, std::string>) {
                return wasmtime::Val(0);
            } else {
                return wasmtime::Val(static_cast<int32_t>(arg));
            }
        },
        val
    );
}

/// @brief Convert SampleValue to wasmtime::Val using the declared type.
/// @note Needed because protobuf stores all numbers as double.
inline wasmtime::Val
sample_to_wasm(const x::telem::SampleValue &val, const types::Type &type) {
    switch (type.kind) {
        case types::Kind::F64:
            return wasmtime::Val(x::telem::cast<double>(val));
        case types::Kind::F32:
            return wasmtime::Val(x::telem::cast<float>(val));
        case types::Kind::I64:
            return wasmtime::Val(x::telem::cast<int64_t>(val));
        case types::Kind::U64:
            return wasmtime::Val(static_cast<int64_t>(x::telem::cast<uint64_t>(val)));
        case types::Kind::Bool:
            return wasmtime::Val(
                static_cast<int32_t>(x::telem::cast<double>(val) != 0 ? 1 : 0)
            );
        default:
            return wasmtime::Val(static_cast<int32_t>(x::telem::cast<int64_t>(val)));
    }
}

/// @brief Convert json value to wasmtime::Val using the declared type.
inline wasmtime::Val json_to_wasm(const x::json::json &val, const types::Type &type) {
    if (val.is_null()) return wasmtime::Val(0);
    if (val.is_boolean()) return wasmtime::Val(val.get<bool>() ? 1 : 0);
    const auto as_double = val.get<double>();
    switch (type.kind) {
        case types::Kind::F64:
            return wasmtime::Val(as_double);
        case types::Kind::F32:
            return wasmtime::Val(static_cast<float>(as_double));
        case types::Kind::I64:
        case types::Kind::U64:
            return wasmtime::Val(static_cast<int64_t>(as_double));
        default:
            return wasmtime::Val(static_cast<int32_t>(as_double));
    }
}

/// Convert wasmtime::Val to SampleValue after WASM function returns
inline x::telem::SampleValue
sample_from_wasm(const wasmtime::Val &val, const types::Type &type) {
    // Check for timestamp (i64 with nanosecond time units)
    if (type.is_timestamp())
        return x::telem::SampleValue(x::telem::TimeStamp(val.i64()));

    switch (type.kind) {
        case types::Kind::U8:
            return x::telem::SampleValue(static_cast<uint8_t>(val.i32()));
        case types::Kind::U16:
            return x::telem::SampleValue(static_cast<uint16_t>(val.i32()));
        case types::Kind::U32:
            return x::telem::SampleValue(static_cast<uint32_t>(val.i32()));
        case types::Kind::U64:
            return x::telem::SampleValue(static_cast<uint64_t>(val.i64()));
        case types::Kind::I8:
            return x::telem::SampleValue(static_cast<int8_t>(val.i32()));
        case types::Kind::I16:
            return x::telem::SampleValue(static_cast<int16_t>(val.i32()));
        case types::Kind::I32:
            return x::telem::SampleValue(val.i32());
        case types::Kind::I64:
            return x::telem::SampleValue(val.i64());
        case types::Kind::F32:
            return x::telem::SampleValue(val.f32());
        case types::Kind::F64:
            return x::telem::SampleValue(val.f64());
        case types::Kind::String:
            return x::telem::SampleValue(val.i32());
        case types::Kind::Bool:
            return x::telem::SampleValue(static_cast<uint8_t>(val.i32() != 0 ? 1 : 0));
        default:
            return x::telem::SampleValue(0);
    }
    return x::telem::SampleValue(0);
}

/// Convert raw memory bits to SampleValue based on Arc type
inline x::telem::SampleValue
sample_from_bits(const uint64_t bits, const types::Type &type) {
    // Check for timestamp (i64 with nanosecond time units)
    if (type.is_timestamp())
        return x::telem::SampleValue(x::telem::TimeStamp(static_cast<int64_t>(bits)));

    switch (type.kind) {
        case types::Kind::U8:
            return x::telem::SampleValue(static_cast<uint8_t>(bits));
        case types::Kind::U16:
            return x::telem::SampleValue(static_cast<uint16_t>(bits));
        case types::Kind::U32:
            return x::telem::SampleValue(static_cast<uint32_t>(bits));
        case types::Kind::U64:
            return x::telem::SampleValue(static_cast<uint64_t>(bits));
        case types::Kind::I8:
            return x::telem::SampleValue(static_cast<int8_t>(bits));
        case types::Kind::I16:
            return x::telem::SampleValue(static_cast<int16_t>(bits));
        case types::Kind::I32:
            return x::telem::SampleValue(static_cast<int32_t>(bits));
        case types::Kind::I64:
            return x::telem::SampleValue(static_cast<int64_t>(bits));
        case types::Kind::F32: {
            const auto bits32 = static_cast<uint32_t>(bits);
            float f = NAN;
            memcpy(&f, &bits32, sizeof(float));
            return x::telem::SampleValue(f);
        }
        case types::Kind::F64: {
            double d = NAN;
            memcpy(&d, &bits, sizeof(double));
            return x::telem::SampleValue(d);
        }
        case types::Kind::String:
            // String outputs are i32 handles in WASM
            return x::telem::SampleValue(static_cast<int32_t>(bits));
        case types::Kind::Bool:
            return x::telem::SampleValue(static_cast<uint8_t>(bits != 0 ? 1 : 0));
        default:
            return x::telem::SampleValue(static_cast<int32_t>(0));
    }
    return x::telem::SampleValue(static_cast<int32_t>(0));
}

const auto BASE_ERROR = errors::BASE.sub("wasm");
const auto INITIALIZATION_ERROR = BASE_ERROR.sub("initialization");

/// @brief address space Wasmtime sets aside for a module's linear memory, and again for
/// each growth past it. Wasmtime's 4 GiB default counts against RLIMIT_MEMLOCK once the
/// Driver locks its memory.
constexpr std::uint64_t MEMORY_RESERVATION = std::uint64_t{2} * 1024 * 1024;
/// @brief size of each guard region around a module's linear memory.
constexpr std::uint64_t MEMORY_GUARD_SIZE = std::uint64_t{64} * 1024;
/// @brief name of the exported i64 global that counts the loop iterations and function
/// calls a guest can still run. The guest traps when it takes the last one. No Arc
/// function can export the name, because an Arc identifier cannot hold '$'.
constexpr auto LOOP_BUDGET_EXPORT = "$loop_budget";
/// @brief loop iterations and function calls that a program can run in one cycle by
/// default.
constexpr std::int64_t DEFAULT_LOOP_BUDGET = 100'000'000;
/// @brief bytes of guest memory that each node's batch call can take by default.
constexpr std::uint32_t DEFAULT_BATCH_CAPACITY = 1024 * 1024;

struct ModuleConfig {
    program::Program program;
    std::vector<std::shared_ptr<stl::Module>> modules;
    std::shared_ptr<stl::strings::State> strings;
    std::uint32_t stack_size = 2 * 1024 * 1024; // 2MB (Wasmtime default)
    std::uint32_t host_managed_heap_size = 10 * 1024 * 1024; // 10MB
    /// @brief loop iterations and function calls that the guest can run between calls
    /// to reset_loop_budget.
    std::int64_t loop_budget = DEFAULT_LOOP_BUDGET;
    /// @brief bytes of guest memory that each node's batch call can take. A longer
    /// series runs in slices that fit.
    std::uint32_t batch_capacity = DEFAULT_BATCH_CAPACITY;
};

class Module {
    ModuleConfig cfg;
    wasmtime::Module module;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Memory memory;
    wasmtime::Instance instance;
    wasmtime::Global loop_budget;

public:
    Module(
        ModuleConfig cfg,
        wasmtime::Module module,
        wasmtime::Engine engine,
        wasmtime::Store store,
        wasmtime::Memory memory,
        wasmtime::Instance instance,
        wasmtime::Global loop_budget
    ):
        cfg(std::move(cfg)),
        module(std::move(module)),
        engine(std::move(engine)),
        store(std::move(store)),
        memory(memory),
        instance(instance),
        loop_budget(loop_budget) {}

    static std::pair<std::shared_ptr<Module>, x::errors::Error>
    open(const ModuleConfig &cfg) {
        if (cfg.program.wasm.empty())
            return {
                nullptr,
                x::errors::Error(x::errors::VALIDATION, "wasm bytes are empty")
            };
        if (cfg.loop_budget <= 0)
            return {
                nullptr,
                x::errors::Error(x::errors::VALIDATION, "loop budget must be positive")
            };

        wasmtime::Config engine_cfg;
        engine_cfg.memory_reservation(MEMORY_RESERVATION);
        engine_cfg.memory_reservation_for_growth(MEMORY_RESERVATION);
        engine_cfg.memory_guard_size(MEMORY_GUARD_SIZE);
        engine_cfg.epoch_interruption(true);
        wasmtime::Engine engine(std::move(engine_cfg));
        wasmtime::Store store(engine);
        // The default deadline is the current epoch, which traps every call at once.
        store.context().set_epoch_deadline(1);

        auto &wasm_bytes = const_cast<std::vector<uint8_t> &>(cfg.program.wasm);
        auto mod_result = wasmtime::Module::compile(
            engine,
            wasmtime::Span<uint8_t>(wasm_bytes.data(), wasm_bytes.size())
        );
        if (!mod_result) {
            auto err = mod_result.err();
            auto msg = err.message();
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "failed to compile module: " + std::string(msg.data(), msg.size())
                )
            };
        }
        const auto mod = mod_result.ok();

        wasmtime::Linker linker(engine);
        for (const auto &m: cfg.modules)
            m->bind_to(linker, store);

        auto inst_result = linker.instantiate(store, mod);
        if (!inst_result) {
            auto trap_err = inst_result.err();
            auto msg = trap_err.message();
            return {
                nullptr,
                x::errors::Error(
                    INITIALIZATION_ERROR,
                    "failed to instantiate module: " +
                        std::string(msg.data(), msg.size())
                )
            };
        }
        auto instance = inst_result.ok();

        const auto mem_opt = instance.get(store, "memory");
        if (!mem_opt)
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "WASM module does not export 'memory'"
                )
            };

        const auto *mem_ptr = std::get_if<wasmtime::Memory>(&*mem_opt);
        if (!mem_ptr)
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "export 'memory' is not a Memory type"
                )
            };
        auto mem = *mem_ptr;

        const auto budget_opt = instance.get(store, LOOP_BUDGET_EXPORT);
        const wasmtime::Global *budget_ptr = nullptr;
        if (budget_opt) budget_ptr = std::get_if<wasmtime::Global>(&*budget_opt);
        if (budget_ptr == nullptr || !budget_ptr->type(store)->is_mutable() ||
            budget_ptr->get(store).kind() != wasmtime::ValKind::I64)
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    std::string("WASM module does not export a mutable i64 '") +
                        LOOP_BUDGET_EXPORT + "' global"
                )
            };

        auto module = std::make_shared<Module>(
            cfg,
            mod,
            std::move(engine),
            std::move(store),
            mem,
            instance,
            *budget_ptr
        );
        for (auto &m: module->cfg.modules)
            m->set_wasm_context(&module->store, &module->memory);
        return {module, x::errors::NIL};
    }

    Module(Module &&other) noexcept = default;
    Module &operator=(Module &&other) noexcept = default;

    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    /// @brief calls the batch wrapper that the compiler exports beside an element-wise
    /// function. The wrapper loops over a whole series inside the guest. Its params are
    /// the sample count, the output block's base, then a base and byte stride per
    /// input.
    class Batch {
        Module &module;
        wasmtime::Func fn;
        types::Params inputs;
        types::Type output;
        /// @brief the bits of each literal input; unused for a streamed input.
        std::vector<uint64_t> literals;
        std::vector<wasmtime::Val> args;
        uint32_t ptr = 0;
        uint32_t capacity = 0;
        /// @brief true once a failed grow has been reported.
        bool warned = false;

        static constexpr uint64_t align8(const uint64_t v) { return (v + 7) & ~7ULL; }

        /// @brief returns the bytes of a region that holds count samples, where varies
        /// marks the inputs that hold a sample per output sample.
        [[nodiscard]] uint64_t
        region_size(const uint64_t count, const std::vector<bool> &varies) const {
            uint64_t size = align8(count * this->output.density());
            for (size_t i = 0; i < this->inputs.size(); i++)
                size += align8(
                    (varies[i] ? count : 1) * this->inputs[i].type.density()
                );
            return size;
        }

        /// @brief runs samples [start, start + count) of the function into out.
        x::errors::Error call_slice(
            const std::vector<const x::telem::Series *> &series,
            const std::vector<bool> &varies,
            const size_t start,
            const size_t count,
            x::telem::Series &out
        ) {
            const auto out_density = this->output.density();
            auto *buf = this->module.memory.data(this->module.store).data() + this->ptr;
            uint64_t offset = align8(count * out_density);
            for (size_t i = 0; i < this->inputs.size(); i++) {
                const auto density = this->inputs[i].type.density();
                auto *dst = buf + offset;
                const auto *in = series[i];
                if (in == nullptr)
                    std::memcpy(dst, &this->literals[i], density);
                else if (!varies[i])
                    std::memcpy(dst, in->data(), density);
                else if (in->size() >= start + count)
                    std::memcpy(dst, in->data() + (start * density), count * density);
                else
                    for (size_t j = 0; j < count; j++)
                        std::memcpy(
                            dst + (j * density),
                            in->data() + (((start + j) % in->size()) * density),
                            density
                        );
                this->args[2 + (2 * i)] = wasmtime::Val(
                    static_cast<int32_t>(this->ptr + offset)
                );
                this->args[3 + (2 * i)] = wasmtime::Val(
                    static_cast<int32_t>(varies[i] ? density : 0)
                );
                offset += align8((varies[i] ? count : 1) * density);
            }
            this->args[0] = wasmtime::Val(static_cast<int32_t>(count));
            this->args[1] = wasmtime::Val(static_cast<int32_t>(this->ptr));
            if (auto result = this->fn.call(this->module.store, this->args); !result)
                return this->module.call_error(result.err());
            // The guest can grow memory, which can move it.
            buf = this->module.memory.data(this->module.store).data() + this->ptr;
            std::memcpy(out.data() + (start * out_density), buf, count * out_density);
            return x::errors::NIL;
        }

    public:
        Batch(
            Module &module,
            wasmtime::Func fn,
            const types::Params &inputs,
            types::Type output
        ):
            module(module),
            fn(fn),
            inputs(inputs),
            output(std::move(output)),
            literals(inputs.size()),
            args(2 + (2 * inputs.size()), wasmtime::Val(0)) {
            for (size_t i = 0; i < inputs.size(); i++) {
                if (inputs[i].value.is_null()) continue;
                const auto sv = types::to_sample_value(inputs[i].value, inputs[i].type);
                if (sv.has_value())
                    this->literals[i] = wasm_bits(sample_to_wasm(*sv, inputs[i].type));
            }
        }

        /// @brief writes count samples of the function into out, which must hold count
        /// samples. series holds each streamed input, and nullptr for a literal one. A
        /// series longer than the batch region holds runs in slices that fit it.
        /// @returns false when a series does not match the wrapper's layout or guest
        /// memory cannot hold one sample, so the caller must run the samples one at a
        /// time, and the error of the call.
        std::pair<bool, x::errors::Error> call(
            const std::vector<const x::telem::Series *> &series,
            const size_t count,
            x::telem::Series &out
        ) {
            if (!same_bits(out.data_type(), this->output))
                return {false, x::errors::NIL};
            std::vector<bool> varies(this->inputs.size(), false);
            for (size_t i = 0; i < this->inputs.size(); i++) {
                const auto *in = series[i];
                if (in == nullptr) continue;
                if (!same_bits(in->data_type(), this->inputs[i].type))
                    return {false, x::errors::NIL};
                varies[i] = in->size() != 1;
            }
            const uint64_t cap = this->module.cfg.batch_capacity;
            const uint64_t fixed = this->region_size(0, varies);
            const uint64_t per_sample = this->region_size(1, varies) - fixed;
            // Alignment adds at most 7 bytes for each block past the fixed size.
            const uint64_t slack = 8 * (1 + this->inputs.size());
            if (fixed + slack + per_sample > cap) return {false, x::errors::NIL};
            const uint64_t slice = std::min<uint64_t>(
                count,
                (cap - fixed - slack) / per_sample
            );
            const uint64_t size = this->region_size(slice, varies);
            if (size > this->capacity) {
                // Doubling bounds what a node abandons as its series grow.
                const auto want = std::min(
                    std::max(size, uint64_t{2} * this->capacity),
                    cap
                );
                const auto region = this->module.grow(want);
                if (!region.has_value()) {
                    if (!this->warned)
                        LOG(WARNING)
                            << "[arc.wasm] guest memory cannot grow by " << want
                            << " bytes, so a node runs one sample at a time";
                    this->warned = true;
                    return {false, x::errors::NIL};
                }
                std::tie(this->ptr, this->capacity) = *region;
            }
            for (size_t start = 0; start < count; start += slice)
                if (auto err = this->call_slice(
                        series,
                        varies,
                        start,
                        std::min<size_t>(slice, count - start),
                        out
                    ))
                    return {true, err};
            if (this->output.kind == types::Kind::Bool) {
                auto *bytes = reinterpret_cast<uint8_t *>(out.data());
                for (size_t j = 0; j < count; j++)
                    bytes[j] = bytes[j] != 0 ? 1 : 0;
            }
            return {true, x::errors::NIL};
        }
    };

    class Function {
    public:
        struct Result {
            x::telem::SampleValue value;
            bool changed = false;
        };

    private:
        Module &module;
        wasmtime::Func fn;
        types::Params outputs;
        /// @brief one flag per input, true when streamed per sample (nil value).
        std::vector<bool> streamed;
        uint32_t base;
        std::vector<wasmtime::Val> args;
        std::vector<uint32_t> offsets;
        std::optional<Batch> batch_call;

    public:
        Function(
            Module &module,
            wasmtime::Func fn,
            const types::Params &outputs,
            const types::Params &inputs,
            const uint32_t base,
            std::optional<Batch> batch = std::nullopt
        ):
            module(module),
            fn(fn),
            outputs(outputs),
            base(base),
            batch_call(std::move(batch)) {
            this->args.resize(inputs.size(), wasmtime::Val(0));
            this->streamed.assign(inputs.size(), false);
            // An input with a nil value is edge-fed and streams in call(); a literal
            // input is set once here.
            for (size_t i = 0; i < inputs.size(); i++) {
                // A var input re-reads its variable each pass; never bake its
                // declared initial.
                if (inputs[i].type.kind == types::Kind::VarRef) {
                    this->streamed[i] = true;
                    continue;
                }
                if (inputs[i].value.is_null()) {
                    this->streamed[i] = true;
                    continue;
                }
                if (inputs[i].type.kind == types::Kind::String) {
                    // A literal string gets a stable handle, not cleared by flush().
                    // strings is null only in tests without host bindings.
                    const auto &s = inputs[i].value.get<std::string>();
                    if (module.cfg.strings != nullptr)
                        this->args[i] = wasmtime::Val(
                            static_cast<int32_t>(module.cfg.strings->create_literal(s))
                        );
                    continue;
                }
                auto sv = types::to_sample_value(inputs[i].value, inputs[i].type);
                if (sv.has_value()) this->args[i] = sample_to_wasm(*sv, inputs[i].type);
            }
            uint32_t offset = base + 8;
            for (const auto &param: outputs) {
                this->offsets.push_back(offset);
                if (param.type.kind == types::Kind::String)
                    offset += 4; // strings are i32 handles in WASM
                else
                    offset += static_cast<uint32_t>(param.type.density());
            }
        }

        /// @brief returns the wrapper that runs the function over a whole series, or
        /// nullptr when the node's shape does not admit one.
        [[nodiscard]] Batch *batch() {
            return this->batch_call.has_value() ? &*this->batch_call : nullptr;
        }

        x::errors::Error call(
            const std::vector<x::telem::SampleValue> &input_vals,
            std::vector<Result> &output_vals
        ) {
            output_vals.assign(this->outputs.size(), Result{});

            for (size_t i = 0; i < input_vals.size(); i++)
                if (this->streamed[i]) this->args[i] = sample_to_wasm(input_vals[i]);

            auto result = fn.call(this->module.store, this->args);
            if (!result) return this->module.call_error(result.err());

            const auto results = result.ok();
            if (this->base == 0) {
                if (!output_vals.empty() && !results.empty())
                    output_vals[0] = Result{
                        .value = sample_from_wasm(results[0], this->outputs[0].type),
                        .changed = true
                    };
                return x::errors::NIL;
            }

            const auto mem_span = this->module.memory.data(this->module.store);
            const uint8_t *mem_data = mem_span.data();
            const size_t mem_size = mem_span.size();

            if (this->base + sizeof(uint64_t) > mem_size)
                return x::errors::Error(
                    errors::WASM_TRAP,
                    "The base address is out of memory bounds."
                );

            uint64_t dirty_flags = 0;
            memcpy(&dirty_flags, mem_data + base, sizeof(uint64_t));
            for (size_t i = 0; i < this->outputs.size(); i++) {
                if ((dirty_flags & 1ULL << i) == 0) continue;
                const auto output = this->outputs[i];
                const uint32_t offset = this->offsets[i];
                const size_t byte_size = output.type.kind == types::Kind::String
                                           ? 4
                                           : output.type.density();
                if (offset + byte_size > mem_size) continue;
                uint64_t raw_value = 0;
                memcpy(&raw_value, mem_data + offset, byte_size);
                output_vals[i] = Result{
                    .value = sample_from_bits(raw_value, output.type),
                    .changed = true
                };
            }

            return x::errors::NIL;
        }
    };

    /// @brief returns the runtime error of a failed guest call.
    x::errors::Error call_error(const wasmtime::TrapError &trap) {
        if (const auto *t = std::get_if<wasmtime::Trap>(&trap.data);
            t != nullptr && t->code() == WASMTIME_TRAP_CODE_INTERRUPT)
            return x::errors::Error(
                errors::INTERRUPTED,
                "The runtime stopped the call."
            );
        if (this->loop_budget_used()) {
            // Later calls would each trap at once and report the error again.
            this->interrupt();
            return x::errors::Error(
                errors::LOOP_BUDGET,
                "The program ran more than " + std::to_string(this->cfg.loop_budget) +
                    " loop iterations and function calls in one cycle."
            );
        }
        auto msg = trap.message();
        return x::errors::Error(
            errors::WASM_TRAP,
            trap_reason({msg.data(), msg.size()})
        );
    }

    /// @brief takes a region of at least size bytes past the end of guest memory.
    /// @returns the base and capacity of the region, or nullopt when memory cannot
    /// grow. A 32-bit guest memory cannot grow past 4 GiB, so both fit 32 bits.
    std::optional<std::pair<uint32_t, uint32_t>> grow(const uint64_t size) {
        constexpr uint64_t PAGE_SIZE = 65536;
        const uint64_t pages = (size + PAGE_SIZE - 1) / PAGE_SIZE;
        const auto prev = this->memory.grow(this->store, pages);
        if (!prev) return std::nullopt;
        return std::pair{
            static_cast<uint32_t>(prev.ok() * PAGE_SIZE),
            static_cast<uint32_t>(pages * PAGE_SIZE)
        };
    }

    /// @brief gives the guest its whole loop budget. Call it before each cycle. A call
    /// that uses the whole budget interrupts the module until resume().
    void reset_loop_budget() {
        // open checked that the global is a mutable i64, so the set cannot fail.
        std::ignore = this->loop_budget.set(
            this->store,
            wasmtime::Val(this->cfg.loop_budget)
        );
    }

    /// @brief returns true when the guest used its whole loop budget since the last
    /// call to reset_loop_budget.
    [[nodiscard]] bool loop_budget_used() {
        return this->loop_budget.get(this->store).i64() <= 0;
    }

    [[nodiscard]] std::shared_ptr<stl::strings::State> strings() const {
        return this->cfg.strings;
    }

    /// @brief makes a running call, and every later call until resume(), return
    /// errors::INTERRUPTED. Safe to call from any thread.
    void interrupt() const { this->engine.increment_epoch(); }

    /// @brief lets calls run again after interrupt(). Call only while no call runs.
    void resume() { this->store.context().set_epoch_deadline(1); }

    [[nodiscard]] bool has_func(const std::string &name) {
        const auto export_opt = this->instance.get(this->store, name);
        if (!export_opt) return false;
        return std::get_if<wasmtime::Func>(&*export_opt) != nullptr;
    }

    /// @brief returns the batch wrapper of the function name for a node with inputs,
    /// or nullopt when the guest exports none or the node's shape does not match one: a
    /// single returned value, and fixed-width values with no references throughout.
    std::optional<Batch> batch(
        const std::string &name,
        const ir::Function &func,
        const types::Params &inputs,
        const uint32_t base
    ) {
        if (base != 0 || func.outputs.size() != 1 ||
            func.outputs[0].type.density() == 0 || inputs.size() != func.inputs.size())
            return std::nullopt;
        for (size_t i = 0; i < inputs.size(); i++)
            if (func.inputs[i].type.density() == 0 || inputs[i].type.density() == 0)
                return std::nullopt;
        const auto export_opt = this->instance.get(this->store, name + BATCH_SUFFIX);
        if (!export_opt) return std::nullopt;
        const auto *fn = std::get_if<wasmtime::Func>(&*export_opt);
        if (fn == nullptr) return std::nullopt;
        return Batch(*this, *fn, inputs, func.outputs[0].type);
    }

    /// @brief Wraps the WASM export `name` using its declared inputs.
    std::pair<Function, x::errors::Error> func(const std::string &name) {
        return this->func(name, {});
    }

    /// @brief Wraps the WASM export `name` for a node's inputs. An input with a nil
    /// value is edge-fed (streamed per sample); a non-nil value is a literal set once.
    std::pair<Function, x::errors::Error>
    func(const std::string &name, const types::Params &node_inputs) {
        const auto export_opt = this->instance.get(this->store, name);
        const Function zero_func(*this, wasmtime::Func({}), {}, {}, 0);
        if (!export_opt) return {zero_func, x::errors::NOT_FOUND};

        const auto *func_ptr = std::get_if<wasmtime::Func>(&*export_opt);
        if (!func_ptr)
            return {
                zero_func,
                x::errors::Error(x::errors::VALIDATION, "export is not a function")
            };

        const auto &func = this->cfg.program.function(name);

        uint32_t base = 0;
        if (const auto base_it = this->cfg.program.output_memory_bases.find(name);
            base_it != this->cfg.program.output_memory_bases.end()) {
            base = base_it->second;
        }

        for (const auto &output: func.outputs) {
            if (!output.type.is_valid())
                return {
                    zero_func,
                    x::errors::Error(
                        x::errors::VALIDATION,
                        "function '" + name + "' has output '" + output.name +
                            "' with unresolved type"
                    )
                };
        }

        const auto &inputs_to_use = node_inputs.empty() ? func.inputs : node_inputs;
        return {
            Function(
                *this,
                *func_ptr,
                func.outputs,
                inputs_to_use,
                base,
                this->batch(name, func, inputs_to_use, base)
            ),
            x::errors::NIL
        };
    }
};
}
