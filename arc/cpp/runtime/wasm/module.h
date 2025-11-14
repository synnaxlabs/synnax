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
#include "wasm.h"      // Standard WebAssembly C API
#include "wasmtime.h"  // Wasmtime-specific extensions

namespace arc::runtime::wasm {
namespace bindings {
class Runtime;
}

const auto BASE_ERROR = errors::RUNTIME.sub("wasm");
const auto INITIALIZATION_ERROR = BASE_ERROR.sub("initialization");

struct ModuleConfig {
    module::Module module;
    bindings::Runtime *runtime = nullptr;
    std::uint32_t stack_size = 2 * 1024 * 1024;      // 2MB (Wasmtime default)
    std::uint32_t host_managed_heap_size = 10 * 1024 * 1024;  // 10MB
};

class Module {
    ModuleConfig cfg;
    // Wasmtime types (owned via unique_ptr for proper RAII)
    std::unique_ptr<wasm_engine_t, decltype(&wasm_engine_delete)> engine;
    std::unique_ptr<wasmtime_store_t, decltype(&wasmtime_store_delete)> store;  // Use wasmtime_store_t for context data support
    std::unique_ptr<wasm_module_t, decltype(&wasm_module_delete)> module;
    std::unique_ptr<wasm_instance_t, decltype(&wasm_instance_delete)> instance;
    wasm_memory_t *memory;  // Non-owning pointer to memory export

public:
    Module(
        const ModuleConfig &cfg,
        std::unique_ptr<wasm_engine_t, decltype(&wasm_engine_delete)> engine,
        std::unique_ptr<wasmtime_store_t, decltype(&wasmtime_store_delete)> store,
        std::unique_ptr<wasm_module_t, decltype(&wasm_module_delete)> module,
        std::unique_ptr<wasm_instance_t, decltype(&wasm_instance_delete)> instance,
        wasm_memory_t *memory
    ):
        cfg(cfg),
        engine(std::move(engine)),
        store(std::move(store)),
        module(std::move(module)),
        instance(std::move(instance)),
        memory(memory) {}

    static std::pair<std::shared_ptr<Module>, xerrors::Error>
    open(const ModuleConfig &cfg) {
        if (cfg.module.wasm.empty()) {
            return {nullptr, xerrors::Error("wasm bytes are empty")};
        }

        // Debug: print wasm bytes before loading
        std::printf("wasm::Module::open - size: %zu\n", cfg.module.wasm.size());
        std::printf("wasm::Module::open - first 8 bytes: ");
        for (size_t i = 0; i < 8 && i < cfg.module.wasm.size(); i++) {
            std::printf("%02x ", cfg.module.wasm[i]);
        }
        std::printf("\n");

        // Step 1: Create Engine
        std::unique_ptr<wasm_engine_t, decltype(&wasm_engine_delete)> engine(
            wasm_engine_new(),
            wasm_engine_delete
        );
        if (!engine) {
            return {nullptr, INITIALIZATION_ERROR.wrap("failed to create wasmtime engine")};
        }
        std::printf("Engine created successfully\n");

        // Step 2: Create Store (use wasmtime_store_new for context data support)
        std::unique_ptr<wasmtime_store_t, decltype(&wasmtime_store_delete)> store(
            wasmtime_store_new(engine.get(), nullptr, nullptr),  // engine, data, finalizer
            wasmtime_store_delete
        );
        if (!store) {
            return {nullptr, INITIALIZATION_ERROR.wrap("failed to create wasmtime store")};
        }
        std::printf("Store created successfully\n");

        // Step 3: Compile Module
        wasm_byte_vec_t wasm_bytes;
        wasm_bytes.size = cfg.module.wasm.size();
        wasm_bytes.data = reinterpret_cast<char*>(const_cast<uint8_t*>(cfg.module.wasm.data()));

        std::unique_ptr<wasm_module_t, decltype(&wasm_module_delete)> module(
            wasm_module_new(store.get(), &wasm_bytes),
            wasm_module_delete
        );
        if (!module) {
            return {nullptr, xerrors::Error("failed to compile wasm module")};
        }
        std::printf("Module compiled successfully\n");

        // Step 4: Set Runtime in store data (must be done before creating imports)
        if (cfg.runtime != nullptr) {
            bindings::set_runtime_in_store(store.get(), cfg.runtime);
            std::printf("Runtime set in store data\n");
        }

        // Step 5: Create imports (host functions)
        wasm_extern_vec_t imports;
        if (cfg.runtime != nullptr) {
            imports = bindings::create_imports(store.get(), cfg.runtime);
        } else {
            wasm_extern_vec_new_empty(&imports);
        }

        // Step 6: Instantiate Module
        wasm_trap_t *trap = nullptr;
        std::unique_ptr<wasm_instance_t, decltype(&wasm_instance_delete)> instance(
            wasm_instance_new(store.get(), module.get(), &imports, &trap),
            wasm_instance_delete
        );

        // Clean up imports vector (functions are owned by store)
        bindings::delete_imports(&imports);

        if (trap) {
            wasm_message_t message;
            wasm_trap_message(trap, &message);
            std::string error_msg(message.data, message.size);
            wasm_byte_vec_delete(&message);
            wasm_trap_delete(trap);
            return {nullptr, xerrors::Error("instantiation trapped: " + error_msg)};
        }

        if (!instance) {
            return {nullptr, xerrors::Error("failed to instantiate wasm module")};
        }
        std::printf("Module instantiated successfully\n");

        // Step 7: Get memory export (if exists)
        wasm_extern_vec_t exports;
        wasm_instance_exports(instance.get(), &exports);
        wasm_memory_t *memory = nullptr;
        for (size_t i = 0; i < exports.size; i++) {
            if (wasm_extern_kind(exports.data[i]) == WASM_EXTERN_MEMORY) {
                memory = wasm_extern_as_memory(exports.data[i]);
                break;
            }
        }
        if (memory) {
            std::printf("Found memory export\n");
        }

        return {
            std::make_shared<Module>(
                cfg,
                std::move(engine),
                std::move(store),
                std::move(module),
                std::move(instance),
                memory
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
        wasm_func_t *fn;  // Non-owning pointer to function from exports
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
            wasm_func_t *fn,
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

            // Convert params to wasm_val_t array
            std::vector<wasm_val_t> args;
            args.reserve(params.size());
            for (const auto p: params) {
                wasm_val_t val;
                val.kind = WASM_I32;
                val.of.i32 = static_cast<uint32_t>(p);
                args.push_back(val);
            }

            // Prepare args and results vectors
            wasm_val_vec_t args_vec;
            args_vec.size = args.size();
            args_vec.data = args.data();

            wasm_val_vec_t results_vec;
            results_vec.size = 1;  // Assuming single return value for now
            wasm_val_t result_val;
            results_vec.data = &result_val;

            // Call function
            wasm_trap_t *trap = wasm_func_call(fn, &args_vec, &results_vec);
            if (trap) {
                wasm_message_t message;
                wasm_trap_message(trap, &message);
                std::string error_msg(message.data, message.size);
                wasm_byte_vec_delete(&message);
                wasm_trap_delete(trap);
                return {{}, xerrors::Error("WASM execution failed: " + error_msg)};
            }

            // Handle output extraction
            if (base == 0) {
                // Direct return value
                if (!output_values.empty() && results_vec.size > 0) {
                    this->output_values[0] = Result{
                        static_cast<uint64_t>(results_vec.data[0].of.i32),
                        true
                    };
                }
                return {this->output_values, xerrors::NIL};
            }

            // Access memory for outputs
            if (!module.memory) {
                return {{}, xerrors::Error("module has no memory export")};
            }

            byte_t *mem_data = wasm_memory_data(module.memory);
            size_t mem_size = wasm_memory_data_size(module.memory);

            // Check bounds for base address
            if (base + sizeof(uint64_t) > mem_size) {
                return {{}, xerrors::Error("base address out of memory bounds")};
            }

            // Read dirty flags
            uint64_t dirty_flags = 0;
            memcpy(&dirty_flags, mem_data + base, sizeof(uint64_t));

            // Extract output values
            for (size_t i = 0; i < this->outputs.size(); i++) {
                if ((dirty_flags & (1ULL << i)) != 0) {
                    uint32_t offset = this->offsets[i];
                    if (offset + this->outputs[i].type.density() > mem_size) continue;

                    uint64_t value = 0;
                    memcpy(&value, mem_data + offset, this->outputs[i].type.density());
                    this->output_values[i] = Result{value, true};
                }
            }

            return {this->output_values, xerrors::NIL};
        }
    };

    std::pair<Function, xerrors::Error> func(const std::string &name) {
        // Use Wasmtime-specific API to lookup export by name
        // wasmtime_instance_export_get is in wasmtime.h (not wasm.h)
        wasm_extern_t *export_extern = nullptr;
        bool found = wasmtime_instance_export_get(
            wasmtime_store_context(store.get()),
            instance.get(),
            name.c_str(),
            name.size(),
            &export_extern
        );

        if (!found || !export_extern) {
            return {Function(*this, nullptr, {}, 0), xerrors::NOT_FOUND};
        }

        // Check if it's a function
        if (wasm_extern_kind(export_extern) != WASM_EXTERN_FUNC) {
            return {Function(*this, nullptr, {}, 0), xerrors::Error("export is not a function")};
        }

        wasm_func_t *fn = wasm_extern_as_func(export_extern);
        if (fn == nullptr) {
            return {Function(*this, nullptr, {}, 0), xerrors::NOT_FOUND};
        }

        const auto func_it = this->cfg.module.find_function(name);
        if (func_it == this->cfg.module.functions.end()) {
            return {Function(*this, nullptr, {}, 0), xerrors::NOT_FOUND};
        }

        uint32_t base = 0;
        if (const auto base_it = this->cfg.module.output_memory_bases.find(name);
            base_it != this->cfg.module.output_memory_bases.end()) {
            base = base_it->second;
        }

        return {Function(*this, fn, func_it->outputs, base), xerrors::NIL};
    }
};
}
