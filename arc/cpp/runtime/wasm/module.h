// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/errors/errors.h"
#include "arc/cpp/module/module.h"
#include "arc/cpp/runtime/wasm/bindings.h"
#include "arc/cpp/types/types.h"
#include "wasmtime.hh"

namespace arc::runtime::wasm {

/// Convert SampleValue to wasmtime::Val for WASM function calls
inline wasmtime::Val sample_to_wasm(const telem::SampleValue &val) {
    return std::visit([](auto &&arg) -> wasmtime::Val {
        using T = std::decay_t<decltype(arg)>;
        if constexpr (std::is_same_v<T, double>) {
            return wasmtime::Val(arg);
        } else if constexpr (std::is_same_v<T, float>) {
            return wasmtime::Val(arg);
        } else if constexpr (std::is_same_v<T, int64_t>) {
            return wasmtime::Val(arg);
        } else if constexpr (std::is_same_v<T, uint64_t>) {
            return wasmtime::Val(static_cast<int64_t>(arg));
        } else if constexpr (std::is_same_v<T, telem::TimeStamp>) {
            return wasmtime::Val(static_cast<int64_t>(arg.nanoseconds()));
        } else if constexpr (std::is_same_v<T, std::string>) {
            // Strings are passed as handles (uint32_t) which should already be converted
            return wasmtime::Val(static_cast<int32_t>(0));
        } else {
            // int32_t, int16_t, int8_t, uint32_t, uint16_t, uint8_t
            return wasmtime::Val(static_cast<int32_t>(arg));
        }
    }, val);
}

/// Convert wasmtime::Val to SampleValue after WASM function returns
inline telem::SampleValue sample_from_wasm(const wasmtime::Val &val, const types::Type &type) {
    switch (type.kind) {
        case types::Kind::U8:
            return telem::SampleValue(static_cast<uint8_t>(val.i32()));
        case types::Kind::U16:
            return telem::SampleValue(static_cast<uint16_t>(val.i32()));
        case types::Kind::U32:
            return telem::SampleValue(static_cast<uint32_t>(val.i32()));
        case types::Kind::U64:
            return telem::SampleValue(static_cast<uint64_t>(val.i64()));
        case types::Kind::I8:
            return telem::SampleValue(static_cast<int8_t>(val.i32()));
        case types::Kind::I16:
            return telem::SampleValue(static_cast<int16_t>(val.i32()));
        case types::Kind::I32:
            return telem::SampleValue(val.i32());
        case types::Kind::I64:
            return telem::SampleValue(val.i64());
        case types::Kind::F32:
            return telem::SampleValue(val.f32());
        case types::Kind::F64:
            return telem::SampleValue(val.f64());
        case types::Kind::TimeStamp:
            return telem::SampleValue(telem::TimeStamp(val.i64()));
        default:
            return telem::SampleValue(static_cast<int32_t>(0));
    }
}

/// Convert raw memory bits to SampleValue based on Arc type
inline telem::SampleValue sample_from_bits(const uint64_t bits, const types::Type &type) {
    switch (type.kind) {
        case types::Kind::U8:
            return telem::SampleValue(static_cast<uint8_t>(bits));
        case types::Kind::U16:
            return telem::SampleValue(static_cast<uint16_t>(bits));
        case types::Kind::U32:
            return telem::SampleValue(static_cast<uint32_t>(bits));
        case types::Kind::U64:
            return telem::SampleValue(static_cast<uint64_t>(bits));
        case types::Kind::I8:
            return telem::SampleValue(static_cast<int8_t>(bits));
        case types::Kind::I16:
            return telem::SampleValue(static_cast<int16_t>(bits));
        case types::Kind::I32:
            return telem::SampleValue(static_cast<int32_t>(bits));
        case types::Kind::I64:
            return telem::SampleValue(static_cast<int64_t>(bits));
        case types::Kind::F32: {
            const auto bits32 = static_cast<uint32_t>(bits);
            float f;
            memcpy(&f, &bits32, sizeof(float));
            return telem::SampleValue(f);
        }
        case types::Kind::F64: {
            double d;
            memcpy(&d, &bits, sizeof(double));
            return telem::SampleValue(d);
        }
        case types::Kind::TimeStamp:
            return telem::SampleValue(telem::TimeStamp(bits));
        default:
            return telem::SampleValue(static_cast<int32_t>(0));
    }
}


const auto BASE_ERROR = errors::RUNTIME.sub("wasm");
const auto INITIALIZATION_ERROR = BASE_ERROR.sub("initialization");

struct ModuleConfig {
    module::Module module;
    Bindings *bindings = nullptr;
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

    static std::pair<std::shared_ptr<Module>, xerrors::Error>
    open(const ModuleConfig &cfg) {
        if (cfg.module.wasm.empty()) return {nullptr, xerrors::Error("wasm bytes are empty")};

        wasmtime::Engine engine;
        wasmtime::Store store(engine);

        // TODO: don't unwrap this
        // Note: cfg.module.wasm is std::vector<uint8_t> (already decoded from base64)
        // We need to convert to Span, but cfg is const, so we cast to non-const for the span
        auto &wasm_bytes = const_cast<std::vector<uint8_t>&>(cfg.module.wasm);
        auto mod = wasmtime::Module::compile(engine,
            wasmtime::Span<uint8_t>(wasm_bytes.data(), wasm_bytes.size())).unwrap();

        // Create imports if bindings are provided
        std::vector<wasmtime::Extern> imports;
        if (cfg.bindings != nullptr)
            imports = create_imports(store, cfg.bindings);

        auto instance = wasmtime::Instance::create(store, mod, imports).unwrap();
        auto mem_opt = instance.get(store, "memory");
        if (!mem_opt) {
            return {nullptr, xerrors::Error("WASM module does not export 'memory'")};
        }

        const auto *mem_ptr = std::get_if<wasmtime::Memory>(&*mem_opt);
        if (!mem_ptr) {
            return {nullptr, xerrors::Error("export 'memory' is not a Memory type")};
        }
        auto mem = *mem_ptr;

        // Set memory in bindings after instantiation
        if (cfg.bindings != nullptr) {
            cfg.bindings->set_memory(&mem);
        }
        return {
            std::make_shared<Module>(
                cfg,
                std::move(mod),
                std::move(engine),
                std::move(store),
                std::move(mem),
                std::move(instance)
            ),
            xerrors::NIL
        };
    }

    ~Module() {
        // Wasmtime uses RAII - unique_ptrs will automatically clean up
        // in reverse order: instance, module, store, engine
        std::printf("Module destructor - cleaning up Wasmtime resources\n");
    }

    // Move semantics - default implementation works with unique_ptrs
    Module(Module &&other) noexcept = default;
    Module &operator=(Module &&other) noexcept = default;

    // Delete copy operations
    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    class Function {
        Module &module;
        wasmtime::Func fn;
        ir::Params outputs;
        uint32_t base;
        std::vector<wasmtime::Val> args;
        std::vector<uint32_t> offsets;
        struct Result {
            telem::SampleValue value;
            bool changed = false;
        };
        std::vector<Result> output_values;

    public:
        Function(
            Module &module,
            wasmtime::Func fn,
            const ir::Params &outputs,
            const ir::Params &inputs,
            const uint32_t base
        ):
            module(module), fn(std::move(fn)), outputs(outputs), base(base) {
            this->output_values.resize(outputs.size(), Result{});
            this->args.resize(inputs.size(), wasmtime::Val(0));
            uint32_t offset = base + 8;
            for (const auto &param: outputs) {
                this->offsets.push_back(offset);
                offset += static_cast<uint32_t>(param.type.density());
            }
        }

        std::pair<std::vector<Result>, xerrors::Error>
        call(const std::vector<telem::SampleValue> &params) {
            for (auto &[_, changed]: this->output_values)
                changed = false;

            for (size_t i = 0; i < params.size(); i++)
                args[i] = sample_to_wasm(params[i]);


            auto result = fn.call(module.store, args);
            if (!result) {
                auto trap = result.err();
                auto msg = trap.message();
                std::string trap_msg(msg.data(), msg.size());
                std::fprintf(stderr, "WASM trap: %s\n", trap_msg.c_str());
                return {{}, xerrors::Error("WASM execution failed: " + trap_msg)};
            }

            const auto results = result.ok();

            if (base == 0) {
                if (!output_values.empty() && !results.empty())
                    this->output_values[0] = Result{
                        .value=sample_from_wasm(results[0], this->outputs[0].type),
                        .changed=true
                    };
                return {this->output_values, xerrors::NIL};
            }

            // Access memory for outputs using C++ API
            const auto mem_span = module.memory.data(module.store);
            const uint8_t *mem_data = mem_span.data();
            const size_t mem_size = mem_span.size();

            // Check bounds for base address
            if (base + sizeof(uint64_t) > mem_size)
                return {{}, xerrors::Error("base address out of memory bounds")};

            // Read dirty flags
            uint64_t dirty_flags = 0;
            memcpy(&dirty_flags, mem_data + base, sizeof(uint64_t));

            // Extract output values
            for (size_t i = 0; i < this->outputs.size(); i++) {
                if ((dirty_flags & 1ULL << i) != 0) {
                    const uint32_t offset = this->offsets[i];
                    if (offset + this->outputs[i].type.density() > mem_size) continue;

                    uint64_t raw_value = 0;
                    memcpy(&raw_value, mem_data + offset, this->outputs[i].type.density());

                    // Store as Val - caller will extract based on type
                    this->output_values[i] = Result{
                        .value = sample_from_bits(raw_value, this->outputs[i].type),
                        .changed = true
                    };
                }
            }

            return {this->output_values, xerrors::NIL};
        }
    };


    std::pair<Function, xerrors::Error> func(const std::string &name) {
        // Use C++ API to lookup export by name
        const auto export_opt = instance.get(store, name);
        const Function zero_func(*this, wasmtime::Func({}), {}, {}, 0);
        if (!export_opt)
            return {zero_func,xerrors::NOT_FOUND};

        const auto *func_ptr = std::get_if<wasmtime::Func>(&*export_opt);
        if (!func_ptr)
            return {
                zero_func,
                xerrors::Error(xerrors::VALIDATION, "export is not a function")
            };

        const auto func_it = this->cfg.module.find_function(name);
        if (func_it == this->cfg.module.functions.end())
            return {zero_func,xerrors::NOT_FOUND};

        uint32_t base = 0;
        if (const auto base_it = this->cfg.module.output_memory_bases.find(name);
            base_it != this->cfg.module.output_memory_bases.end()) {
            base = base_it->second;
        }

        return {
            Function(*this, *func_ptr, func_it->outputs, func_it->inputs, base),
            xerrors::NIL
        };
    }
};
}
