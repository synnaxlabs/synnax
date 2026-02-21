// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/module/module.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"
#include "wasmtime.hh"

namespace arc::runtime::wasm {

/// @brief Convert SampleValue to wasmtime::Val for WASM function calls.
inline wasmtime::Val sample_to_wasm(const x::telem::SampleValue &val) {
    return std::visit(
        []<typename T0>(T0 &&arg) -> wasmtime::Val {
            using T = std::decay_t<T0>;
            if constexpr (std::is_same_v<T, double>) {
                return wasmtime::Val(arg);
            } else if constexpr (std::is_same_v<T, float>) {
                return wasmtime::Val(arg);
            } else if constexpr (std::is_same_v<T, int64_t>) {
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
    const auto as_double = x::telem::cast<double>(val);
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
        case types::Kind::Invalid:
        case types::Kind::String:
        case types::Kind::Chan:
        case types::Kind::Series:
            return x::telem::SampleValue(0);
    }
    return x::telem::SampleValue(0);
}

/// Convert raw memory bits to SampleValue based on Arc type
inline x::telem::SampleValue
sample_from_bits(const uint64_t bits, const types::Type &type) {
    // Check for timestamp (i64 with nanosecond time units)
    if (type.is_timestamp()) return x::telem::SampleValue(x::telem::TimeStamp(bits));

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
            float f;
            memcpy(&f, &bits32, sizeof(float));
            return x::telem::SampleValue(f);
        }
        case types::Kind::F64: {
            double d;
            memcpy(&d, &bits, sizeof(double));
            return x::telem::SampleValue(d);
        }
        case types::Kind::Invalid:
        case types::Kind::String:
        case types::Kind::Chan:
        case types::Kind::Series:
            return x::telem::SampleValue(static_cast<int32_t>(0));
    }
    return x::telem::SampleValue(static_cast<int32_t>(0));
}

const auto BASE_ERROR = errors::BASE.sub("wasm");
const auto INITIALIZATION_ERROR = BASE_ERROR.sub("initialization");

struct ModuleConfig {
    module::Module module;
    std::vector<std::shared_ptr<stl::Module>> modules;
    std::uint32_t stack_size = 2 * 1024 * 1024; // 2MB (Wasmtime default)
    std::uint32_t host_managed_heap_size = 10 * 1024 * 1024; // 10MB
};

class Module {
    ModuleConfig cfg;
    wasmtime::Module module;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Memory memory;
    wasmtime::Instance instance;

public:
    Module(
        const ModuleConfig &cfg,
        wasmtime::Module module,
        wasmtime::Engine engine,
        wasmtime::Store store,
        wasmtime::Memory memory,
        wasmtime::Instance instance
    ):
        cfg(cfg),
        module(std::move(module)),
        engine(std::move(engine)),
        store(std::move(store)),
        memory(std::move(memory)),
        instance(std::move(instance)) {}

    static std::pair<std::shared_ptr<Module>, x::errors::Error>
    open(const ModuleConfig &cfg) {
        if (cfg.module.wasm.empty())
            return {
                nullptr,
                x::errors::Error(x::errors::VALIDATION, "wasm bytes are empty")
            };

        wasmtime::Engine engine;
        wasmtime::Store store(engine);

        auto &wasm_bytes = const_cast<std::vector<uint8_t> &>(cfg.module.wasm);
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
        for (auto &m: cfg.modules)
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

        auto module = std::make_shared<Module>(
            cfg,
            std::move(mod),
            std::move(engine),
            std::move(store),
            std::move(mem),
            std::move(instance)
        );
        for (auto &m: module->cfg.modules)
            m->set_wasm_context(&module->store, &module->memory);
        return {module, x::errors::NIL};
    }

    Module(Module &&other) noexcept = default;
    Module &operator=(Module &&other) noexcept = default;

    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    class Function {
    public:
        struct Result {
            x::telem::SampleValue value;
            bool changed = false;
        };

    private:
        Module &module;
        wasmtime::Func fn;
        ir::Params outputs;
        size_t config_count;
        uint32_t base;
        std::vector<wasmtime::Val> args;
        std::vector<uint32_t> offsets;

    public:
        Function(
            Module &module,
            wasmtime::Func fn,
            const ir::Params &outputs,
            const ir::Params &config,
            const ir::Params &inputs,
            const uint32_t base
        ):
            module(module),
            fn(std::move(fn)),
            outputs(outputs),
            config_count(config.size()),
            base(base) {
            this->args.resize(config.size() + inputs.size(), wasmtime::Val(0));
            for (size_t i = 0; i < config.size(); i++)
                if (config[i].value.has_value())
                    this->args[i] = sample_to_wasm(*config[i].value, config[i].type);
            uint32_t offset = base + 8;
            for (const auto &param: outputs) {
                this->offsets.push_back(offset);
                offset += static_cast<uint32_t>(param.type.density());
            }
        }

        x::errors::Error call(
            const std::vector<x::telem::SampleValue> &input_vals,
            std::vector<Result> &output_vals
        ) {
            output_vals.assign(this->outputs.size(), Result{});

            for (size_t i = 0; i < input_vals.size(); i++)
                this->args[this->config_count + i] = sample_to_wasm(input_vals[i]);

            auto result = fn.call(this->module.store, this->args);
            if (!result) {
                auto trap = result.err();
                auto msg = trap.message();
                std::string trap_msg(msg.data(), msg.size());
                return x::errors::Error("WASM execution failed: " + trap_msg);
            }

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
                return x::errors::Error("base address out of memory bounds");

            uint64_t dirty_flags = 0;
            memcpy(&dirty_flags, mem_data + base, sizeof(uint64_t));
            for (size_t i = 0; i < this->outputs.size(); i++) {
                if ((dirty_flags & 1ULL << i) == 0) continue;
                const auto output = this->outputs[i];
                const uint32_t offset = this->offsets[i];
                if (offset + output.type.density() > mem_size) continue;
                uint64_t raw_value = 0;
                memcpy(&raw_value, mem_data + offset, output.type.density());
                output_vals[i] = Result{
                    .value = sample_from_bits(raw_value, output.type),
                    .changed = true
                };
            }

            return x::errors::NIL;
        }
    };

    [[nodiscard]] bool has_func(const std::string &name) {
        const auto export_opt = this->instance.get(this->store, name);
        if (!export_opt) return false;
        return std::get_if<wasmtime::Func>(&*export_opt) != nullptr;
    }

    /// @brief Returns a WASM function wrapper for the given function name.
    /// @param name The function name.
    /// @param node_config The node's config params with values. If empty, uses the
    /// function's config.
    std::pair<Function, x::errors::Error>
    func(const std::string &name, const ir::Params &node_config = {}) {
        const auto export_opt = this->instance.get(this->store, name);
        const Function zero_func(*this, wasmtime::Func({}), {}, {}, {}, 0);
        if (!export_opt) return {zero_func, x::errors::NOT_FOUND};

        const auto *func_ptr = std::get_if<wasmtime::Func>(&*export_opt);
        if (!func_ptr)
            return {
                zero_func,
                x::errors::Error(x::errors::VALIDATION, "export is not a function")
            };

        const auto &func = this->cfg.module.function(name);

        uint32_t base = 0;
        if (const auto base_it = this->cfg.module.output_memory_bases.find(name);
            base_it != this->cfg.module.output_memory_bases.end()) {
            base = base_it->second;
        }

        const auto &config_to_use = node_config.empty() ? func.config : node_config;
        return {
            Function(*this, *func_ptr, func.outputs, config_to_use, func.inputs, base),
            x::errors::NIL
        };
    }
};
}
