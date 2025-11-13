// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/errors/errors.h"
#include "arc/cpp/module/module.h"
#include "wasm_export.h"

namespace arc::runtime::wasm {
const auto BASE_ERROR = errors::RUNTIME.sub("wasm");
const auto INITIALIZATION_ERROR = BASE_ERROR.sub("initialization");

struct ModuleConfig {
    module::Module module;
    std::uint32_t stack_size = 64 * 1024;
    std::uint32_t host_managed_heap_size = 64 * 1024;
};

class Module {
    ModuleConfig cfg;
    wasm_module_t module;
    wasm_module_inst_t module_inst;
    wasm_exec_env_t exec_env;

public:
    Module(
        const ModuleConfig &cfg,
        const wasm_module_t module,
        const wasm_module_inst_t module_inst,
        const wasm_exec_env_t exec_env
    ):
        cfg(cfg), module(module), module_inst(module_inst), exec_env(exec_env) {}

    static std::pair<std::shared_ptr<Module>, xerrors::Error>
    open(const ModuleConfig &cfg) {
        if (cfg.module.wasm.empty()) {
            return {nullptr, xerrors::Error("wasm bytes are empty")};
        }
        if (!wasm_runtime_init()) return {nullptr, INITIALIZATION_ERROR};
        char error_buffer[1024];
        auto module = wasm_runtime_load(
            const_cast<uint8_t *>(cfg.module.wasm.data()),
            cfg.module.wasm.size(),
            error_buffer,
            sizeof(error_buffer)
        );

        auto module_inst = wasm_runtime_instantiate(
            module,
            cfg.stack_size,
            cfg.host_managed_heap_size,
            error_buffer,
            sizeof(error_buffer)
        );

        auto exec_env = wasm_runtime_create_exec_env(module_inst, cfg.stack_size);
        return {
            std::make_shared<Module>(cfg, module, module_inst, exec_env),
            xerrors::NIL
        };
    }

    ~Module() {
        wasm_runtime_destroy_exec_env(exec_env);
        wasm_runtime_deinstantiate(module_inst);
        wasm_runtime_unload(module);
        wasm_runtime_destroy();
    }

    Module(Module &&other) noexcept;
    Module &operator=(Module &&other) noexcept;
    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    class Function {
        Module &module;
        wasm_function_inst_t fn;
        ir::Params outputs;
        uint32_t base;
        std::vector<uint32_t> offsets;
        struct Result {
            std::uint64_t value;
            bool changed;
        };
        std::vector<Result> output_values;

    public:
        Function(
            Module &module,
            const wasm_function_inst_t fn,
            const ir::Params &outputs,
            const uint32_t base
        ):
            module(module), fn(fn), outputs(outputs), base(base) {
            this->output_values.resize(outputs.size(), Result{0, false});
            uint32_t offset = base + 8;
            for (const auto &param: outputs) {
                this->offsets.push_back(offset);
                offset += static_cast<uint32_t>(param.type.density());
            }
        }

        std::pair<std::vector<Result>, xerrors::Error>
        call(const std::vector<uint64_t> &params) {
            for (auto &[_, changed]: this->output_values)
                changed = false;
            std::vector<uint32_t> args;
            args.reserve(params.size());
            for (const auto p: params)
                args.push_back(static_cast<uint32_t>(p));

            if (!wasm_runtime_call_wasm(
                    this->module.exec_env,
                    this->fn,
                    static_cast<uint32_t>(args.size()),
                    args.data()
                )) {
                const char *exception = wasm_runtime_get_exception(module.module_inst);
                return {
                    {},
                    xerrors::Error(
                        std::string("WASM execution failed: ") +
                        (exception ? exception : "unknown error")
                    )
                };
            }

            if (base == 0) {
                if (!output_values.empty())
                    this->output_values[0] = Result{args[0], true};
                return {this->output_values, xerrors::NIL};
            }

            uint64_t dirty_flags = 0;
            const void *dirty_ptr = wasm_runtime_addr_app_to_native(
                this->module.module_inst,
                this->base
            );
            if (!dirty_ptr)
                return {
                    {},
                    xerrors::Error("failed to access WASM memory at base address")
                };

            memcpy(&dirty_flags, dirty_ptr, sizeof(uint64_t));

            for (size_t i = 0; i < this->outputs.size(); i++) {
                if ((dirty_flags & 1ULL << i) != 0) {
                    const void *value_ptr = wasm_runtime_addr_app_to_native(
                        this->module.module_inst,
                        this->offsets[i]
                    );
                    if (value_ptr == nullptr) continue;
                    uint64_t value = 0;
                    const auto density = this->outputs[i].type.density();
                    memcpy(&value, value_ptr, density);
                    this->output_values[i] = Result{value, true};
                }
            }

            return {this->output_values, xerrors::NIL};
        }
    };

    std::pair<Function, xerrors::Error> func(const std::string &name) {
        const wasm_function_inst_t fn = wasm_runtime_lookup_function(
            this->module_inst,
            name.c_str()
        );
        if (fn == nullptr) return {Function(*this, nullptr, {}, 0), xerrors::NOT_FOUND};
        const auto func_it = this->cfg.module.find_function(name);
        if (func_it == this->cfg.module.functions.end())
            return {Function(*this, nullptr, {}, 0), xerrors::NOT_FOUND};
        uint32_t base = 0;
        if (const auto base_it = this->cfg.module.output_memory_bases.find(name);
            base_it != this->cfg.module.output_memory_bases.end())
            base = base_it->second;
        return {Function(*this, fn, func_it->outputs, base), xerrors::NIL};
    }
};
}
