// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/runtime.h"

#include <cstring>

// WAMR includes
#include "wasm_export.h"

namespace arc {

// Static initialization flag
static bool g_wamr_initialized = false;

xerrors::Error Runtime::initialize_runtime() {
    if (g_wamr_initialized) {
        return xerrors::NIL;  // Already initialized
    }

    // Initialize WAMR runtime with default settings
    // For now, use simple initialization - we can add custom allocator later
    if (!wasm_runtime_init()) {
        return xerrors::Error("arc.runtime.init_failed");
    }

    g_wamr_initialized = true;
    return xerrors::NIL;
}

void Runtime::destroy_runtime() {
    if (g_wamr_initialized) {
        wasm_runtime_destroy();
        g_wamr_initialized = false;
    }
}

Runtime::~Runtime() {
    if (exec_env_ != nullptr) {
        wasm_runtime_destroy_exec_env(exec_env_);
        exec_env_ = nullptr;
    }

    if (module_inst_ != nullptr) {
        wasm_runtime_deinstantiate(module_inst_);
        module_inst_ = nullptr;
    }

    if (module_ != nullptr) {
        wasm_runtime_unload(module_);
        module_ = nullptr;
    }
}

Runtime::Runtime(Runtime &&other) noexcept
    : module_(other.module_),
      module_inst_(other.module_inst_),
      exec_env_(other.exec_env_),
      initialized_(other.initialized_) {
    other.module_ = nullptr;
    other.module_inst_ = nullptr;
    other.exec_env_ = nullptr;
    other.initialized_ = false;
}

Runtime &Runtime::operator=(Runtime &&other) noexcept {
    if (this != &other) {
        // Clean up existing resources
        this->~Runtime();

        // Move from other
        module_ = other.module_;
        module_inst_ = other.module_inst_;
        exec_env_ = other.exec_env_;
        initialized_ = other.initialized_;

        other.module_ = nullptr;
        other.module_inst_ = nullptr;
        other.exec_env_ = nullptr;
        other.initialized_ = false;
    }
    return *this;
}

xerrors::Error Runtime::load_aot_module(const std::vector<uint8_t> &aot_bytes) {
    if (module_ != nullptr) {
        return xerrors::Error("arc.runtime.already_loaded");
    }

    if (!g_wamr_initialized) {
        return xerrors::Error("arc.runtime.not_initialized",
                            "Call Runtime::initialize_runtime() first");
    }

    char error_buf[128];
    module_ = wasm_runtime_load(
        const_cast<uint8_t *>(aot_bytes.data()),
        static_cast<uint32_t>(aot_bytes.size()),
        error_buf,
        sizeof(error_buf)
    );

    if (module_ == nullptr) {
        return xerrors::Error("arc.runtime.load_failed", std::string(error_buf));
    }

    return xerrors::NIL;
}

xerrors::Error Runtime::instantiate(uint32_t stack_size, uint32_t heap_size) {
    if (module_ == nullptr) {
        return xerrors::Error("arc.runtime.no_module", "Call load_aot_module() first");
    }

    if (module_inst_ != nullptr) {
        return xerrors::Error("arc.runtime.already_instantiated");
    }

    char error_buf[128];
    module_inst_ = wasm_runtime_instantiate(
        module_,
        stack_size,
        heap_size,
        error_buf,
        sizeof(error_buf)
    );

    if (module_inst_ == nullptr) {
        return xerrors::Error("arc.runtime.instantiate_failed", std::string(error_buf));
    }

    // Create execution environment
    exec_env_ = wasm_runtime_create_exec_env(module_inst_, stack_size);
    if (exec_env_ == nullptr) {
        wasm_runtime_deinstantiate(module_inst_);
        module_inst_ = nullptr;
        return xerrors::Error("arc.runtime.exec_env_failed");
    }

    initialized_ = true;
    return xerrors::NIL;
}

void Runtime::set_user_data(void *user_data) {
    if (exec_env_ != nullptr) {
        wasm_runtime_set_user_data(exec_env_, user_data);
    }
}

std::pair<wasm_function_inst_t, xerrors::Error>
Runtime::find_function(const std::string &name) {
    if (module_inst_ == nullptr) {
        return {nullptr, xerrors::Error("arc.runtime.not_instantiated")};
    }

    wasm_function_inst_t func =
        wasm_runtime_lookup_function(module_inst_, name.c_str());

    if (func == nullptr) {
        return {nullptr,
                xerrors::Error("arc.runtime.function_not_found", "Function: " + name)};
    }

    return {func, xerrors::NIL};
}

xerrors::Error Runtime::call_function(wasm_function_inst_t func,
                                     std::span<const WasmValue> args,
                                     std::span<WasmValue> results) {
    if (exec_env_ == nullptr) {
        return xerrors::Error("arc.runtime.not_ready");
    }

    if (args.size() > MAX_ARGS) {
        return xerrors::Error("arc.runtime.too_many_args");
    }

    if (results.size() > MAX_RESULTS) {
        return xerrors::Error("arc.runtime.too_many_results");
    }

    // Convert WasmValue to WAMR's uint32 representation
    for (size_t i = 0; i < args.size(); i++) {
        switch (args[i].kind) {
        case WasmValue::Kind::I32:
            arg_buffer_[i] = static_cast<uint32_t>(args[i].i32);
            break;
        case WasmValue::Kind::I64:
            // 64-bit values use two slots
            std::memcpy(&arg_buffer_[i * 2], &args[i].i64, sizeof(int64_t));
            break;
        case WasmValue::Kind::F32:
            std::memcpy(&arg_buffer_[i], &args[i].f32, sizeof(float));
            break;
        case WasmValue::Kind::F64:
            std::memcpy(&arg_buffer_[i * 2], &args[i].f64, sizeof(double));
            break;
        }
    }

    // Call WASM function
    const bool success = wasm_runtime_call_wasm(
        exec_env_,
        func,
        static_cast<uint32_t>(args.size()),
        arg_buffer_.data()
    );

    if (!success) {
        const char *exception = wasm_runtime_get_exception(module_inst_);
        if (exception != nullptr) {
            return xerrors::Error("arc.runtime.wasm_trap", std::string(exception));
        }
        return xerrors::Error("arc.runtime.call_failed");
    }

    // Extract results (for now, assume single return value in arg_buffer_[0])
    // WAMR stores results in the same buffer as arguments
    if (!results.empty() && results.size() == 1) {
        // For now, assume i32 result (we'll handle other types later)
        results[0] = WasmValue(static_cast<int32_t>(arg_buffer_[0]));
    }

    // TODO: Use result_buffer_ for proper multi-value result handling
    (void)result_buffer_;  // Suppress unused warning for now

    return xerrors::NIL;
}

}  // namespace arc
